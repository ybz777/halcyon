package middleware



import (

	"net/http"

	"strings"



	"github.com/go-chi/chi/v5/middleware"

	appauth "github.com/halcyon-io/halcyon/services/gateway/internal/application/auth"

	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"

	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"

	platformratelimit "github.com/halcyon-io/halcyon/pkg/platform/ratelimit"

)



// RateLimit enforces distributed multi-dimension rate limits.

func RateLimit(

	limiter platformratelimit.Limiter,

	policies platformratelimit.PolicySet,

	enabled bool,

	trustProxy bool,

	metrics *platformratelimit.Metrics,

	proxyMetrics ports.ProxyMetrics,

	security ports.SecurityEventPublisher,

) func(http.Handler) http.Handler {

	return platformratelimit.Middleware(platformratelimit.HTTPOptions{

		Limiter:    limiter,

		Enabled:    enabled,

		TrustProxy: trustProxy,

		Metrics:    metrics,

		Skip: func(r *http.Request) bool {

			path := strings.TrimSuffix(r.URL.Path, "/")

			return path == "/api/v1/health" || path == "/api/v1/ready"

		},

		Checks: func(r *http.Request) []platformratelimit.DimensionCheck {

			return buildChecks(r, policies, trustProxy)

		},

		OnRejected: func(r *http.Request, policy platformratelimit.Policy, _ platformratelimit.Decision) {

			class := string(appauth.ClassifyRoute(r.Method, r.URL.Path))

			if proxyMetrics != nil {

				proxyMetrics.RecordRateLimited(class)

			}

			if security != nil {

				reqID := middleware.GetReqID(r.Context())

				_ = security.PublishRateLimited(r.Context(), reqID, reqID, class, platformratelimit.ClientIP(r, trustProxy))

			}

		},

		RequestID: func(r *http.Request) string {

			return middleware.GetReqID(r.Context())

		},

	})

}



func buildChecks(r *http.Request, policies platformratelimit.PolicySet, trustProxy bool) []platformratelimit.DimensionCheck {

	class := appauth.ClassifyRoute(r.Method, r.URL.Path)

	policy := policyForClass(class, policies)

	ip := platformratelimit.ClientIP(r, trustProxy)

	checks := []platformratelimit.DimensionCheck{

		{Identity: platformratelimit.IdentityKey(platformratelimit.IdentityIP, ip), Policy: policy},

	}



	if principal, ok := PrincipalFromContext(r.Context()); ok && principal.IsAuthenticated() {

		switch principal.Type() {

		case domain.PrincipalTypeAPIKey:

			if id := principal.ID(); id != "" {

				checks = append(checks, platformratelimit.DimensionCheck{

					Identity: platformratelimit.IdentityKey(platformratelimit.IdentityAPIKey, id),

					Policy:   policies.API,

				})

			}

			if orgID := principal.OrganizationID(); orgID != "" {

				checks = append(checks, platformratelimit.DimensionCheck{

					Identity: platformratelimit.IdentityKey(platformratelimit.IdentityOrg, orgID),

					Policy:   policies.API,

				})

			}

		case domain.PrincipalTypeUser:

			if id := principal.UserID(); id != "" {

				checks = append(checks, platformratelimit.DimensionCheck{

					Identity: platformratelimit.IdentityKey(platformratelimit.IdentityUser, id),

					Policy:   policies.API,

				})

			}

		}

	}

	return checks

}



func policyForClass(class appauth.RouteClass, policies platformratelimit.PolicySet) platformratelimit.Policy {

	switch class {

	case appauth.RouteClassAuth:

		return policies.Auth

	case appauth.RouteClassWorkflow:

		return policies.Workflow

	case appauth.RouteClassInvitation:

		return policies.Public

	default:

		return policies.API

	}

}


