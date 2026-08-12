package auth

// PrincipalType classifies who is calling the API.
type PrincipalType string

const (
	PrincipalUser    PrincipalType = "user"
	PrincipalService PrincipalType = "service"
	PrincipalSystem  PrincipalType = "system"
)

// Method identifies how the caller authenticated.
type Method string

const (
	MethodBearerJWT    Method = "bearer_jwt"
	MethodAPIKey       Method = "api_key"
	MethodServiceToken Method = "service_token"
	MethodSession      Method = "session"
)

// Identity is the normalized authenticated caller attached to request context.
// Services may wrap or map this into domain-specific principals.
type Identity struct {
	SubjectID      string
	PrincipalType  PrincipalType
	OrganizationID string
	Roles          []string
	Permissions    []string
	Method         Method
}

// HasPermission reports whether the identity includes a permission string.
func (i Identity) HasPermission(permission string) bool {
	for _, p := range i.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasRole reports whether the identity includes a role string.
func (i Identity) HasRole(role string) bool {
	for _, r := range i.Roles {
		if r == role {
			return true
		}
	}
	return false
}
