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
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	platformhealth "github.com/halcyon-io/halcyon/pkg/platform/health"
	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
	httpapi "github.com/halcyon-io/halcyon/services/notifications/internal/api/http"
	appnotification "github.com/halcyon-io/halcyon/services/notifications/internal/application/notification"
	apppreference "github.com/halcyon-io/halcyon/services/notifications/internal/application/preference"
	"github.com/halcyon-io/halcyon/services/notifications/internal/config"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/postgres"
	jwtinfra "github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/security/jwt"
)

func main() {
	if err := run(); err != nil {
		slog.Error("notifications failed", "error", err)
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

	var notificationHandler *httpapi.NotificationHandler
	var preferenceHandler *httpapi.PreferenceHandler
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

		notifications := postgres.NewNotificationRepository(pool)
		preferences := postgres.NewPreferenceRepository(pool)
		notificationHandler = httpapi.NewNotificationHandler(appnotification.NewService(notifications))
		preferenceHandler = httpapi.NewPreferenceHandler(apppreference.NewService(preferences))

		validator, err = jwtinfra.NewValidator([]byte(cfg.JWTSecret), cfg.AuthServiceName)
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

	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewRouter(httpapi.RouterDeps{
			Logger: logger, ServiceName: cfg.ServiceName,
			NotificationHandler: notificationHandler, PreferenceHandler: preferenceHandler,
			Validator: validator, APIEnabled: apiEnabled, Readiness: readiness,
			GatewayTrust: gatewayTrust, Environment: cfg.Environment,
		}),
	}
	httputil.ApplyServerTimeouts(server, httputil.DefaultServerTimeouts())

	go func() {
		logger.Info("notifications listening", "port", cfg.Port)
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
