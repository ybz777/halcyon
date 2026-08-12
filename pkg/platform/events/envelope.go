package events

import (
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/eventtopics"
	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

// Envelope is the canonical event wrapper for Kafka messages.
type Envelope struct {
	EventID        string
	EventType      string
	EventVersion   string
	AggregateID    string
	AggregateType  string
	OrganizationID string
	CorrelationID  string
	RequestID      string
	ActorID        string
	SubjectID      string
	CausationID    string
	OccurredAt     time.Time
	Producer       string
	Payload        []byte
	TraceID        string
	SpanID         string
}

// NewEnvelope builds an envelope from a domain event and metadata.
func NewEnvelope(event DomainEvent, meta Metadata) (Envelope, error) {
	payload, err := event.Payload()
	if err != nil {
		return Envelope{}, fmt.Errorf("marshal payload: %w", err)
	}

	eventID, err := ids.Generate("evt")
	if err != nil {
		return Envelope{}, fmt.Errorf("generate event id: %w", err)
	}

	correlationID := meta.CorrelationID
	if correlationID == "" {
		correlationID = eventID
	}

	return Envelope{
		EventID:        eventID,
		EventType:      event.EventType(),
		EventVersion:   event.EventVersion(),
		AggregateID:    event.AggregateID(),
		AggregateType:  event.AggregateType(),
		OrganizationID: event.OrganizationID(),
		CorrelationID:  correlationID,
		RequestID:      meta.RequestID,
		ActorID:        meta.ActorID,
		SubjectID:      meta.SubjectID,
		CausationID:    meta.CausationID,
		OccurredAt:     event.OccurredAt().UTC(),
		Producer:       meta.Producer,
		Payload:        payload,
		TraceID:        meta.TraceID,
		SpanID:         meta.SpanID,
	}, nil
}

// TopicForEventType maps an event type to its Kafka topic.
func TopicForEventType(eventType string) string {
	return eventtopics.TopicForEventType(eventType)
}

// DLQTopicFor returns the dead-letter topic for a source topic.
func DLQTopicFor(topic string) string {
	return eventtopics.DLQTopicFor(topic)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
