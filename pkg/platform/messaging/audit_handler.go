package messaging

import (
	"context"
	"log/slog"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
)

// AuditHandler logs organization domain events for audit indexing.
func AuditHandler(log *slog.Logger) events.Handler {
	if log == nil {
		log = slog.Default()
	}
	return events.HandlerFunc(func(ctx context.Context, envelope events.Envelope) error {
		log.Info("audit_event_indexed",
			slog.String("event_id", envelope.EventID),
			slog.String("event_type", envelope.EventType),
			slog.String("event_version", envelope.EventVersion),
			slog.String("organization_id", envelope.OrganizationID),
			slog.String("actor_id", envelope.ActorID),
			slog.String("correlation_id", envelope.CorrelationID),
			slog.String("request_id", envelope.RequestID),
		)
		return nil
	})
}
