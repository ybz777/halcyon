package postgres

import (
	"context"
	"fmt"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PreferenceRepository struct {
	pool *pgxpool.Pool
}

func NewPreferenceRepository(pool *pgxpool.Pool) *PreferenceRepository {
	return &PreferenceRepository{pool: pool}
}

func (r *PreferenceRepository) ListByUser(ctx context.Context, userID string, limit int, cursor string) ([]domain.NotificationPreference, string, error) {
	limit = platformpagination.NormalizePageSize(limit, domain.DefaultPageSize, domain.MaxPageSize)
	args := []any{userID}
	where := "user_id = $1"
	argPos := 2
	if cursor != "" {
		decoded, err := platformpagination.DecodeKeysetCursor(cursor)
		if err != nil {
			return nil, "", domain.ErrInvalidCursor
		}
		where += fmt.Sprintf(" AND (notification_type > $%d OR (notification_type = $%d AND channel > $%d))", argPos, argPos, argPos+1)
		args = append(args, decoded.Primary, decoded.ID)
		argPos += 2
	}
	args = append(args, limit+1)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT user_id, notification_type, channel, enabled, created_at, updated_at
		FROM notification_preferences WHERE %s
		ORDER BY notification_type ASC, channel ASC
		LIMIT $%d
	`, where, argPos), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := make([]domain.NotificationPreference, 0, limit)
	for rows.Next() {
		var p domain.NotificationPreference
		var channel string
		if err := rows.Scan(&p.UserID, &p.NotificationType, &channel, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, "", err
		}
		p.Channel = domain.Channel(channel)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(out) > limit {
		last := out[limit-1]
		next = platformpagination.EncodeKeysetCursor("notification_type", "asc", string(last.NotificationType), string(last.Channel))
		out = out[:limit]
	}
	return out, next, nil
}

func (r *PreferenceRepository) Upsert(ctx context.Context, pref domain.NotificationPreference) error {
	now := time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO notification_preferences (user_id, notification_type, channel, enabled, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$5)
		ON CONFLICT (user_id, notification_type, channel)
		DO UPDATE SET enabled=EXCLUDED.enabled, updated_at=EXCLUDED.updated_at
	`, pref.UserID, string(pref.NotificationType), string(pref.Channel), pref.Enabled, now)
	return err
}

func (r *PreferenceRepository) PreferencesForUser(ctx context.Context, userID string, notificationType domain.NotificationType) (map[domain.Channel]bool, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT channel, enabled FROM notification_preferences
		WHERE user_id=$1 AND notification_type=$2
	`, userID, string(notificationType))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[domain.Channel]bool)
	for rows.Next() {
		var channel string
		var enabled bool
		if err := rows.Scan(&channel, &enabled); err != nil {
			return nil, err
		}
		out[domain.Channel(channel)] = enabled
	}
	return out, rows.Err()
}
