package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := platformconfig.LoadOutboxRelay(platformconfig.FromEnv())
	if err != nil {
		log.Error("config failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	brokers := splitCSV(cfg.KafkaBrokers)
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := migrations.Run(cfg.DatabaseURL); err != nil {
		log.Error("apply migrations", slog.String("error", err.Error()))
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("connect database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	kafkaCfg := kafka.Config{Brokers: brokers, ClientID: cfg.ClientID}
	if err := kafka.EnsurePlatformTopics(ctx, kafkaCfg); err != nil {
		log.Warn("ensure topics", slog.String("error", err.Error()))
	}

	relayCfg := outbox.DefaultRelayConfig()
	relayCfg.BatchSize = cfg.BatchSize
	relayCfg.PollInterval = cfg.PollInterval

	producer := kafka.NewWriterProducer(kafkaCfg)
	defer producer.Close()

	metrics := messaging.NewMetrics()
	store := outbox.NewPostgresStore(pool)
	relay := outbox.NewRelay(store, producer, metrics, relayCfg, log)

	log.Info("outbox relay started",
		slog.Any("brokers", brokers),
		slog.Int("batch_size", relayCfg.BatchSize),
		slog.Duration("poll_interval", relayCfg.PollInterval),
	)
	if err := relay.Run(ctx); err != nil && err != context.Canceled {
		log.Error("relay stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
