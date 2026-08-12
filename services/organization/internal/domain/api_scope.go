package domain

import (
	"fmt"
	"strings"
)

// APIScope identifies a fine-grained API key capability aligned with auth service scopes.
type APIScope string

const (
	ScopeUsersRead      APIScope = "users.read"
	ScopeUsersWrite     APIScope = "users.write"
	ScopeIncidentsRead  APIScope = "incidents.read"
	ScopeIncidentsWrite APIScope = "incidents.write"
	ScopeWorkflowsRead  APIScope = "workflows.read"
	ScopeWorkflowsExecute APIScope = "workflows.execute"
	ScopeAuditRead      APIScope = "audit.read"
	ScopeIntegrationsRead  APIScope = "integrations.read"
	ScopeIntegrationsWrite APIScope = "integrations.write"
	ScopeAdmin          APIScope = "admin"
)

// ParseAPIScope validates a scope string.
func ParseAPIScope(raw string) (APIScope, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return "", fmt.Errorf("%w: empty scope", ErrInvalidPermission)
	}
	switch APIScope(raw) {
	case ScopeUsersRead, ScopeUsersWrite, ScopeIncidentsRead, ScopeIncidentsWrite,
		ScopeWorkflowsRead, ScopeWorkflowsExecute, ScopeAuditRead,
		ScopeIntegrationsRead, ScopeIntegrationsWrite, ScopeAdmin:
		return APIScope(raw), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidPermission, raw)
	}
}

// ScopeGrantsPermission maps API key scopes to organization permissions.
func ScopeGrantsPermission(scopes []APIScope, required Permission) bool {
	for _, scope := range scopes {
		if scope == ScopeAdmin {
			return true
		}
		for _, perm := range permissionsForScope(scope) {
			if perm == required {
				return true
			}
		}
	}
	return false
}

func permissionsForScope(scope APIScope) []Permission {
	switch scope {
	case ScopeUsersRead:
		return []Permission{PermissionUsersRead, PermissionMembersRead, PermissionOrganizationRead}
	case ScopeUsersWrite:
		return []Permission{PermissionUsersRead, PermissionUsersUpdate, PermissionMembersRead, PermissionMembersUpdate}
	case ScopeIncidentsRead:
		return []Permission{PermissionIncidentsRead, PermissionOrganizationRead}
	case ScopeIncidentsWrite:
		return []Permission{
			PermissionIncidentsRead, PermissionIncidentsCreate, PermissionIncidentsUpdate,
			PermissionOrganizationRead,
		}
	case ScopeWorkflowsRead:
		return []Permission{PermissionOrganizationRead}
	case ScopeWorkflowsExecute:
		return []Permission{PermissionOrganizationRead}
	case ScopeAuditRead:
		return []Permission{PermissionAuditRead, PermissionOrganizationRead}
	case ScopeIntegrationsRead:
		return []Permission{PermissionIntegrationsRead, PermissionOrganizationRead}
	case ScopeIntegrationsWrite:
		return []Permission{PermissionIntegrationsRead, PermissionIntegrationsWrite, PermissionOrganizationRead}
	default:
		return nil
	}
}
