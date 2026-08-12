package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	httpmiddleware "github.com/halcyon-io/halcyon/services/auth/internal/api/http/middleware"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// RouterDeps holds dependencies for HTTP routing.
type RouterDeps struct {
	Logger        *slog.Logger
	ServiceName   string
	Readiness     ports.ReadinessChecker
	Metrics       *observability.HTTPMetrics
	Login         *LoginHandler
	Register      *RegisterHandler
	Refresh       *RefreshHandler
	Auth          *AuthHandler
	APIKeys       *APIKeyHandler
	Introspect    *IntrospectAPIKeyHandler
	ValidateToken *appauth.ValidateTokenUseCase
	JWKS          http.Handler
	AuthEnabled   bool
	Idempotency   func(http.Handler) http.Handler
}

// NewRouter builds the HTTP router with versioned API routes and observability.
func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(httpmiddleware.Correlation)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(deps.Logger))
	r.Use(middleware.Timeout(30 * time.Second))

	health := NewHealthHandler(deps.ServiceName, deps.Readiness)

	metrics := deps.Metrics

	r.Group(func(r chi.Router) {
		if metrics != nil {
			r.Use(metrics.Middleware("/metrics"))
		}

		r.Get("/metrics", promhttp.Handler().ServeHTTP)
	})

	r.Route("/api/v1", func(v1 chi.Router) {
		v1.Group(func(r chi.Router) {
			if metrics != nil {
				r.Use(metrics.Middleware("/api/v1/health"))
			}

			r.Get("/health", health.Health)
		})

		v1.Group(func(r chi.Router) {
			if metrics != nil {
				r.Use(metrics.Middleware("/api/v1/ready"))
			}

			r.Get("/ready", health.Ready)
		})

		if deps.AuthEnabled {
			v1.Route("/auth", func(auth chi.Router) {
				if metrics != nil {
					auth.Use(metrics.Middleware("/api/v1/auth"))
				}

				if deps.JWKS != nil {
					auth.Get("/jwks", deps.JWKS.ServeHTTP)
				}

				if deps.Login != nil {
					auth.Post("/login", deps.Login.Login)
				}

				if deps.Register != nil {
					auth.Post("/register", deps.Register.Register)
				}

				if deps.Refresh != nil {
					auth.Post("/refresh", deps.Refresh.Refresh)
				}

				if deps.Introspect != nil {
					auth.Post("/api-keys/introspect", deps.Introspect.Introspect)
				}

				if deps.Auth != nil {
					auth.Post("/logout", deps.Auth.Logout)

					auth.Group(func(protected chi.Router) {
						if deps.ValidateToken != nil {
							protected.Use(
								httpmiddleware.Authenticate(deps.ValidateToken),
							)
						}

						if deps.Idempotency != nil {
							protected.Use(deps.Idempotency)
						}

						protected.With(
							httpmiddleware.RequirePermission(
								domain.PermissionProfileRead,
							),
						).Get("/me", deps.Auth.Me)

						if deps.APIKeys != nil {
							protected.With(
								httpmiddleware.RequirePermission(
									domain.PermissionAPIKeyManage,
								),
							).Route("/api-keys", func(keys chi.Router) {
								mountAPIKeyRoutes(keys, deps.APIKeys)
							})
						}
					})
				}
			})

			if deps.APIKeys != nil && deps.ValidateToken != nil {
				v1.Group(func(protected chi.Router) {
					protected.Use(
						httpmiddleware.Authenticate(deps.ValidateToken),
					)

					if deps.Idempotency != nil {
						protected.Use(deps.Idempotency)
					}

					protected.With(
						httpmiddleware.RequirePermission(
							domain.PermissionAPIKeyManage,
						),
					).Route("/api-keys", func(keys chi.Router) {
						mountAPIKeyRoutes(keys, deps.APIKeys)
					})
				})
			}
		}
	})

	return r
}

func mountAPIKeyRoutes(keys chi.Router, handler *APIKeyHandler) {
	keys.Post("/", handler.Create)
	keys.Get("/", handler.List)
	keys.Get("/{id}", handler.Get)
	keys.Post("/{id}/revoke", handler.RevokePost)
	keys.Delete("/{id}", handler.Revoke)
	keys.Post("/{id}/rotate", handler.Rotate)
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

			logger.Info(
				"request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}