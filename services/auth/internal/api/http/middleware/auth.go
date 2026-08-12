package middleware

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwt"
)

// Authenticate validates bearer tokens and attaches the principal to context.
func Authenticate(validator *appauth.ValidateTokenUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := jwtinfra.ExtractBearerToken(r.Header.Get("Authorization"))
			if err != nil {
				writeUnauthorized(w, r, err)
				return
			}

			principal, err := validator.Execute(r.Context(), appauth.ValidateTokenInput{
				AccessToken: token.String(),
			})
			if err != nil {
				writeUnauthorized(w, r, err)
				return
			}

			ctx := WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermission enforces RBAC permission checks.
func RequirePermission(permission domain.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				writeForbidden(w, r)
				return
			}

			if !principal.HasPermission(permission) {
				writeForbidden(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, r *http.Request, err error) {
	detail := "authentication required"
	switch {
	case errors.Is(err, domain.ErrTokenExpired):
		detail = "token expired"
	case errors.Is(err, domain.ErrInvalidAccessToken):
		detail = "invalid access token"
	}

	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Type:     "https://halcyon.io/problems/unauthorized",
		Title:    "Unauthorized",
		Status:   http.StatusUnauthorized,
		Detail:   detail,
		Instance: middleware.GetReqID(r.Context()),
	})
}

func writeForbidden(w http.ResponseWriter, r *http.Request) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Type:     "https://halcyon.io/problems/forbidden",
		Title:    "Forbidden",
		Status:   http.StatusForbidden,
		Detail:   "insufficient permissions",
		Instance: middleware.GetReqID(r.Context()),
	})
}
