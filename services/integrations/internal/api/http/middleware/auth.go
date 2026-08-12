package middleware

import (
	"context"
	"net/http"
	"strings"

	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	appwebhook "github.com/halcyon-io/halcyon/services/integrations/internal/application/webhook"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
)

type principalKey struct{}

type TokenValidator interface {
	Validate(ctx context.Context, bearerToken string) (appwebhook.Principal, error)
}

func WithPrincipal(ctx context.Context, principal appwebhook.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (appwebhook.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(appwebhook.Principal)
	return principal, ok
}

type AuthenticateDeps struct {
	Validator    TokenValidator
	GatewayTrust *platformauth.GatewayTrust
}

func Authenticate(deps AuthenticateDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKeyID := strings.TrimSpace(r.Header.Get("X-Halcyon-API-Key-ID")); apiKeyID != "" {
				if deps.GatewayTrust == nil {
					platformauth.WriteUnauthorized(w, "authentication required")
					return
				}
				if _, err := deps.GatewayTrust.ValidateRequest(r); err != nil {
					platformauth.WriteUnauthorized(w, "authentication required")
					return
				}
				principal := appwebhook.Principal{
					UserID:      apiKeyID,
					Permissions: permissionsFromScopes(r.Header.Get("X-Halcyon-API-Key-Scopes")),
				}
				ctx := platformauth.WithIdentity(r.Context(), platformauth.Identity{
					SubjectID:     apiKeyID,
					PrincipalType: platformauth.PrincipalService,
					Method:        platformauth.MethodAPIKey,
					Permissions:   mapKeys(principal.Permissions),
				})
				next.ServeHTTP(w, r.WithContext(WithPrincipal(ctx, principal)))
				return
			}
			if deps.Validator == nil {
				platformauth.WriteUnauthorized(w, "authentication required")
				return
			}
			principal, err := deps.Validator.Validate(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				platformauth.WriteUnauthorized(w, "authentication required")
				return
			}
			ctx := platformauth.WithIdentity(r.Context(), platformauth.Identity{
				SubjectID:     principal.UserID,
				PrincipalType: platformauth.PrincipalUser,
				Method:        platformauth.MethodBearerJWT,
			})
			next.ServeHTTP(w, r.WithContext(WithPrincipal(ctx, principal)))
		})
	}
}

func mapKeys(perms map[string]struct{}) []string {
	out := make([]string, 0, len(perms))
	for perm := range perms {
		out = append(out, perm)
	}
	return out
}

func permissionsFromScopes(raw string) map[string]struct{} {
	perms := make(map[string]struct{})
	for _, scope := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(scope)) {
		case "integrations.read":
			perms[domain.PermissionIntegrationsRead] = struct{}{}
		case "integrations.write":
			perms[domain.PermissionIntegrationsWrite] = struct{}{}
		case "admin":
			perms[domain.PermissionIntegrationsRead] = struct{}{}
			perms[domain.PermissionIntegrationsWrite] = struct{}{}
		}
	}
	return perms
}

func OrganizationFromRequest(r *http.Request) string {
	if header := strings.TrimSpace(r.Header.Get("X-Halcyon-Organization-ID")); header != "" {
		return header
	}
	return strings.TrimSpace(r.Header.Get("X-Organization-ID"))
}

func RequireOrganization() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if OrganizationFromRequest(r) == "" {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Bad Request", Status: http.StatusBadRequest, Detail: "organization context required"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok || !principal.HasPermission(permission) {
				platformauth.WriteForbidden(w, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
