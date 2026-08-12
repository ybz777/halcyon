package httpapi

import (
	"net/http"

	platformidempotency "github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	httpmiddleware "github.com/halcyon-io/halcyon/services/auth/internal/api/http/middleware"
)

// AuthPolicies defines idempotency behavior for auth mutation routes.
var AuthPolicies = platformidempotency.PolicyMap{
	"POST /api/v1/auth/api-keys":              platformidempotency.Required,
	"POST /api/v1/auth/api-keys/":             platformidempotency.Required,
	"POST /api/v1/api-keys":                   platformidempotency.Required,
	"POST /api/v1/api-keys/":                  platformidempotency.Required,
	"POST /api/v1/auth/api-keys/{id}/rotate":  platformidempotency.Required,
	"POST /api/v1/api-keys/{id}/rotate":       platformidempotency.Required,
}

// ResolveAuthScope builds idempotency scope from authenticated user context.
func ResolveAuthScope(r *http.Request) (platformidempotency.Scope, bool) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		return platformidempotency.Scope{}, false
	}
	return platformidempotency.Scope{
		Service:       "auth",
		PrincipalType: "user",
		PrincipalID:   principal.UserID.String(),
	}, true
}
