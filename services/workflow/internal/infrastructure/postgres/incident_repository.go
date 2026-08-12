package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IncidentRepository persists incidents in PostgreSQL.
type IncidentRepository struct {
	pool *pgxpool.Pool
}

// NewIncidentRepository creates an IncidentRepository.
func NewIncidentRepository(pool *pgxpool.Pool) *IncidentRepository {
	return &IncidentRepository{pool: pool}
}

func (r *IncidentRepository) db(ctx context.Context) DBTX {
	return QuerierFromContext(ctx, r.pool)
}

// Create inserts a new incident.
func (r *IncidentRepository) Create(ctx context.Context, incident *domain.Incident) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO incidents (
			external_id, organization_id, title, description, severity, priority,
			status, source, version, created_at, updated_at, resolved_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		incident.ID().String(),
		incident.OrganizationID().String(),
		incident.Title().String(),
		incident.Description().String(),
		incident.Severity().String(),
		incident.Priority().String(),
		incident.Status().String(),
		incident.Source().String(),
		incident.Version(),
		incident.CreatedAt(),
		incident.UpdatedAt(),
		incident.ResolvedAt(),
	)
	return wrapError("create incident", err)
}

// Update persists incident changes with optimistic locking.
func (r *IncidentRepository) Update(ctx context.Context, incident *domain.Incident) error {
	expectedVersion := incident.Version() - 1
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE incidents
		SET title = $1, description = $2, severity = $3, priority = $4, status = $5,
			source = $6, version = $7, updated_at = $8, resolved_at = $9
		WHERE external_id = $10 AND organization_id = $11 AND version = $12
	`,
		incident.Title().String(),
		incident.Description().String(),
		incident.Severity().String(),
		incident.Priority().String(),
		incident.Status().String(),
		incident.Source().String(),
		incident.Version(),
		incident.UpdatedAt(),
		incident.ResolvedAt(),
		incident.ID().String(),
		incident.OrganizationID().String(),
		expectedVersion,
	)
	if err != nil {
		return wrapError("update incident", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

// FindByID loads an incident scoped to an organization.
func (r *IncidentRepository) FindByID(ctx context.Context, organizationID domain.OrganizationID, id domain.IncidentID) (*domain.Incident, error) {
	row := r.db(ctx).QueryRow(ctx, `
		SELECT external_id, organization_id, title, description, severity, priority, status, source,
			created_at, updated_at, resolved_at, version
		FROM incidents
		WHERE external_id = $1 AND organization_id = $2
	`, id.String(), organizationID.String())

	var externalID, orgID, title, description, severity, priority, status, source string
	var createdAt, updatedAt time.Time
	var resolvedAt *time.Time
	var version int
	if err := row.Scan(
		&externalID, &orgID, &title, &description, &severity, &priority, &status, &source,
		&createdAt, &updatedAt, &resolvedAt, &version,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrIncidentNotFound
		}
		return nil, fmt.Errorf("find incident: %w", err)
	}
	return toDomainIncident(externalID, orgID, title, description, severity, priority, status, source, createdAt, updatedAt, resolvedAt, version)
}

// Search lists incidents with filters, sorting, and pagination.
// Implemented in incident_search.go.

var _ ports.IncidentRepository = (*IncidentRepository)(nil)
