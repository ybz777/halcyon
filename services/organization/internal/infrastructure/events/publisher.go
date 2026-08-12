package events

import (
	"context"
	"fmt"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	"github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxPublisher stores domain events in the transactional outbox.
type OutboxPublisher struct {
	pool *pgxpool.Pool
}

// NewOutboxPublisher creates an outbox-backed event publisher.
func NewOutboxPublisher(pool *pgxpool.Pool) *OutboxPublisher {
	return &OutboxPublisher{pool: pool}
}

// Publish implements ports.EventPublisher.
func (p *OutboxPublisher) Publish(ctx context.Context, event platformevents.DomainEvent) error {
	if tx, ok := postgres.TxFromContext(ctx); ok {
		store := outbox.NewTxStore(tx)
		return outbox.NewPublisher(store, "organization").Publish(ctx, event)
	}
	store := outbox.NewPostgresStore(p.pool)
	return outbox.NewPublisher(store, "organization").Publish(ctx, event)
}

var _ ports.EventPublisher = (*OutboxPublisher)(nil)

// PublishDomainEvents publishes uncommitted domain events.
func PublishDomainEvents(ctx context.Context, publisher ports.EventPublisher, events []domain.Event) error {
	for _, event := range events {
		record, err := FromDomainEvent(event)
		if err != nil {
			return err
		}
		if err := publisher.Publish(ctx, record); err != nil {
			return fmt.Errorf("publish event: %w", err)
		}
	}
	return nil
}

// FromDomainEvent maps a domain event to a platform record.
func FromDomainEvent(event domain.Event) (platformevents.Record, error) {
	switch e := event.(type) {
	case domain.OrganizationCreated:
		return platformevents.BuildRecord(
			platformevents.TypeOrganizationCreated,
			e.OrganizationID.String(), "organization", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.OrganizationCreatedPayload{OrganizationID: e.OrganizationID.String(), Name: e.Name.String(), Slug: e.Slug.String()},
		)
	case domain.OrganizationUpdated:
		return platformevents.BuildRecord(
			platformevents.TypeOrganizationUpdated,
			e.OrganizationID.String(), "organization", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.OrganizationUpdatedPayload{OrganizationID: e.OrganizationID.String(), Name: e.Name.String()},
		)
	case domain.OrganizationDeleted:
		return platformevents.BuildRecord(
			platformevents.TypeOrganizationDeleted,
			e.OrganizationID.String(), "organization", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.OrganizationDeletedPayload{OrganizationID: e.OrganizationID.String()},
		)
	case domain.MembershipCreated:
		return platformevents.BuildRecord(
			platformevents.TypeMembershipCreated,
			e.UserID.String(), "membership", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.MembershipCreatedPayload{OrganizationID: e.OrganizationID.String(), UserID: e.UserID.String(), Role: e.Role.String()},
		)
	case domain.MembershipRoleChanged:
		return platformevents.BuildRecord(
			platformevents.TypeMembershipRoleChanged,
			e.UserID.String(), "membership", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.MembershipRoleChangedPayload{OrganizationID: e.OrganizationID.String(), UserID: e.UserID.String(), FromRole: e.FromRole.String(), ToRole: e.ToRole.String()},
		)
	case domain.MembershipSuspended:
		return platformevents.BuildRecord(
			platformevents.TypeMembershipSuspended,
			e.UserID.String(), "membership", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.MembershipSuspendedPayload{OrganizationID: e.OrganizationID.String(), UserID: e.UserID.String()},
		)
	case domain.MembershipRemoved:
		return platformevents.BuildRecord(
			platformevents.TypeMembershipRemoved,
			e.UserID.String(), "membership", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.MembershipRemovedPayload{OrganizationID: e.OrganizationID.String(), UserID: e.UserID.String()},
		)
	case domain.InvitationCreated:
		return platformevents.BuildRecord(
			platformevents.TypeInvitationCreated,
			e.OrganizationID.String(), "invitation", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.InvitationCreatedPayload{OrganizationID: e.OrganizationID.String(), Email: e.Email.String(), Role: e.Role.String()},
		)
	case domain.InvitationAccepted:
		return platformevents.BuildRecord(
			platformevents.TypeInvitationAccepted,
			e.UserID.String(), "invitation", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.InvitationAcceptedPayload{OrganizationID: e.OrganizationID.String(), UserID: e.UserID.String(), Role: e.Role.String()},
		)
	case domain.InvitationRevoked:
		return platformevents.BuildRecord(
			platformevents.TypeInvitationRevoked,
			e.OrganizationID.String(), "invitation", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.InvitationRevokedPayload{OrganizationID: e.OrganizationID.String(), Email: e.Email.String()},
		)
	case domain.MemberInvited:
		return platformevents.BuildRecord(
			platformevents.TypeMemberInvited,
			e.OrganizationID.String(), "membership", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.MemberInvitedPayload{OrganizationID: e.OrganizationID.String(), UserID: "", Role: e.Role.String()},
		)
	case domain.MemberJoined:
		return platformevents.BuildRecord(
			platformevents.TypeMemberJoined,
			e.UserID.String(), "membership", e.OrganizationID.String(), e.OccurredAt(),
			platformevents.MemberJoinedPayload{OrganizationID: e.OrganizationID.String(), UserID: e.UserID.String(), Role: e.Role.String()},
		)
	default:
		return platformevents.Record{}, fmt.Errorf("unsupported domain event %T", event)
	}
}

// NowUTC returns current UTC time.
func NowUTC() time.Time { return time.Now().UTC() }
