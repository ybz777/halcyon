package tenant

import (
	"net/http"
	"strings"

	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
)

// HasOrganizationHeader reports whether the request carries tenant scope headers.
func HasOrganizationHeader(r *http.Request) bool {
	if r == nil {
		return false
	}
	if strings.TrimSpace(r.Header.Get("X-Halcyon-Organization-ID")) != "" {
		return true
	}
	return strings.TrimSpace(r.Header.Get("X-Organization-ID")) != ""
}

// OrganizationFromRequest returns the organization ID from trusted internal headers.
func OrganizationFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if header := strings.TrimSpace(r.Header.Get("X-Halcyon-Organization-ID")); header != "" {
		return header
	}
	return strings.TrimSpace(r.Header.Get("X-Organization-ID"))
}

// RequireGatewayForOrganizationHeaders rejects organization headers in production unless
// the gateway service token attests the request.
func RequireGatewayForOrganizationHeaders(trust *platformauth.GatewayTrust, env platformconfig.Environment) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasOrganizationHeader(r) {
				next.ServeHTTP(w, r)
				return
			}
			if env != platformconfig.EnvProduction {
				next.ServeHTTP(w, r)
				return
			}
			if trust == nil {
				platformauth.WriteUnauthorized(w, "organization context must be gateway-attested")
				return
			}
			if strings.TrimSpace(r.Header.Get(servicetoken.HeaderName)) == "" {
				platformauth.WriteUnauthorized(w, "organization context must be gateway-attested")
				return
			}
			if _, err := trust.ValidateRequest(r); err != nil {
				platformauth.WriteUnauthorized(w, "organization context must be gateway-attested")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
