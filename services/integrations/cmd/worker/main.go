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
	"github.com/halcyon-io/halcyon/services/integrations/internal/config"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/crypto"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/postgres"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/worker"
)

func main() {
	log := observability.NewLogger(envOrDefault("LOG_LEVEL", "info"), envOrDefault("LOG_FORMAT", "json"))
	slog.SetDefault(log)

	cfg := config.Config{
		DatabaseURL:       os.Getenv("INTEGRATIONS_DATABASE_URL"),
		EncryptionKey:     os.Getenv("INTEGRATIONS_ENCRYPTION_KEY"),
		AllowHTTPWebhooks: envBool("WEBHOOK_ALLOW_HTTP", false),
		WebhookTimeout:    envDuration("WEBHOOK_TIMEOUT", 10*time.Second),
		MaxRetryAttempts:  envInt("WEBHOOK_MAX_RETRY_ATTEMPTS", 6),
		FailureThreshold:  envInt("WEBHOOK_FAILURE_THRESHOLD", 10),
	}
	if cfg.DatabaseURL == "" || cfg.EncryptionKey == "" {
		log.Error("INTEGRATIONS_DATABASE_URL and INTEGRATIONS_ENCRYPTION_KEY are required")
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

	cipher, err := crypto.NewSecretCipher(cfg.EncryptionKey)
	if err != nil {
		log.Error("cipher init failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	w := worker.NewWorker(
		postgres.NewDeliveryRepository(pool),
		postgres.NewEndpointRepository(pool),
		cipher,
		worker.Config{
			WorkerID: envOrDefault("WORKER_ID", worker.DefaultWorkerID()),
			PollInterval: envDuration("WORKER_POLL_INTERVAL", time.Second),
			LeaseDuration: envDuration("DELIVERY_LEASE_DURATION", 30*time.Second),
			MaxConcurrent: envInt("MAX_CONCURRENT_DELIVERIES", 8),
			ClaimBatchSize: envInt("DELIVERY_CLAIM_BATCH_SIZE", 10),
			MaxAttempts: cfg.MaxRetryAttempts,
			FailureThreshold: cfg.FailureThreshold,
			AllowHTTP: cfg.AllowHTTPWebhooks,
			Timeout: cfg.WebhookTimeout,
		},
		log,
	)
	log.Info("webhook worker started")
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
