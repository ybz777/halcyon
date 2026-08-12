package auth

import (
	"net/http"
	"strings"
)

// RouteClass categorizes routes for rate limiting and auth policy.
type RouteClass string

const (
	RouteClassPublic       RouteClass = "public"
	RouteClassAuth         RouteClass = "auth"
	RouteClassInvitation   RouteClass = "invitation"
	RouteClassWorkflow     RouteClass = "workflow"
	RouteClassProtected    RouteClass = "protected"
)

// ClassifyRoute determines route class from the request path.
func ClassifyRoute(method, path string) RouteClass {
	path = strings.TrimSuffix(path, "/")

	if path == "/api/v1/health" ||
		path == "/api/v1/ready" ||
		path == "/metrics" {
		return RouteClassPublic
	}

	// Public authentication endpoints.
	// These endpoints must be reachable without an existing credential.
	if strings.HasPrefix(path, "/api/v1/auth/login") ||
		strings.HasPrefix(path, "/api/v1/auth/register") ||
		strings.HasPrefix(path, "/api/v1/auth/refresh") ||
		strings.HasPrefix(path, "/api/v1/auth/password-reset") ||
		strings.HasPrefix(path, "/api/v1/auth/verify") {
		return RouteClassAuth
	}

	if strings.Contains(path, "/executions") &&
		strings.HasPrefix(path, "/api/v1/workflows") {
		return RouteClassWorkflow
	}

	if strings.HasPrefix(path, "/api/v1/workflow-executions/") &&
		strings.HasSuffix(path, "/cancel") {
		return RouteClassWorkflow
	}

	if strings.HasPrefix(path, "/api/v1/jobs") {
		return RouteClassWorkflow
	}

	if strings.HasPrefix(path, "/api/v1/auth/jwks") {
		return RouteClassPublic
	}

	if method == http.MethodPost &&
		strings.HasPrefix(path, "/api/v1/invitations/") &&
		strings.HasSuffix(path, "/accept") {
		return RouteClassInvitation
	}

	if strings.HasPrefix(path, "/api/v1/auth/") ||
		strings.HasPrefix(path, "/api/v1/organizations/") ||
		strings.HasPrefix(path, "/api/v1/invitations/") {
		return RouteClassProtected
	}

	return RouteClassProtected
}

// RequiresAuthentication reports whether the route requires a valid credential.
func RequiresAuthentication(class RouteClass) bool {
	switch class {
	case RouteClassPublic, RouteClassAuth:
		return false
	default:
		return true
	}
}

// UpstreamService resolves the configured upstream for a path.
func UpstreamService(path string) string {
	if strings.HasPrefix(path, "/api/v1/auth/") {
		return "auth"
	}

	if strings.HasPrefix(path, "/api/v1/audit-events") {
		return "audit"
	}

	if strings.HasPrefix(path, "/api/v1/jobs") {
		return "jobs"
	}

	if strings.HasPrefix(path, "/api/v1/webhook-endpoints") ||
		strings.HasPrefix(path, "/api/v1/webhook-deliveries") {
		return "integrations"
	}

	if strings.HasPrefix(path, "/api/v1/notifications") ||
		strings.HasPrefix(path, "/api/v1/notification-preferences") {
		return "notifications"
	}

	if strings.HasPrefix(path, "/api/v1/auth/api-keys") ||
		strings.HasPrefix(path, "/api/v1/api-keys") {
		return "auth"
	}

	if path == "/api/v1/organizations" ||
		strings.HasPrefix(path, "/api/v1/organizations/") ||
		strings.HasPrefix(path, "/api/v1/invitations/") {
		return "organization"
	}

	return ""
}