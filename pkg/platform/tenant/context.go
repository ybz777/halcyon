package tenant

import "context"

type organizationKey struct{}

// WithOrganizationID stores the authoritative organization tenant ID in context.
func WithOrganizationID(ctx context.Context, organizationID string) context.Context {
	return context.WithValue(ctx, organizationKey{}, organizationID)
}

// OrganizationIDFromContext returns the tenant organization ID when present.
func OrganizationIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(organizationKey{}).(string)
	return id, ok && id != ""
}
