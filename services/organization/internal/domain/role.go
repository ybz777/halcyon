package domain

import (
	"fmt"
	"strings"
)

// MemberRole is an organization-scoped RBAC role.
type MemberRole string

const (
	RoleOwner  MemberRole = "owner"
	RoleAdmin  MemberRole = "admin"
	RoleMember MemberRole = "member"
	RoleViewer MemberRole = "viewer"
)

// ParseMemberRole validates a role name.
func ParseMemberRole(raw string) (MemberRole, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch MemberRole(raw) {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return MemberRole(raw), nil
	case MemberRole("operator"):
		return RoleMember, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidRole, raw)
	}
}

func (r MemberRole) String() string { return string(r) }
func (r MemberRole) IsOwner() bool  { return r == RoleOwner }

// Permission identifies an organization capability.
type Permission string

const (
	PermissionOrganizationRead   Permission = "organization.read"
	PermissionOrganizationUpdate Permission = "organization.update"
	PermissionOrganizationDelete Permission = "organization.delete"
	PermissionMembersRead        Permission = "members.read"
	PermissionMembersInvite      Permission = "members.invite"
	PermissionMembersUpdate      Permission = "members.update"
	PermissionMembersRemove      Permission = "members.remove"
	PermissionRolesRead          Permission = "roles.read"
	PermissionRolesManage        Permission = "roles.manage"
	PermissionUsersRead          Permission = "users.read"
	PermissionUsersUpdate        Permission = "users.update"
	PermissionIncidentsRead      Permission = "incidents.read"
	PermissionIncidentsCreate    Permission = "incidents.create"
	PermissionIncidentsUpdate    Permission = "incidents.update"
	PermissionIncidentsDelete    Permission = "incidents.delete"
	PermissionAuditRead          Permission = "audit.read"
	PermissionAuditReadSecurity  Permission = "audit.read.security"
	PermissionAuditReadPlatform  Permission = "audit.read.platform"
	PermissionAPIKeysRead        Permission = "api_keys.read"
	PermissionAPIKeysCreate      Permission = "api_keys.create"
	PermissionAPIKeysRevoke      Permission = "api_keys.revoke"
	PermissionIntegrationsRead   Permission = "integrations.read"
	PermissionIntegrationsWrite  Permission = "integrations.write"
)

// ParsePermission validates a permission identifier.
func ParsePermission(raw string) (Permission, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidPermission)
	}
	return Permission(raw), nil
}

func (p Permission) String() string { return string(p) }

// rolePermissionMatrix is the single source of truth for RBAC policy.
var rolePermissionMatrix = map[MemberRole][]Permission{
	RoleOwner: allPermissions(),
	RoleAdmin: {
		PermissionOrganizationRead, PermissionOrganizationUpdate,
		PermissionMembersRead, PermissionMembersInvite, PermissionMembersUpdate, PermissionMembersRemove,
		PermissionRolesRead, PermissionRolesManage,
		PermissionUsersRead, PermissionUsersUpdate,
		PermissionIncidentsRead, PermissionIncidentsCreate, PermissionIncidentsUpdate, PermissionIncidentsDelete,
		PermissionAuditRead, PermissionAuditReadSecurity,
		PermissionAPIKeysRead, PermissionAPIKeysCreate, PermissionAPIKeysRevoke,
		PermissionIntegrationsRead, PermissionIntegrationsWrite,
	},
	RoleMember: {
		PermissionOrganizationRead,
		PermissionMembersRead,
		PermissionUsersRead,
		PermissionIncidentsRead, PermissionIncidentsCreate, PermissionIncidentsUpdate,
	},
	RoleViewer: {
		PermissionOrganizationRead,
		PermissionMembersRead,
		PermissionUsersRead,
		PermissionIncidentsRead,
	},
}

func allPermissions() []Permission {
	return []Permission{
		PermissionOrganizationRead, PermissionOrganizationUpdate, PermissionOrganizationDelete,
		PermissionMembersRead, PermissionMembersInvite, PermissionMembersUpdate, PermissionMembersRemove,
		PermissionRolesRead, PermissionRolesManage,
		PermissionUsersRead, PermissionUsersUpdate,
		PermissionIncidentsRead, PermissionIncidentsCreate, PermissionIncidentsUpdate, PermissionIncidentsDelete,
		PermissionAuditRead, PermissionAuditReadSecurity, PermissionAuditReadPlatform,
		PermissionAPIKeysRead, PermissionAPIKeysCreate, PermissionAPIKeysRevoke,
		PermissionIntegrationsRead, PermissionIntegrationsWrite,
	}
}

// PermissionsForRole returns permissions granted to a role.
func PermissionsForRole(role MemberRole) []Permission {
	perms, ok := rolePermissionMatrix[role]
	if !ok {
		return nil
	}
	out := make([]Permission, len(perms))
	copy(out, perms)
	return out
}

// HasPermission reports whether role grants the permission.
func HasPermission(role MemberRole, permission Permission) bool {
	for _, p := range PermissionsForRole(role) {
		if p == permission {
			return true
		}
	}
	return false
}

// AuthorizationService evaluates organization RBAC rules.
type AuthorizationService struct{}

// NewAuthorizationService creates an AuthorizationService.
func NewAuthorizationService() AuthorizationService {
	return AuthorizationService{}
}

// CanPerform checks whether a member role grants a permission.
func (AuthorizationService) CanPerform(role MemberRole, permission Permission) bool {
	return HasPermission(role, permission)
}

// CanRemoveMember validates membership removal invariants.
func (s AuthorizationService) CanRemoveMember(target MemberRef, ownerCount int) error {
	if target.Role.IsOwner() && ownerCount <= 1 {
		return ErrLastOwner
	}
	return nil
}

// CanChangeRole validates role change invariants.
func (s AuthorizationService) CanChangeRole(actor MemberRef, target MemberRef, newRole MemberRole, ownerCount int) error {
	if !HasPermission(actor.Role, PermissionMembersUpdate) {
		return ErrForbidden
	}
	if target.Role.IsOwner() && newRole != RoleOwner && ownerCount <= 1 {
		return ErrLastOwner
	}
	if newRole == RoleOwner && !HasPermission(actor.Role, PermissionRolesManage) {
		return ErrForbidden
	}
	return nil
}

// MemberRef pairs a user with a membership role for authorization checks.
type MemberRef struct {
	UserID UserID
	Role   MemberRole
}
