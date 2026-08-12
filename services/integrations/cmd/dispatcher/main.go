package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
	"github.com/halcyon-io/halcyon/pkg/platform/migrations"
	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	jobpostgres "github.com/halcyon-io/halcyon/pkg/platform/jobs/postgres"
	"github.com/halcyon-io/halcyon/services/integrations/internal/application/dispatcher"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/postgres"
)

const consumerGroup = "webhook-dispatcher"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	databaseURL := os.Getenv("INTEGRATIONS_DATABASE_URL")
	if databaseURL == "" {
		log.Error("INTEGRATIONS_DATABASE_URL is required")
		os.Exit(1)
	}
	brokers := splitCSV(os.Getenv("KAFKA_BROKERS"))
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := postgres.RunMigrations(databaseURL); err != nil {
		log.Error("migrations failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	if err := migrations.Run(databaseURL); err != nil {
		log.Error("platform migrations failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	pool, err := postgres.NewPool(ctx, databaseURL, 25, 2)
	if err != nil {
		log.Error("database connect failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	kafkaCfg := kafka.Config{Brokers: brokers, ClientID: "webhook-dispatcher"}
	_ = kafka.EnsurePlatformTopics(ctx, kafkaCfg)
	producer := kafka.NewWriterProducer(kafkaCfg)
	defer producer.Close()

	endpoints := postgres.NewEndpointRepository(pool)
	deliveries := postgres.NewDeliveryRepository(pool)
	var jobEnqueuer dispatcher.JobEnqueuer
	if jobsURL := os.Getenv("JOBS_DATABASE_URL"); jobsURL != "" {
		jobsPool, err := postgres.NewPool(ctx, jobsURL, 10, 2)
		if err != nil {
			log.Error("jobs database connect failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer jobsPool.Close()
		jobEnqueuer = platformjobs.NewProducer(jobpostgres.NewRepository(jobsPool))
	}
	handler := dispatcher.NewHandler(endpoints, deliveries, pool, consumerGroup, jobEnqueuer)

	topics := []string{"halcyon.auth", "halcyon.organizations", "halcyon.workflows"}
	for _, topic := range topics {
		topic := topic
		go func() {
			consumer := kafka.NewReaderConsumer(kafkaCfg)
			defer consumer.Close()
			pipeline := messaging.NewPipeline(consumer, messaging.PipelineConfig{
				Topic: topic, GroupID: consumerGroup, Registry: platformevents.DefaultRegistry(),
				Producer: producer, Log: log, Concurrency: envInt("CONSUMER_CONCURRENCY", 4),
				DLQEnabled: true,
			})
			eventHandler := platformevents.HandlerFunc(handler.Handle)
			if err := pipeline.Run(ctx, eventHandler); err != nil && err != context.Canceled {
				log.Error("dispatcher stopped", slog.String("topic", topic), slog.String("error", err.Error()))
			}
		}()
	}
	<-ctx.Done()
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
