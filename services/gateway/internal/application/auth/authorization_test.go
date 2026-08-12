package auth_test

import (
	"net/http"
	"testing"

	appauth "github.com/halcyon-io/halcyon/services/gateway/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"
)

func TestRequiredPermission_APIKeys(t *testing.T) {
	t.Parallel()

	perm, ok := appauth.RequiredPermission(http.MethodGet, "/api/v1/auth/api-keys")
	if !ok {
		t.Fatal("expected permission requirement")
	}
	if perm != domain.PermissionAPIKeyManage {
		t.Fatalf("permission = %q, want %q", perm, domain.PermissionAPIKeyManage)
	}

	_, ok = appauth.RequiredPermission(http.MethodGet, "/api/v1/auth/me")
	if ok {
		t.Fatal("did not expect permission requirement for /me")
	}
}
