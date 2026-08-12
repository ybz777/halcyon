package ports

import (
	"context"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
)

// EventPublisher stores domain events for asynchronous delivery.
type EventPublisher interface {
	Publish(ctx context.Context, event platformevents.DomainEvent) error
}

// NoopEventPublisher discards events (tests and degraded mode).
type NoopEventPublisher struct{}

func (NoopEventPublisher) Publish(context.Context, platformevents.DomainEvent) error { return nil }
