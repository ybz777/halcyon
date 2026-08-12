package httpapi

import (
	"net/http"
	"strings"

	platformidempotency "github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	httpmiddleware "github.com/halcyon-io/halcyon/services/workflow/internal/api/http/middleware"
)

// WorkflowPolicies defines idempotency behavior for workflow routes.
var WorkflowPolicies = platformidempotency.PolicyMap{
	"POST /api/v1/incidents":   platformidempotency.Required,
	"POST /api/v1/workflows":   platformidempotency.Required,
	"POST /incidents":          platformidempotency.Required,
	"POST /workflows":          platformidempotency.Required,
	// StartExecution uses domain-level Postgres idempotency; HTTP replay disabled to avoid double-claim.
	"POST /api/v1/workflows/{workflow_id}/executions": platformidempotency.Disabled,
	"POST /workflows/{workflow_id}/executions":          platformidempotency.Disabled,
}

// ResolveWorkflowScope builds idempotency scope from request context.
func ResolveWorkflowScope(r *http.Request) (platformidempotency.Scope, bool) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		return platformidempotency.Scope{}, false
	}
	return platformidempotency.Scope{
		Service:       "workflow",
		PrincipalType: "user",
		PrincipalID:   principal.UserID.String(),
		OrganizationID: organizationIDFromHeader(r),
	}, true
}

func organizationIDFromHeader(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Halcyon-Organization-ID")); v != "" {
		return v
	}
	return strings.TrimSpace(r.Header.Get("X-Organization-ID"))
}
