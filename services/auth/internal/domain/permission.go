package domain

import (
	"fmt"
	"strings"
)

// Permission identifies an authorization capability.
// Permissions are designed for future organization-scoped enforcement.
type Permission string

const (
	PermissionProfileRead  Permission = "profile:read"
	PermissionProfileWrite Permission = "profile:write"
	PermissionUsersRead    Permission = "users:read"
	PermissionUsersWrite   Permission = "users:write"
	PermissionAdminAccess  Permission = "admin:access"
	PermissionAPIKeyManage Permission = "apikey:manage"
)

// ParsePermission validates a permission string.
func ParsePermission(raw string) (Permission, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidPermission)
	}

	switch Permission(raw) {
	case PermissionProfileRead, PermissionProfileWrite,
		PermissionUsersRead, PermissionUsersWrite, PermissionAdminAccess, PermissionAPIKeyManage:
		return Permission(raw), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidPermission, raw)
	}
}

// String returns the permission identifier.
func (p Permission) String() string {
	return string(p)
}

// PermissionsForRoles resolves the union of permissions granted by roles.
func PermissionsForRoles(roles []Role) []Permission {
	seen := make(map[Permission]struct{})
	var perms []Permission

	for _, role := range roles {
		for _, perm := range permissionsForRole(role.Name()) {
			if _, ok := seen[perm]; ok {
				continue
			}
			seen[perm] = struct{}{}
			perms = append(perms, perm)
		}
	}

	return perms
}

// HasPermission reports whether the permission set includes the required permission.
func HasPermission(permissions []Permission, required Permission) bool {
	for _, perm := range permissions {
		if perm == required {
			return true
		}
	}
	return false
}

func permissionsForRole(name RoleName) []Permission {
	switch name {
	case RoleAdmin:
		return []Permission{
			PermissionProfileRead,
			PermissionProfileWrite,
			PermissionUsersRead,
			PermissionUsersWrite,
			PermissionAdminAccess,
			PermissionAPIKeyManage,
		}
	default:
		return []Permission{PermissionProfileRead}
	}
}
