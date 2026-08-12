package postgres

import (
	"context"
	"fmt"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

func (r *MembershipRepository) ListActiveByOrganizationPage(
	ctx context.Context,
	organizationID domain.OrganizationID,
	limit int,
	cursor string,
) ([]*domain.Membership, string, error) {
	limit = platformpagination.NormalizePageSize(limit, platformpagination.DefaultPageSize, platformpagination.MaxPageSize)
	args := []any{organizationID.String()}
	where := "organization_external_id = $1 AND status = 'active'"
	argPos := 2
	if cursor != "" {
		cursorTime, cursorID, err := platformpagination.DecodeTimeIDCursor(cursor)
		if err != nil {
			return nil, "", domain.ErrInvalidCursor
		}
		where += fmt.Sprintf(" AND (joined_at < $%d OR (joined_at = $%d AND external_id < $%d))", argPos, argPos, argPos+1)
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}
	args = append(args, limit+1)
	rows, err := r.db(ctx).Query(ctx, fmt.Sprintf(`
		SELECT external_id, organization_external_id, user_external_id, role, status, joined_at, updated_at
		FROM memberships
		WHERE %s
		ORDER BY joined_at DESC, external_id DESC
		LIMIT $%d
	`, where, argPos), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	members := make([]*domain.Membership, 0, limit)
	for rows.Next() {
		var externalID, orgID, usrID, role, status string
		var joinedAt, updatedAt time.Time
		if err := rows.Scan(&externalID, &orgID, &usrID, &role, &status, &joinedAt, &updatedAt); err != nil {
			return nil, "", err
		}
		member, err := toDomainMembership(externalID, orgID, usrID, role, status, joinedAt, updatedAt)
		if err != nil {
			return nil, "", err
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(members) > limit {
		last := members[limit-1]
		next = platformpagination.EncodeTimeIDCursor(last.JoinedAt(), last.ID().String())
		members = members[:limit]
	}
	return members, next, nil
}
