package organization

import (
	"context"
	"fmt"
	"time"

	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

// GetOrganizationInput identifies an organization to load.
type GetOrganizationInput struct {
	OrganizationID domain.OrganizationID
	Principal      appauth.Principal
}

// GetOrganizationUseCase loads an organization for authorized members.
type GetOrganizationUseCase struct {
	orgs   ports.OrganizationRepository
	authz  *Authorizer
	metrics ports.OrgMetrics
}

// NewGetOrganizationUseCase creates the use case.
func NewGetOrganizationUseCase(
	orgs ports.OrganizationRepository,
	authz *Authorizer,
	metrics ports.OrgMetrics,
) *GetOrganizationUseCase {
	return &GetOrganizationUseCase{orgs: orgs, authz: authz, metrics: metrics}
}

// Execute returns an organization when the principal has read access.
func (uc *GetOrganizationUseCase) Execute(ctx context.Context, input GetOrganizationInput) (*domain.Organization, error) {
	start := time.Now()
	defer func() {
		if uc.metrics != nil {
			uc.metrics.ObserveOrganizationLookup(start)
		}
	}()

	if err := uc.authz.RequirePermission(input.Principal, input.OrganizationID, domain.PermissionOrganizationRead); err != nil {
		return nil, err
	}

	org, err := uc.orgs.FindByID(ctx, input.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("find organization: %w", err)
	}
	if org.Status() == domain.OrganizationStatusDeleted {
		return nil, domain.ErrOrganizationNotFound
	}
	return org, nil
}
