package auth

import (
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

// AuthMethod identifies how a caller authenticated.
type AuthMethod string

const (
	AuthMethodUserJWT        AuthMethod = "user_jwt"
	AuthMethodAPIKey         AuthMethod = "api_key"
	AuthMethodServiceAccount AuthMethod = "service_account"
)

// Principal is an immutable authenticated organization context.
type Principal struct {
	authMethod       AuthMethod
	userID           domain.UserID
	email            domain.Email
	organizationID   domain.OrganizationID
	membershipID     domain.MembershipID
	serviceAccountID string
	role             domain.MemberRole
	permissions      []domain.Permission
	scopes           []domain.APIScope
}

// NewUserPrincipal creates a JWT-authenticated principal with resolved membership.
func NewUserPrincipal(
	userID domain.UserID,
	email domain.Email,
	organizationID domain.OrganizationID,
	membershipID domain.MembershipID,
	role domain.MemberRole,
) Principal {
	return Principal{
		authMethod:     AuthMethodUserJWT,
		userID:         userID,
		email:          email,
		organizationID: organizationID,
		membershipID:   membershipID,
		role:           role,
		permissions:    domain.PermissionsForRole(role),
	}
}

// NewServiceAccountPrincipal creates an API-key-authenticated principal scoped to one organization.
func NewServiceAccountPrincipal(
	organizationID domain.OrganizationID,
	serviceAccountID string,
	scopes []domain.APIScope,
) Principal {
	return Principal{
		authMethod:       AuthMethodServiceAccount,
		organizationID:   organizationID,
		serviceAccountID: serviceAccountID,
		scopes:           append([]domain.APIScope(nil), scopes...),
	}
}

// AuthMethod returns the authentication method.
func (p Principal) AuthMethod() AuthMethod { return p.authMethod }

// UserID returns the authenticated user ID.
func (p Principal) UserID() domain.UserID { return p.userID }

// Email returns the authenticated user email.
func (p Principal) Email() domain.Email { return p.email }

// OrganizationID returns the authoritative tenant organization ID.
func (p Principal) OrganizationID() domain.OrganizationID { return p.organizationID }

// MembershipID returns the active membership ID.
func (p Principal) MembershipID() domain.MembershipID { return p.membershipID }

// Role returns the organization role.
func (p Principal) Role() domain.MemberRole { return p.role }

// Permissions returns resolved organization permissions.
func (p Principal) Permissions() []domain.Permission { return append([]domain.Permission(nil), p.permissions...) }

// ServiceAccountID returns the service account identifier for API-key auth.
func (p Principal) ServiceAccountID() string { return p.serviceAccountID }

// Scopes returns API key scopes when authenticated via service account.
func (p Principal) Scopes() []domain.APIScope { return append([]domain.APIScope(nil), p.scopes...) }

// HasPermission reports whether the principal has a permission.
func (p Principal) HasPermission(permission domain.Permission) bool {
	switch p.authMethod {
	case AuthMethodAPIKey, AuthMethodServiceAccount:
		return domain.ScopeGrantsPermission(p.scopes, permission)
	default:
		return domain.HasPermission(p.role, permission)
	}
}

// EnsureOrganization verifies the principal belongs to the requested organization.
func (p Principal) EnsureOrganization(organizationID domain.OrganizationID) error {
	if !p.organizationID.Equal(organizationID) {
		return domain.ErrCrossTenantAccess
	}
	return nil
}
