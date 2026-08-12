package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

type PersistInput struct {
	EventID          string
	ConsumerGroup    string
	OrganizationID   string
	Notification     domain.Notification
	EmailDelivery    *domain.EmailDelivery
}

// PersistIdempotent inserts notification and optional email delivery in one transaction.
func (r *NotificationRepository) PersistIdempotent(ctx context.Context, input PersistInput) (duplicate bool, err error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var claimed bool
	err = tx.QueryRow(ctx, `
		INSERT INTO processed_events (event_id, consumer_group, organization_id, status)
		VALUES ($1, $2, $3, 'processing')
		ON CONFLICT (event_id, consumer_group) DO NOTHING
		RETURNING true
	`, input.EventID, input.ConsumerGroup, input.OrganizationID).Scan(&claimed)
	if err != nil {
		if err == pgx.ErrNoRows {
			return true, nil
		}
		return false, err
	}

	metadata, _ := json.Marshal(input.Notification.Metadata)
	tag, err := tx.Exec(ctx, `
		INSERT INTO notifications (
			id, organization_id, user_id, event_id, event_type, notification_type,
			title, body, resource_type, resource_id, metadata, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (event_id, user_id, notification_type) DO NOTHING
	`, input.Notification.ID.String(), input.Notification.OrganizationID, input.Notification.UserID,
		input.Notification.EventID, input.Notification.EventType, string(input.Notification.NotificationType),
		input.Notification.Title, input.Notification.Body, input.Notification.ResourceType,
		input.Notification.ResourceID, metadata, input.Notification.CreatedAt.UTC())
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		_, _ = tx.Exec(ctx, `UPDATE processed_events SET status='completed', processed_at=now() WHERE event_id=$1 AND consumer_group=$2`,
			input.EventID, input.ConsumerGroup)
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return true, nil
	}

	if input.EmailDelivery != nil {
		d := input.EmailDelivery
		_, err = tx.Exec(ctx, `
			INSERT INTO email_deliveries (
				id, notification_id, user_id, email, template, status, attempt_count,
				available_at, created_at
			) VALUES ($1,$2,$3,$4,$5,$6,0,$7,$7)
		`, d.ID.String(), d.NotificationID.String(), d.UserID, d.Email, d.Template,
			string(d.Status), d.AvailableAt.UTC())
		if err != nil {
			return false, err
		}
	}

	_, err = tx.Exec(ctx, `UPDATE processed_events SET status='completed', processed_at=now() WHERE event_id=$1 AND consumer_group=$2`,
		input.EventID, input.ConsumerGroup)
	if err != nil {
		return false, err
	}
	return false, tx.Commit(ctx)
}

func (r *NotificationRepository) FindByID(ctx context.Context, orgID, userID string, id domain.NotificationID) (domain.Notification, error) {
	row := r.pool.QueryRow(ctx, selectNotificationSQL()+` WHERE id=$1 AND organization_id=$2 AND user_id=$3`,
		id.String(), orgID, userID)
	return scanNotification(row)
}

func (r *NotificationRepository) FindByIDOnly(ctx context.Context, id domain.NotificationID) (domain.Notification, error) {
	row := r.pool.QueryRow(ctx, selectNotificationSQL()+` WHERE id=$1`, id.String())
	return scanNotification(row)
}

func (r *NotificationRepository) ListByUser(ctx context.Context, orgID, userID string, limit int, cursor string) ([]domain.Notification, string, error) {
	limit = platformpagination.NormalizePageSize(limit, domain.DefaultPageSize, domain.MaxPageSize)
	args := []any{userID, orgID}
	where := "user_id=$1 AND organization_id=$2"
	argPos := 3
	if cursor != "" {
		cursorTime, cursorID, err := platformpagination.DecodeTimeIDCursor(cursor)
		if err != nil {
			return nil, "", domain.ErrInvalidCursor
		}
		where += fmt.Sprintf(" AND (created_at < $%d OR (created_at = $%d AND id < $%d))", argPos, argPos, argPos+1)
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}
	args = append(args, limit+1)
	query := fmt.Sprintf(`%s WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d`, selectNotificationSQL(), where, argPos)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	return scanNotificationPage(rows, limit)
}

func (r *NotificationRepository) UnreadCount(ctx context.Context, orgID, userID string) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications
		WHERE user_id=$1 AND organization_id=$2 AND read_at IS NULL
	`, userID, orgID).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkRead(ctx context.Context, orgID, userID string, id domain.NotificationID, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at=COALESCE(read_at, $4)
		WHERE id=$1 AND organization_id=$2 AND user_id=$3
	`, id.String(), orgID, userID, at.UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, orgID, userID string, at time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE notifications SET read_at=$3
		WHERE organization_id=$1 AND user_id=$2 AND read_at IS NULL
	`, orgID, userID, at.UTC())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func selectNotificationSQL() string {
	return `SELECT id, organization_id, user_id, event_id, event_type, notification_type,
		title, body, resource_type, resource_id, metadata, read_at, created_at FROM notifications`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNotification(row rowScanner) (domain.Notification, error) {
	var (
		id, orgID, userID, eventID, eventType, nType, title, body, resourceType, resourceID string
		metadataJSON                                                                         []byte
		readAt                                                                               *time.Time
		createdAt                                                                            time.Time
	)
	if err := row.Scan(&id, &orgID, &userID, &eventID, &eventType, &nType, &title, &body,
		&resourceType, &resourceID, &metadataJSON, &readAt, &createdAt); err != nil {
		if err == pgx.ErrNoRows {
			return domain.Notification{}, domain.ErrNotificationNotFound
		}
		return domain.Notification{}, err
	}
	parsedID, _ := domain.ParseNotificationID(id)
	meta := map[string]string{}
	_ = json.Unmarshal(metadataJSON, &meta)
	return domain.Notification{
		ID: parsedID, OrganizationID: orgID, UserID: userID, EventID: eventID, EventType: eventType,
		NotificationType: domain.NotificationType(nType), Title: title, Body: body,
		ResourceType: resourceType, ResourceID: resourceID, Metadata: meta,
		ReadAt: readAt, CreatedAt: createdAt.UTC(),
	}, nil
}

func scanNotificationPage(rows pgx.Rows, limit int) ([]domain.Notification, string, error) {
	items := make([]domain.Notification, 0, limit)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	var next string
	if len(items) > limit {
		last := items[limit-1]
		next = platformpagination.EncodeTimeIDCursor(last.CreatedAt, last.ID.String())
		items = items[:limit]
	}
	return items, next, rows.Err()
}
