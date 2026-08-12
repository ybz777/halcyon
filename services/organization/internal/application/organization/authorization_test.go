package organization_test

import (
	"testing"
	"time"

	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	apporg "github.com/halcyon-io/halcyon/services/organization/internal/application/organization"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

type noopMetrics struct{}

func (noopMetrics) RecordOrganizationCreated()       {}
func (noopMetrics) RecordOrganizationUpdated()       {}
func (noopMetrics) RecordOrganizationDeleted()       {}
func (noopMetrics) RecordMembershipCreated()         {}
func (noopMetrics) RecordMembershipRemoved()         {}
func (noopMetrics) RecordMembershipRoleChanged()     {}
func (noopMetrics) RecordInvitationCreated()         {}
func (noopMetrics) RecordInvitationAccepted()        {}
func (noopMetrics) RecordInvitationRevoked()         {}
func (noopMetrics) ObserveOrganizationLookup(time.Time) {}
func (noopMetrics) RecordPermissionCheck(bool)       {}
func (noopMetrics) RecordAuthorizationDenied(bool)   {}

func newPrincipal(orgID domain.OrganizationID, role domain.MemberRole) appauth.Principal {
	userID, _ := domain.ParseUserID("usr_authztest000000001")
	memID, _ := domain.NewMembershipID()
	email, _ := domain.ParseEmail("authz@example.com")
	return appauth.NewUserPrincipal(userID, email, orgID, memID, role)
}

func TestAuthorizer_CrossTenantAccessDenied(t *testing.T) {
	t.Parallel()

	authz := apporg.NewAuthorizer(noopMetrics{})
	orgA, _ := domain.ParseOrganizationID("org_authztest000000001")
	orgB, _ := domain.ParseOrganizationID("org_authztest000000002")
	principal := newPrincipal(orgA, domain.RoleAdmin)

	err := authz.RequirePermission(principal, orgB, domain.PermissionOrganizationRead)
	if err != domain.ErrCrossTenantAccess {
		t.Fatalf("error = %v, want ErrCrossTenantAccess", err)
	}
}

func TestAuthorizer_ViewerCannotModifyMembers(t *testing.T) {
	t.Parallel()

	authz := apporg.NewAuthorizer(noopMetrics{})
	orgID, _ := domain.ParseOrganizationID("org_authztest000000001")
	principal := newPrincipal(orgID, domain.RoleViewer)

	err := authz.RequirePermission(principal, orgID, domain.PermissionMembersUpdate)
	if err != domain.ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestAuthorizer_AdminCannotDeleteOrganization(t *testing.T) {
	t.Parallel()

	authz := apporg.NewAuthorizer(noopMetrics{})
	orgID, _ := domain.ParseOrganizationID("org_authztest000000001")
	principal := newPrincipal(orgID, domain.RoleAdmin)

	err := authz.RequirePermission(principal, orgID, domain.PermissionOrganizationDelete)
	if err != domain.ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestAuthorizer_MemberCannotManageRoles(t *testing.T) {
	t.Parallel()

	authz := apporg.NewAuthorizer(noopMetrics{})
	orgID, _ := domain.ParseOrganizationID("org_authztest000000001")
	principal := newPrincipal(orgID, domain.RoleMember)

	err := authz.RequirePermission(principal, orgID, domain.PermissionRolesManage)
	if err != domain.ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestAuthorizer_NonOwnerCannotAccessOtherOrganization(t *testing.T) {
	t.Parallel()

	authz := apporg.NewAuthorizer(noopMetrics{})
	orgA, _ := domain.ParseOrganizationID("org_authztest000000001")
	orgB, _ := domain.ParseOrganizationID("org_authztest000000002")
	principal := newPrincipal(orgA, domain.RoleMember)

	err := authz.RequirePermission(principal, orgB, domain.PermissionOrganizationRead)
	if err != domain.ErrCrossTenantAccess {
		t.Fatalf("error = %v, want ErrCrossTenantAccess", err)
	}
}

func TestAuthorizer_ViewerCannotDeleteOrganization(t *testing.T) {
	t.Parallel()

	authz := apporg.NewAuthorizer(noopMetrics{})
	orgID, _ := domain.ParseOrganizationID("org_authztest000000001")
	principal := newPrincipal(orgID, domain.RoleViewer)

	err := authz.RequirePermission(principal, orgID, domain.PermissionOrganizationDelete)
	if err != domain.ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestAuthorizationService_LastOwnerProtection(t *testing.T) {
	t.Parallel()

	svc := domain.NewAuthorizationService()
	target := domain.MemberRef{Role: domain.RoleOwner}
	if err := svc.CanRemoveMember(target, 1); err != domain.ErrLastOwner {
		t.Fatalf("error = %v, want ErrLastOwner", err)
	}
}

func TestAuthorizationService_CannotDemoteLastOwner(t *testing.T) {
	t.Parallel()

	svc := domain.NewAuthorizationService()
	actor := domain.MemberRef{Role: domain.RoleOwner}
	target := domain.MemberRef{Role: domain.RoleOwner}
	if err := svc.CanChangeRole(actor, target, domain.RoleAdmin, 1); err != domain.ErrLastOwner {
		t.Fatalf("error = %v, want ErrLastOwner", err)
	}
}
