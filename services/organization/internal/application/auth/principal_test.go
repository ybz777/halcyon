package auth_test

import (
	"testing"

	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

func TestServiceAccountPrincipal_ScopeIntersection(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_test00000000000001")
	principal := appauth.NewServiceAccountPrincipal(orgID, "sa_test", []domain.APIScope{
		domain.ScopeIncidentsRead,
	})

	if !principal.HasPermission(domain.PermissionIncidentsRead) {
		t.Fatal("expected incidents.read from scope")
	}
	if principal.HasPermission(domain.PermissionMembersInvite) {
		t.Fatal("scope must not grant members.invite")
	}
}

func TestServiceAccountPrincipal_CrossTenantDenied(t *testing.T) {
	t.Parallel()

	orgA, _ := domain.ParseOrganizationID("org_test00000000000001")
	orgB, _ := domain.ParseOrganizationID("org_test00000000000002")
	principal := appauth.NewServiceAccountPrincipal(orgA, "sa_test", []domain.APIScope{domain.ScopeAdmin})

	if err := principal.EnsureOrganization(orgB); err == nil {
		t.Fatal("expected cross-tenant error")
	}
}

func TestUserPrincipal_RolePermissions(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_test00000000000001")
	userID, _ := domain.ParseUserID("usr_test00000000000001")
	memID, _ := domain.NewMembershipID()
	email, _ := domain.ParseEmail("user@example.com")

	principal := appauth.NewUserPrincipal(userID, email, orgID, memID, domain.RoleViewer)
	if principal.HasPermission(domain.PermissionMembersUpdate) {
		t.Fatal("viewer must not update members")
	}
	if !principal.HasPermission(domain.PermissionOrganizationRead) {
		t.Fatal("viewer must read organization")
	}
}
