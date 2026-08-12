package organization

import (
	"context"
	"time"

	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	infraevents "github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/events"
)

// UpdateOrganizationInput carries organization update data.
type UpdateOrganizationInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
	Name           string
}

// UpdateOrganizationUseCase updates organization metadata.
type UpdateOrganizationUseCase struct {
	orgs    ports.OrganizationRepository
	tx      ports.TransactionManager
	authz   *Authorizer
	events  ports.EventPublisher
	metrics ports.OrgMetrics
}

// NewUpdateOrganizationUseCase creates the use case.
func NewUpdateOrganizationUseCase(
	orgs ports.OrganizationRepository,
	tx ports.TransactionManager,
	authz *Authorizer,
	events ports.EventPublisher,
	metrics ports.OrgMetrics,
) *UpdateOrganizationUseCase {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &UpdateOrganizationUseCase{orgs: orgs, tx: tx, authz: authz, events: events, metrics: metrics}
}

// Execute updates an organization.
func (uc *UpdateOrganizationUseCase) Execute(ctx context.Context, input UpdateOrganizationInput) (*domain.Organization, error) {
	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionOrganizationUpdate); err != nil {
		return nil, err
	}

	org, err := uc.orgs.FindByID(ctx, input.OrganizationID)
	if err != nil {
		return nil, err
	}
	name, err := domain.ParseOrganizationName(input.Name)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := org.UpdateName(name, now); err != nil {
		return nil, err
	}

	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.orgs.Update(txCtx, org); err != nil {
			return err
		}
		return infraevents.PublishDomainEvents(txCtx, uc.events, org.UncommittedEvents())
	})
	if err != nil {
		return nil, err
	}
	org.ClearUncommittedEvents()
	if uc.metrics != nil {
		uc.metrics.RecordOrganizationUpdated()
	}
	return org, nil
}

// DeleteOrganizationInput carries organization deletion data.
type DeleteOrganizationInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
}

// DeleteOrganizationUseCase soft-deletes an organization.
type DeleteOrganizationUseCase struct {
	orgs    ports.OrganizationRepository
	tx      ports.TransactionManager
	authz   *Authorizer
	events  ports.EventPublisher
	metrics ports.OrgMetrics
}

// NewDeleteOrganizationUseCase creates the use case.
func NewDeleteOrganizationUseCase(
	orgs ports.OrganizationRepository,
	tx ports.TransactionManager,
	authz *Authorizer,
	events ports.EventPublisher,
	metrics ports.OrgMetrics,
) *DeleteOrganizationUseCase {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &DeleteOrganizationUseCase{orgs: orgs, tx: tx, authz: authz, events: events, metrics: metrics}
}

// Execute soft-deletes an organization.
func (uc *DeleteOrganizationUseCase) Execute(ctx context.Context, input DeleteOrganizationInput) error {
	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionOrganizationDelete); err != nil {
		return err
	}
	org, err := uc.orgs.FindByID(ctx, input.OrganizationID)
	if err != nil {
		return err
	}
	if err := org.SoftDelete(time.Now().UTC()); err != nil {
		return err
	}
	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.orgs.Update(txCtx, org); err != nil {
			return err
		}
		return infraevents.PublishDomainEvents(txCtx, uc.events, org.UncommittedEvents())
	})
	if err != nil {
		return err
	}
	org.ClearUncommittedEvents()
	if uc.metrics != nil {
		uc.metrics.RecordOrganizationDeleted()
	}
	return nil
}
