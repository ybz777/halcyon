package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/services/jobs/internal/application/ports"
)

type principalKey struct{}

func WithPrincipal(ctx context.Context, principal ports.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (ports.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(ports.Principal)
	return principal, ok
}

func Authenticate(validator ports.TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, err := validator.Validate(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				slog.Warn("authentication failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Unauthorized", Status: http.StatusUnauthorized, Detail: "authentication required"})
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

func BlockClientOrgOverride() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			requested := strings.TrimSpace(r.URL.Query().Get("organization_id"))
			trusted := OrganizationFromRequest(r)
			if requested != "" && requested != trusted && !principal.HasPermission(ports.PermissionJobsReadPlatform) {
				httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: "Forbidden", Status: http.StatusForbidden, Detail: "cross-tenant access denied"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
