package organization

import (
	"context"

	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

// MembershipPermissionChecker validates permissions using membership roles.
type MembershipPermissionChecker struct {
	members   ports.MembershipRepository
	evaluator ports.PermissionEvaluator
	metrics   ports.OrgMetrics
}

// NewMembershipPermissionChecker creates a permission checker.
func NewMembershipPermissionChecker(
	members ports.MembershipRepository,
	evaluator ports.PermissionEvaluator,
	metrics ports.OrgMetrics,
) *MembershipPermissionChecker {
	return &MembershipPermissionChecker{members: members, evaluator: evaluator, metrics: metrics}
}

// Check verifies the actor has the required permission in the organization.
func (c *MembershipPermissionChecker) Check(
	ctx context.Context,
	organizationID domain.OrganizationID,
	actorID domain.UserID,
	permission domain.Permission,
) error {
	membership, err := c.members.FindActiveByOrganizationAndUser(ctx, organizationID, actorID)
	if err != nil {
		return domain.ErrForbidden
	}
	if err := membership.EnsureActive(); err != nil {
		return domain.ErrForbidden
	}

	granted := c.evaluator.HasPermission(membership.Role(), permission)
	if c.metrics != nil {
		c.metrics.RecordPermissionCheck(granted)
	}
	if !granted {
		return domain.ErrForbidden
	}
	return nil
}
