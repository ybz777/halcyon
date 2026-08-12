package rbac_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	"github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/rbac"
)

func TestEvaluator_HasPermission(t *testing.T) {
	t.Parallel()

	evaluator := rbac.NewEvaluator()
	if !evaluator.HasPermission(domain.RoleAdmin, domain.PermissionOrganizationUpdate) {
		t.Fatal("admin should update organization")
	}
	if evaluator.HasPermission(domain.RoleViewer, domain.PermissionOrganizationUpdate) {
		t.Fatal("viewer should not update organization")
	}
}
