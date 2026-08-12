package organization_test

import (
	"context"
	"testing"
	"time"

	apporg "github.com/halcyon-io/halcyon/services/organization/internal/application/organization"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	"github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/invitation"
)

type stubInviteRepo struct {
	inv *domain.Invitation
}

func (s *stubInviteRepo) Create(context.Context, *domain.Invitation) error { return nil }
func (s *stubInviteRepo) FindByTokenHash(context.Context, string) (*domain.Invitation, error) {
	return s.inv, nil
}
func (s *stubInviteRepo) FindByID(context.Context, domain.OrganizationID, domain.InvitationID) (*domain.Invitation, error) {
	return s.inv, nil
}
func (s *stubInviteRepo) ListByOrganization(context.Context, domain.OrganizationID, int, string) ([]*domain.Invitation, string, error) {
	return nil, "", nil
}
func (s *stubInviteRepo) Update(context.Context, *domain.Invitation) error { return nil }

type stubMemberRepo struct{}

func (stubMemberRepo) Create(context.Context, *domain.Membership) error { return nil }
func (stubMemberRepo) Update(context.Context, *domain.Membership) error   { return nil }
func (stubMemberRepo) FindActiveByOrganizationAndUser(context.Context, domain.OrganizationID, domain.UserID) (*domain.Membership, error) {
	return nil, domain.ErrMembershipNotFound
}
func (stubMemberRepo) ListActiveByOrganization(context.Context, domain.OrganizationID, int, string) ([]*domain.Membership, string, error) {
	return nil, "", nil
}
func (stubMemberRepo) CountActiveOwners(context.Context, domain.OrganizationID) (int, error) {
	return 1, nil
}

type stubOrgRepo struct {
	org *domain.Organization
}

func (s stubOrgRepo) Create(context.Context, *domain.Organization) error { return nil }
func (s stubOrgRepo) Update(context.Context, *domain.Organization) error  { return nil }
func (s stubOrgRepo) FindByID(context.Context, domain.OrganizationID) (*domain.Organization, error) {
	return s.org, nil
}
func (s stubOrgRepo) ExistsBySlug(context.Context, domain.Slug) (bool, error) { return false, nil }

type passthroughTx struct{}

func (passthroughTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestAcceptInvitationUseCase_Success(t *testing.T) {
	t.Parallel()

	orgID, _ := domain.ParseOrganizationID("org_integrationtest0601")
	userID, _ := domain.ParseUserID("usr_integrationtest0601")
	email, _ := domain.ParseEmail("invitee@example.com")
	invID, _ := domain.NewInvitationID()
	inviter, _ := domain.ParseUserID("usr_integrationtest0602")

	inv, err := domain.NewInvitation(
		invID, orgID, email, domain.RoleViewer,
		invitation.Hash("token-raw"),
		inviter,
		time.Now().Add(time.Hour),
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("NewInvitation() error = %v", err)
	}

	name, _ := domain.ParseOrganizationName("Acme")
	slug, _ := domain.ParseSlug("acme")
	org, _ := domain.CreateOrganization(orgID, name, slug, time.Now().UTC())

	uc := apporg.NewAcceptInvitationUseCase(
		stubMemberRepo{},
		&stubInviteRepo{inv: inv},
		stubOrgRepo{org: org},
		passthroughTx{},
		invitation.NewTokenService(),
		ports.NoopEventPublisher{},
		nil,
	)

	membership, err := uc.Execute(context.Background(), apporg.AcceptInvitationInput{
		Actor: ports.AuthenticatedUser{UserID: userID, Email: email},
		Token: "token-raw",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if membership.Role() != domain.RoleViewer {
		t.Fatalf("role = %q", membership.Role())
	}
}
