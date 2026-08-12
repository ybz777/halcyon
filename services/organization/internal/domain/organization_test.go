package domain

import (
	"testing"
	"time"
)

func TestCreateOrganization(t *testing.T) {
	t.Parallel()

	id, _ := ParseOrganizationID("org_integrationtest0001")
	name, _ := ParseOrganizationName("Acme Corp")
	slug, _ := ParseSlug("acme-corp")
	now := time.Now().UTC()

	org, err := CreateOrganization(id, name, slug, now)
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	if org.Slug().String() != "acme-corp" {
		t.Fatalf("slug = %q", org.Slug().String())
	}
}

func TestSlugFromName(t *testing.T) {
	t.Parallel()

	name, _ := ParseOrganizationName("My Company Inc.")
	slug, err := SlugFromName(name)
	if err != nil {
		t.Fatalf("SlugFromName() error = %v", err)
	}
	if slug.String() != "my-company-inc" {
		t.Fatalf("slug = %q", slug.String())
	}
}

func TestHasPermission_Owner(t *testing.T) {
	t.Parallel()

	if !HasPermission(RoleOwner, PermissionMembersInvite) {
		t.Fatal("owner should invite users")
	}
	if HasPermission(RoleViewer, PermissionMembersInvite) {
		t.Fatal("viewer should not invite users")
	}
}

func TestCanRemoveMember_LastOwner(t *testing.T) {
	t.Parallel()

	svc := NewAuthorizationService()
	err := svc.CanRemoveMember(MemberRef{Role: RoleOwner}, 1)
	if err != ErrLastOwner {
		t.Fatalf("error = %v, want ErrLastOwner", err)
	}
}
