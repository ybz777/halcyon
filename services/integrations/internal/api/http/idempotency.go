package httpapi

import (
	"net/http"

	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	platformidempotency "github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	httpmiddleware "github.com/halcyon-io/halcyon/services/integrations/internal/api/http/middleware"
)

// IntegrationPolicies defines idempotency behavior for integrations routes.
var IntegrationPolicies = platformidempotency.PolicyMap{
	"POST /api/v1/webhook-endpoints": platformidempotency.Required,
}

// ResolveIntegrationScope builds idempotency scope from request context.
func ResolveIntegrationScope(r *http.Request) (platformidempotency.Scope, bool) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		return platformidempotency.Scope{}, false
	}
	principalType := "user"
	principalID := principal.UserID
	if identity, ok := platformauth.IdentityFromContext(r.Context()); ok && identity.Method == platformauth.MethodAPIKey {
		principalType = "apikey"
		principalID = identity.SubjectID
	}
	return platformidempotency.Scope{
		Service:        "integrations",
		PrincipalType:  principalType,
		PrincipalID:    principalID,
		OrganizationID: httpmiddleware.OrganizationFromRequest(r),
	}, true
}
