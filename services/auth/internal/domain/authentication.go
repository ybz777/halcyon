package domain

// AuthenticationService encapsulates authentication domain rules.
type AuthenticationService struct{}

// NewAuthenticationService creates an AuthenticationService.
func NewAuthenticationService() AuthenticationService {
	return AuthenticationService{}
}

// EnsureUserCanAuthenticate validates that a user may receive credentials.
func (AuthenticationService) EnsureUserCanAuthenticate(user *User) error {
	if user == nil {
		return ErrInvalidCredentials
	}
	return user.RequireActive()
}

// ResolvePermissions derives permissions from assigned roles.
func (AuthenticationService) ResolvePermissions(roles []Role) []Permission {
	if len(roles) == 0 {
		roles = []Role{{name: RoleUser}}
	}
	return PermissionsForRoles(roles)
}

// DefaultRoles returns the platform default roles for a user without explicit assignments.
func (AuthenticationService) DefaultRoles() []Role {
	return []Role{{name: RoleUser}}
}
