package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	jobpostgres "github.com/halcyon-io/halcyon/pkg/platform/jobs/postgres"
	"github.com/halcyon-io/halcyon/pkg/platform/observability"
	intjobhandler "github.com/halcyon-io/halcyon/services/integrations/pkg/jobhandler"
	"github.com/halcyon-io/halcyon/services/jobs/internal/config"
	jobmetrics "github.com/halcyon-io/halcyon/services/jobs/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/jobs/internal/infrastructure/postgres"
	notifjobhandler "github.com/halcyon-io/halcyon/services/notifications/pkg/jobhandler"
)

func main() {
	log := observability.NewLogger(envOrDefault("LOG_LEVEL", "info"), envOrDefault("LOG_FORMAT", "json"))
	slog.SetDefault(log)

	cfg, err := config.Load()
	if err != nil {
		log.Error("config failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if cfg.DatabaseURL == "" {
		log.Error("JOBS_DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.DatabaseMigrate {
		if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
			log.Error("migrations failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}

	jobsPool, err := postgres.NewPool(ctx, postgres.PoolConfig{URL: cfg.DatabaseURL, MaxConns: cfg.DatabaseMaxConns, MinConns: cfg.DatabaseMinConns})
	if err != nil {
		log.Error("jobs database connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer jobsPool.Close()

	repo := jobpostgres.NewRepository(jobsPool)
	attempts := jobpostgres.NewAttemptRepository(jobsPool)
	metrics := jobmetrics.NewCollector()
	registry := platformjobs.NewRegistry()

	closeWebhook, err := intjobhandler.RegisterWebhookDelivery(ctx, registry, cfg.IntegrationsDBURL, cfg.IntegrationsEncKey,
		intjobhandler.DefaultWebhookConfig(
			envInt("WEBHOOK_MAX_RETRY_ATTEMPTS", 6),
			envInt("WEBHOOK_FAILURE_THRESHOLD", 10),
			envBool("WEBHOOK_ALLOW_HTTP", false),
			envDuration("WEBHOOK_TIMEOUT", 10*time.Second),
		))
	if err != nil {
		log.Error("webhook handler registration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer closeWebhook()

	closeEmail, err := notifjobhandler.RegisterEmailDelivery(ctx, registry, cfg.NotificationsDBURL,
		envOrDefault("EMAIL_PROVIDER", "log"),
		notifjobhandler.DefaultEmailConfig(
			envInt("EMAIL_MAX_RETRY_ATTEMPTS", 6),
			envOrDefault("EMAIL_FROM_NAME", "Halcyon"),
			envOrDefault("EMAIL_FROM_ADDRESS", "noreply@halcyon.local"),
			envDuration("EMAIL_TIMEOUT", 15*time.Second),
		))
	if err != nil {
		log.Error("email handler registration failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer 	closeEmail()

	workerCfg, err := platformjobs.LoadConfig()
	if err != nil {
		log.Error("worker config failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	pool := platformjobs.NewPool(repo, registry, workerCfg.PoolConfigFrom(
		envOrDefault("WORKER_ID", defaultWorkerID()),
		[]platformjobs.Type{platformjobs.TypeWebhookDelivery, platformjobs.TypeEmailDelivery},
	), metrics, log, platformjobs.WithAttemptRepository(attempts))

	log.Info("job worker started", slog.String("worker_id", envOrDefault("WORKER_ID", defaultWorkerID())))
	if err := pool.Run(ctx); err != nil && err != context.Canceled {
		log.Error("worker stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func defaultWorkerID() string {
	host, _ := os.Hostname()
	if host == "" {
		return "job-worker-1"
	}
	return "job-worker-" + host
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
