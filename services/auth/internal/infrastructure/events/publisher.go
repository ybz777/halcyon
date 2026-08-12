package events

import (
	"context"
	"fmt"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres"
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
		return outbox.NewPublisher(store, "auth").Publish(ctx, event)
	}

	store := outbox.NewPostgresStore(p.pool)
	return outbox.NewPublisher(store, "auth").Publish(ctx, event)
}

var _ ports.EventPublisher = (*OutboxPublisher)(nil)

// PublishDomainEvent maps and stores an auth domain event.
func PublishDomainEvent(ctx context.Context, publisher ports.EventPublisher, event domain.Event, organizationID string) error {
	record, err := FromDomainEvent(event, organizationID)
	if err != nil {
		return err
	}
	return publisher.Publish(ctx, record)
}

// FromDomainEvent maps an auth domain event to a platform record.
func FromDomainEvent(event domain.Event, organizationID string) (platformevents.Record, error) {
	if organizationID == "" {
		organizationID = platformevents.SystemOrganizationID
	}

	switch e := event.(type) {
	case domain.UserRegistered:
		return platformevents.BuildRecord(
			platformevents.TypeUserRegistered,
			e.AggregateID().String(),
			"user",
			organizationID,
			e.OccurredAt(),
			platformevents.UserRegisteredPayload{UserID: e.AggregateID().String(), Email: e.Email.String()},
		)
	case domain.UserLoggedIn:
		return platformevents.BuildRecord(
			platformevents.TypeUserLoggedIn,
			e.AggregateID().String(),
			"user",
			organizationID,
			e.OccurredAt(),
			platformevents.UserLoggedInPayload{UserID: e.AggregateID().String(), Email: e.Email.String()},
		)
	case domain.UserLoginFailed:
		aggregateID := e.AggregateID().String()
		if aggregateID == "" {
			aggregateID = "login"
		}
		return platformevents.BuildRecord(
			platformevents.TypeUserLoginFailed,
			aggregateID,
			"user",
			organizationID,
			e.OccurredAt(),
			platformevents.UserLoginFailedPayload{UserID: e.AggregateID().String(), Reason: e.Reason},
		)
	default:
		return platformevents.Record{}, fmt.Errorf("unsupported domain event %T", event)
	}
}

// APIKeyCreatedRecord builds a platform record for API key creation.
func APIKeyCreatedRecord(key *domain.APIKey, organizationID string) (platformevents.Record, error) {
	return platformevents.BuildRecord(
		platformevents.TypeAPIKeyCreated,
		key.ID().String(),
		"api_key",
		organizationID,
		key.CreatedAt(),
		platformevents.APIKeyCreatedPayload{
			APIKeyID:         key.ID().String(),
			KeyID:            key.KeyID().String(),
			ServiceAccountID: key.ServiceAccountID().String(),
			OrganizationID:   organizationID,
			Name:             key.Name(),
		},
	)
}

// APIKeyRevokedRecord builds a platform record for API key revocation.
func APIKeyRevokedRecord(key *domain.APIKey, organizationID string, at time.Time) (platformevents.Record, error) {
	return platformevents.BuildRecord(
		platformevents.TypeAPIKeyRevoked,
		key.ID().String(),
		"api_key",
		organizationID,
		at,
		platformevents.APIKeyRevokedPayload{
			APIKeyID:         key.ID().String(),
			ServiceAccountID: key.ServiceAccountID().String(),
			OrganizationID:   organizationID,
		},
	)
}

// APIKeyRotatedRecord builds a platform record for API key rotation.
func APIKeyRotatedRecord(oldKey, newKey *domain.APIKey, organizationID string, at time.Time) (platformevents.Record, error) {
	return platformevents.BuildRecord(
		platformevents.TypeAPIKeyRotated,
		newKey.ID().String(),
		"api_key",
		organizationID,
		at,
		platformevents.APIKeyRotatedPayload{
			OldAPIKeyID:      oldKey.ID().String(),
			NewAPIKeyID:      newKey.ID().String(),
			ServiceAccountID: newKey.ServiceAccountID().String(),
			OrganizationID:   organizationID,
		},
	)
}

// APIKeyAuthenticationFailedRecord builds a platform record for failed API key authentication.
func APIKeyAuthenticationFailedRecord(apiKeyID, keyPrefix, organizationID, reason string, at time.Time) (platformevents.Record, error) {
	aggregateID := apiKeyID
	if aggregateID == "" {
		aggregateID = keyPrefix
	}
	if aggregateID == "" {
		aggregateID = "unknown"
	}
	if organizationID == "" {
		organizationID = platformevents.SystemOrganizationID
	}
	return platformevents.BuildRecord(
		platformevents.TypeAPIKeyAuthenticationFailed,
		aggregateID,
		"api_key",
		organizationID,
		at,
		platformevents.APIKeyAuthenticationFailedPayload{
			APIKeyID: apiKeyID, KeyPrefix: keyPrefix, OrganizationID: organizationID, Reason: reason,
		},
	)
}
