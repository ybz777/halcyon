package contextx

import "context"

type correlationKey struct{}
type causationKey struct{}

// WithCorrelationID stores a correlation identifier in context.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey{}, id)
}

// CorrelationIDFromContext returns the correlation identifier when present.
func CorrelationIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(correlationKey{}).(string)
	return id
}

// WithCausationID stores a causation identifier in context.
func WithCausationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, causationKey{}, id)
}

// CausationIDFromContext returns the causation identifier when present.
func CausationIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(causationKey{}).(string)
	return id
}
