package events

import (
	"context"
	"encoding/json"
	"fmt"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxPublisher stores domain events in the transactional outbox.
type OutboxPublisher struct {
	pool *pgxpool.Pool
}

// NewOutboxPublisher creates an outbox-backed event publisher.
func NewOutboxPublisher(pool *pgxpool.Pool) *OutboxPublisher {
	return &OutboxPublisher{pool: pool}
}

// Publish implements ports.EventPublisher.
func (p *OutboxPublisher) Publish(ctx context.Context, event domain.Event) error {
	record, err := toPlatformRecord(event)
	if err != nil {
		return err
	}

	if tx, ok := postgres.TxFromContext(ctx); ok {
		store := outbox.NewTxStore(tx)
		return outbox.NewPublisher(store, "workflow").Publish(ctx, record)
	}

	store := outbox.NewPostgresStore(p.pool)
	return outbox.NewPublisher(store, "workflow").Publish(ctx, record)
}

var _ ports.EventPublisher = (*OutboxPublisher)(nil)

type eventRecord struct {
	EventName string         `json:"event_name"`
	Payload   map[string]any `json:"payload"`
}

func toPlatformRecord(event domain.Event) (platformevents.Record, error) {
	payload, orgID, aggregateID, err := eventPayload(event)
	if err != nil {
		return platformevents.Record{}, err
	}

	bytes, err := json.Marshal(eventRecord{EventName: event.EventName(), Payload: payload})
	if err != nil {
		return platformevents.Record{}, fmt.Errorf("marshal event payload: %w", err)
	}

	return platformevents.Record{
		Type:              event.EventName(),
		Version:           platformevents.VersionV1,
		AggregateIDVal:    aggregateID,
		AggregateTypeVal:  aggregateTypeFor(event),
		OrganizationIDVal: orgID,
		At:                event.OccurredAt().UTC(),
		PayloadBytes:      bytes,
	}, nil
}

func aggregateTypeFor(event domain.Event) string {
	if _, ok := event.(domain.WorkflowEvent); ok {
		return "workflow_execution"
	}
	return "incident"
}

func eventPayload(event domain.Event) (map[string]any, string, string, error) {
	switch e := event.(type) {
	case domain.IncidentCreated:
		return map[string]any{
			"incident_id": e.IncidentID.String(),
			"title":       e.Title.String(),
			"severity":    e.Severity.String(),
			"priority":    e.Priority.String(),
			"source":      e.Source.String(),
		}, e.OrganizationID.String(), e.IncidentID.String(), nil
	case domain.IncidentAcknowledged:
		return map[string]any{"incident_id": e.IncidentID.String()}, e.OrganizationID.String(), e.IncidentID.String(), nil
	case domain.IncidentEscalated:
		return map[string]any{
			"incident_id":   e.IncidentID.String(),
			"from_severity": e.FromSeverity.String(),
			"to_severity":   e.ToSeverity.String(),
		}, e.OrganizationID.String(), e.IncidentID.String(), nil
	case domain.IncidentResolved:
		return map[string]any{"incident_id": e.IncidentID.String()}, e.OrganizationID.String(), e.IncidentID.String(), nil
	case domain.IncidentClosed:
		return map[string]any{"incident_id": e.IncidentID.String()}, e.OrganizationID.String(), e.IncidentID.String(), nil
	case domain.StateChanged:
		return map[string]any{
			"incident_id": e.IncidentID.String(),
			"from_status": e.FromStatus.String(),
			"to_status":   e.ToStatus.String(),
		}, e.OrganizationID.String(), e.IncidentID.String(), nil
	case domain.WorkflowExecutionStarted:
		return map[string]any{
			"execution_id": e.ExecutionID().String(), "workflow_id": e.WorkflowID().String(),
			"correlation_id": e.CorrelationID, "triggered_by": e.TriggeredBy,
		}, e.OrganizationID().String(), e.ExecutionID().String(), nil
	case domain.WorkflowExecutionSucceeded:
		return map[string]any{"execution_id": e.ExecutionID().String(), "workflow_id": e.WorkflowID().String()},
			e.OrganizationID().String(), e.ExecutionID().String(), nil
	case domain.WorkflowExecutionFailed:
		return map[string]any{"execution_id": e.ExecutionID().String(), "workflow_id": e.WorkflowID().String(), "reason": e.Reason},
			e.OrganizationID().String(), e.ExecutionID().String(), nil
	case domain.WorkflowExecutionCancelled:
		return map[string]any{"execution_id": e.ExecutionID().String(), "workflow_id": e.WorkflowID().String()},
			e.OrganizationID().String(), e.ExecutionID().String(), nil
	case domain.WorkflowStepStarted:
		return map[string]any{"execution_id": e.ExecutionID().String(), "step_key": e.StepKey(), "attempt": e.Attempt},
			e.OrganizationID().String(), e.ExecutionID().String(), nil
	case domain.WorkflowStepSucceeded:
		return map[string]any{"execution_id": e.ExecutionID().String(), "step_key": e.StepKey()},
			e.OrganizationID().String(), e.ExecutionID().String(), nil
	case domain.WorkflowStepFailed:
		return map[string]any{"execution_id": e.ExecutionID().String(), "step_key": e.StepKey(), "reason": e.Reason},
			e.OrganizationID().String(), e.ExecutionID().String(), nil
	case domain.PlatformEmitEvent:
		return map[string]any{
			"execution_id": e.ExecutionID.String(), "event_type": e.EventType, "payload": e.Payload,
		}, e.OrganizationID.String(), e.ExecutionID.String(), nil
	default:
		return nil, "", "", fmt.Errorf("unsupported event type %T", event)
	}
}

// NoopPublisher discards events (for tests without outbox).
type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, domain.Event) error { return nil }

var _ ports.EventPublisher = NoopPublisher{}
