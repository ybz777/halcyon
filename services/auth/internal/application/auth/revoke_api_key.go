package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	infraevents "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/events"
)

// RevokeAPIKeyCommand revokes an API key by ID.
type RevokeAPIKeyCommand struct {
	APIKeyID domain.APIKeyID
}

// RevokeAPIKeyHandler revokes API keys.
type RevokeAPIKeyHandler struct {
	apiKeys         ports.APIKeyRepository
	serviceAccounts ports.ServiceAccountRepository
	tx              ports.TransactionManager
	events          ports.EventPublisher
	metrics         ports.AuthMetrics
}

// NewRevokeAPIKeyHandler creates a RevokeAPIKeyHandler.
func NewRevokeAPIKeyHandler(
	apiKeys ports.APIKeyRepository,
	serviceAccounts ports.ServiceAccountRepository,
	tx ports.TransactionManager,
	events ports.EventPublisher,
	metrics ports.AuthMetrics,
) *RevokeAPIKeyHandler {
	if tx == nil {
		tx = ports.NoopTransactionManager{}
	}
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &RevokeAPIKeyHandler{
		apiKeys:         apiKeys,
		serviceAccounts: serviceAccounts,
		tx:              tx,
		events:          events,
		metrics:         metrics,
	}
}

// Handle revokes the API key.
func (h *RevokeAPIKeyHandler) Handle(ctx context.Context, cmd RevokeAPIKeyCommand) error {
	key, err := h.apiKeys.FindByID(ctx, cmd.APIKeyID)
	if err != nil {
		if errors.Is(err, domain.ErrAPIKeyNotFound) {
			return domain.ErrAPIKeyNotFound
		}
		return fmt.Errorf("find api key: %w", err)
	}

	if key.Status() == domain.APIKeyStatusRevoked {
		return nil
	}

	key.MarkRevoked(time.Now().UTC())
	account, err := h.serviceAccounts.FindByID(ctx, key.ServiceAccountID())
	if err != nil {
		return fmt.Errorf("find service account: %w", err)
	}

	if err := h.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := h.apiKeys.Update(txCtx, key); err != nil {
			return fmt.Errorf("revoke api key: %w", err)
		}
		record, err := infraevents.APIKeyRevokedRecord(key, account.OrganizationID().String(), time.Now().UTC())
		if err != nil {
			return err
		}
		return h.events.Publish(txCtx, record)
	}); err != nil {
		return err
	}

	if h.metrics != nil {
		h.metrics.RecordAPIKeyRevoked()
	}
	slog.Info("api_key_revoked", "api_key_id", key.ID().String(), "prefix", key.RedactedPrefix())
	return nil
}
