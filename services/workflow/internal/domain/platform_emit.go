package domain

import (
	"encoding/json"
	"time"
)

// PlatformEmitEvent is emitted by EVENT_EMIT workflow steps via outbox.
type PlatformEmitEvent struct {
	EventType      string
	OrganizationID OrganizationID
	ExecutionID    ExecutionID
	Payload        json.RawMessage
	at             time.Time
}

func NewPlatformEmitEvent(eventType string, org OrganizationID, execution ExecutionID, payload json.RawMessage, at time.Time) PlatformEmitEvent {
	return PlatformEmitEvent{
		EventType: eventType, OrganizationID: org, ExecutionID: execution, Payload: payload, at: at,
	}
}

func (e PlatformEmitEvent) EventName() string     { return e.EventType }
func (e PlatformEmitEvent) OccurredAt() time.Time { return e.at }
