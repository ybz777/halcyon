package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrganizationRepository persists organizations in PostgreSQL.
type OrganizationRepository struct {
	pool *pgxpool.Pool
}

// NewOrganizationRepository creates an OrganizationRepository.
func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

func (r *OrganizationRepository) db(ctx context.Context) DBTX {
	return QuerierFromContext(ctx, r.pool)
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO organizations (external_id, name, slug, status, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, org.ID().String(), org.Name().String(), org.Slug().String(), string(org.Status()),
		org.Version(), org.CreatedAt(), org.UpdatedAt())
	return wrapError("create organization", err)
}

func (r *OrganizationRepository) Update(ctx context.Context, org *domain.Organization) error {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE organizations
		SET name = $2, slug = $3, status = $4, version = $5, updated_at = $6
		WHERE external_id = $1 AND version = $7
	`, org.ID().String(), org.Name().String(), org.Slug().String(), string(org.Status()),
		org.Version(), org.UpdatedAt(), org.Version()-1)
	if err != nil {
		return wrapError("update organization", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

func (r *OrganizationRepository) FindByID(ctx context.Context, id domain.OrganizationID) (*domain.Organization, error) {
	row := r.db(ctx).QueryRow(ctx, `
		SELECT external_id, name, slug, status, version, created_at, updated_at
		FROM organizations WHERE external_id = $1
	`, id.String())

	var externalID, name, slug, status string
	var version int
	var createdAt, updatedAt time.Time
	if err := row.Scan(&externalID, &name, &slug, &status, &version, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("find organization: %w", err)
	}
	return toDomainOrganization(externalID, name, slug, status, createdAt, updatedAt, version)
}

func (r *OrganizationRepository) ExistsBySlug(ctx context.Context, slug domain.Slug) (bool, error) {
	row := r.db(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM organizations WHERE slug = $1 AND status <> 'deleted'
		)
	`, slug.String())
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("exists slug: %w", err)
	}
	return exists, nil
}

var _ ports.OrganizationRepository = (*OrganizationRepository)(nil)
