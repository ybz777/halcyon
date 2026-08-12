package jobs

import (
	"context"
	"encoding/json"
	"time"
)

// Producer enqueues durable jobs.
type Producer struct {
	repo    Repository
	metrics MetricsCollector
}

func NewProducer(repo Repository) *Producer {
	return &Producer{repo: repo, metrics: NoopMetrics()}
}

// WithMetrics attaches a metrics collector to the producer.
func (p *Producer) WithMetrics(metrics MetricsCollector) *Producer {
	if metrics != nil {
		p.metrics = metrics
	}
	return p
}

// Enqueue creates a job, returning duplicate=true when idempotency key collides.
func (p *Producer) Enqueue(ctx context.Context, input CreateInput) (Job, bool, error) {
	if len(input.Payload) > MaxPayloadBytes {
		return Job{}, false, ErrPayloadTooLarge
	}
	if err := ValidateType(input.Type); err != nil {
		return Job{}, false, err
	}
	if input.Priority == "" {
		input.Priority = PriorityNormal
	}
	if input.MaxAttempts <= 0 {
		input.MaxAttempts = 6
	}
	if input.AvailableAt.IsZero() {
		input.AvailableAt = time.Now().UTC()
	}
	job, duplicate, err := p.repo.Create(ctx, input)
	if err == nil && !duplicate {
		p.metrics.IncCreated(input.Type)
	}
	return job, duplicate, err
}

// WebhookDeliveryPayload references a webhook delivery row.
type WebhookDeliveryPayload struct {
	DeliveryID string `json:"delivery_id"`
}

// EmailDeliveryPayload references an email delivery row.
type EmailDeliveryPayload struct {
	DeliveryID string `json:"delivery_id"`
}

func MarshalWebhookDelivery(deliveryID string) ([]byte, error) {
	return json.Marshal(WebhookDeliveryPayload{DeliveryID: deliveryID})
}

func MarshalEmailDelivery(deliveryID string) ([]byte, error) {
	return json.Marshal(EmailDeliveryPayload{DeliveryID: deliveryID})
}
