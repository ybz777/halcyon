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
	"github.com/halcyon-io/halcyon/services/notifications/internal/config"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/email"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/postgres"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/worker"
)

func main() {
	log := observability.NewLogger(envOrDefault("LOG_LEVEL", "info"), envOrDefault("LOG_FORMAT", "json"))
	slog.SetDefault(log)

	cfg := config.Config{
		DatabaseURL:    os.Getenv("NOTIFICATIONS_DATABASE_URL"),
		EmailProvider:  envOrDefault("EMAIL_PROVIDER", "log"),
		EmailFromName:  envOrDefault("EMAIL_FROM_NAME", "Halcyon"),
		EmailFromAddress: envOrDefault("EMAIL_FROM_ADDRESS", "noreply@halcyon.local"),
		EmailTimeout:   envDuration("EMAIL_TIMEOUT", 10*time.Second),
		MaxRetryAttempts: envInt("EMAIL_MAX_RETRY_ATTEMPTS", 6),
	}
	if cfg.DatabaseURL == "" {
		log.Error("NOTIFICATIONS_DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := postgres.RunMigrations(cfg.DatabaseURL); err != nil {
		log.Error("migrations failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := migrations.Run(cfg.DatabaseURL); err != nil {
		log.Error("platform migrations failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, 25, 2)
	if err != nil {
		log.Error("database connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	w := worker.NewWorker(
		postgres.NewEmailDeliveryRepository(pool),
		postgres.NewNotificationRepository(pool),
		email.NewSender(cfg.EmailProvider),
		worker.Config{
			WorkerID:       envOrDefault("WORKER_ID", worker.DefaultWorkerID()),
			PollInterval:   envDuration("WORKER_POLL_INTERVAL", time.Second),
			LeaseDuration:  envDuration("DELIVERY_LEASE_DURATION", 30*time.Second),
			MaxConcurrent:  envInt("MAX_CONCURRENT_EMAILS", 8),
			ClaimBatchSize: envInt("EMAIL_CLAIM_BATCH_SIZE", 10),
			MaxAttempts:    cfg.MaxRetryAttempts,
			FromName:       cfg.EmailFromName,
			FromAddress:    cfg.EmailFromAddress,
			Timeout:        cfg.EmailTimeout,
		},
		log,
	)
	log.Info("email worker started")
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
