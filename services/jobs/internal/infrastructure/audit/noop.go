package audit

import "context"

// NoopPublisher discards audit events.
type NoopPublisher struct{}

func (NoopPublisher) PublishJobCancelled(context.Context, string, string, string) {}
