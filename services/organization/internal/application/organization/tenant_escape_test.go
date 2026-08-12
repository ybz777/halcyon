package organization_test

import (
	"testing"

	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	apporg "github.com/halcyon-io/halcyon/services/organization/internal/application/organization"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

func TestTenantEscape_CrossOrganizationReadDenied(t *testing.T) {
	t.Parallel()
	authz := apporg.NewAuthorizer(noopMetrics{})
	orgA, _ := domain.ParseOrganizationID("org_tenanttest00000001")
	orgB, _ := domain.ParseOrganizationID("org_tenanttest00000002")
	principal := newPrincipal(orgA, domain.RoleAdmin)

	err := authz.RequirePermission(principal, orgB, domain.PermissionOrganizationRead)
	if err != domain.ErrCrossTenantAccess {
		t.Fatalf("error = %v, want ErrCrossTenantAccess", err)
	}
}

func TestTenantEscape_CrossOrganizationUpdateDenied(t *testing.T) {
	t.Parallel()
	authz := apporg.NewAuthorizer(noopMetrics{})
	orgA, _ := domain.ParseOrganizationID("org_tenanttest00000001")
	orgB, _ := domain.ParseOrganizationID("org_tenanttest00000002")
	principal := newPrincipal(orgA, domain.RoleOwner)

	err := authz.RequirePermission(principal, orgB, domain.PermissionOrganizationUpdate)
	if err != domain.ErrCrossTenantAccess {
		t.Fatalf("error = %v, want ErrCrossTenantAccess", err)
	}
}

func TestIDOR_NonOwnerCannotManageRolesInOtherOrg(t *testing.T) {
	t.Parallel()
	authz := apporg.NewAuthorizer(noopMetrics{})
	orgA, _ := domain.ParseOrganizationID("org_tenanttest00000001")
	orgB, _ := domain.ParseOrganizationID("org_tenanttest00000002")
	memberInA := newPrincipal(orgA, domain.RoleMember)

	err := authz.RequirePermission(memberInA, orgB, domain.PermissionRolesManage)
	if err != domain.ErrCrossTenantAccess {
		t.Fatalf("error = %v, want ErrCrossTenantAccess", err)
	}
}

func TestMassAssignment_PrincipalPermissionsFromRoleOnly(t *testing.T) {
	t.Parallel()
	orgID, _ := domain.ParseOrganizationID("org_tenanttest00000001")
	userID, _ := domain.ParseUserID("usr_tenanttest00000001")
	memID, _ := domain.NewMembershipID()
	email, _ := domain.ParseEmail("member@example.com")

	principal := appauth.NewUserPrincipal(userID, email, orgID, memID, domain.RoleViewer)
	if principal.HasPermission(domain.PermissionMembersUpdate) {
		t.Fatal("viewer must not receive member update permission via client input")
	}
	if principal.HasPermission(domain.PermissionOrganizationDelete) {
		t.Fatal("viewer must not receive delete permission")
	}
}

func TestPrivilegeEscalation_MemberCannotBecomeOwnerViaPermissionCheck(t *testing.T) {
	t.Parallel()
	svc := domain.NewAuthorizationService()
	actor := domain.MemberRef{UserID: mustUserID("usr_actor00000000001"), Role: domain.RoleMember}
	target := domain.MemberRef{UserID: mustUserID("usr_target0000000001"), Role: domain.RoleMember}

	err := svc.CanChangeRole(actor, target, domain.RoleOwner, 2)
	if err != domain.ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func mustUserID(raw string) domain.UserID {
	id, err := domain.ParseUserID(raw)
	if err != nil {
		panic(err)
	}
	return id
}
