package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	platformhealth "github.com/halcyon-io/halcyon/pkg/platform/health"
	platformtenant "github.com/halcyon-io/halcyon/pkg/platform/tenant"
	httpmiddleware "github.com/halcyon-io/halcyon/services/notifications/internal/api/http/middleware"
	jwtinfra "github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/security/jwt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type RouterDeps struct {
	Logger              *slog.Logger
	ServiceName         string
	NotificationHandler *NotificationHandler
	PreferenceHandler   *PreferenceHandler
	Validator           *jwtinfra.Validator
	GatewayTrust        *platformauth.GatewayTrust
	APIEnabled          bool
	Readiness           platformhealth.PoolReadiness
	Environment         platformconfig.Environment
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
		if deps.APIEnabled && deps.Validator != nil {
			v1.Group(func(api chi.Router) {
				if deps.GatewayTrust != nil {
					api.Use(deps.GatewayTrust.Middleware())
				}
				api.Use(platformtenant.RequireGatewayForOrganizationHeaders(deps.GatewayTrust, deps.Environment))
				api.Use(httpmiddleware.Authenticate(deps.Validator))
				api.Use(httpmiddleware.RequireOrganization())
				api.Use(httpmiddleware.RequireSelf())
				if deps.NotificationHandler != nil {
					api.Get("/notifications", deps.NotificationHandler.List)
					api.Get("/notifications/unread-count", deps.NotificationHandler.UnreadCount)
					api.Get("/notifications/{id}", deps.NotificationHandler.Get)
					api.Post("/notifications/{id}/read", deps.NotificationHandler.MarkRead)
					api.Post("/notifications/read-all", deps.NotificationHandler.MarkAllRead)
				}
				if deps.PreferenceHandler != nil {
					api.Get("/notification-preferences", deps.PreferenceHandler.List)
					api.Put("/notification-preferences/{notification_type}", deps.PreferenceHandler.Update)
				}
			})
		}
	})
	return r
}
