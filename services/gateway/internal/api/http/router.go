package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	platformratelimit "github.com/halcyon-io/halcyon/pkg/platform/ratelimit"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	appauth "github.com/halcyon-io/halcyon/services/gateway/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
	gwmiddleware "github.com/halcyon-io/halcyon/services/gateway/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/gateway/internal/config"
	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/proxy"
	"github.com/halcyon-io/halcyon/services/gateway/internal/infrastructure/securityevents"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RouterDeps holds router dependencies.
type RouterDeps struct {
	Logger             *slog.Logger
	ServiceName        string
	Config             config.Config
	Readiness          ports.ReadinessChecker
	Metrics            *observability.HTTPMetrics
	ProxyMetrics       ports.ProxyMetrics
	Validator          ports.TokenValidator
	APIKeys            ports.APIKeyAuthenticator
	RateLimiter        platformratelimit.Limiter
	RateLimitMetrics   *platformratelimit.Metrics
	RateLimitPolicy    platformratelimit.PolicySet
	ConcurrencyLimiter *platformratelimit.ConcurrencyLimiter
	Security           ports.SecurityEventPublisher
	Upstreams          map[string]*proxy.Upstream
}

// NewRouter builds the gateway HTTP router.
func NewRouter(deps RouterDeps) http.Handler {
	if deps.Security == nil {
		deps.Security = securityevents.NewPublisher(nil)
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(gwmiddleware.RequestID)
	r.Use(gwmiddleware.Correlation)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(gwmiddleware.SecurityHeaders(deps.Config.SecurityHeaders))
	bodyOpts := platformratelimit.BodyLimitOptions{
		MaxBytes: deps.Config.MaxRequestBodyBytes,
		Metrics:  deps.RateLimitMetrics,
		RequestID: func(r *http.Request) string {
			return middleware.GetReqID(r.Context())
		},
	}
	r.Use(platformratelimit.BodyLimitMiddleware(bodyOpts))
	r.Use(platformratelimit.RecoverBodyLimitMiddleware(bodyOpts))
	r.Use(requestLogger(deps.Logger))
	r.Use(middleware.Timeout(60 * time.Second))

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

		v1.Group(func(api chi.Router) {
			api.Use(gwmiddleware.Authenticate(deps.Validator, deps.APIKeys, deps.ProxyMetrics, deps.Security))
			api.Use(gwmiddleware.Authorize())
			api.Use(gwmiddleware.RateLimit(
				deps.RateLimiter,
				deps.RateLimitPolicy,
				deps.Config.RateLimitEnabled,
				deps.Config.RateLimitTrustProxy,
				deps.RateLimitMetrics,
				deps.ProxyMetrics,
				deps.Security,
			))
			api.Use(platformratelimit.ConcurrencyMiddleware(
				deps.ConcurrencyLimiter,
				deps.Config.RateLimitEnabled,
				func(r *http.Request) (string, bool) {
					class := appauth.ClassifyRoute(r.Method, r.URL.Path)
					if class != appauth.RouteClassWorkflow {
						return "", false
					}
					if principal, ok := gwmiddleware.PrincipalFromContext(r.Context()); ok && principal.OrganizationID() != "" {
						return platformratelimit.IdentityKey(platformratelimit.IdentityOrg, principal.OrganizationID()), true
					}
					return platformratelimit.IdentityKey(platformratelimit.IdentityIP, platformratelimit.ClientIP(r, deps.Config.RateLimitTrustProxy)), true
				},
				deps.RateLimitMetrics,
				func(r *http.Request) string { return middleware.GetReqID(r.Context()) },
			))
			api.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
				proxyRequest(w, r, deps.Upstreams, deps.ProxyMetrics)
			})
		})
	})

	return r
}

func proxyRequest(w http.ResponseWriter, r *http.Request, upstreams map[string]*proxy.Upstream, metrics ports.ProxyMetrics) {
	service := appauth.UpstreamService(r.URL.Path)
	if service == "" {
		http.NotFound(w, r)
		return
	}
	upstream, ok := upstreams[service]
	if !ok || upstream == nil {
		if metrics != nil {
			metrics.RecordUpstreamError(service, "not_configured")
		}
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	if strings.Contains(r.URL.Path, "..") {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	upstream.ServeHTTP(w, r)
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
			logger.Info("gateway request completed",
				"method", r.Method,
				"route", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
