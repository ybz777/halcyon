package events_test

import (
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
)

func TestValidate_RejectsMissingFields(t *testing.T) {
	t.Parallel()
	err := events.Validate(events.Envelope{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidate_RejectsUnknownEventType(t *testing.T) {
	t.Parallel()
	err := events.Validate(events.Envelope{
		EventID: "evt_test12345678901", EventType: "unknown.event", EventVersion: events.VersionV1,
		AggregateID: "agg_1", AggregateType: "x", OccurredAt: time.Now().UTC(), Producer: "test",
		Payload: []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected unsupported event type")
	}
}

func TestValidate_RejectsMissingOrganizationID(t *testing.T) {
	t.Parallel()
	payload, _ := events.MarshalPayload(events.TypeMembershipCreated, events.MembershipCreatedPayload{
		OrganizationID: "org_1", UserID: "usr_1", Role: "member",
	})
	err := events.Validate(events.Envelope{
		EventID: "evt_test12345678901", EventType: events.TypeMembershipCreated,
		EventVersion: events.VersionV1, AggregateID: "mem_1", AggregateType: "membership",
		OccurredAt: time.Now().UTC(), Producer: "organization", Payload: payload,
	})
	if err == nil {
		t.Fatal("expected missing organization_id error")
	}
}

func TestJSONEnvelopeRoundTrip(t *testing.T) {
	t.Parallel()
	payload, _ := events.MarshalPayload(events.TypeUserRegistered, events.UserRegisteredPayload{
		UserID: "usr_1", Email: "a@example.com",
	})
	original := events.Envelope{
		EventID: events.TypeUserRegistered, EventType: events.TypeUserRegistered,
		EventVersion: events.VersionV1, AggregateID: "usr_1", AggregateType: "user",
		OrganizationID: events.SystemOrganizationID, CorrelationID: "corr_1", RequestID: "req_1",
		ActorID: "usr_1", OccurredAt: time.Now().UTC(), Producer: "auth", Payload: payload,
	}
	original.EventID = "evt_test12345678901"

	data, err := events.MarshalJSONEnvelope(original)
	if err != nil {
		t.Fatalf("MarshalJSONEnvelope() error = %v", err)
	}
	got, err := events.UnmarshalJSONEnvelope(data)
	if err != nil {
		t.Fatalf("UnmarshalJSONEnvelope() error = %v", err)
	}
	if got.EventID != original.EventID || got.EventType != original.EventType {
		t.Fatalf("mismatch: %+v", got)
	}
}

func TestDLQMessageRoundTrip(t *testing.T) {
	t.Parallel()
	envelope := events.Envelope{
		EventID: "evt_dlqtest000001", EventType: events.TypeOrganizationCreated,
		EventVersion: events.VersionV1, AggregateID: "org_1", AggregateType: "organization",
		OrganizationID: "org_1", OccurredAt: time.Now().UTC(), Producer: "test", Payload: []byte(`{}`),
	}
	msg := events.NewDLQMessage(envelope, "halcyon.organizations", 0, 42, "test-group", "bad payload", "permanent", 3)
	data, err := events.MarshalDLQMessage(msg)
	if err != nil {
		t.Fatalf("MarshalDLQMessage() error = %v", err)
	}
	got, err := events.UnmarshalDLQMessage(data)
	if err != nil {
		t.Fatalf("UnmarshalDLQMessage() error = %v", err)
	}
	if got.FailureCategory != "permanent" || got.OriginalTopic != "halcyon.organizations" {
		t.Fatalf("dlq mismatch: %+v", got)
	}
}
