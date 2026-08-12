package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	apporg "github.com/halcyon-io/halcyon/services/organization/internal/application/organization"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	httpmiddleware "github.com/halcyon-io/halcyon/services/organization/internal/api/http/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)
type RouterDeps struct {
	Logger      *slog.Logger
	ServiceName string
	Readiness   ports.ReadinessChecker
	Metrics     *observability.HTTPMetrics
	Handler     *OrganizationHandler
	Validator   ports.TokenValidator
	Resolver    httpmiddleware.OrganizationResolver
	Authorizer  *apporg.Authorizer
	APIEnabled  bool
	Idempotency func(http.Handler) http.Handler
}

// NewRouter builds the HTTP router.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(deps.Logger))
	r.Use(middleware.Timeout(30 * time.Second))

	health := NewHealthHandler(deps.ServiceName, deps.Readiness)

	r.Group(func(r chi.Router) {
		if deps.Metrics != nil {
			r.Use(deps.Metrics.Middleware("/metrics"))
		}
		r.Get("/metrics", promhttp.Handler().ServeHTTP)
	})

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/health", health.Health)
		v1.Get("/ready", health.Ready)

		if deps.APIEnabled && deps.Handler != nil && deps.Validator != nil {
			v1.Group(func(api chi.Router) {
				api.Use(httpmiddleware.Authenticate(deps.Validator))
				if deps.Idempotency != nil {
					api.Use(deps.Idempotency)
				}

				api.Post("/organizations", deps.Handler.CreateOrganization)

				api.Route("/organizations/{organization_id}", func(org chi.Router) {
					org.Use(httpmiddleware.OrganizationContext(deps.Resolver))
					org.Get("/", deps.Handler.GetOrganization)
					org.Patch("/", deps.Handler.UpdateOrganization)
					org.Delete("/", deps.Handler.DeleteOrganization)
					org.Get("/members", deps.Handler.ListMembers)
					org.Patch("/members/{user_id}", deps.Handler.UpdateMember)
					org.Delete("/members/{user_id}", deps.Handler.RemoveMember)
					org.Post("/invitations", deps.Handler.CreateInvitation)
					org.Get("/invitations", deps.Handler.ListInvitations)
					org.Delete("/invitations/{id}", deps.Handler.RevokeInvitation)
				})

				api.Post("/invitations/{token}/accept", deps.Handler.AcceptInvitation)
			})
		}
	})

	return r
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("request completed",
				"method", r.Method, "path", r.URL.Path, "status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
