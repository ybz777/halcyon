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

	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	platformidempotency "github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	httpapi "github.com/halcyon-io/halcyon/services/workflow/internal/api/http"
	appengine "github.com/halcyon-io/halcyon/services/workflow/internal/application/engine"
	appincident "github.com/halcyon-io/halcyon/services/workflow/internal/application/incident"
	appworkflow "github.com/halcyon-io/halcyon/services/workflow/internal/application/workflow"
	"github.com/halcyon-io/halcyon/services/workflow/internal/config"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/events"
	infrahealth "github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/health"
	workflowmetrics "github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/postgres"
	jwtinfra "github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/security/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
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
	var incidentHandler *httpapi.IncidentHandler
	var workflowHandler *httpapi.WorkflowHandler
	var validator *jwtinfra.Validator

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
				return fmt.Errorf("run workflow migrations: %w", err)
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
			return fmt.Errorf("create jwt validator: %w", err)
		}

		incidentMetrics := workflowmetrics.New()
		engineMetrics := workflowmetrics.NewEngineMetrics()
		limits := domain.DefaultWorkflowLimits()

		incidentRepo := postgres.NewIncidentRepository(pool)
		historyRepo := postgres.NewHistoryRepository(pool)
		txManager := postgres.NewTransactionManager(pool)
		publisher := events.NewOutboxPublisher(pool)

		workflowRepo := postgres.NewWorkflowRepository(pool)
		executionRepo := postgres.NewExecutionRepository(pool)
		jobRepo := postgres.NewJobRepository(pool)
		idempotencyRepo := postgres.NewIdempotencyRepository(pool)

		transitionUC := appincident.NewTransitionIncidentUseCase(incidentRepo, historyRepo, publisher, txManager, incidentMetrics)

		incidentHandler = httpapi.NewIncidentHandler(
			appincident.NewCreateIncidentUseCase(incidentRepo, historyRepo, publisher, txManager, incidentMetrics),
			appincident.NewGetIncidentUseCase(incidentRepo),
			appincident.NewListIncidentsUseCase(incidentRepo),
			appincident.NewSearchIncidentsUseCase(incidentRepo),
			transitionUC,
			appincident.NewUpdateSeverityUseCase(incidentRepo, historyRepo, publisher, txManager, incidentMetrics),
			appincident.NewResolveIncidentUseCase(transitionUC),
			appincident.NewCloseIncidentUseCase(transitionUC),
		)

		workflowHandler = httpapi.NewWorkflowHandler(
			appworkflow.NewCreateWorkflowUseCase(workflowRepo, txManager, limits),
			appworkflow.NewGetWorkflowUseCase(workflowRepo),
			appworkflow.NewListWorkflowsUseCase(workflowRepo),
			appworkflow.NewStartExecutionUseCase(workflowRepo, executionRepo, jobRepo, idempotencyRepo, publisher, txManager, engineMetrics, limits),
			appworkflow.NewGetExecutionUseCase(executionRepo),
			appworkflow.NewCancelExecutionUseCase(executionRepo, publisher, txManager, engineMetrics),
		)

		_ = appengine.NewRegistry(
			appengine.NewHTTPActionExecutor(),
			appengine.NewDelayExecutor(),
			appengine.NewEventEmitExecutor(publisher),
		)
		apiEnabled = true
	}

	var idemMiddleware func(http.Handler) http.Handler
	var redisClient *redis.Client
	if apiEnabled && cfg.IdempotencyEnabled {
		idemCfg, err := platformidempotency.LoadConfig(cfg.ServiceName)
		if err != nil {
			return fmt.Errorf("load idempotency config: %w", err)
		}
		mw, client, err := platformidempotency.RedisMiddleware(ctx, cfg.RedisURL, idemCfg, platformidempotency.HTTPOptions{
			Metrics: platformidempotency.NewMetrics(cfg.ServiceName), Policies: httpapi.WorkflowPolicies,
			ResolveScope: httpapi.ResolveWorkflowScope,
		})
		if err != nil {
			logger.Warn("idempotency disabled; redis unavailable", "error", err)
		} else if mw != nil {
			idemMiddleware = mw
			redisClient = client
			defer redisClient.Close()
		}
	}

	router := httpapi.NewRouter(httpapi.RouterDeps{
		Logger:          logger,
		ServiceName:     cfg.ServiceName,
		Readiness:       infrahealth.NewPostgresChecker(cfg.DatabaseURL, pool),
		Metrics:         observability.NewHTTPMetrics(cfg.ServiceName),
		IncidentHandler: incidentHandler,
		WorkflowHandler: workflowHandler,
		Validator:       validator,
		APIEnabled:      apiEnabled,
		Idempotency:     idemMiddleware,
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("starting server", "service", cfg.ServiceName, "port", cfg.Port, "api_enabled", apiEnabled)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return fmt.Errorf("listen and serve: %w", err)
	case <-stop:
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}
