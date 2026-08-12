package domain_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

func allPermissions() []domain.Permission {
	return []domain.Permission{
		domain.PermissionOrganizationRead, domain.PermissionOrganizationUpdate, domain.PermissionOrganizationDelete,
		domain.PermissionMembersRead, domain.PermissionMembersInvite, domain.PermissionMembersUpdate, domain.PermissionMembersRemove,
		domain.PermissionRolesRead, domain.PermissionRolesManage,
		domain.PermissionUsersRead, domain.PermissionUsersUpdate,
		domain.PermissionIncidentsRead, domain.PermissionIncidentsCreate, domain.PermissionIncidentsUpdate, domain.PermissionIncidentsDelete,
		domain.PermissionAuditRead,
		domain.PermissionAPIKeysRead, domain.PermissionAPIKeysCreate, domain.PermissionAPIKeysRevoke,
	}
}

func allRoles() []domain.MemberRole {
	return []domain.MemberRole{domain.RoleOwner, domain.RoleAdmin, domain.RoleMember, domain.RoleViewer}
}

func expectedAllow(role domain.MemberRole, permission domain.Permission) bool {
	switch role {
	case domain.RoleOwner:
		return true
	case domain.RoleAdmin:
		switch permission {
		case domain.PermissionOrganizationDelete:
			return false
		default:
			return domain.HasPermission(role, permission)
		}
	default:
		return domain.HasPermission(role, permission)
	}
}

func TestRBACMatrix_AllRolesAndPermissions(t *testing.T) {
	t.Parallel()

	for _, role := range allRoles() {
		for _, permission := range allPermissions() {
			role, permission := role, permission
			t.Run(string(role)+"/"+string(permission), func(t *testing.T) {
				t.Parallel()
				got := domain.HasPermission(role, permission)
				want := expectedAllow(role, permission)
				if got != want {
					t.Fatalf("role=%s permission=%s got=%v want=%v", role, permission, got, want)
				}
			})
		}
	}
}

func TestRBACMatrix_OwnerHasFullAccess(t *testing.T) {
	t.Parallel()
	for _, permission := range allPermissions() {
		if !domain.HasPermission(domain.RoleOwner, permission) {
			t.Fatalf("owner missing permission %s", permission)
		}
	}
}

func TestRBACMatrix_ViewerIsReadOnly(t *testing.T) {
	t.Parallel()
	readOnly := map[domain.Permission]bool{
		domain.PermissionOrganizationRead: true,
		domain.PermissionMembersRead:      true,
		domain.PermissionUsersRead:        true,
		domain.PermissionIncidentsRead:    true,
	}
	for _, permission := range allPermissions() {
		got := domain.HasPermission(domain.RoleViewer, permission)
		if got != readOnly[permission] {
			t.Fatalf("viewer permission %s got=%v want=%v", permission, got, readOnly[permission])
		}
	}
}
