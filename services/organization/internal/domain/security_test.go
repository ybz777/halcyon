package domain_test

import (
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
)

func TestInvitation_ExpiredCannotBeAccepted(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_sectest0000000001")
	email, _ := domain.ParseEmail("invitee@example.com")
	invID, _ := domain.NewInvitationID()
	inviter, _ := domain.ParseUserID("usr_sectest0000000001")
	now := time.Now().UTC()

	inv, err := domain.NewInvitation(
		invID, orgID, email, domain.RoleMember, "hash",
		inviter, now.Add(-time.Hour), now.Add(-2*time.Hour),
	)
	if err != nil {
		t.Fatalf("NewInvitation() error = %v", err)
	}

	userID, _ := domain.ParseUserID("usr_sectest0000000002")

	err = inv.Accept(userID, now)
	if err == nil {
		t.Fatal("expected error accepting expired invitation")
	}
}

func TestInvitation_RevokedCannotBeAccepted(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_sectest0000000001")
	email, _ := domain.ParseEmail("invitee@example.com")
	invID, _ := domain.NewInvitationID()
	inviter, _ := domain.ParseUserID("usr_sectest0000000001")
	now := time.Now().UTC()

	inv, err := domain.NewInvitation(
		invID, orgID, email, domain.RoleMember, "hash",
		inviter, now.Add(time.Hour), now,
	)
	if err != nil {
		t.Fatalf("NewInvitation() error = %v", err)
	}
	if err := inv.Revoke(now); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}

	userID, _ := domain.ParseUserID("usr_sectest0000000002")

	err = inv.Accept(userID, now)
	if err == nil {
		t.Fatal("expected error accepting revoked invitation")
	}
}

func TestInvitation_AcceptedCannotBeReplayed(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_sectest0000000001")
	email, _ := domain.ParseEmail("invitee@example.com")
	invID, _ := domain.NewInvitationID()
	inviter, _ := domain.ParseUserID("usr_sectest0000000001")
	now := time.Now().UTC()

	inv, err := domain.NewInvitation(
		invID, orgID, email, domain.RoleMember, "hash",
		inviter, now.Add(time.Hour), now,
	)
	if err != nil {
		t.Fatalf("NewInvitation() error = %v", err)
	}

	userID, _ := domain.ParseUserID("usr_sectest0000000002")

	if err := inv.Accept(userID, now); err != nil {
		t.Fatalf("first Accept() error = %v", err)
	}
	otherUser, _ := domain.ParseUserID("usr_sectest0000000099")
	if err := inv.Accept(otherUser, now); err == nil {
		t.Fatal("expected replay by different user to fail")
	}
}

func TestOrganization_DeletedCannotOperate(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_sectest0000000001")
	name, _ := domain.ParseOrganizationName("Acme")
	slug, _ := domain.ParseSlug("acme")
	now := time.Now().UTC()

	org, err := domain.CreateOrganization(orgID, name, slug, now)
	if err != nil {
		t.Fatalf("CreateOrganization() error = %v", err)
	}
	if err := org.SoftDelete(now); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}
	newName, _ := domain.ParseOrganizationName("Renamed")
	if err := org.UpdateName(newName, now); err != domain.ErrOrganizationDeleted {
		t.Fatalf("deleted organization update error = %v, want ErrOrganizationDeleted", err)
	}
}

func TestMembership_SuspendedCannotOperate(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_sectest0000000001")
	userID, _ := domain.ParseUserID("usr_sectest0000000001")
	memID, _ := domain.NewMembershipID()
	now := time.Now().UTC()

	mem, err := domain.NewMembership(memID, orgID, userID, domain.RoleMember, now)
	if err != nil {
		t.Fatalf("NewMembership() error = %v", err)
	}
	if err := mem.Suspend(now); err != nil {
		t.Fatalf("Suspend() error = %v", err)
	}
	if err := mem.EnsureActive(); err == nil {
		t.Fatal("suspended membership must not be active")
	}
}

func TestMembership_RemovedCannotAuthenticate(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_sectest0000000001")
	userID, _ := domain.ParseUserID("usr_sectest0000000001")
	memID, _ := domain.NewMembershipID()
	now := time.Now().UTC()

	mem, err := domain.NewMembership(memID, orgID, userID, domain.RoleMember, now)
	if err != nil {
		t.Fatalf("NewMembership() error = %v", err)
	}
	if err := mem.Remove(now); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if err := mem.EnsureActive(); err == nil {
		t.Fatal("removed membership must not be active")
	}
}
