package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	httpmiddleware "github.com/halcyon-io/halcyon/services/workflow/internal/api/http/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RouterDeps holds HTTP dependencies.
type RouterDeps struct {
	Logger          *slog.Logger
	ServiceName     string
	Readiness       ports.ReadinessChecker
	Metrics         *observability.HTTPMetrics
	IncidentHandler *IncidentHandler
	WorkflowHandler *WorkflowHandler
	Validator       ports.TokenValidator
	APIEnabled      bool
	Idempotency     func(http.Handler) http.Handler
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

		if deps.APIEnabled && deps.Validator != nil {
			v1.Group(func(api chi.Router) {
				api.Use(httpmiddleware.Authenticate(deps.Validator))
				if deps.Idempotency != nil {
					api.Use(deps.Idempotency)
				}

				if deps.IncidentHandler != nil {
					api.Post("/incidents", deps.IncidentHandler.CreateIncident)
					api.Get("/incidents", deps.IncidentHandler.ListIncidents)
					api.Get("/incidents/{id}", deps.IncidentHandler.GetIncident)
					api.Patch("/incidents/{id}/status", deps.IncidentHandler.UpdateStatus)
					api.Patch("/incidents/{id}/severity", deps.IncidentHandler.UpdateSeverity)
					api.Post("/incidents/{id}/resolve", deps.IncidentHandler.ResolveIncident)
					api.Post("/incidents/{id}/close", deps.IncidentHandler.CloseIncident)
				}

				if deps.WorkflowHandler != nil {
					api.Post("/workflows", deps.WorkflowHandler.CreateWorkflow)
					api.Get("/workflows", deps.WorkflowHandler.ListWorkflows)
					api.Get("/workflows/{workflow_id}", deps.WorkflowHandler.GetWorkflow)
					api.Post("/workflows/{workflow_id}/executions", deps.WorkflowHandler.StartExecution)
					api.Get("/workflow-executions/{execution_id}", deps.WorkflowHandler.GetExecution)
					api.Post("/workflow-executions/{execution_id}/cancel", deps.WorkflowHandler.CancelExecution)
				}
			})
		}
	})

	return r
}
