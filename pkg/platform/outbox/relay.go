package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
)

// RelayConfig configures the outbox relay worker.
type RelayConfig struct {
	BatchSize    int
	PollInterval time.Duration
	MaxAttempts  int
}

// DefaultRelayConfig returns sensible relay defaults.
func DefaultRelayConfig() RelayConfig {
	return RelayConfig{
		BatchSize:    100,
		PollInterval: time.Second,
		MaxAttempts:  10,
	}
}

// Relay publishes pending outbox entries to Kafka.
type Relay struct {
	store   Store
	prod    kafka.Producer
	metrics *messaging.Metrics
	cfg     RelayConfig
	log     *slog.Logger
}

// NewRelay creates an outbox relay worker.
func NewRelay(store Store, prod kafka.Producer, metrics *messaging.Metrics, cfg RelayConfig, log *slog.Logger) *Relay {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 10
	}
	if log == nil {
		log = slog.Default()
	}
	return &Relay{store: store, prod: prod, metrics: metrics, cfg: cfg, log: log}
}

// Run polls the outbox and publishes events until the context is cancelled.
func (r *Relay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()

	for {
		if err := r.ProcessBatch(ctx); err != nil {
			r.log.Error("outbox relay batch failed", slog.String("error", err.Error()))
		}
		r.refreshOutboxMetric(ctx)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// ProcessBatch publishes a single batch of pending outbox entries.
func (r *Relay) ProcessBatch(ctx context.Context) error {
	_ = r.store.RequeueStalePublishing(ctx, 5*time.Minute)

	entries, err := r.store.ClaimPending(ctx, r.cfg.BatchSize)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := r.publishEntry(ctx, entry); err != nil {
			r.log.Error("event_publish_failure",
				slog.String("event_id", entry.EventID),
				slog.String("event_type", entry.EventType),
				slog.String("error", err.Error()),
			)
		}
	}
	return nil
}

func (r *Relay) publishEntry(ctx context.Context, entry Entry) error {
	start := time.Now()
	attempts := entry.Attempts + 1
	envelope := entry.Envelope()

	data, err := events.SerializeEnvelope(envelope)
	if err != nil {
		_ = r.store.MarkFailed(ctx, entry.ID, attempts, err.Error(), nil)
		return err
	}

	msg := kafka.Message{
		Topic:         entry.Topic,
		Key:           []byte(entry.PartitionKey),
		Value:         data,
		EventID:       entry.EventID,
		EventType:     entry.EventType,
		EventVersion:  entry.EventVersion,
		CorrelationID: entry.CorrelationID,
		RequestID:     entry.RequestID,
		Producer:      entry.Producer,
		TraceID:       entry.TraceID,
		SpanID:        entry.SpanID,
	}

	if err := r.prod.Publish(ctx, msg); err != nil {
		if r.metrics != nil {
			r.metrics.RecordOutboxPublishFailure(entry.EventType)
		}
		if attempts >= r.cfg.MaxAttempts {
			if dlqErr := r.deadLetter(ctx, entry, attempts, err); dlqErr != nil {
				return dlqErr
			}
			return err
		}
		next := NextRetryAt(attempts, time.Now().UTC())
		_ = r.store.MarkFailed(ctx, entry.ID, attempts, err.Error(), &next)
		return err
	}

	now := time.Now().UTC()
	if err := r.store.MarkPublished(ctx, entry.ID, now); err != nil {
		return err
	}

	if r.metrics != nil {
		r.metrics.RecordOutboxPublished(entry.EventType, entry.Topic)
		r.metrics.ObserveOutboxPublishLatency(entry.EventType, time.Since(start))
	}
	r.log.Info("event_publish_success",
		slog.String("event_id", entry.EventID),
		slog.String("event_type", entry.EventType),
		slog.String("topic", entry.Topic),
	)
	return nil
}

func (r *Relay) deadLetter(ctx context.Context, entry Entry, attempts int, cause error) error {
	reason := cause.Error()
	if err := r.store.MarkDeadLetter(ctx, entry.ID, reason); err != nil {
		return err
	}

	dlqTopic := events.DLQTopicFor(entry.Topic)
	dlqPayload, err := events.MarshalDLQMessage(events.NewDLQMessage(
		entry.Envelope(), entry.Topic, 0, 0, "outbox-relay",
		reason, "transient", attempts,
	))
	if err != nil {
		return err
	}

	dlqMsg := kafka.Message{
		Topic:         dlqTopic,
		Key:           []byte(entry.PartitionKey),
		Value:         dlqPayload,
		EventID:       entry.EventID,
		EventType:     entry.EventType,
		EventVersion:  entry.EventVersion,
		CorrelationID: entry.CorrelationID,
		RequestID:     entry.RequestID,
	}
	if err := r.prod.Publish(ctx, dlqMsg); err != nil {
		return fmt.Errorf("publish dead letter: %w", err)
	}

	if r.metrics != nil {
		r.metrics.RecordDeadLetter(dlqTopic, entry.EventType)
	}
	r.log.Warn("event_dead_lettered",
		slog.String("event_id", entry.EventID),
		slog.String("event_type", entry.EventType),
		slog.Int("attempts", attempts),
		slog.String("reason", reason),
	)
	return nil
}

func (r *Relay) refreshOutboxMetric(ctx context.Context) {
	if r.metrics == nil {
		return
	}
	count, err := r.store.PendingCount(ctx)
	if err != nil {
		r.log.Warn("failed to refresh outbox metric", slog.String("error", err.Error()))
		return
	}
	r.metrics.SetOutboxPending(count)
}
