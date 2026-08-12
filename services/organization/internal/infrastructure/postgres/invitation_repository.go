package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// InvitationRepository persists invitations in PostgreSQL.
type InvitationRepository struct {
	pool *pgxpool.Pool
}

// NewInvitationRepository creates an InvitationRepository.
func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{pool: pool}
}

func (r *InvitationRepository) db(ctx context.Context) DBTX {
	return QuerierFromContext(ctx, r.pool)
}

func (r *InvitationRepository) Create(ctx context.Context, invitation *domain.Invitation) error {
	userID := ""
	if !invitation.UserID().IsZero() {
		userID = invitation.UserID().String()
	}
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO invitations (
			external_id, organization_external_id, user_external_id, email, role,
			token_hash, invited_by_user_id, status, expires_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, invitation.ID().String(), invitation.OrganizationID().String(), nullString(userID),
		invitation.Email().String(), invitation.Role().String(), invitation.TokenHash(),
		invitation.InvitedBy().String(), string(invitation.Status()), invitation.ExpiresAt(), invitation.CreatedAt())
	if err != nil {
		return fmt.Errorf("create invitation: %w", err)
	}
	return nil
}

func (r *InvitationRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Invitation, error) {
	return r.scanInvitation(r.db(ctx).QueryRow(ctx, `
		SELECT external_id, organization_external_id, COALESCE(email, ''), COALESCE(user_external_id, ''),
			role, token_hash, invited_by_user_id, status, expires_at, accepted_at, revoked_at, created_at
		FROM invitations WHERE token_hash = $1
	`, tokenHash))
}

func (r *InvitationRepository) FindByID(ctx context.Context, organizationID domain.OrganizationID, id domain.InvitationID) (*domain.Invitation, error) {
	return r.scanInvitation(r.db(ctx).QueryRow(ctx, `
		SELECT external_id, organization_external_id, COALESCE(email, ''), COALESCE(user_external_id, ''),
			role, token_hash, invited_by_user_id, status, expires_at, accepted_at, revoked_at, created_at
		FROM invitations WHERE external_id = $1 AND organization_external_id = $2
	`, id.String(), organizationID.String()))
}

func (r *InvitationRepository) ListByOrganization(ctx context.Context, organizationID domain.OrganizationID, limit int, cursor string) ([]*domain.Invitation, string, error) {
	limit = platformpagination.NormalizePageSize(limit, platformpagination.DefaultPageSize, platformpagination.MaxPageSize)
	args := []any{organizationID.String()}
	where := "organization_external_id = $1"
	argPos := 2
	if cursor != "" {
		cursorTime, cursorID, err := platformpagination.DecodeTimeIDCursor(cursor)
		if err != nil {
			return nil, "", domain.ErrInvalidCursor
		}
		where += fmt.Sprintf(" AND (created_at < $%d OR (created_at = $%d AND external_id < $%d))", argPos, argPos, argPos+1)
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}
	args = append(args, limit+1)
	rows, err := r.db(ctx).Query(ctx, fmt.Sprintf(`
		SELECT external_id, organization_external_id, COALESCE(email, ''), COALESCE(user_external_id, ''),
			role, token_hash, invited_by_user_id, status, expires_at, accepted_at, revoked_at, created_at
		FROM invitations WHERE %s ORDER BY created_at DESC, external_id DESC LIMIT $%d
	`, where, argPos), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list invitations: %w", err)
	}
	defer rows.Close()

	invitations := make([]*domain.Invitation, 0, limit)
	for rows.Next() {
		inv, err := r.scanInvitationRows(rows.Scan)
		if err != nil {
			return nil, "", err
		}
		invitations = append(invitations, inv)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(invitations) > limit {
		last := invitations[limit-1]
		next = platformpagination.EncodeTimeIDCursor(last.CreatedAt(), last.ID().String())
		invitations = invitations[:limit]
	}
	return invitations, next, nil
}

func (r *InvitationRepository) Update(ctx context.Context, invitation *domain.Invitation) error {
	userID := ""
	if !invitation.UserID().IsZero() {
		userID = invitation.UserID().String()
	}
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE invitations
		SET status = $2, user_external_id = NULLIF($3, ''), accepted_at = $4, revoked_at = $5
		WHERE external_id = $1 AND status IN ('pending', 'accepted')
	`, invitation.ID().String(), string(invitation.Status()), userID, invitation.AcceptedAt(), invitation.RevokedAt())
	if err != nil {
		return fmt.Errorf("update invitation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInvitationUsed
	}
	return nil
}

func (r *InvitationRepository) scanInvitation(row pgx.Row) (*domain.Invitation, error) {
	return r.scanInvitationRows(row.Scan)
}

func (r *InvitationRepository) scanInvitationRows(scan func(dest ...any) error) (*domain.Invitation, error) {
	var externalID, orgID, email, userID, role, hash, invitedBy, status string
	var expiresAt, createdAt time.Time
	var acceptedAt, revokedAt *time.Time
	if err := scan(&externalID, &orgID, &email, &userID, &role, &hash, &invitedBy, &status, &expiresAt, &acceptedAt, &revokedAt, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, fmt.Errorf("scan invitation: %w", err)
	}
	return toDomainInvitation(externalID, orgID, email, userID, role, hash, invitedBy, status, expiresAt, acceptedAt, revokedAt, createdAt)
}

func nullString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

var _ ports.InvitationRepository = (*InvitationRepository)(nil)
