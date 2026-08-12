package audit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/services/audit/internal/application/ports"
)

// IndexUseCase persists audit events from platform envelopes.
type IndexUseCase struct {
	indexer ports.TransactionalIndexer
	metrics AuditMetrics
	log     *slog.Logger
}

// AuditMetrics records audit consumer metrics.
type AuditMetrics interface {
	RecordConsumed(eventType string)
	RecordPersisted(eventType, sourceService, result string)
	RecordFailed(eventType string)
	RecordDuplicate(eventType string)
	ObservePersistence(eventType string, seconds float64)
}

type noopMetrics struct{}

func (noopMetrics) RecordConsumed(string)                             {}
func (noopMetrics) RecordPersisted(string, string, string)            {}
func (noopMetrics) RecordFailed(string)                               {}
func (noopMetrics) RecordDuplicate(string)                            {}
func (noopMetrics) ObservePersistence(string, float64)                {}

// NewIndexUseCase creates an IndexUseCase.
func NewIndexUseCase(indexer ports.TransactionalIndexer, metrics AuditMetrics, log *slog.Logger) *IndexUseCase {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &IndexUseCase{indexer: indexer, metrics: metrics, log: log}
}

// HandleEnvelope maps and persists an audit event idempotently.
func (uc *IndexUseCase) HandleEnvelope(ctx context.Context, envelope platformevents.Envelope, consumerGroup, sourceService string) error {
	uc.metrics.RecordConsumed(envelope.EventType)

	event, ok, err := MapEnvelope(envelope, sourceService)
	if err != nil {
		uc.metrics.RecordFailed(envelope.EventType)
		return fmt.Errorf("map envelope: %w", err)
	}
	if !ok {
		return nil
	}

	start := time.Now()
	duplicate, err := uc.indexer.PersistIdempotent(ctx, envelope.EventID, consumerGroup, event.OrganizationID, event)
	if err != nil {
		uc.metrics.RecordFailed(envelope.EventType)
		return err
	}
	if duplicate {
		uc.metrics.RecordDuplicate(envelope.EventType)
		uc.log.Info("audit_event_duplicate",
			slog.String("event_id", envelope.EventID),
			slog.String("event_type", envelope.EventType),
			slog.String("organization_id", event.OrganizationID),
			slog.String("consumer_group", consumerGroup),
		)
		return nil
	}

	uc.metrics.RecordPersisted(event.EventType, event.SourceService, string(event.Result))
	uc.metrics.ObservePersistence(event.EventType, time.Since(start).Seconds())
	uc.log.Info("audit_event_persisted",
		slog.String("event_id", envelope.EventID),
		slog.String("event_type", event.EventType),
		slog.String("organization_id", event.OrganizationID),
		slog.String("actor_id", event.ActorID),
		slog.String("target_id", event.TargetID),
		slog.String("request_id", event.RequestID),
		slog.String("correlation_id", event.CorrelationID),
		slog.String("consumer_group", consumerGroup),
		slog.String("source_service", event.SourceService),
	)
	return nil
}
