package auth

// RoutePolicy describes route-level access requirements.
type RoutePolicy string

const (
	PolicyPublic             RoutePolicy = "public"
	PolicyAuthenticated      RoutePolicy = "authenticated"
	PolicyPermissionRequired RoutePolicy = "permission_required"
	PolicyAdminOnly          RoutePolicy = "admin_only"
	PolicyServiceOnly        RoutePolicy = "service_only"
)

// Authorizer performs centralized permission checks.
type Authorizer interface {
	Can(identity Identity, permission string) bool
}

// StaticAuthorizer grants permissions present on the identity.
type StaticAuthorizer struct{}

// Can reports whether identity includes the permission.
func (StaticAuthorizer) Can(identity Identity, permission string) bool {
	return identity.HasPermission(permission)
}

// RequirePermission returns forbidden when the permission is missing.
func RequirePermission(authz Authorizer, identity Identity, permission string) error {
	if authz == nil {
		authz = StaticAuthorizer{}
	}
	if authz.Can(identity, permission) {
		return nil
	}
	return ErrForbidden
}
