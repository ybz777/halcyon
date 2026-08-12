package auth

import "context"

type identityKey struct{}
type serviceCallerKey struct{}

// WithIdentity stores a normalized identity in context.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

// IdentityFromContext returns the normalized identity when present.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}

// WithServiceCaller records the validated gateway/service identity for internal hops.
func WithServiceCaller(ctx context.Context, serviceID string) context.Context {
	return context.WithValue(ctx, serviceCallerKey{}, serviceID)
}

// ServiceCallerFromContext returns the validated internal service caller ID.
func ServiceCallerFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(serviceCallerKey{}).(string)
	return id, ok
}
