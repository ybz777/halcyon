package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MembershipRepository persists memberships in PostgreSQL.
type MembershipRepository struct {
	pool *pgxpool.Pool
}

// NewMembershipRepository creates a MembershipRepository.
func NewMembershipRepository(pool *pgxpool.Pool) *MembershipRepository {
	return &MembershipRepository{pool: pool}
}

func (r *MembershipRepository) db(ctx context.Context) DBTX {
	return QuerierFromContext(ctx, r.pool)
}

func (r *MembershipRepository) Create(ctx context.Context, membership *domain.Membership) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO memberships (
			external_id, organization_external_id, user_external_id, role, status, joined_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, membership.ID().String(), membership.OrganizationID().String(), membership.UserID().String(),
		membership.Role().String(), string(membership.Status()), membership.JoinedAt(), membership.UpdatedAt())
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrMembershipExists
		}
		return fmt.Errorf("create membership: %w", err)
	}
	return nil
}

func (r *MembershipRepository) Update(ctx context.Context, membership *domain.Membership) error {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE memberships
		SET role = $3, status = $4, updated_at = $5
		WHERE organization_external_id = $1 AND user_external_id = $2 AND status <> 'removed'
	`, membership.OrganizationID().String(), membership.UserID().String(),
		membership.Role().String(), string(membership.Status()), membership.UpdatedAt())
	if err != nil {
		return fmt.Errorf("update membership: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMembershipNotFound
	}
	return nil
}

func (r *MembershipRepository) FindActiveByOrganizationAndUser(
	ctx context.Context,
	organizationID domain.OrganizationID,
	userID domain.UserID,
) (*domain.Membership, error) {
	row := r.db(ctx).QueryRow(ctx, `
		SELECT external_id, organization_external_id, user_external_id, role, status, joined_at, updated_at
		FROM memberships
		WHERE organization_external_id = $1 AND user_external_id = $2 AND status = 'active'
	`, organizationID.String(), userID.String())

	var externalID, orgID, usrID, role, status string
	var joinedAt, updatedAt time.Time
	if err := row.Scan(&externalID, &orgID, &usrID, &role, &status, &joinedAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMembershipNotFound
		}
		return nil, fmt.Errorf("find membership: %w", err)
	}
	return toDomainMembership(externalID, orgID, usrID, role, status, joinedAt, updatedAt)
}

func (r *MembershipRepository) ListActiveByOrganization(
	ctx context.Context,
	organizationID domain.OrganizationID,
	limit int,
	cursor string,
) ([]*domain.Membership, string, error) {
	return r.ListActiveByOrganizationPage(ctx, organizationID, limit, cursor)
}

func (r *MembershipRepository) CountActiveOwners(ctx context.Context, organizationID domain.OrganizationID) (int, error) {
	row := r.db(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM memberships
		WHERE organization_external_id = $1 AND role = 'owner' AND status = 'active'
	`, organizationID.String())
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count owners: %w", err)
	}
	return count, nil
}

var _ ports.MembershipRepository = (*MembershipRepository)(nil)
