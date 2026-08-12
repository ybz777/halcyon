package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	platformhealth "github.com/halcyon-io/halcyon/pkg/platform/health"
	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	platformidempotency "github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
	httpapi "github.com/halcyon-io/halcyon/services/integrations/internal/api/http"
	appwebhook "github.com/halcyon-io/halcyon/services/integrations/internal/application/webhook"
	"github.com/halcyon-io/halcyon/services/integrations/internal/config"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/crypto"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/postgres"
	jwtinfra "github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/security/jwt"
	"github.com/redis/go-redis/v9"
)

func main() {
	if err := run(); err != nil {
		slog.Error("integrations failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := observability.NewLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)
	ctx := context.Background()

	var handler *httpapi.WebhookHandler
	var validator *jwtinfra.Validator
	var gatewayTrust *platformauth.GatewayTrust
	var readiness platformhealth.PoolReadiness
	apiEnabled := false

	if cfg.DatabaseURL != "" {
		if cfg.DatabaseMigrate {
			if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
				return fmt.Errorf("migrations: %w", err)
			}
			if err := migrations.Run(cfg.DatabaseURL); err != nil {
				return fmt.Errorf("platform migrations: %w", err)
			}
		}
		pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns, cfg.DatabaseMinConns)
		if err != nil {
			return err
		}
		defer pool.Close()
		readiness = platformhealth.PoolReadiness{Pool: pool}

		cipher, err := crypto.NewSecretCipher(cfg.EncryptionKey)
		if err != nil {
			return err
		}
		endpoints := postgres.NewEndpointRepository(pool)
		deliveries := postgres.NewDeliveryRepository(pool)
		service := appwebhook.NewService(endpoints, deliveries, cipher, cfg.AllowHTTPWebhooks, cfg.WebhookTimeout)
		handler = httpapi.NewWebhookHandler(service)

		validator, err = jwtinfra.NewValidator([]byte(cfg.JWTSecret), cfg.JWTIssuer, cfg.JWTAudience)
		if err != nil {
			return err
		}
		if cfg.ServiceAuth.PublicKeyPEM.IsSet() {
			serviceValidator, err := servicetoken.NewValidator(
				cfg.ServiceAuth.PublicKeyPEM.Value(),
				cfg.ServiceAuth.Issuer,
				cfg.ServiceAuth.Audience,
			)
			if err != nil {
				return fmt.Errorf("service token validator: %w", err)
			}
			gatewayTrust = platformauth.NewGatewayTrust(serviceValidator, platformauth.NewPrometheusMetrics(cfg.ServiceName))
		} else {
			gatewayTrust = platformauth.NewGatewayTrust(nil, platformauth.NewPrometheusMetrics(cfg.ServiceName))
		}
		apiEnabled = true
	}

	var idemMiddleware func(http.Handler) http.Handler
	var redisClient *redis.Client
	if apiEnabled && cfg.IdempotencyEnabled {
		idemCfg, err := platformidempotency.LoadConfig(cfg.ServiceName)
		if err != nil {
			return err
		}
		mw, client, err := platformidempotency.RedisMiddleware(ctx, cfg.RedisURL, idemCfg, platformidempotency.HTTPOptions{
			Metrics: platformidempotency.NewMetrics(cfg.ServiceName), Policies: httpapi.IntegrationPolicies,
			ResolveScope: httpapi.ResolveIntegrationScope,
		})
		if err != nil {
			logger.Warn("idempotency disabled; redis unavailable", "error", err)
		} else if mw != nil {
			idemMiddleware = mw
			redisClient = client
			defer redisClient.Close()
		}
	}

	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewRouter(httpapi.RouterDeps{
			Logger: logger, ServiceName: cfg.ServiceName,
			WebhookHandler: handler, Validator: validator, GatewayTrust: gatewayTrust,
			APIEnabled: apiEnabled, Idempotency: idemMiddleware, Readiness: readiness,
			Environment: cfg.Environment,
		}),
	}
	httputil.ApplyServerTimeouts(server, httputil.DefaultServerTimeouts())

	go func() {
		logger.Info("integrations listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
