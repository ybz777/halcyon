package events

import (
	"context"
	"time"
)

// DomainEvent is the platform abstraction for domain events.
// Service domain layers map their events to this interface at the application boundary.
type DomainEvent interface {
	EventType() string
	EventVersion() string
	AggregateID() string
	AggregateType() string
	OrganizationID() string
	OccurredAt() time.Time
	Payload() ([]byte, error)
}

// Publisher stores events for eventual delivery. Implementations must use the outbox pattern.
type Publisher interface {
	Publish(ctx context.Context, event DomainEvent) error
}

// Handler processes a deserialized envelope.
type Handler interface {
	Handle(ctx context.Context, envelope Envelope) error
}

// HandlerFunc adapts a function to Handler.
type HandlerFunc func(ctx context.Context, envelope Envelope) error

// Handle implements Handler.
func (f HandlerFunc) Handle(ctx context.Context, envelope Envelope) error {
	return f(ctx, envelope)
}

// Consumer subscribes to events from a topic.
type Consumer interface {
	Run(ctx context.Context, topic string, groupID string, handler Handler) error
}

// Metadata carries cross-cutting event context.
type Metadata struct {
	CorrelationID string
	RequestID     string
	CausationID   string
	ActorID       string
	SubjectID     string
	Producer      string
	TraceID       string
	SpanID        string
}

// Record is a concrete domain event implementation for registry-defined types.
type Record struct {
	Type           string
	Version        string
	AggregateIDVal      string
	AggregateTypeVal    string
	OrganizationIDVal   string
	At             time.Time
	PayloadBytes   []byte
}

func (r Record) EventType() string        { return r.Type }
func (r Record) EventVersion() string     { return r.Version }
func (r Record) AggregateID() string      { return r.AggregateIDVal }
func (r Record) AggregateType() string    { return r.AggregateTypeVal }
func (r Record) OrganizationID() string   { return r.OrganizationIDVal }
func (r Record) OccurredAt() time.Time    { return r.At }
func (r Record) Payload() ([]byte, error) { return r.PayloadBytes, nil }
