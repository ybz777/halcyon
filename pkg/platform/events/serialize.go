package events

import (
	"bytes"
	"encoding/json"
	"time"
)

// SerializeEnvelope marshals an envelope to JSON bytes (canonical format).
func SerializeEnvelope(e Envelope) ([]byte, error) {
	return MarshalJSONEnvelope(e)
}

// DeserializeEnvelope unmarshals envelope bytes (JSON or legacy protobuf).
func DeserializeEnvelope(data []byte) (Envelope, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return UnmarshalJSONEnvelope(trimmed)
	}
	e, err := unmarshalEnvelope(data)
	if err != nil {
		return Envelope{}, err
	}
	return e, nil
}

// SerializePayloadJSON marshals a typed payload to JSON bytes.
func SerializePayloadJSON(payload any) ([]byte, error) {
	return json.Marshal(payload)
}

// BuildRecord creates a platform Record from a typed payload.
func BuildRecord(eventType, aggregateID, aggregateType, orgID string, at time.Time, payload any) (Record, error) {
	bytes, err := MarshalPayload(eventType, payload)
	if err != nil {
		return Record{}, err
	}
	return Record{
		Type:           eventType,
		Version:        VersionV1,
		AggregateIDVal: aggregateID,
		AggregateTypeVal:  aggregateType,
		OrganizationIDVal: orgID,
		At:             at.UTC(),
		PayloadBytes:   bytes,
	}, nil
}
