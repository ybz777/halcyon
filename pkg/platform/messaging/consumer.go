package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/contextx"
	"github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
)

// PipelineConfig configures the event consumer pipeline.
type PipelineConfig struct {
	Topic         string
	GroupID       string
	Retry         RetryConfig
	Registry      *events.Registry
	Processed     ProcessedEventStore
	Idempotent    *IdempotentProcessor
	Metrics       *Metrics
	Producer      kafka.Producer
	Log           *slog.Logger
	Concurrency   int
	DLQEnabled    bool
}

// Pipeline consumes Kafka events with idempotency, retry, and DLQ support.
type Pipeline struct {
	consumer kafka.Consumer
	cfg      PipelineConfig
}

// NewPipeline creates a consumer pipeline.
func NewPipeline(consumer kafka.Consumer, cfg PipelineConfig) *Pipeline {
	if cfg.Registry == nil {
		cfg.Registry = events.DefaultRegistry()
	}
	if cfg.Retry.MaxAttempts <= 0 {
		cfg.Retry = DefaultRetryConfig()
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 1
	}
	if cfg.Log == nil {
		cfg.Log = slog.Default()
	}
	if !cfg.DLQEnabled {
		cfg.DLQEnabled = cfg.Producer != nil
	}
	return &Pipeline{consumer: consumer, cfg: cfg}
}

// Run starts consuming until the context is cancelled.
func (p *Pipeline) Run(ctx context.Context, handler events.Handler) error {
	if concurrent, ok := p.consumer.(kafka.ConcurrentConsumer); ok && p.cfg.Concurrency > 1 {
		return concurrent.RunConcurrent(ctx, p.cfg.Topic, p.cfg.GroupID, p.cfg.Concurrency, func(ctx context.Context, msg kafka.Message) error {
			return p.handleMessage(ctx, handler, msg)
		})
	}
	return p.consumer.Run(ctx, p.cfg.Topic, p.cfg.GroupID, func(ctx context.Context, msg kafka.Message) error {
		return p.handleMessage(ctx, handler, msg)
	})
}

func (p *Pipeline) handleMessage(ctx context.Context, handler events.Handler, msg kafka.Message) error {
	start := time.Now()

	envelope, err := events.DeserializeEnvelope(msg.Value)
	if err != nil {
		if p.cfg.Metrics != nil {
			p.cfg.Metrics.RecordFailure(msg.EventType, string(FailureDeserialize))
			p.cfg.Metrics.RecordConsumeFailure(msg.EventType, string(FailureDeserialize))
		}
		return p.sendToDLQ(ctx, msg, events.Envelope{}, err, 0, FailureDeserialize)
	}

	if msg.EventID == "" {
		msg.EventID = envelope.EventID
	}
	if msg.EventType == "" {
		msg.EventType = envelope.EventType
	}

	ctx = kafka.ExtractTraceContext(ctx, msg.Propagation)
	ctx = ContextWithTrace(ctx, envelope.TraceID, envelope.SpanID)
	if envelope.CorrelationID != "" {
		ctx = contextx.WithCorrelationID(ctx, envelope.CorrelationID)
	}
	if envelope.RequestID != "" {
		ctx = contextx.WithRequestID(ctx, envelope.RequestID)
	}

	if p.cfg.Processed != nil {
		processed, err := p.cfg.Processed.IsProcessed(ctx, envelope.EventID, p.cfg.GroupID)
		if err != nil {
			return err
		}
		if processed {
			p.cfg.Log.Info("event_duplicate",
				slog.String("event_id", envelope.EventID),
				slog.String("event_type", envelope.EventType),
				slog.String("consumer_group", p.cfg.GroupID),
			)
			if p.cfg.Metrics != nil {
				p.cfg.Metrics.RecordDuplicate(envelope.EventType, p.cfg.GroupID)
				p.cfg.Metrics.RecordConsumed(envelope.EventType, p.cfg.Topic)
			}
			return nil
		}
	}

	process := func(ctx context.Context) error {
		return p.invokeWithRetry(ctx, handler, envelope)
	}

	if p.cfg.Idempotent != nil {
		result, err := p.cfg.Idempotent.ProcessOnce(ctx, envelope.EventID, p.cfg.GroupID, envelope.OrganizationID, process)
		if err != nil {
			category := ClassifyFailure(err)
			if !ShouldRetry(err) {
				if p.cfg.Metrics != nil {
					p.cfg.Metrics.RecordConsumeFailure(envelope.EventType, string(category))
				}
				return p.sendToDLQ(ctx, msg, envelope, err, p.cfg.Retry.MaxAttempts, category)
			}
			return err
		}
		if result.Duplicate {
			if p.cfg.Metrics != nil {
				p.cfg.Metrics.RecordDuplicate(envelope.EventType, p.cfg.GroupID)
			}
			return nil
		}
	} else if err := process(ctx); err != nil {
		return err
	} else if p.cfg.Processed != nil {
		if err := p.cfg.Processed.MarkProcessed(ctx, envelope.EventID, p.cfg.GroupID); err != nil {
			return err
		}
	}

	if p.cfg.Metrics != nil {
		p.cfg.Metrics.RecordConsumed(envelope.EventType, p.cfg.Topic)
		p.cfg.Metrics.RecordEventConsumed(envelope.EventType, p.cfg.Topic)
		p.cfg.Metrics.ObserveLatency(envelope.EventType, time.Since(start))
	}
	p.cfg.Log.Info("event_consumed",
		slog.String("event_id", envelope.EventID),
		slog.String("event_type", envelope.EventType),
		slog.String("organization_id", envelope.OrganizationID),
		slog.String("consumer_group", p.cfg.GroupID),
	)
	return nil
}

func (p *Pipeline) invokeWithRetry(ctx context.Context, handler events.Handler, envelope events.Envelope) error {
	var lastErr error
	attempts := p.cfg.Retry.MaxAttempts
	for attempt := 1; attempt <= attempts; attempt++ {
		lastErr = handler.Handle(ctx, envelope)
		if lastErr == nil {
			return nil
		}
		if !ShouldRetry(lastErr) {
			break
		}
		if attempt < attempts {
			if p.cfg.Metrics != nil {
				p.cfg.Metrics.RecordRetry(envelope.EventType, p.cfg.GroupID)
			}
			delay := BackoffWithJitter(p.cfg.Retry, attempt)
			p.cfg.Log.Warn("event_retry",
				slog.String("event_id", envelope.EventID),
				slog.Int("attempt", attempt),
				slog.Duration("backoff", delay),
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	if p.cfg.Metrics != nil {
		p.cfg.Metrics.RecordFailure(envelope.EventType, "handler")
		p.cfg.Metrics.RecordConsumeFailure(envelope.EventType, string(ClassifyFailure(lastErr)))
	}
	category := ClassifyFailure(lastErr)
	return p.sendToDLQ(ctx, kafka.Message{
		EventID: envelope.EventID, EventType: envelope.EventType,
	}, envelope, lastErr, attempts, category)
}

func (p *Pipeline) sendToDLQ(ctx context.Context, msg kafka.Message, envelope events.Envelope, cause error, attempts int, category FailureCategory) error {
	if !p.cfg.DLQEnabled || p.cfg.Producer == nil {
		return cause
	}

	dlqTopic := events.DLQTopicFor(p.cfg.Topic)
	dlqPayload, err := events.MarshalDLQMessage(events.NewDLQMessage(
		envelope, p.cfg.Topic, msg.Partition, msg.Offset, p.cfg.GroupID,
		FailureReason(cause), string(category), attempts,
	))
	if err != nil {
		return fmt.Errorf("marshal dlq: %w (original: %v)", err, cause)
	}

	dlqMsg := kafka.Message{
		Topic:          dlqTopic,
		Key:            msg.Key,
		Value:          dlqPayload,
		EventID:        msg.EventID,
		EventType:      msg.EventType,
		EventVersion:   envelope.EventVersion,
		CorrelationID:  envelope.CorrelationID,
		RequestID:      envelope.RequestID,
		TraceID:        msg.TraceID,
		SpanID:         msg.SpanID,
	}

	if err := p.cfg.Producer.Publish(ctx, dlqMsg); err != nil {
		return fmt.Errorf("publish to dlq: %w (original: %v)", err, cause)
	}

	p.cfg.Log.Error("event_dlq",
		slog.String("event_id", msg.EventID),
		slog.String("topic", dlqTopic),
		slog.String("failure_category", string(category)),
		slog.Int("attempts", attempts),
	)

	if p.cfg.Metrics != nil {
		p.cfg.Metrics.RecordDLQ(dlqTopic, msg.EventType)
		p.cfg.Metrics.RecordEventDLQ(dlqTopic, msg.EventType, string(category))
	}
	return nil
}
func BackoffWithJitter(cfg RetryConfig, attempt int) time.Duration {
	base := Backoff(cfg, attempt)
	jitter := time.Duration(rand.Int63n(int64(base / 5)))
	return base + jitter
}

// KafkaEventConsumer adapts the messaging pipeline to events.Consumer.
type KafkaEventConsumer struct {
	pipeline *Pipeline
}

// NewKafkaEventConsumer creates an events.Consumer backed by Kafka.
func NewKafkaEventConsumer(consumer kafka.Consumer, cfg PipelineConfig) *KafkaEventConsumer {
	return &KafkaEventConsumer{pipeline: NewPipeline(consumer, cfg)}
}

// Run implements events.Consumer.
func (c *KafkaEventConsumer) Run(ctx context.Context, topic, groupID string, handler events.Handler) error {
	c.pipeline.cfg.Topic = topic
	c.pipeline.cfg.GroupID = groupID
	return c.pipeline.Run(ctx, handler)
}
