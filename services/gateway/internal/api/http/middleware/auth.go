package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	appauth "github.com/halcyon-io/halcyon/services/gateway/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/jwt"
)

type principalKey struct{}

// WithPrincipal stores the authenticated principal in context.
func WithPrincipal(ctx context.Context, principal domain.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFromContext returns the authenticated principal.
func PrincipalFromContext(ctx context.Context) (domain.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(domain.Principal)
	return principal, ok
}

// Authenticate validates JWT or passes API keys through for downstream validation.
func Authenticate(validator ports.TokenValidator, apiKeys ports.APIKeyAuthenticator, metrics ports.ProxyMetrics, security ports.SecurityEventPublisher) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			class := appauth.ClassifyRoute(r.Method, r.URL.Path)
			if !appauth.RequiresAuthentication(class) {
				next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), domain.NewAnonymousPrincipal())))
				return
			}

			token, err := jwtinfra.ExtractBearer(r.Header.Get("Authorization"))
			if err != nil {
				if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
					if principal, err := authenticateAPIKey(r, apiKeys, apiKey); err == nil {
						next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
						return
					}
					writeAuthFailure(w, r, metrics, security, "invalid_api_key")
					return
				}
				writeAuthFailure(w, r, metrics, security, "missing_token")
				return
			}

			if jwtinfra.IsAPIKeyToken(token) {
				if principal, err := authenticateAPIKey(r, apiKeys, token); err == nil {
					next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
					return
				}
				writeAuthFailure(w, r, metrics, security, "invalid_api_key")
				return
			}

			claims, err := validator.Validate(r.Context(), token)
			if err != nil {
				reason := "invalid_token"
				if errors.Is(err, domain.ErrTokenExpired) {
					reason = "expired_token"
				}
				writeAuthFailure(w, r, metrics, security, reason)
				return
			}

			principal := domain.NewUserPrincipal(claims.UserID, claims.Email, claims.Roles, claims.Permissions)
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

// Authorize enforces route-level permissions declared by the gateway policy.
func Authorize() func(http.Handler) http.Handler {
	authz := platformauth.StaticAuthorizer{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			required, ok := appauth.RequiredPermission(r.Method, r.URL.Path)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			principal, ok := PrincipalFromContext(r.Context())
			if !ok || !principal.IsAuthenticated() {
				writeAuthFailure(w, r, nil, nil, "missing_token")
				return
			}

			if err := platformauth.RequirePermission(authz, principal.Identity(), required.String()); err != nil {
				writeForbidden(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func authenticateAPIKey(r *http.Request, apiKeys ports.APIKeyAuthenticator, rawKey string) (domain.Principal, error) {
	if apiKeys == nil {
		return domain.Principal{}, domain.ErrUnauthorized
	}
	claims, err := apiKeys.Authenticate(r.Context(), rawKey)
	if err != nil {
		return domain.Principal{}, err
	}
	return domain.NewAPIKeyPrincipal(claims.APIKeyID, claims.OrganizationID, claims.Scopes), nil
}

func writeAuthFailure(w http.ResponseWriter, r *http.Request, metrics ports.ProxyMetrics, security ports.SecurityEventPublisher, reason string) {
	if metrics != nil {
		metrics.RecordAuthFailure(reason)
	}
	if security != nil && (reason == "invalid_token" || reason == "expired_token" || reason == "missing_token") {
		reqID := middleware.GetReqID(r.Context())
		_ = security.PublishInvalidToken(r.Context(), reqID, reqID, reason, r.RemoteAddr, r.UserAgent())
	}
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Type:     "https://halcyon.io/problems/unauthorized",
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   "authentication required",
		Instance: middleware.GetReqID(r.Context()),
	})
}

func writeForbidden(w http.ResponseWriter, r *http.Request) {
	platformauth.WriteForbidden(w, "insufficient permissions")
	_ = r
}
