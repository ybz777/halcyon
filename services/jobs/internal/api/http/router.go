package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	platformtenant "github.com/halcyon-io/halcyon/pkg/platform/tenant"
	httpmiddleware "github.com/halcyon-io/halcyon/services/jobs/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/jobs/internal/application/ports"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RouterDeps holds HTTP dependencies.
type RouterDeps struct {
	Logger      *slog.Logger
	ServiceName string
	Readiness   ports.ReadinessChecker
	Metrics     *observability.HTTPMetrics
	JobHandler  *JobHandler
	Validator   ports.TokenValidator
	GatewayTrust *platformauth.GatewayTrust
	Environment  platformconfig.Environment
	APIEnabled  bool
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
		if deps.APIEnabled && deps.Validator != nil && deps.JobHandler != nil {
			v1.Group(func(api chi.Router) {
				if deps.GatewayTrust != nil {
					api.Use(deps.GatewayTrust.Middleware())
				}
				api.Use(platformtenant.RequireGatewayForOrganizationHeaders(deps.GatewayTrust, deps.Environment))
				api.Use(httpmiddleware.Authenticate(deps.Validator))
				api.Use(httpmiddleware.BlockClientOrgOverride())
				api.Get("/jobs", deps.JobHandler.List)
				api.Get("/jobs/{id}", deps.JobHandler.Get)
				api.Post("/jobs/{id}/cancel", deps.JobHandler.Cancel)
				api.Get("/jobs/{id}/attempts", deps.JobHandler.ListAttempts)
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
