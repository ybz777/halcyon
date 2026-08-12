package auth

import (
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// AuthMethod identifies how a caller authenticated.
type AuthMethod string

const (
	AuthMethodUserJWT        AuthMethod = "user_jwt"
	AuthMethodAPIKey         AuthMethod = "api_key"
	AuthMethodServiceAccount AuthMethod = "service_account"
	// AuthMethodJWT is retained for backward compatibility.
	AuthMethodJWT AuthMethod = "user_jwt"
)

// Principal represents an authenticated caller with optional organization context.
type Principal struct {
	AuthMethod       AuthMethod
	UserID           domain.UserID
	Email            domain.Email
	OrganizationID   domain.OrganizationID
	MembershipID     string
	ServiceAccountID domain.ServiceAccountID
	APIKeyID         domain.APIKeyID
	OrgRole          string
	Roles            []domain.Role
	Permissions      []domain.Permission
	Scopes           []domain.APIScope
}

// HasPermission reports whether the principal has the required permission.
func (p Principal) HasPermission(required domain.Permission) bool {
	if p.AuthMethod == AuthMethodAPIKey || p.AuthMethod == AuthMethodServiceAccount {
		if !domain.ScopeGrantsPermission(p.Scopes, required) {
			return false
		}
	}
	return domain.HasPermission(p.Permissions, required)
}

// HasScope reports whether the principal has the required API scope.
func (p Principal) HasScope(required domain.APIScope) bool {
	return domain.HasScope(p.Scopes, required)
}

// EnsureOrganization verifies the principal belongs to the requested organization.
func (p Principal) EnsureOrganization(organizationID domain.OrganizationID) error {
	if p.OrganizationID.IsZero() || !p.OrganizationID.Equal(organizationID) {
		return domain.ErrForbidden
	}
	return nil
}
