package authorizer_test

import (
	"testing"

	"github.com/halcyon-io/halcyon/services/jobs/internal/application/authorizer"
	"github.com/halcyon-io/halcyon/services/jobs/internal/application/ports"
)

func TestAuthorizerTenantIsolation(t *testing.T) {
	authz := authorizer.New()
	orgAdmin := ports.Principal{Permissions: map[string]struct{}{ports.PermissionJobsRead: {}}}
	if !authz.CanRead(orgAdmin, "org_a") {
		t.Fatal("org admin should read own org jobs")
	}
	if authz.CanRead(orgAdmin, "") {
		t.Fatal("org admin cannot read platform jobs")
	}
	platform := ports.Principal{Permissions: map[string]struct{}{ports.PermissionJobsReadPlatform: {}}}
	if !authz.CanRead(platform, "") {
		t.Fatal("platform admin should read all jobs")
	}
}

func TestAuthorizerCancel(t *testing.T) {
	authz := authorizer.New()
	user := ports.Principal{Permissions: map[string]struct{}{}}
	if authz.CanCancel(user, "org_a") {
		t.Fatal("normal user cannot cancel")
	}
	admin := ports.Principal{Permissions: map[string]struct{}{ports.PermissionJobsCancel: {}}}
	if !authz.CanCancel(admin, "org_a") {
		t.Fatal("admin can cancel")
	}
}
