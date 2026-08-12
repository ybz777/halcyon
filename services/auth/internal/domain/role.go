package domain

import (
	"fmt"
	"strings"
)

// RoleName identifies a named role in the RBAC model.
type RoleName string

const (
	RoleUser  RoleName = "user"
	RoleAdmin RoleName = "admin"
)

// Role is a named authorization role, optionally scoped to an organization.
type Role struct {
	name  RoleName
	orgID string
}

// ParseRole validates a role name.
func ParseRole(raw string) (Role, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch RoleName(raw) {
	case RoleUser, RoleAdmin:
		return Role{name: RoleName(raw)}, nil
	default:
		return Role{}, fmt.Errorf("%w: %q", ErrInvalidRole, raw)
	}
}

// NewRole creates a role, optionally scoped to an organization.
func NewRole(name RoleName, orgID string) (Role, error) {
	switch name {
	case RoleUser, RoleAdmin:
		return Role{name: name, orgID: strings.TrimSpace(orgID)}, nil
	default:
		return Role{}, fmt.Errorf("%w: %q", ErrInvalidRole, name)
	}
}

// Name returns the role name.
func (r Role) Name() RoleName {
	return r.name
}

// OrgID returns the organization scope when set.
func (r Role) OrgID() string {
	return r.orgID
}

// IsGlobal reports whether the role applies platform-wide.
func (r Role) IsGlobal() bool {
	return r.orgID == ""
}

// Equal reports whether two roles are identical.
func (r Role) Equal(other Role) bool {
	return r.name == other.name && r.orgID == other.orgID
}

// String returns the role name.
func (r Role) String() string {
	return string(r.name)
}
