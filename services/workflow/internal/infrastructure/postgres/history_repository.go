package postgres

import (
	"context"
	"fmt"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HistoryRepository persists incident transition history.
type HistoryRepository struct {
	pool *pgxpool.Pool
}

// NewHistoryRepository creates a HistoryRepository.
func NewHistoryRepository(pool *pgxpool.Pool) *HistoryRepository {
	return &HistoryRepository{pool: pool}
}

func (r *HistoryRepository) db(ctx context.Context) DBTX {
	return QuerierFromContext(ctx, r.pool)
}

// Append stores transition records.
func (r *HistoryRepository) Append(ctx context.Context, records []domain.TransitionRecord) error {
	for _, record := range records {
		var fromSeverity, toSeverity *string
		if record.FromSeverity != nil {
			v := record.FromSeverity.String()
			fromSeverity = &v
		}
		if record.ToSeverity != nil {
			v := record.ToSeverity.String()
			toSeverity = &v
		}

		_, err := r.db(ctx).Exec(ctx, `
			INSERT INTO incident_history (
				incident_id, organization_id, from_status, to_status,
				from_severity, to_severity, actor_id, note, occurred_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`,
			record.IncidentID.String(),
			record.OrganizationID.String(),
			nullIfEmpty(record.FromStatus.String()),
			record.ToStatus.String(),
			fromSeverity,
			toSeverity,
			record.ActorID.String(),
			record.Note,
			record.OccurredAt,
		)
		if err != nil {
			return fmt.Errorf("append incident history: %w", err)
		}
	}
	return nil
}

func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

var _ ports.HistoryRepository = (*HistoryRepository)(nil)
