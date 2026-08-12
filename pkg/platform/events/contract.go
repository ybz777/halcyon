package events

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ContentTypeJSON is the JSON envelope media type.
const ContentTypeJSON = "application/vnd.halcyon.event+json"

// ContentTypeProtobuf is the legacy protobuf envelope media type.
const ContentTypeProtobuf = "application/vnd.halcyon.event+protobuf"

// EventEnvelope is the canonical JSON-serializable event wrapper.
// Versioning uses separate event_type and event_version fields.
type EventEnvelope struct {
	EventID        string          `json:"event_id"`
	EventType      string          `json:"event_type"`
	EventVersion   string          `json:"event_version"`
	OccurredAt     time.Time       `json:"occurred_at"`
	Producer       string          `json:"producer"`
	CorrelationID  string          `json:"correlation_id"`
	RequestID      string          `json:"request_id,omitempty"`
	OrganizationID string          `json:"organization_id,omitempty"`
	ActorID        string          `json:"actor_id,omitempty"`
	SubjectID      string          `json:"subject_id,omitempty"`
	AggregateID    string          `json:"aggregate_id"`
	AggregateType  string          `json:"aggregate_type"`
	CausationID    string          `json:"causation_id,omitempty"`
	TraceID        string          `json:"trace_id,omitempty"`
	SpanID         string          `json:"span_id,omitempty"`
	Payload        json.RawMessage `json:"payload"`
}

// ToEnvelope converts a JSON envelope to the internal Envelope type.
func (e EventEnvelope) ToEnvelope() Envelope {
	return Envelope{
		EventID:        e.EventID,
		EventType:      e.EventType,
		EventVersion:   e.EventVersion,
		AggregateID:    e.AggregateID,
		AggregateType:  e.AggregateType,
		OrganizationID: e.OrganizationID,
		CorrelationID:  e.CorrelationID,
		RequestID:      e.RequestID,
		ActorID:        e.ActorID,
		SubjectID:      e.SubjectID,
		CausationID:    e.CausationID,
		OccurredAt:     e.OccurredAt.UTC(),
		Producer:       e.Producer,
		Payload:        envelopePayloadFromJSON(e.Payload),
		TraceID:        e.TraceID,
		SpanID:         e.SpanID,
	}
}

// FromEnvelope converts an internal envelope to JSON form.
func FromEnvelope(e Envelope) EventEnvelope {
	return EventEnvelope{
		EventID:        e.EventID,
		EventType:      e.EventType,
		EventVersion:   e.EventVersion,
		OccurredAt:     e.OccurredAt.UTC(),
		Producer:       e.Producer,
		CorrelationID:  e.CorrelationID,
		RequestID:      e.RequestID,
		OrganizationID: e.OrganizationID,
		ActorID:        e.ActorID,
		SubjectID:      e.SubjectID,
		AggregateID:    e.AggregateID,
		AggregateType:  e.AggregateType,
		CausationID:    e.CausationID,
		TraceID:        e.TraceID,
		SpanID:         e.SpanID,
		Payload:        envelopePayloadForJSON(e.Payload),
	}
}

func envelopePayloadForJSON(payload []byte) json.RawMessage {
	if len(payload) == 0 {
		return json.RawMessage(`""`)
	}
	trimmed := bytes.TrimSpace(payload)
	if json.Valid(trimmed) {
		return json.RawMessage(append([]byte(nil), payload...))
	}
	encoded, err := json.Marshal(base64.StdEncoding.EncodeToString(payload))
	if err != nil {
		return json.RawMessage(`""`)
	}
	return json.RawMessage(encoded)
}

func envelopePayloadFromJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return append([]byte(nil), raw...)
	}
	if trimmed[0] == '"' {
		var encoded string
		if err := json.Unmarshal(trimmed, &encoded); err == nil {
			if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil {
				return decoded
			}
		}
	}
	return append([]byte(nil), raw...)
}

// Validate checks required envelope fields before publish/consume.
func Validate(envelope Envelope) error {
	if strings.TrimSpace(envelope.EventID) == "" {
		return fmt.Errorf("%w: event_id required", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(envelope.EventType) == "" {
		return fmt.Errorf("%w: event_type required", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(envelope.EventVersion) == "" {
		return fmt.Errorf("%w: event_version required", ErrInvalidEnvelope)
	}
	if envelope.OccurredAt.IsZero() {
		return fmt.Errorf("%w: occurred_at required", ErrInvalidEnvelope)
	}
	if strings.TrimSpace(envelope.Producer) == "" {
		return fmt.Errorf("%w: producer required", ErrInvalidEnvelope)
	}
	if len(envelope.Payload) == 0 {
		return fmt.Errorf("%w: payload required", ErrInvalidEnvelope)
	}
	if requiresOrganization(envelope.EventType) && strings.TrimSpace(envelope.OrganizationID) == "" {
		return fmt.Errorf("%w: organization_id required for %s", ErrInvalidEnvelope, envelope.EventType)
	}
	if _, ok := DefaultRegistry().Lookup(envelope.EventType); !ok {
		return fmt.Errorf("%w: unsupported event_type %q", ErrUnsupportedVersion, envelope.EventType)
	}
	return nil
}

func requiresOrganization(eventType string) bool {
	switch {
	case hasPrefix(eventType, "organizations."):
		return eventType != TypeOrganizationCreated
	case hasPrefix(eventType, "incidents."):
		return true
	default:
		return false
	}
}

// MarshalJSONEnvelope serializes an envelope as JSON.
func MarshalJSONEnvelope(e Envelope) ([]byte, error) {
	if err := Validate(e); err != nil {
		return nil, err
	}
	return json.Marshal(FromEnvelope(e))
}

// UnmarshalJSONEnvelope deserializes a JSON envelope.
func UnmarshalJSONEnvelope(data []byte) (Envelope, error) {
	var wire EventEnvelope
	if err := json.Unmarshal(data, &wire); err != nil {
		return Envelope{}, fmt.Errorf("unmarshal json envelope: %w", err)
	}
	envelope := wire.ToEnvelope()
	if err := Validate(envelope); err != nil {
		return Envelope{}, err
	}
	return envelope, nil
}
