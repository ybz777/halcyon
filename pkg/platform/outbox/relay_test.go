package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/kafka"
	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
)

type dlqCaptureProducer struct {
	msgs []kafka.Message
}

func (p *dlqCaptureProducer) Publish(_ context.Context, msg kafka.Message) error {
	p.msgs = append(p.msgs, msg)
	return nil
}

func (p *dlqCaptureProducer) Close() error { return nil }

type relayDLQProducer struct {
	cap     *dlqCaptureProducer
	attempt int
}

func (p *relayDLQProducer) Publish(ctx context.Context, msg kafka.Message) error {
	p.attempt++
	if p.attempt == 1 {
		return context.DeadlineExceeded
	}
	return p.cap.Publish(ctx, msg)
}

func (p *relayDLQProducer) Close() error { return nil }

func TestRelayDeadLetterUsesStructuredDLQ(t *testing.T) {
	t.Parallel()

	payload, err := events.MarshalPayload(events.TypeUserRegistered, events.UserRegisteredPayload{
		UserID: "usr_test12345678901", Email: "a@example.com",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	store := &memoryStore{entries: []outbox.Entry{{
		ID: "outbox-1", EventID: "evt_relaydlq0000001", EventType: events.TypeUserRegistered,
		EventVersion: events.VersionV1, AggregateID: "usr_test12345678901", AggregateType: "user",
		OrganizationID: events.SystemOrganizationID, OccurredAt: time.Now().UTC(), Producer: "auth",
		Payload: payload, Topic: "halcyon.auth", PartitionKey: "usr_test12345678901",
		Status: outbox.StatusPending, Attempts: 9,
	}}}

	producer := &dlqCaptureProducer{}
	relay := outbox.NewRelay(store, &relayDLQProducer{cap: producer}, messaging.NewMetrics(), outbox.RelayConfig{
		BatchSize: 1, MaxAttempts: 10,
	}, nil)

	if err := relay.ProcessBatch(context.Background()); err != nil {
		t.Fatalf("process batch: %v", err)
	}
	if len(producer.msgs) != 1 {
		t.Fatalf("published messages = %d, want 1", len(producer.msgs))
	}
	if producer.msgs[0].Topic != "halcyon.auth.dlq" {
		t.Fatalf("dlq topic = %q", producer.msgs[0].Topic)
	}

	dlq, err := events.UnmarshalDLQMessage(producer.msgs[0].Value)
	if err != nil {
		t.Fatalf("unmarshal dlq: %v", err)
	}
	if dlq.OriginalTopic != "halcyon.auth" || dlq.OriginalEvent.EventID != "evt_relaydlq0000001" {
		t.Fatalf("dlq payload mismatch: %+v", dlq)
	}
	if store.entries[0].Status != outbox.StatusFailed {
		t.Fatalf("status = %q, want failed", store.entries[0].Status)
	}
}
