package organization

import (
	"context"

	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

// Authorizer performs centralized authorization checks.
type Authorizer struct {
	metrics ports.OrgMetrics
}

// NewAuthorizer creates an Authorizer.
func NewAuthorizer(metrics ports.OrgMetrics) *Authorizer {
	return &Authorizer{metrics: metrics}
}

// RequirePermission verifies the principal has the required permission in the organization.
func (a *Authorizer) RequirePermission(principal appauth.Principal, organizationID domain.OrganizationID, permission domain.Permission) error {
	if err := principal.EnsureOrganization(organizationID); err != nil {
		a.recordDenied(true)
		return err
	}
	granted := principal.HasPermission(permission)
	a.recordDenied(!granted)
	if !granted {
		return domain.ErrForbidden
	}
	return nil
}

// RequireRole verifies the principal has one of the required roles.
func (a *Authorizer) RequireRole(principal appauth.Principal, organizationID domain.OrganizationID, roles ...domain.MemberRole) error {
	if err := principal.EnsureOrganization(organizationID); err != nil {
		a.recordDenied(true)
		return err
	}
	for _, role := range roles {
		if principal.Role() == role {
			return nil
		}
	}
	a.recordDenied(false)
	return domain.ErrForbidden
}

// RequireOrganization verifies tenant context matches.
func (a *Authorizer) RequireOrganization(principal appauth.Principal, organizationID domain.OrganizationID) error {
	if err := principal.EnsureOrganization(organizationID); err != nil {
		a.recordDenied(true)
		return err
	}
	return nil
}

func (a *Authorizer) recordDenied(crossTenant bool) {
	if a.metrics != nil {
		a.metrics.RecordPermissionCheck(false)
		a.metrics.RecordAuthorizationDenied(crossTenant)
	}
}

// ResolvePrincipal loads membership and builds an organization principal.
func ResolvePrincipal(
	ctx context.Context,
	members ports.MembershipRepository,
	orgs ports.OrganizationRepository,
	user ports.AuthenticatedUser,
	organizationID domain.OrganizationID,
) (appauth.Principal, error) {
	org, err := orgs.FindByID(ctx, organizationID)
	if err != nil {
		return appauth.Principal{}, domain.ErrForbidden
	}
	if org.Status() != domain.OrganizationStatusActive {
		return appauth.Principal{}, domain.ErrForbidden
	}

	membership, err := members.FindActiveByOrganizationAndUser(ctx, organizationID, user.UserID)
	if err != nil {
		return appauth.Principal{}, domain.ErrForbidden
	}
	if err := membership.EnsureActive(); err != nil {
		return appauth.Principal{}, domain.ErrForbidden
	}

	return appauth.NewUserPrincipal(user.UserID, user.Email, organizationID, membership.ID(), membership.Role()), nil
}
