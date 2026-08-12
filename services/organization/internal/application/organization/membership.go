package organization

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	infraevents "github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/events"
)

// InvitationTokenGenerator creates secure invitation tokens.
type InvitationTokenGenerator interface {
	Generate() (raw string, hash string, err error)
}

// InvitationTokenHasher hashes invitation tokens for lookup.
type InvitationTokenHasher interface {
	Hash(raw string) string
}

// InviteMemberInput carries invitation details.
type InviteMemberInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
	Email          string
	Role           string
}

// InviteMemberOutput contains the invitation token for the invitee.
type InviteMemberOutput struct {
	InvitationID    string
	InvitationToken string
	ExpiresAt       time.Time
}

// InviteMemberUseCase invites a user by email.
type InviteMemberUseCase struct {
	invitations ports.InvitationRepository
	authz       *Authorizer
	metrics     ports.OrgMetrics
	tokenGen    InvitationTokenGenerator
	events      ports.EventPublisher
}

// NewInviteMemberUseCase creates the use case.
func NewInviteMemberUseCase(
	invitations ports.InvitationRepository,
	authz *Authorizer,
	metrics ports.OrgMetrics,
	tokenGen InvitationTokenGenerator,
	events ports.EventPublisher,
) *InviteMemberUseCase {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &InviteMemberUseCase{invitations: invitations, authz: authz, metrics: metrics, tokenGen: tokenGen, events: events}
}

// Execute creates a pending invitation.
func (uc *InviteMemberUseCase) Execute(ctx context.Context, input InviteMemberInput) (InviteMemberOutput, error) {
	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionMembersInvite); err != nil {
		return InviteMemberOutput{}, err
	}

	email, err := domain.ParseEmail(input.Email)
	if err != nil {
		return InviteMemberOutput{}, err
	}
	role, err := domain.ParseMemberRole(input.Role)
	if err != nil {
		return InviteMemberOutput{}, err
	}
	if role == domain.RoleOwner {
		return InviteMemberOutput{}, domain.ErrForbidden
	}

	raw, hash, err := uc.tokenGen.Generate()
	if err != nil {
		return InviteMemberOutput{}, err
	}
	invID, err := domain.NewInvitationID()
	if err != nil {
		return InviteMemberOutput{}, err
	}
	now := time.Now().UTC()
	expires := now.Add(7 * 24 * time.Hour)
	invitation, err := domain.NewInvitation(
		invID, input.OrganizationID, email, role, hash, input.Principal.UserID(), expires, now,
	)
	if err != nil {
		return InviteMemberOutput{}, err
	}
	if err := uc.invitations.Create(ctx, invitation); err != nil {
		return InviteMemberOutput{}, fmt.Errorf("create invitation: %w", err)
	}
	_ = infraevents.PublishDomainEvents(ctx, uc.events, invitation.UncommittedEvents())
	invitation.ClearUncommittedEvents()

	if uc.metrics != nil {
		uc.metrics.RecordInvitationCreated()
	}
	slog.Info("invitation_created",
		"organization_id", input.OrganizationID.String(),
		"actor_id", input.Principal.UserID().String(),
	)
	return InviteMemberOutput{InvitationID: invID.String(), InvitationToken: raw, ExpiresAt: expires}, nil
}

// AcceptInvitationInput carries invitation acceptance data.
type AcceptInvitationInput struct {
	Actor ports.AuthenticatedUser
	Token string
}

// AcceptInvitationUseCase accepts invitations and creates memberships.
type AcceptInvitationUseCase struct {
	members     ports.MembershipRepository
	invitations ports.InvitationRepository
	orgs        ports.OrganizationRepository
	tx          ports.TransactionManager
	tokenHasher InvitationTokenHasher
	events      ports.EventPublisher
	metrics     ports.OrgMetrics
}

// NewAcceptInvitationUseCase creates the use case.
func NewAcceptInvitationUseCase(
	members ports.MembershipRepository,
	invitations ports.InvitationRepository,
	orgs ports.OrganizationRepository,
	tx ports.TransactionManager,
	hasher InvitationTokenHasher,
	events ports.EventPublisher,
	metrics ports.OrgMetrics,
) *AcceptInvitationUseCase {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &AcceptInvitationUseCase{
		members: members, invitations: invitations, orgs: orgs, tx: tx,
		tokenHasher: hasher, events: events, metrics: metrics,
	}
}

// Execute accepts an invitation and creates membership atomically.
func (uc *AcceptInvitationUseCase) Execute(ctx context.Context, input AcceptInvitationInput) (*domain.Membership, error) {
	invitation, err := uc.invitations.FindByTokenHash(ctx, uc.tokenHasher.Hash(input.Token))
	if err != nil {
		return nil, domain.ErrInvitationNotFound
	}
	if !invitation.Email().Equal(input.Actor.Email) {
		return nil, domain.ErrForbidden
	}
	org, err := uc.orgs.FindByID(ctx, invitation.OrganizationID())
	if err != nil || org.Status() != domain.OrganizationStatusActive {
		return nil, domain.ErrForbidden
	}

	now := time.Now().UTC()
	if err := invitation.Accept(input.Actor.UserID, now); err != nil {
		return nil, err
	}

	var membership *domain.Membership
	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if existing, err := uc.members.FindActiveByOrganizationAndUser(txCtx, invitation.OrganizationID(), input.Actor.UserID); err == nil {
			membership = existing
			return uc.invitations.Update(txCtx, invitation)
		} else if !errors.Is(err, domain.ErrMembershipNotFound) {
			return err
		}

		memberID, err := domain.NewMembershipID()
		if err != nil {
			return err
		}
		membership, err = domain.NewMembership(memberID, invitation.OrganizationID(), input.Actor.UserID, invitation.Role(), now)
		if err != nil {
			return err
		}
		if err := uc.members.Create(txCtx, membership); err != nil {
			return err
		}
		if err := uc.invitations.Update(txCtx, invitation); err != nil {
			return err
		}
		events := append(invitation.UncommittedEvents(), membership.UncommittedEvents()...)
		return infraevents.PublishDomainEvents(txCtx, uc.events, events)
	})
	if err != nil {
		return nil, err
	}
	invitation.ClearUncommittedEvents()
	if membership != nil {
		membership.ClearUncommittedEvents()
	}
	if uc.metrics != nil {
		uc.metrics.RecordInvitationAccepted()
		uc.metrics.RecordMembershipCreated()
	}
	return membership, nil
}

// ListMembersInput identifies an organization membership list request.
type ListMembersInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
	Limit          int
	Cursor         string
}

// ListMembersUseCase lists organization members.
type ListMembersUseCase struct {
	members ports.MembershipRepository
	authz   *Authorizer
}

// NewListMembersUseCase creates the use case.
func NewListMembersUseCase(members ports.MembershipRepository, authz *Authorizer) *ListMembersUseCase {
	return &ListMembersUseCase{members: members, authz: authz}
}

// Execute returns members when the principal has read access.
func (uc *ListMembersUseCase) Execute(ctx context.Context, input ListMembersInput) ([]*domain.Membership, string, error) {
	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionMembersRead); err != nil {
		return nil, "", err
	}
	return uc.members.ListActiveByOrganization(ctx, input.OrganizationID, input.Limit, input.Cursor)
}

// UpdateMemberInput carries membership update data.
type UpdateMemberInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
	TargetUserID   domain.UserID
	Role           *string
	Status         *string
}

// UpdateMemberUseCase updates membership role or status.
type UpdateMemberUseCase struct {
	members ports.MembershipRepository
	authz   *Authorizer
	authzDom domain.AuthorizationService
	tx      ports.TransactionManager
	events  ports.EventPublisher
	metrics ports.OrgMetrics
}

// NewUpdateMemberUseCase creates the use case.
func NewUpdateMemberUseCase(
	members ports.MembershipRepository,
	authz *Authorizer,
	tx ports.TransactionManager,
	events ports.EventPublisher,
	metrics ports.OrgMetrics,
) *UpdateMemberUseCase {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &UpdateMemberUseCase{
		members: members, authz: authz, authzDom: domain.NewAuthorizationService(),
		tx: tx, events: events, metrics: metrics,
	}
}

// Execute updates a membership.
func (uc *UpdateMemberUseCase) Execute(ctx context.Context, input UpdateMemberInput) (*domain.Membership, error) {
	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionMembersUpdate); err != nil {
		return nil, err
	}

	target, err := uc.members.FindActiveByOrganizationAndUser(ctx, input.OrganizationID, input.TargetUserID)
	if err != nil {
		return nil, domain.ErrMembershipNotFound
	}
	now := time.Now().UTC()

	if input.Role != nil {
		newRole, err := domain.ParseMemberRole(*input.Role)
		if err != nil {
			return nil, err
		}
		ownerCount, err := uc.members.CountActiveOwners(ctx, input.OrganizationID)
		if err != nil {
			return nil, err
		}
		if err := uc.authzDom.CanChangeRole(
			domain.MemberRef{UserID: input.Principal.UserID(), Role: input.Principal.Role()},
			domain.MemberRef{UserID: target.UserID(), Role: target.Role()},
			newRole, ownerCount,
		); err != nil {
			return nil, err
		}
		if err := target.ChangeRole(newRole, now); err != nil {
			return nil, err
		}
	}
	if input.Status != nil {
		switch *input.Status {
		case string(domain.MembershipStatusSuspended):
			if err := target.Suspend(now); err != nil {
				return nil, err
			}
		case string(domain.MembershipStatusActive):
			return nil, domain.ErrForbidden
		default:
			return nil, domain.ErrInvalidMembershipStatus
		}
	}

	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.members.Update(txCtx, target); err != nil {
			return err
		}
		return infraevents.PublishDomainEvents(txCtx, uc.events, target.UncommittedEvents())
	})
	if err != nil {
		return nil, err
	}
	target.ClearUncommittedEvents()
	if uc.metrics != nil && input.Role != nil {
		uc.metrics.RecordMembershipRoleChanged()
	}
	return target, nil
}

// RemoveMemberInput carries member removal data.
type RemoveMemberInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
	TargetUserID   domain.UserID
}

// RemoveMemberUseCase removes a member from an organization.
type RemoveMemberUseCase struct {
	members ports.MembershipRepository
	authz   *Authorizer
	authzDom domain.AuthorizationService
	tx      ports.TransactionManager
	events  ports.EventPublisher
	metrics ports.OrgMetrics
}

// NewRemoveMemberUseCase creates the use case.
func NewRemoveMemberUseCase(
	members ports.MembershipRepository,
	authz *Authorizer,
	tx ports.TransactionManager,
	events ports.EventPublisher,
	metrics ports.OrgMetrics,
) *RemoveMemberUseCase {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &RemoveMemberUseCase{
		members: members, authz: authz, authzDom: domain.NewAuthorizationService(),
		tx: tx, events: events, metrics: metrics,
	}
}

// Execute removes a member when permitted and invariants hold.
func (uc *RemoveMemberUseCase) Execute(ctx context.Context, input RemoveMemberInput) error {
	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionMembersRemove); err != nil {
		return err
	}
	target, err := uc.members.FindActiveByOrganizationAndUser(ctx, input.OrganizationID, input.TargetUserID)
	if err != nil {
		return domain.ErrMembershipNotFound
	}
	ownerCount, err := uc.members.CountActiveOwners(ctx, input.OrganizationID)
	if err != nil {
		return fmt.Errorf("count owners: %w", err)
	}
	if err := uc.authzDom.CanRemoveMember(domain.MemberRef{UserID: target.UserID(), Role: target.Role()}, ownerCount); err != nil {
		return err
	}
	if err := target.Remove(time.Now().UTC()); err != nil {
		return err
	}
	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.members.Update(txCtx, target); err != nil {
			return err
		}
		return infraevents.PublishDomainEvents(txCtx, uc.events, target.UncommittedEvents())
	})
	if err != nil {
		return err
	}
	target.ClearUncommittedEvents()
	if uc.metrics != nil {
		uc.metrics.RecordMembershipRemoved()
	}
	return nil
}

// ListInvitationsInput identifies invitation list request.
type ListInvitationsInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
	Limit          int
	Cursor         string
}

// ListInvitationsUseCase lists organization invitations.
type ListInvitationsUseCase struct {
	invitations ports.InvitationRepository
	authz       *Authorizer
}

// NewListInvitationsUseCase creates the use case.
func NewListInvitationsUseCase(invitations ports.InvitationRepository, authz *Authorizer) *ListInvitationsUseCase {
	return &ListInvitationsUseCase{invitations: invitations, authz: authz}
}

// Execute lists invitations for authorized users.
func (uc *ListInvitationsUseCase) Execute(ctx context.Context, input ListInvitationsInput) ([]*domain.Invitation, string, error) {
	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionMembersRead); err != nil {
		return nil, "", err
	}
	return uc.invitations.ListByOrganization(ctx, input.OrganizationID, input.Limit, input.Cursor)
}

// RevokeInvitationInput carries invitation revocation data.
type RevokeInvitationInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
	InvitationID   domain.InvitationID
}

// RevokeInvitationUseCase revokes pending invitations.
type RevokeInvitationUseCase struct {
	invitations ports.InvitationRepository
	authz       *Authorizer
	events      ports.EventPublisher
	metrics     ports.OrgMetrics
}

// NewRevokeInvitationUseCase creates the use case.
func NewRevokeInvitationUseCase(
	invitations ports.InvitationRepository,
	authz *Authorizer,
	events ports.EventPublisher,
	metrics ports.OrgMetrics,
) *RevokeInvitationUseCase {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &RevokeInvitationUseCase{invitations: invitations, authz: authz, events: events, metrics: metrics}
}

// Execute revokes an invitation.
func (uc *RevokeInvitationUseCase) Execute(ctx context.Context, input RevokeInvitationInput) error {
	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionMembersInvite); err != nil {
		return err
	}
	invitation, err := uc.invitations.FindByID(ctx, input.OrganizationID, input.InvitationID)
	if err != nil {
		return err
	}
	if err := invitation.Revoke(time.Now().UTC()); err != nil {
		return err
	}
	if err := uc.invitations.Update(ctx, invitation); err != nil {
		return err
	}
	_ = infraevents.PublishDomainEvents(ctx, uc.events, invitation.UncommittedEvents())
	invitation.ClearUncommittedEvents()
	if uc.metrics != nil {
		uc.metrics.RecordInvitationRevoked()
	}
	return nil
}
