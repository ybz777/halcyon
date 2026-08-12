package domain

import (
	"fmt"
	"strings"
)

// APIScope identifies a fine-grained API key capability.
type APIScope string

const (
	ScopeAuthRead       APIScope = "auth.read"
	ScopeAuthWrite      APIScope = "auth.write"
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
		return "", fmt.Errorf("%w: empty scope", ErrInvalidAPIScope)
	}
	switch APIScope(raw) {
	case ScopeAuthRead, ScopeAuthWrite, ScopeUsersRead, ScopeUsersWrite,
		ScopeIncidentsRead, ScopeIncidentsWrite,
		ScopeWorkflowsRead, ScopeWorkflowsExecute, ScopeAuditRead,
		ScopeIntegrationsRead, ScopeIntegrationsWrite, ScopeAdmin:
		return APIScope(raw), nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidAPIScope, raw)
	}
}

// ParseAPIScopes validates a list of scope strings.
func ParseAPIScopes(raw []string) ([]APIScope, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: at least one scope required", ErrInvalidAPIScope)
	}
	scopes := make([]APIScope, 0, len(raw))
	seen := make(map[APIScope]struct{}, len(raw))
	for _, item := range raw {
		scope, err := ParseAPIScope(item)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}
	return scopes, nil
}

// String returns the scope identifier.
func (s APIScope) String() string { return string(s) }

// HasScope reports whether scopes include the required scope or admin.
func HasScope(scopes []APIScope, required APIScope) bool {
	for _, scope := range scopes {
		if scope == ScopeAdmin || scope == required {
			return true
		}
	}
	return false
}

// ScopeGrantsPermission maps API scopes to domain permissions.
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
	case ScopeAuthRead:
		return []Permission{PermissionProfileRead}
	case ScopeAuthWrite:
		return []Permission{PermissionProfileRead, PermissionProfileWrite}
	case ScopeUsersRead:
		return []Permission{PermissionUsersRead}
	case ScopeUsersWrite:
		return []Permission{PermissionUsersRead, PermissionUsersWrite}
	case ScopeIncidentsRead, ScopeIncidentsWrite:
		return nil
	case ScopeWorkflowsRead:
		return nil
	case ScopeWorkflowsExecute:
		return nil
	case ScopeAuditRead:
		return nil
	case ScopeIntegrationsRead, ScopeIntegrationsWrite:
		return nil
	default:
		return nil
	}
}
