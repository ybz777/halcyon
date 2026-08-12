package postgres

import (
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

func mapOrganizationError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "organizations_slug_unique" {
			return domain.ErrSlugExists
		}
		return domain.ErrOrganizationExists
	}
	return err
}

func toDomainOrganization(
	externalID, name, slug, status string,
	createdAt, updatedAt time.Time,
	version int,
) (*domain.Organization, error) {
	id, err := domain.ParseOrganizationID(externalID)
	if err != nil {
		return nil, err
	}
	orgName, err := domain.ParseOrganizationName(name)
	if err != nil {
		return nil, err
	}
	orgSlug, err := domain.ParseSlug(slug)
	if err != nil {
		return nil, err
	}
	return domain.ReconstituteOrganization(id, orgName, orgSlug, domain.OrganizationStatus(status), createdAt, updatedAt, version)
}

func toDomainMembership(
	externalID, orgExternalID, userExternalID, role, status string,
	joinedAt, updatedAt time.Time,
) (*domain.Membership, error) {
	id, err := domain.ParseMembershipID(externalID)
	if err != nil {
		return nil, err
	}
	orgID, err := domain.ParseOrganizationID(orgExternalID)
	if err != nil {
		return nil, err
	}
	userID, err := domain.ParseUserID(userExternalID)
	if err != nil {
		return nil, err
	}
	memberRole, err := domain.ParseMemberRole(role)
	if err != nil {
		return nil, err
	}
	return domain.ReconstituteMembership(id, orgID, userID, memberRole, domain.MembershipStatus(status), joinedAt, updatedAt)
}

func toDomainInvitation(
	externalID, orgExternalID, email, userExternalID, role, tokenHash, invitedBy, status string,
	expiresAt time.Time,
	acceptedAt, revokedAt *time.Time,
	createdAt time.Time,
) (*domain.Invitation, error) {
	id, err := domain.ParseInvitationID(externalID)
	if err != nil {
		return nil, err
	}
	orgID, err := domain.ParseOrganizationID(orgExternalID)
	if err != nil {
		return nil, err
	}
	parsedEmail, err := domain.ParseEmail(email)
	if err != nil {
		return nil, err
	}
	var userID domain.UserID
	if userExternalID != "" {
		userID, err = domain.ParseUserID(userExternalID)
		if err != nil {
			return nil, err
		}
	}
	memberRole, err := domain.ParseMemberRole(role)
	if err != nil {
		return nil, err
	}
	invitedByID, err := domain.ParseUserID(invitedBy)
	if err != nil {
		return nil, err
	}
	return domain.ReconstituteInvitation(
		id, orgID, parsedEmail, userID, memberRole, tokenHash, invitedByID,
		domain.InvitationStatus(status), expiresAt, acceptedAt, revokedAt, createdAt,
	)
}

func wrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, mapOrganizationError(err))
}
