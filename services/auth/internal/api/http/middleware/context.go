package middleware

import (
	"context"

	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
)

type principalKey struct{}

// WithPrincipal stores the authenticated principal in context.
func WithPrincipal(ctx context.Context, principal appauth.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, principal)
}

// PrincipalFromContext returns the authenticated principal when present.
func PrincipalFromContext(ctx context.Context) (appauth.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(appauth.Principal)
	return principal, ok
}
