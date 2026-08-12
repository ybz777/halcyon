package domain

import platformauth "github.com/halcyon-io/halcyon/pkg/platform/auth"

// PrincipalType identifies the authenticated caller category.
type PrincipalType string

const (
	PrincipalTypeAnonymous      PrincipalType = "anonymous"
	PrincipalTypeUser           PrincipalType = "user"
	PrincipalTypeAPIKey         PrincipalType = "api_key"
	PrincipalTypeServiceAccount PrincipalType = "service_account"
)

// Principal is an immutable authenticated gateway context.
type Principal struct {
	principalType  PrincipalType
	principalID    string
	userID         string
	email          string
	organizationID string
	roles          []string
	permissions    []string
	scopes         []string
}

// NewAnonymousPrincipal returns an unauthenticated principal.
func NewAnonymousPrincipal() Principal {
	return Principal{principalType: PrincipalTypeAnonymous}
}

// NewUserPrincipal creates a JWT-authenticated user principal.
func NewUserPrincipal(userID, email string, roles, permissions []string) Principal {
	return Principal{
		principalType: PrincipalTypeUser,
		principalID:   userID,
		userID:        userID,
		email:         email,
		roles:         cloneStrings(roles),
		permissions:   cloneStrings(permissions),
	}
}

// NewAPIKeyPrincipal creates a fully-resolved API-key principal.
func NewAPIKeyPrincipal(apiKeyID, organizationID string, scopes []string) Principal {
	return Principal{
		principalType:  PrincipalTypeAPIKey,
		principalID:    apiKeyID,
		organizationID: organizationID,
		scopes:         cloneStrings(scopes),
	}
}

// OrganizationID returns the tenant organization for API key callers.
func (p Principal) OrganizationID() string { return p.organizationID }

// Scopes returns API key scopes when present.
func (p Principal) Scopes() []string { return cloneStrings(p.scopes) }

// Roles returns JWT role names when present.
func (p Principal) Roles() []string { return cloneStrings(p.roles) }

// Permissions returns JWT permission strings when present.
func (p Principal) Permissions() []string { return cloneStrings(p.permissions) }

// HasPermission reports whether the principal has the required permission.
// User JWT permissions and API key scopes are both checked.
func (p Principal) HasPermission(permission Permission) bool {
	target := permission.String()
	for _, perm := range p.permissions {
		if perm == target {
			return true
		}
	}
	for _, scope := range p.scopes {
		if scope == target {
			return true
		}
	}
	return false
}

// Identity maps the gateway principal to the shared platform identity model.
func (p Principal) Identity() platformauth.Identity {
	switch p.principalType {
	case PrincipalTypeUser:
		return platformauth.Identity{
			SubjectID:     p.userID,
			PrincipalType: platformauth.PrincipalUser,
			Roles:         p.Roles(),
			Permissions:   p.Permissions(),
			Method:        platformauth.MethodBearerJWT,
		}
	case PrincipalTypeAPIKey:
		return platformauth.Identity{
			SubjectID:      p.principalID,
			PrincipalType:  platformauth.PrincipalUser,
			OrganizationID: p.organizationID,
			Permissions:    p.Scopes(),
			Method:         platformauth.MethodAPIKey,
		}
	default:
		return platformauth.Identity{}
	}
}

// Type returns the principal type.
func (p Principal) Type() PrincipalType { return p.principalType }

// ID returns the principal identifier.
func (p Principal) ID() string { return p.principalID }

// UserID returns the authenticated user ID when present.
func (p Principal) UserID() string { return p.userID }

// Email returns the authenticated user email when present.
func (p Principal) Email() string { return p.email }

// IsAuthenticated reports whether the caller is authenticated.
func (p Principal) IsAuthenticated() bool {
	return p.principalType != PrincipalTypeAnonymous
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}
