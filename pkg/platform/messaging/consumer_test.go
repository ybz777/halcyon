package messaging_test

import (
	"context"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
)

type fakeConsumer struct {
	messages []kafka.Message
}

func (f *fakeConsumer) Run(ctx context.Context, topic, groupID string, handler kafka.ConsumerHandler) error {
	for _, msg := range f.messages {
		if err := handler(ctx, msg); err != nil {
			return err
		}
	}
	return nil
}

func (f *fakeConsumer) Close() error { return nil }

type recordingProducer struct {
	messages []kafka.Message
	failNext bool
}

func (p *recordingProducer) Publish(_ context.Context, msg kafka.Message) error {
	if p.failNext {
		p.failNext = false
		return context.DeadlineExceeded
	}
	p.messages = append(p.messages, msg)
	return nil
}

func (p *recordingProducer) Close() error { return nil }

type memoryProcessed struct {
	seen map[string]struct{}
}

func (m *memoryProcessed) key(eventID, group string) string {
	return eventID + "|" + group
}

func (m *memoryProcessed) MarkProcessed(_ context.Context, eventID, consumerGroup string) error {
	if m.seen == nil {
		m.seen = make(map[string]struct{})
	}
	m.seen[m.key(eventID, consumerGroup)] = struct{}{}
	return nil
}

func (m *memoryProcessed) IsProcessed(_ context.Context, eventID, consumerGroup string) (bool, error) {
	if m.seen == nil {
		return false, nil
	}
	_, ok := m.seen[m.key(eventID, consumerGroup)]
	return ok, nil
}

func TestPipeline_SkipsDuplicateEvent(t *testing.T) {
	t.Parallel()

	payload, err := events.MarshalPayload(events.TypeOrganizationCreated, events.OrganizationCreatedPayload{
		OrganizationID: "org_test12345678901",
		Name:           "Acme",
		Slug:           "acme",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	envelope := events.Envelope{
		EventID:        "evt_dupetest0000001",
		EventType:      events.TypeOrganizationCreated,
		EventVersion:   events.VersionV1,
		AggregateID:    "org_test12345678901",
		AggregateType:  "organization",
		OrganizationID: "org_test12345678901",
		OccurredAt:     time.Now().UTC(),
		Producer:       "test",
		Payload:        payload,
	}
	data, err := events.SerializeEnvelope(envelope)
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}

	processed := &memoryProcessed{}
	_ = processed.MarkProcessed(context.Background(), envelope.EventID, "test-group")

	handled := 0
	handler := events.HandlerFunc(func(ctx context.Context, env events.Envelope) error {
		handled++
		return nil
	})

	consumer := &fakeConsumer{messages: []kafka.Message{{
		Topic: "halcyon.organizations", Value: data, EventID: envelope.EventID, EventType: envelope.EventType,
	}}}

	pipeline := messaging.NewPipeline(consumer, messaging.PipelineConfig{
		Topic: "halcyon.organizations", GroupID: "test-group", Processed: processed,
	})
	if err := pipeline.Run(context.Background(), handler); err != nil {
		t.Fatalf("run: %v", err)
	}
	if handled != 0 {
		t.Fatalf("handler called %d times, want 0", handled)
	}
}

func TestPipeline_RoutesPermanentFailureToDLQ(t *testing.T) {
	t.Parallel()

	payload, _ := events.MarshalPayload(events.TypeOrganizationCreated, events.OrganizationCreatedPayload{
		OrganizationID: "org_test12345678901", Name: "Acme", Slug: "acme",
	})
	envelope := events.Envelope{
		EventID: "evt_dlqtest00000001", EventType: events.TypeOrganizationCreated,
		EventVersion: events.VersionV1, AggregateID: "org_test12345678901", AggregateType: "organization",
		OrganizationID: "org_test12345678901", OccurredAt: time.Now().UTC(), Producer: "test", Payload: payload,
	}
	data, _ := events.SerializeEnvelope(envelope)

	producer := &recordingProducer{}
	handler := events.HandlerFunc(func(ctx context.Context, env events.Envelope) error {
		return messaging.Permanent(context.Canceled)
	})

	consumer := &fakeConsumer{messages: []kafka.Message{{
		Topic: "halcyon.organizations", Partition: 1, Offset: 7, Value: data,
		EventID: envelope.EventID, EventType: envelope.EventType,
	}}}

	pipeline := messaging.NewPipeline(consumer, messaging.PipelineConfig{
		Topic: "halcyon.organizations", GroupID: "test-group", Producer: producer, DLQEnabled: true,
		Retry: messaging.RetryConfig{MaxAttempts: 1},
	})
	if err := pipeline.Run(context.Background(), handler); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(producer.messages) != 1 {
		t.Fatalf("dlq messages = %d, want 1", len(producer.messages))
	}
	if producer.messages[0].Topic != "halcyon.organizations.dlq" {
		t.Fatalf("dlq topic = %q", producer.messages[0].Topic)
	}
}

func TestPipeline_RejectsInvalidEnvelopeToDLQ(t *testing.T) {
	t.Parallel()

	producer := &recordingProducer{}
	consumer := &fakeConsumer{messages: []kafka.Message{{
		Topic: "halcyon.organizations", Value: []byte("not-json"),
	}}}

	pipeline := messaging.NewPipeline(consumer, messaging.PipelineConfig{
		Topic: "halcyon.organizations", GroupID: "test-group", Producer: producer, DLQEnabled: true,
	})
	if err := pipeline.Run(context.Background(), events.HandlerFunc(func(ctx context.Context, env events.Envelope) error {
		return nil
	})); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(producer.messages) != 1 {
		t.Fatalf("expected dlq publish")
	}
}
