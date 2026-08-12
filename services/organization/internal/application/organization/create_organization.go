package organization

import (
	"context"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	infraevents "github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/events"
)

// CreateOrganizationInput carries organization creation data.
type CreateOrganizationInput struct {
	ActorUserID domain.UserID
	Name        string
}

// CreateOrganizationUseCase creates organizations and owner memberships.
type CreateOrganizationUseCase struct {
	orgs    ports.OrganizationRepository
	members ports.MembershipRepository
	tx      ports.TransactionManager
	events  ports.EventPublisher
	metrics ports.OrgMetrics
}

// NewCreateOrganizationUseCase creates the use case.
func NewCreateOrganizationUseCase(
	orgs ports.OrganizationRepository,
	members ports.MembershipRepository,
	tx ports.TransactionManager,
	events ports.EventPublisher,
	metrics ports.OrgMetrics,
) *CreateOrganizationUseCase {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &CreateOrganizationUseCase{orgs: orgs, members: members, tx: tx, events: events, metrics: metrics}
}

// Execute creates an organization and assigns the actor as owner atomically.
func (uc *CreateOrganizationUseCase) Execute(ctx context.Context, input CreateOrganizationInput) (*domain.Organization, error) {
	name, err := domain.ParseOrganizationName(input.Name)
	if err != nil {
		return nil, err
	}
	slug, err := domain.SlugFromName(name)
	if err != nil {
		return nil, err
	}
	exists, err := uc.orgs.ExistsBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("check slug: %w", err)
	}
	if exists {
		return nil, domain.ErrSlugExists
	}

	orgID, err := domain.NewOrganizationID()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	org, err := domain.CreateOrganization(orgID, name, slug, now)
	if err != nil {
		return nil, err
	}

	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.orgs.Create(txCtx, org); err != nil {
			return err
		}
		memberID, err := domain.NewMembershipID()
		if err != nil {
			return err
		}
		membership, err := domain.NewMembership(memberID, org.ID(), input.ActorUserID, domain.RoleOwner, now)
		if err != nil {
			return err
		}
		if err := uc.members.Create(txCtx, membership); err != nil {
			return err
		}
		if err := infraevents.PublishDomainEvents(txCtx, uc.events, org.UncommittedEvents()); err != nil {
			return err
		}
		if err := infraevents.PublishDomainEvents(txCtx, uc.events, membership.UncommittedEvents()); err != nil {
			return err
		}
		membership.ClearUncommittedEvents()
		return nil
	})
	if err != nil {
		return nil, err
	}

	org.ClearUncommittedEvents()
	if uc.metrics != nil {
		uc.metrics.RecordOrganizationCreated()
		uc.metrics.RecordMembershipCreated()
	}
	return org, nil
}
