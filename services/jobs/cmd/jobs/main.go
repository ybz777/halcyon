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

	jobpostgres "github.com/halcyon-io/halcyon/pkg/platform/jobs/postgres"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
	httpapi "github.com/halcyon-io/halcyon/services/jobs/internal/api/http"
	appquery "github.com/halcyon-io/halcyon/services/jobs/internal/application/query"
	"github.com/halcyon-io/halcyon/services/jobs/internal/config"
	auditinfra "github.com/halcyon-io/halcyon/services/jobs/internal/infrastructure/audit"
	"github.com/halcyon-io/halcyon/services/jobs/internal/infrastructure/health"
	jobspostgres "github.com/halcyon-io/halcyon/services/jobs/internal/infrastructure/postgres"
	jobmetrics "github.com/halcyon-io/halcyon/services/jobs/internal/infrastructure/metrics"
	jwtinfra "github.com/halcyon-io/halcyon/services/jobs/internal/infrastructure/security/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("jobs service failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	logger := observability.NewLogger(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)
	ctx := context.Background()

	var dbPool *pgxpool.Pool
	var handler *httpapi.JobHandler
	var validator *jwtinfra.Validator
	var gatewayTrust *platformauth.GatewayTrust
	apiEnabled := false

	if cfg.DatabaseURL != "" {
		dbPool, err = jobspostgres.NewPool(ctx, jobspostgres.PoolConfig{
			URL: cfg.DatabaseURL, MaxConns: cfg.DatabaseMaxConns, MinConns: cfg.DatabaseMinConns,
		})
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		defer dbPool.Close()

		if cfg.DatabaseMigrate {
			if err := jobspostgres.RunMigrations(cfg.DatabaseURL); err != nil {
				return fmt.Errorf("run migrations: %w", err)
			}
		}
		if err := jobspostgres.Ping(ctx, dbPool); err != nil {
			return fmt.Errorf("validate database: %w", err)
		}

		validator, err = jwtinfra.NewValidator([]byte(cfg.JWTSecret), cfg.AuthServiceName)
		if err != nil {
			return fmt.Errorf("jwt validator: %w", err)
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
		repo := jobpostgres.NewRepository(dbPool)
		attemptRepo := jobpostgres.NewAttemptRepository(dbPool)
		query := appquery.New(repo, jobspostgres.NewAttemptReader(attemptRepo), auditinfra.NoopPublisher{})
		handler = httpapi.NewJobHandler(query)
		apiEnabled = true
		_ = jobmetrics.NewCollector()
	}

	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewRouter(httpapi.RouterDeps{
			Logger: logger, ServiceName: cfg.ServiceName, Readiness: health.NewReadinessChecker(dbPool),
			Metrics: observability.NewHTTPMetrics(cfg.ServiceName), JobHandler: handler,
			Validator: validator, GatewayTrust: gatewayTrust, Environment: cfg.Environment, APIEnabled: apiEnabled,
		}),
	}
	httputil.ApplyServerTimeouts(server, httputil.DefaultServerTimeouts())
	logger.Info("jobs service listening", slog.String("port", cfg.Port))

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stopCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-errCh:
		return err
	case <-stopCtx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
