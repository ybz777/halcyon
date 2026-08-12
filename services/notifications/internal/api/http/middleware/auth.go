package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	jwtinfra "github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/security/jwt"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

type principalKey struct{}

type TokenValidator interface {
	Validate(ctx context.Context, bearerToken string) (jwtinfra.Principal, error)
}

func WithPrincipal(ctx context.Context, principal jwtinfra.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (jwtinfra.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(jwtinfra.Principal)
	return principal, ok
}

func Authenticate(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := validator.Validate(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Unauthorized", Status: http.StatusUnauthorized})
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
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

func RequireSelf() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok || principal.UserID == "" {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Unauthorized", Status: http.StatusUnauthorized})
				return
			}
			_ = domain.ErrForbidden
			next.ServeHTTP(w, r)
		})
	}
}
