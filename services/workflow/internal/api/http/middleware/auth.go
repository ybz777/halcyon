package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
)

type principalKey struct{}

// WithPrincipal stores the authenticated principal in context.
func WithPrincipal(ctx context.Context, principal ports.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFromContext returns the authenticated principal.
func PrincipalFromContext(ctx context.Context) (ports.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(ports.Principal)
	return principal, ok
}

// Authenticate validates JWT bearer tokens.
func Authenticate(validator ports.TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := validator.Validate(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				slog.Warn("authentication failed",
					"request_id", middleware.GetReqID(r.Context()),
					"error", err,
				)
				http.Error(w, `{"title":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}
