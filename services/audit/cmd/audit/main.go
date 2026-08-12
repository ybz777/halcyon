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
	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	"github.com/halcyon-io/halcyon/pkg/platform/servicetoken"
	httpapi "github.com/halcyon-io/halcyon/services/audit/internal/api/http"
	appaudit "github.com/halcyon-io/halcyon/services/audit/internal/application/audit"
	"github.com/halcyon-io/halcyon/services/audit/internal/config"
	"github.com/halcyon-io/halcyon/services/audit/internal/infrastructure/health"
	auditmetrics "github.com/halcyon-io/halcyon/services/audit/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/audit/internal/infrastructure/postgres"
	jwtinfra "github.com/halcyon-io/halcyon/services/audit/internal/infrastructure/security/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
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

	var pool *pgxpool.Pool
	apiEnabled := false
	var handler *httpapi.AuditHandler
	var validator *jwtinfra.Validator
	var gatewayTrust *platformauth.GatewayTrust

	if cfg.DatabaseURL != "" {
		pool, err = postgres.NewPool(ctx, postgres.PoolConfig{
			URL:             cfg.DatabaseURL,
			MaxConns:        cfg.DatabaseMaxConns,
			MinConns:        cfg.DatabaseMinConns,
			MaxConnLifetime: cfg.DatabaseConnMaxLife,
			MaxConnIdleTime: cfg.DatabaseConnMaxIdle,
		})
		if err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		defer pool.Close()

		if cfg.DatabaseMigrate {
			if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
				return fmt.Errorf("run migrations: %w", err)
			}
			if err := migrations.Run(cfg.DatabaseURL); err != nil {
				return fmt.Errorf("run platform migrations: %w", err)
			}
		}

		if err := postgres.Ping(ctx, pool); err != nil {
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

		repo := postgres.NewAuditRepository(pool)
		query := appaudit.NewQueryUseCase(repo, appaudit.NewAuthorizer())
		handler = httpapi.NewAuditHandler(query, auditmetrics.NewCollector())
		apiEnabled = true
	}

	readiness := health.NewReadinessChecker(pool)
	httpMetrics := observability.NewHTTPMetrics(cfg.ServiceName)

	server := &http.Server{
		Addr: ":" + cfg.Port,
		Handler: httpapi.NewRouter(httpapi.RouterDeps{
			Logger:       logger,
			ServiceName:  cfg.ServiceName,
			Readiness:    readiness,
			Metrics:      httpMetrics,
			AuditHandler: handler,
			Validator:    validator,
			GatewayTrust: gatewayTrust,
			Environment:  cfg.Environment,
			APIEnabled:   apiEnabled,
		}),
	}
	httputil.ApplyServerTimeouts(server, httputil.DefaultServerTimeouts())

	logger.Info("audit service listening", slog.String("port", cfg.Port))

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
