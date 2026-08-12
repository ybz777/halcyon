package httpapi

import (
	"net/http"

	platformidempotency "github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	httpmiddleware "github.com/halcyon-io/halcyon/services/organization/internal/api/http/middleware"
)

// OrganizationPolicies defines idempotency behavior for organization routes.
var OrganizationPolicies = platformidempotency.PolicyMap{
	"POST /api/v1/organizations":                             platformidempotency.Required,
	"POST /organizations":                                    platformidempotency.Required,
	"POST /api/v1/organizations/{organization_id}/invitations": platformidempotency.Required,
	"POST /organizations/{organization_id}/invitations":        platformidempotency.Required,
	"POST /api/v1/invitations/{token}/accept":              platformidempotency.Required,
	"POST /invitations/{token}/accept":                       platformidempotency.Required,
}

// ResolveOrganizationScope builds idempotency scope from request context.
func ResolveOrganizationScope(r *http.Request) (platformidempotency.Scope, bool) {
	user, ok := httpmiddleware.AuthenticatedUserFromContext(r.Context())
	if !ok {
		return platformidempotency.Scope{}, false
	}
	scope := platformidempotency.Scope{
		Service:       "organization",
		PrincipalType: "user",
		PrincipalID:   user.UserID.String(),
	}
	if principal, ok := httpmiddleware.PrincipalFromContext(r.Context()); ok {
		scope.OrganizationID = principal.OrganizationID().String()
	}
	return scope, true
}
