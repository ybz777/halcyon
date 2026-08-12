package events_test

import (
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
)

func TestSerializeEnvelopeRoundTrip(t *testing.T) {
	t.Parallel()

	original := events.Envelope{
		EventID:        "evt_test12345678901",
		EventType:      events.TypeUserRegistered,
		EventVersion:   events.VersionV1,
		AggregateID:    "usr_test12345678901",
		AggregateType:  "user",
		OrganizationID: events.SystemOrganizationID,
		CorrelationID:  "corr_abc",
		CausationID:    "evt_parent",
		OccurredAt:     time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
		Producer:       "auth",
		Payload:        mustPayload(t, events.TypeUserRegistered, events.UserRegisteredPayload{UserID: "usr_test12345678901", Email: "a@example.com"}),
		TraceID:        "trace123",
		SpanID:         "span456",
	}

	data, err := events.SerializeEnvelope(original)
	if err != nil {
		t.Fatalf("SerializeEnvelope() error = %v", err)
	}

	got, err := events.DeserializeEnvelope(data)
	if err != nil {
		t.Fatalf("DeserializeEnvelope() error = %v", err)
	}

	if got.EventID != original.EventID || got.EventType != original.EventType || got.EventVersion != original.EventVersion {
		t.Fatalf("round trip mismatch: got %+v want %+v", got, original)
	}
}

func mustPayload(t *testing.T, eventType string, payload any) []byte {
	t.Helper()
	data, err := events.MarshalPayload(eventType, payload)
	if err != nil {
		t.Fatalf("MarshalPayload() error = %v", err)
	}
	return data
}

func TestPayloadCodecRoundTrip(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		eventType string
		payload   any
	}{
		{
			name:      "user registered",
			eventType: events.TypeUserRegistered,
			payload:   events.UserRegisteredPayload{UserID: "usr_1", Email: "a@example.com"},
		},
		{
			name:      "organization created",
			eventType: events.TypeOrganizationCreated,
			payload:   events.OrganizationCreatedPayload{OrganizationID: "org_1", Name: "Acme", Slug: "acme"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data, err := events.MarshalPayload(tc.eventType, tc.payload)
			if err != nil {
				t.Fatalf("MarshalPayload() error = %v", err)
			}
			got, err := events.UnmarshalPayload(tc.eventType, data)
			if err != nil {
				t.Fatalf("UnmarshalPayload() error = %v", err)
			}
			if got != tc.payload {
				t.Fatalf("payload mismatch: got %+v want %+v", got, tc.payload)
			}
		})
	}
}

func TestDefaultRegistry(t *testing.T) {
	t.Parallel()

	reg := events.DefaultRegistry()
	for _, eventType := range []string{
		events.TypeUserRegistered,
		events.TypeUserLoggedIn,
		events.TypeAPIKeyCreated,
		events.TypeAPIKeyRevoked,
		events.TypeAPIKeyRotated,
		events.TypeOrganizationCreated,
		events.TypeMemberInvited,
		events.TypeMemberJoined,
	} {
		if _, ok := reg.Lookup(eventType); !ok {
			t.Fatalf("event type %q not registered", eventType)
		}
	}
}

func TestTopicForEventType(t *testing.T) {
	t.Parallel()

	if got := events.TopicForEventType(events.TypeUserRegistered); got != "halcyon.auth" {
		t.Fatalf("topic = %q, want halcyon.auth", got)
	}
	if got := events.TopicForEventType(events.TypeOrganizationCreated); got != "halcyon.organizations" {
		t.Fatalf("topic = %q, want halcyon.organizations", got)
	}
}
