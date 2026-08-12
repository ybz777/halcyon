package postgres

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailDeliveryRepository struct {
	pool *pgxpool.Pool
}

func NewEmailDeliveryRepository(pool *pgxpool.Pool) *EmailDeliveryRepository {
	return &EmailDeliveryRepository{pool: pool}
}

func (r *EmailDeliveryRepository) Claim(ctx context.Context, workerID string, batchSize int, lease time.Duration) ([]domain.EmailDelivery, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	leaseUntil := now.Add(lease)
	rows, err := tx.Query(ctx, `
		SELECT id, notification_id, user_id, email, template, status, attempt_count,
			available_at, last_attempt_at, next_attempt_at, provider, provider_message_id,
			last_error, locked_at, locked_by, created_at, completed_at
		FROM email_deliveries
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

	deliveries := make([]domain.EmailDelivery, 0, batchSize)
	ids := make([]string, 0, batchSize)
	for rows.Next() {
		d, err := scanEmailDelivery(rows)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, d)
		ids = append(ids, d.ID.String())
	}
	if len(ids) == 0 {
		return nil, tx.Commit(ctx)
	}
	_, err = tx.Exec(ctx, `
		UPDATE email_deliveries SET status='DELIVERING', locked_at=$2, locked_by=$3 WHERE id = ANY($1)
	`, ids, leaseUntil, workerID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *EmailDeliveryRepository) Complete(ctx context.Context, delivery domain.EmailDelivery) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE email_deliveries
		SET status=$2, attempt_count=$3, last_attempt_at=$4, next_attempt_at=$5,
			provider=$6, provider_message_id=$7, last_error=$8,
			locked_at=NULL, locked_by='', completed_at=$9
		WHERE id=$1
	`, delivery.ID.String(), string(delivery.Status), delivery.AttemptCount, delivery.LastAttemptAt,
		delivery.NextAttemptAt, delivery.Provider, delivery.ProviderMessageID, delivery.LastError, delivery.CompletedAt)
	return err
}

func (r *EmailDeliveryRepository) RecoverLeases(ctx context.Context, lease time.Duration) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE email_deliveries
		SET status = CASE WHEN status='DELIVERING' THEN 'RETRY_WAIT' ELSE status END,
			locked_at=NULL, locked_by=''
		WHERE status='DELIVERING' AND locked_at < $1
	`, time.Now().UTC().Add(-lease))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *EmailDeliveryRepository) FindByIDOnly(ctx context.Context, id domain.EmailDeliveryID) (domain.EmailDelivery, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, notification_id, user_id, email, template, status, attempt_count,
			available_at, last_attempt_at, next_attempt_at, provider, provider_message_id,
			last_error, locked_at, locked_by, created_at, completed_at
		FROM email_deliveries WHERE id=$1
	`, id.String())
	return scanEmailDelivery(row)
}

func (r *EmailDeliveryRepository) CountReady(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM email_deliveries
		WHERE status IN ('PENDING', 'RETRY_WAIT') AND COALESCE(next_attempt_at, available_at) <= now()
	`).Scan(&count)
	return count, err
}

func scanEmailDelivery(row rowScanner) (domain.EmailDelivery, error) {
	var (
		id, notificationID, userID, email, template, status, provider, providerMessageID, lastError, lockedBy string
		attemptCount                                                                                            int
		availableAt, createdAt                                                                                  time.Time
		lastAttemptAt, nextAttemptAt, lockedAt, completedAt                                                   *time.Time
	)
	if err := row.Scan(&id, &notificationID, &userID, &email, &template, &status, &attemptCount,
		&availableAt, &lastAttemptAt, &nextAttemptAt, &provider, &providerMessageID, &lastError,
		&lockedAt, &lockedBy, &createdAt, &completedAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.EmailDelivery{}, domain.ErrInvalidEmailDeliveryID
		}
		return domain.EmailDelivery{}, err
	}
	parsedID, _ := domain.ParseEmailDeliveryID(id)
	parsedNotificationID, _ := domain.ParseNotificationID(notificationID)
	return domain.EmailDelivery{
		ID: parsedID, NotificationID: parsedNotificationID, UserID: userID, Email: email,
		Template: template, Status: domain.EmailDeliveryStatus(status), AttemptCount: attemptCount,
		AvailableAt: availableAt.UTC(), LastAttemptAt: lastAttemptAt, NextAttemptAt: nextAttemptAt,
		Provider: provider, ProviderMessageID: providerMessageID, LastError: lastError,
		LockedAt: lockedAt, LockedBy: lockedBy, CreatedAt: createdAt.UTC(), CompletedAt: completedAt,
	}, nil
}
