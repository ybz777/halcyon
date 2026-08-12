package postgres

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeliveryRepository struct {
	pool *pgxpool.Pool
}

func NewDeliveryRepository(pool *pgxpool.Pool) *DeliveryRepository {
	return &DeliveryRepository{pool: pool}
}

func (r *DeliveryRepository) CreateIdempotent(ctx context.Context, delivery domain.WebhookDelivery) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO webhook_deliveries (
			id, webhook_endpoint_id, organization_id, event_id, event_type, payload,
			attempt_count, status, available_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (webhook_endpoint_id, event_id) DO NOTHING
	`, delivery.ID.String(), delivery.EndpointID.String(), delivery.OrganizationID,
		delivery.EventID, delivery.EventType, delivery.Payload, delivery.AttemptCount,
		string(delivery.Status), delivery.AvailableAt.UTC(), delivery.CreatedAt.UTC())
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 0, nil
}

func (r *DeliveryRepository) Claim(ctx context.Context, workerID string, batchSize int, lease time.Duration) ([]domain.WebhookDelivery, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	leaseUntil := now.Add(lease)
	rows, err := tx.Query(ctx, `
		SELECT id, webhook_endpoint_id, organization_id, event_id, event_type, payload,
			attempt_count, status, available_at, last_attempt_at, next_attempt_at,
			response_status, response_time_ms, last_error, locked_at, locked_by, created_at, completed_at
		FROM webhook_deliveries
		WHERE status IN ('PENDING', 'RETRY_WAIT')
			AND COALESCE(next_attempt_at, available_at) <= $1
			AND (locked_at IS NULL OR locked_at < $2)
		ORDER BY COALESCE(next_attempt_at, available_at)
		FOR UPDATE SKIP LOCKED
		LIMIT $3
	`, now, now.Add(-lease), batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	deliveries := make([]domain.WebhookDelivery, 0, batchSize)
	ids := make([]string, 0, batchSize)
	for rows.Next() {
		delivery, err := scanDelivery(rows)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, delivery)
		ids = append(ids, delivery.ID.String())
	}
	if len(ids) == 0 {
		return nil, tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status='DELIVERING', locked_at=$2, locked_by=$3
		WHERE id = ANY($1)
	`, ids, leaseUntil, workerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *DeliveryRepository) Complete(ctx context.Context, delivery domain.WebhookDelivery) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status=$2, attempt_count=$3, last_attempt_at=$4, next_attempt_at=$5,
			response_status=$6, response_time_ms=$7, last_error=$8,
			locked_at=NULL, locked_by='', completed_at=$9
		WHERE id=$1
	`, delivery.ID.String(), string(delivery.Status), delivery.AttemptCount, delivery.LastAttemptAt,
		delivery.NextAttemptAt, delivery.ResponseStatus, delivery.ResponseTimeMs, delivery.LastError, delivery.CompletedAt)
	return err
}

func (r *DeliveryRepository) RecoverLeases(ctx context.Context, lease time.Duration) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = CASE WHEN status='DELIVERING' THEN 'RETRY_WAIT' ELSE status END,
			locked_at = NULL, locked_by = ''
		WHERE status='DELIVERING' AND locked_at < $1
	`, time.Now().UTC().Add(-lease))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *DeliveryRepository) FindByID(ctx context.Context, orgID string, id domain.DeliveryID) (domain.WebhookDelivery, error) {
	row := r.pool.QueryRow(ctx, deliverySelectSQL()+` WHERE id=$1 AND organization_id=$2`, id.String(), orgID)
	return scanDelivery(row)
}

func (r *DeliveryRepository) FindByIDOnly(ctx context.Context, id domain.DeliveryID) (domain.WebhookDelivery, error) {
	row := r.pool.QueryRow(ctx, deliverySelectSQL()+` WHERE id=$1`, id.String())
	return scanDelivery(row)
}

func (r *DeliveryRepository) ListByEndpoint(ctx context.Context, orgID string, endpointID domain.EndpointID, limit int, cursor string) ([]domain.WebhookDelivery, string, error) {
	return listDeliveriesPage(ctx, r.pool, endpointID.String(), orgID, limit, cursor)
}

func (r *DeliveryRepository) CountReady(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM webhook_deliveries
		WHERE status IN ('PENDING', 'RETRY_WAIT') AND COALESCE(next_attempt_at, available_at) <= now()
	`).Scan(&count)
	return count, err
}

func deliverySelectSQL() string {
	return `
		SELECT id, webhook_endpoint_id, organization_id, event_id, event_type, payload,
			attempt_count, status, available_at, last_attempt_at, next_attempt_at,
			response_status, response_time_ms, last_error, locked_at, locked_by, created_at, completed_at
		FROM webhook_deliveries`
}

func scanDelivery(row rowScanner) (domain.WebhookDelivery, error) {
	var (
		id, endpointID, orgID, eventID, eventType, status, lastError, lockedBy string
		payload                                                                 []byte
		attemptCount                                                            int
		availableAt, createdAt                                                  time.Time
		lastAttemptAt, nextAttemptAt, lockedAt, completedAt                     *time.Time
		responseStatus, responseTimeMs                                            *int
	)
	if err := row.Scan(&id, &endpointID, &orgID, &eventID, &eventType, &payload,
		&attemptCount, &status, &availableAt, &lastAttemptAt, &nextAttemptAt,
		&responseStatus, &responseTimeMs, &lastError, &lockedAt, &lockedBy, &createdAt, &completedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.WebhookDelivery{}, domain.ErrDeliveryNotFound
		}
		return domain.WebhookDelivery{}, err
	}
	parsedID, _ := domain.ParseDeliveryID(id)
	parsedEndpointID, _ := domain.ParseEndpointID(endpointID)
	return domain.WebhookDelivery{
		ID: parsedID, EndpointID: parsedEndpointID, OrganizationID: orgID,
		EventID: eventID, EventType: eventType, Payload: payload,
		AttemptCount: attemptCount, Status: domain.DeliveryStatus(status),
		AvailableAt: availableAt.UTC(), LastAttemptAt: lastAttemptAt, NextAttemptAt: nextAttemptAt,
		ResponseStatus: responseStatus, ResponseTimeMs: responseTimeMs, LastError: lastError,
		LockedAt: lockedAt, LockedBy: lockedBy, CreatedAt: createdAt.UTC(), CompletedAt: completedAt,
	}, nil
}
