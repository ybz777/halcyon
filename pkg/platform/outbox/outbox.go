package outbox

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/contextx"
	"github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/messaging"
)

// Status represents the publication state of an outbox entry.
type Status string

const (
	StatusPending    Status = "pending"
	StatusPublishing Status = "publishing"
	StatusPublished  Status = "published"
	StatusFailed     Status = "failed"
)

// Entry is a persisted outbox record awaiting relay to Kafka.
type Entry struct {
	ID             string
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
	Topic          string
	PartitionKey   string
	Status         Status
	Attempts       int
	LastError      string
	NextAttemptAt  *time.Time
	CreatedAt      time.Time
	PublishedAt    *time.Time
}

// Store persists and retrieves outbox entries.
type Store interface {
	Insert(ctx context.Context, entry Entry) error
	RequeueStalePublishing(ctx context.Context, olderThan time.Duration) error
	ClaimPending(ctx context.Context, limit int) ([]Entry, error)
	MarkPublished(ctx context.Context, id string, publishedAt time.Time) error
	MarkFailed(ctx context.Context, id string, attempts int, lastError string, nextAttemptAt *time.Time) error
	MarkDeadLetter(ctx context.Context, id string, lastError string) error
	PendingCount(ctx context.Context) (int64, error)
}

// Publisher stores domain events in the outbox within the caller's transaction.
type Publisher struct {
	store    Store
	producer string
	metrics  *messaging.Metrics
}

// NewPublisher creates an outbox-backed event publisher.
func NewPublisher(store Store, producer string) *Publisher {
	return &Publisher{store: store, producer: producer}
}

// WithMetrics attaches observability metrics to the publisher.
func (p *Publisher) WithMetrics(metrics *messaging.Metrics) *Publisher {
	p.metrics = metrics
	return p
}

// Publish implements events.Publisher by writing to the outbox table.
func (p *Publisher) Publish(ctx context.Context, event events.DomainEvent) error {
	meta := events.Metadata{Producer: p.producer}
	if id := contextx.CorrelationIDFromContext(ctx); id != "" {
		meta.CorrelationID = id
	}
	if id := contextx.CausationIDFromContext(ctx); id != "" {
		meta.CausationID = id
	}
	if id := contextx.RequestIDFromContext(ctx); id != "" {
		meta.RequestID = id
	}

	envelope, err := events.NewEnvelope(event, meta)
	if err != nil {
		return err
	}

	entry := Entry{
		EventID:        envelope.EventID,
		EventType:      envelope.EventType,
		EventVersion:   envelope.EventVersion,
		AggregateID:    envelope.AggregateID,
		AggregateType:  envelope.AggregateType,
		OrganizationID: envelope.OrganizationID,
		CorrelationID:  envelope.CorrelationID,
		RequestID:      envelope.RequestID,
		ActorID:        envelope.ActorID,
		SubjectID:      envelope.SubjectID,
		CausationID:    envelope.CausationID,
		OccurredAt:     envelope.OccurredAt,
		Producer:       envelope.Producer,
		Payload:        envelope.Payload,
		TraceID:        envelope.TraceID,
		SpanID:         envelope.SpanID,
		Topic:          events.TopicForEventType(envelope.EventType),
		PartitionKey:   partitionKeyFor(envelope),
		Status:         StatusPending,
	}

	if err := p.store.Insert(ctx, entry); err != nil {
		return err
	}
	if p.metrics != nil {
		p.metrics.RecordOutboxCreated(envelope.EventType)
	}
	return nil
}

func partitionKeyFor(envelope events.Envelope) string {
	if envelope.OrganizationID != "" && envelope.OrganizationID != events.SystemOrganizationID {
		return envelope.OrganizationID
	}
	if envelope.AggregateID != "" {
		return envelope.AggregateID
	}
	return envelope.EventID
}

// EntryFromEnvelope converts an envelope to an outbox entry.
func EntryFromEnvelope(envelope events.Envelope, topic, partitionKey string) Entry {
	return Entry{
		EventID:        envelope.EventID,
		EventType:      envelope.EventType,
		EventVersion:   envelope.EventVersion,
		AggregateID:    envelope.AggregateID,
		AggregateType:  envelope.AggregateType,
		OrganizationID: envelope.OrganizationID,
		CorrelationID:  envelope.CorrelationID,
		RequestID:      envelope.RequestID,
		ActorID:        envelope.ActorID,
		SubjectID:      envelope.SubjectID,
		CausationID:    envelope.CausationID,
		OccurredAt:     envelope.OccurredAt,
		Producer:       envelope.Producer,
		Payload:        envelope.Payload,
		TraceID:        envelope.TraceID,
		SpanID:         envelope.SpanID,
		Topic:          topic,
		PartitionKey:   partitionKey,
		Status:         StatusPending,
	}
}

// Envelope converts an outbox entry back to an event envelope.
func (e Entry) Envelope() events.Envelope {
	return events.Envelope{
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
		OccurredAt:     e.OccurredAt,
		Producer:       e.Producer,
		Payload:        e.Payload,
		TraceID:        e.TraceID,
		SpanID:         e.SpanID,
	}
}
