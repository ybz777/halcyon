package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	platformhealth "github.com/halcyon-io/halcyon/pkg/platform/health"
	platformtenant "github.com/halcyon-io/halcyon/pkg/platform/tenant"
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	httpmiddleware "github.com/halcyon-io/halcyon/services/integrations/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	jwtinfra "github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/security/jwt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RouterDeps struct {
	Logger         *slog.Logger
	ServiceName    string
	WebhookHandler *WebhookHandler
	Validator      *jwtinfra.Validator
	GatewayTrust   *platformauth.GatewayTrust
	APIEnabled     bool
	Idempotency    func(http.Handler) http.Handler
	Readiness      platformhealth.PoolReadiness
	Environment    platformconfig.Environment
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/metrics", promhttp.Handler().ServeHTTP)
	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
			httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": deps.ServiceName})
		})
		v1.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
			checks := map[string]string{"service": "ok"}
			for k, v := range deps.Readiness.Check(r.Context()) {
				checks[k] = v
			}
			status := http.StatusOK
			for _, v := range checks {
				if v != "ok" {
					status = http.StatusServiceUnavailable
					break
				}
			}
			httputil.WriteJSON(w, status, map[string]any{"status": checks})
		})
		if deps.APIEnabled && deps.Validator != nil && deps.WebhookHandler != nil {
			v1.Group(func(api chi.Router) {
				if deps.GatewayTrust != nil {
					api.Use(deps.GatewayTrust.Middleware())
				}
				api.Use(platformtenant.RequireGatewayForOrganizationHeaders(deps.GatewayTrust, deps.Environment))
				api.Use(httpmiddleware.Authenticate(httpmiddleware.AuthenticateDeps{
					Validator:    deps.Validator,
					GatewayTrust: deps.GatewayTrust,
				}))
				api.Use(httpmiddleware.RequireOrganization())
				if deps.Idempotency != nil {
					api.Use(deps.Idempotency)
				}
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsWrite)).
					Post("/webhook-endpoints", deps.WebhookHandler.Create)
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsRead)).
					Get("/webhook-endpoints", deps.WebhookHandler.List)
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsRead)).
					Get("/webhook-endpoints/{id}", deps.WebhookHandler.Get)
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsWrite)).
					Patch("/webhook-endpoints/{id}", deps.WebhookHandler.Patch)
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsWrite)).
					Post("/webhook-endpoints/{id}/enable", deps.WebhookHandler.Enable)
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsWrite)).
					Post("/webhook-endpoints/{id}/disable", deps.WebhookHandler.Disable)
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsWrite)).
					Post("/webhook-endpoints/{id}/test", deps.WebhookHandler.Test)
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsRead)).
					Get("/webhook-endpoints/{id}/deliveries", deps.WebhookHandler.ListDeliveries)
				api.With(httpmiddleware.RequirePermission(domain.PermissionIntegrationsRead)).
					Get("/webhook-deliveries/{delivery_id}", deps.WebhookHandler.GetDelivery)
			})
		}
	})
	return r
}
