package ports

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
)

// OrganizationRepository persists Organization aggregates.
type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	Update(ctx context.Context, org *domain.Organization) error
	FindByID(ctx context.Context, id domain.OrganizationID) (*domain.Organization, error)
	ExistsBySlug(ctx context.Context, slug domain.Slug) (bool, error)
}

// MembershipRepository persists memberships.
type MembershipRepository interface {
	Create(ctx context.Context, membership *domain.Membership) error
	Update(ctx context.Context, membership *domain.Membership) error
	FindActiveByOrganizationAndUser(ctx context.Context, organizationID domain.OrganizationID, userID domain.UserID) (*domain.Membership, error)
	ListActiveByOrganization(ctx context.Context, organizationID domain.OrganizationID, limit int, cursor string) ([]*domain.Membership, string, error)
	CountActiveOwners(ctx context.Context, organizationID domain.OrganizationID) (int, error)
}

// InvitationRepository persists invitations.
type InvitationRepository interface {
	Create(ctx context.Context, invitation *domain.Invitation) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*domain.Invitation, error)
	FindByID(ctx context.Context, organizationID domain.OrganizationID, id domain.InvitationID) (*domain.Invitation, error)
	ListByOrganization(ctx context.Context, organizationID domain.OrganizationID, limit int, cursor string) ([]*domain.Invitation, string, error)
	Update(ctx context.Context, invitation *domain.Invitation) error
}

// TransactionManager runs callbacks inside a database transaction.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// AuthenticatedUser is the validated JWT identity.
type AuthenticatedUser struct {
	UserID domain.UserID
	Email  domain.Email
}

// TokenValidator validates JWT access tokens from the auth service.
type TokenValidator interface {
	Validate(ctx context.Context, bearerToken string) (AuthenticatedUser, error)
}

// PermissionEvaluator checks organization permissions for a member role.
type PermissionEvaluator interface {
	HasPermission(role domain.MemberRole, permission domain.Permission) bool
}

// EventPublisher stores domain events for asynchronous delivery.
type EventPublisher interface {
	Publish(ctx context.Context, event platformevents.DomainEvent) error
}

// NoopEventPublisher discards events.
type NoopEventPublisher struct{}

func (NoopEventPublisher) Publish(context.Context, platformevents.DomainEvent) error { return nil }

// OrgMetrics records organization observability signals.
type OrgMetrics interface {
	RecordOrganizationCreated()
	RecordOrganizationUpdated()
	RecordOrganizationDeleted()
	RecordMembershipCreated()
	RecordMembershipRemoved()
	RecordMembershipRoleChanged()
	RecordInvitationCreated()
	RecordInvitationAccepted()
	RecordInvitationRevoked()
	ObserveOrganizationLookup(start time.Time)
	RecordPermissionCheck(granted bool)
	RecordAuthorizationDenied(crossTenant bool)
}

// ReadinessChecker reports dependency readiness.
type ReadinessChecker interface {
	Check(ctx context.Context) map[string]string
}
