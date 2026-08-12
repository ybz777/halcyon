package domain

import (
	"errors"
	"testing"
)

func TestParseRole(t *testing.T) {
	t.Parallel()

	role, err := ParseRole("admin")
	if err != nil {
		t.Fatalf("ParseRole() error = %v", err)
	}
	if role.Name() != RoleAdmin {
		t.Fatalf("role = %q, want admin", role.Name())
	}
}

func TestParseRole_Invalid(t *testing.T) {
	t.Parallel()

	_, err := ParseRole("superuser")
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("error = %v, want ErrInvalidRole", err)
	}
}

func TestPermissionsForRoles(t *testing.T) {
	t.Parallel()

	admin, _ := ParseRole("admin")
	user, _ := ParseRole("user")

	perms := PermissionsForRoles([]Role{user, admin})
	if !HasPermission(perms, PermissionAdminAccess) {
		t.Fatal("expected admin access permission")
	}
	if !HasPermission(perms, PermissionProfileRead) {
		t.Fatal("expected profile read permission")
	}
}

func TestAuthenticationService_DefaultRoles(t *testing.T) {
	t.Parallel()

	svc := NewAuthenticationService()
	roles := svc.DefaultRoles()
	if len(roles) != 1 || roles[0].Name() != RoleUser {
		t.Fatalf("roles = %+v, want default user role", roles)
	}
}

func TestAuthenticationService_EnsureUserCanAuthenticate_Disabled(t *testing.T) {
	t.Parallel()

	user := testDisabledUser(t)
	svc := NewAuthenticationService()
	if err := svc.EnsureUserCanAuthenticate(user); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("error = %v, want ErrUserDisabled", err)
	}
}

func testDisabledUser(t *testing.T) *User {
	t.Helper()
	id, _ := NewUserID()
	email, _ := ParseEmail("disabled@example.com")
	hash, _ := ParsePasswordHash(validBcryptHash)
	user, err := RegisterUser(id, email, hash, testNow)
	if err != nil {
		t.Fatalf("RegisterUser() error = %v", err)
	}
	if err := user.Disable(testNow); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	return user
}
