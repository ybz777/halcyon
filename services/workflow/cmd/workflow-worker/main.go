package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	appengine "github.com/halcyon-io/halcyon/services/workflow/internal/application/engine"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/events"
	workflowmetrics "github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/postgres"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/worker"
)

func main() {
	log := observability.NewLogger(envOrDefault("LOG_LEVEL", "info"), envOrDefault("LOG_FORMAT", "json"))
	slog.SetDefault(log)

	databaseURL := os.Getenv("WORKFLOW_DATABASE_URL")
	if databaseURL == "" {
		log.Error("WORKFLOW_DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if envBool("WORKFLOW_DATABASE_MIGRATE", true) {
		if err := postgres.RunMigrations(databaseURL); err != nil {
			log.Error("workflow migrations failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if err := migrations.Run(databaseURL); err != nil {
			log.Error("platform migrations failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{
		URL:             databaseURL,
		MaxConns:        int32(envInt("WORKFLOW_DATABASE_MAX_CONNS", 10)),
		MinConns:        int32(envInt("WORKFLOW_DATABASE_MIN_CONNS", 2)),
		MaxConnLifetime: envDuration("WORKFLOW_DATABASE_MAX_CONN_LIFETIME", time.Hour),
		MaxConnIdleTime: envDuration("WORKFLOW_DATABASE_MAX_CONN_IDLE_TIME", 30*time.Minute),
	})
	if err != nil {
		log.Error("connect database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	workflowRepo := postgres.NewWorkflowRepository(pool)
	executionRepo := postgres.NewExecutionRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)
	txManager := postgres.NewTransactionManager(pool)
	publisher := events.NewOutboxPublisher(pool)
	metrics := workflowmetrics.NewEngineMetrics()

	registry := appengine.NewRegistry(
		appengine.NewHTTPActionExecutor(),
		appengine.NewDelayExecutor(),
		appengine.NewEventEmitExecutor(publisher),
	)

	lease := envDuration("JOB_LEASE_DURATION", 30*time.Second)
	orchestrator := appengine.NewOrchestrator(
		workflowRepo, executionRepo, jobRepo, publisher, txManager, registry, metrics,
		appengine.OrchestratorConfig{LeaseDuration: lease},
	)

	w := worker.NewWorker(jobRepo, orchestrator, metrics, worker.Config{
		WorkerID:       envOrDefault("WORKER_ID", "workflow-worker-1"),
		PollInterval:   envDuration("WORKER_POLL_INTERVAL", time.Second),
		LeaseDuration:  lease,
		MaxConcurrent:  envInt("MAX_CONCURRENT_JOBS", 4),
		ClaimBatchSize: envInt("WORKER_CLAIM_BATCH_SIZE", 5),
	}, log)

	log.Info("workflow worker started")
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Error("worker stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
