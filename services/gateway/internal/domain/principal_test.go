package domain_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"
)

func TestPrincipal_HasPermission_FromJWTPermissions(t *testing.T) {
	t.Parallel()

	principal := domain.NewUserPrincipal(
		"usr_test",
		"admin@example.com",
		[]string{"admin"},
		[]string{"apikey:manage", "profile:read"},
	)

	if !principal.HasPermission(domain.PermissionAPIKeyManage) {
		t.Fatal("expected apikey:manage permission")
	}
	if principal.HasPermission(domain.Permission("users:write")) {
		t.Fatal("did not expect users:write permission")
	}
}

func TestPrincipal_HasPermission_FromAPIKeyScopes(t *testing.T) {
	t.Parallel()

	principal := domain.NewAPIKeyPrincipal("key_test", "org_test", []string{"apikey:manage"})

	if !principal.HasPermission(domain.PermissionAPIKeyManage) {
		t.Fatal("expected apikey:manage scope")
	}
}
