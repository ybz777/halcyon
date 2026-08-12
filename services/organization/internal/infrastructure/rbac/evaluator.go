package rbac

import "github.com/halcyon-io/halcyon/services/organization/internal/domain"

// Evaluator implements ports.PermissionEvaluator using domain RBAC rules.
type Evaluator struct {
	authz domain.AuthorizationService
}

// NewEvaluator creates a permission evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{authz: domain.NewAuthorizationService()}
}

// HasPermission reports whether the role grants the permission.
func (e *Evaluator) HasPermission(role domain.MemberRole, permission domain.Permission) bool {
	return e.authz.CanPerform(role, permission)
}
