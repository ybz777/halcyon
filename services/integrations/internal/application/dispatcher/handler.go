package dispatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobEnqueuer interface {
	Enqueue(ctx context.Context, input platformjobs.CreateInput) (platformjobs.Job, bool, error)
}

type Handler struct {
	endpoints  *postgres.EndpointRepository
	deliveries *postgres.DeliveryRepository
	pool       *pgxpool.Pool
	group      string
	jobs       JobEnqueuer
}

func NewHandler(endpoints *postgres.EndpointRepository, deliveries *postgres.DeliveryRepository, pool *pgxpool.Pool, group string, jobs JobEnqueuer) *Handler {
	return &Handler{endpoints: endpoints, deliveries: deliveries, pool: pool, group: group, jobs: jobs}
}

func (h *Handler) Handle(ctx context.Context, envelope platformevents.Envelope) error {
	if envelope.OrganizationID == "" {
		return nil
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var claimed bool
	err = tx.QueryRow(ctx, `
		INSERT INTO processed_events (event_id, consumer_group, organization_id, status)
		VALUES ($1, $2, $3, 'processing')
		ON CONFLICT (event_id, consumer_group) DO NOTHING
		RETURNING true
	`, envelope.EventID, h.group, envelope.OrganizationID).Scan(&claimed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}

	endpoints, err := h.endpoints.ListActiveByOrganization(ctx, envelope.OrganizationID)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(map[string]any{
		"id":             envelope.EventID,
		"type":           envelope.EventType,
		"organizationId": envelope.OrganizationID,
		"occurredAt":     envelope.OccurredAt.UTC().Format(time.RFC3339Nano),
		"data":           json.RawMessage(envelope.Payload),
	})
	if err != nil {
		return err
	}

	created := 0
	now := time.Now().UTC()
	for _, endpoint := range endpoints {
		if !endpoint.SubscribesTo(envelope.EventType) {
			continue
		}
		if created >= domain.MaxDeliveriesPerEvent {
			break
		}
		deliveryID, err := domain.NewDeliveryID()
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			INSERT INTO webhook_deliveries (
				id, webhook_endpoint_id, organization_id, event_id, event_type, payload,
				attempt_count, status, available_at, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,0,'PENDING',$7,$7)
			ON CONFLICT (webhook_endpoint_id, event_id) DO NOTHING
		`, deliveryID.String(), endpoint.ID.String(), envelope.OrganizationID, envelope.EventID,
			envelope.EventType, payload, now)
		if err != nil {
			return err
		}
		if tag.RowsAffected() > 0 {
			created++
			if h.jobs != nil {
				payload, err := platformjobs.MarshalWebhookDelivery(deliveryID.String())
				if err != nil {
					return err
				}
				if _, _, err := h.jobs.Enqueue(ctx, platformjobs.CreateInput{
					OrganizationID: envelope.OrganizationID,
					Type:           platformjobs.TypeWebhookDelivery,
					Payload:        payload,
					IdempotencyKey: deliveryID.String(),
					Priority:       platformjobs.PriorityNormal,
					MaxAttempts:    6,
					AvailableAt:    now,
				}); err != nil {
					return fmt.Errorf("enqueue webhook job: %w", err)
				}
			}
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE processed_events SET status='completed', processed_at=now()
		WHERE event_id=$1 AND consumer_group=$2
	`, envelope.EventID, h.group)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	return tx.Commit(ctx)
}
