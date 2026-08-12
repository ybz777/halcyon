package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	appaudit "github.com/halcyon-io/halcyon/services/audit/internal/application/audit"
	"github.com/halcyon-io/halcyon/services/audit/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/audit/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

const consumerGroup = "audit-log-consumer"

func main() {
	log := slog.New(
		slog.NewJSONHandler(
			os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo},
		),
	)

	databaseURL := os.Getenv("AUDIT_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DATABASE_URL")
	}
	if databaseURL == "" {
		log.Error("AUDIT_DATABASE_URL is required")
		os.Exit(1)
	}

	brokers := splitCSV(os.Getenv("KAFKA_BROKERS"))
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}

	concurrency := envIntOrDefault("CONSUMER_CONCURRENCY", 4)
	retentionDays := envIntOrDefault("AUDIT_RETENTION_DAYS", 365)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	if err := postgres.RunMigrations(databaseURL); err != nil {
		log.Error(
			"apply audit migrations",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	if err := migrations.Run(databaseURL); err != nil {
		log.Error(
			"apply platform migrations",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Error(
			"connect database",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer pool.Close()

	kafkaCfg := kafka.Config{
		Brokers: brokers,
		ClientID: "audit-consumer",
	}

	if err := kafka.EnsurePlatformTopics(ctx, kafkaCfg); err != nil {
		log.Warn(
			"ensure topics",
			slog.String("error", err.Error()),
		)
	}

	producer := kafka.NewWriterProducer(kafkaCfg)
	defer producer.Close()

	repo := postgres.NewAuditRepository(pool)

	collector := metrics.NewCollector()
	indexer := appaudit.NewIndexUseCase(repo, collector, log)

	// Create the messaging metrics ONCE.
	//
	// messaging.NewMetrics() registers Prometheus metrics globally.
	// Calling it once per topic would attempt to register the same
	// metric names multiple times and cause a panic.
	messagingMetrics := messaging.NewMetrics()

	topics := []struct {
		topic   string
		service string
	}{
		{"halcyon.auth", "auth"},
		{"halcyon.organizations", "organization"},
		{"halcyon.workflows", "workflow"},
		{"halcyon.audit", "gateway"},
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(topics))

	for _, spec := range topics {
		spec := spec

		wg.Add(1)

		go func() {
			defer wg.Done()

			topicConsumer := kafka.NewReaderConsumer(kafkaCfg)
			defer topicConsumer.Close()

			pipeline := messaging.NewPipeline(
				topicConsumer,
				messaging.PipelineConfig{
					Topic:       spec.topic,
					GroupID:     consumerGroup,
					Registry:    platformevents.DefaultRegistry(),
					Metrics:     messagingMetrics,
					Producer:    producer,
					Log:         log,
					Concurrency: concurrency,
					DLQEnabled: envBoolOrDefault(
						"EVENT_DLQ_ENABLED",
						true,
					),
					Idempotent: nil,
					Processed: nil,
					Retry: messaging.RetryConfig{
						MaxAttempts: envIntOrDefault(
							"EVENT_MAX_RETRIES",
							4,
						),
						Initial:    100 * time.Millisecond,
						MaxBackoff: 5 * time.Second,
						Multiplier: 2,
					},
				},
			)

			handler := platformevents.HandlerFunc(
				func(
					ctx context.Context,
					envelope platformevents.Envelope,
				) error {
					return indexer.HandleEnvelope(
						ctx,
						envelope,
						consumerGroup,
						spec.service,
					)
				},
			)

			log.Info(
				"audit consumer started",
				slog.String("topic", spec.topic),
				slog.String("group", consumerGroup),
			)

			if err := pipeline.Run(ctx, handler); err != nil &&
				err != context.Canceled {
				errCh <- err
			}
		}()
	}

	go runRetentionJob(
		ctx,
		pool,
		retentionDays,
		log,
	)

	select {
	case <-ctx.Done():
	case err := <-errCh:
		log.Error(
			"consumer stopped",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	wg.Wait()
}

func runRetentionJob(
	ctx context.Context,
	pool *pgxpool.Pool,
	retentionDays int,
	log *slog.Logger,
) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			cutoff := time.Now().
				UTC().
				AddDate(0, 0, -retentionDays)

			tag, err := pool.Exec(
				ctx,
				`DELETE FROM audit_events WHERE occurred_at < $1`,
				cutoff,
			)

			if err != nil {
				log.Error(
					"audit retention job failed",
					slog.String("error", err.Error()),
				)
				continue
			}

			log.Info(
				"audit retention job completed",
				slog.Int64("deleted", tag.RowsAffected()),
				slog.Time("cutoff", cutoff),
			)
		}
	}
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)

	if v == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return parsed
}

func envBoolOrDefault(key string, fallback bool) bool {
	v := strings.TrimSpace(
		strings.ToLower(os.Getenv(key)),
	)

	if v == "" {
		return fallback
	}

	return v == "1" ||
		v == "true" ||
		v == "yes"
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