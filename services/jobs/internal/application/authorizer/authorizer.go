package authorizer

import (
	"github.com/halcyon-io/halcyon/services/jobs/internal/application/ports"
)

// Authorizer enforces job admin access rules.
type Authorizer struct{}

func New() *Authorizer { return &Authorizer{} }

func (a *Authorizer) CanRead(principal ports.Principal, orgID string) bool {
	if principal.HasPermission(ports.PermissionJobsReadPlatform) {
		return true
	}
	return orgID != "" && principal.HasPermission(ports.PermissionJobsRead)
}

func (a *Authorizer) CanCancel(principal ports.Principal, orgID string) bool {
	if principal.HasPermission(ports.PermissionJobsCancelPlatform) {
		return true
	}
	return orgID != "" && principal.HasPermission(ports.PermissionJobsCancel)
}

func (a *Authorizer) IsPlatformReader(principal ports.Principal) bool {
	return principal.HasPermission(ports.PermissionJobsReadPlatform)
}
