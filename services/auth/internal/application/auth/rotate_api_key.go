package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	infraevents "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/events"
)

// RotateAPIKeyCommand rotates an API key by ID.
type RotateAPIKeyCommand struct {
	APIKeyID    domain.APIKeyID
	Environment string
	RotatedBy   domain.UserID
}

// RotateAPIKeyHandler rotates API keys and returns the new plaintext once.
type RotateAPIKeyHandler struct {
	apiKeys         ports.APIKeyRepository
	serviceAccounts ports.ServiceAccountRepository
	hasher          ports.APIKeyHasher
	events          ports.EventPublisher
	metrics         ports.AuthMetrics
}

// NewRotateAPIKeyHandler creates a RotateAPIKeyHandler.
func NewRotateAPIKeyHandler(
	apiKeys ports.APIKeyRepository,
	serviceAccounts ports.ServiceAccountRepository,
	hasher ports.APIKeyHasher,
	events ports.EventPublisher,
	metrics ports.AuthMetrics,
) *RotateAPIKeyHandler {
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &RotateAPIKeyHandler{
		apiKeys:         apiKeys,
		serviceAccounts: serviceAccounts,
		hasher:          hasher,
		events:          events,
		metrics:         metrics,
	}
}

// Handle revokes the old key and issues a replacement in one transaction.
func (h *RotateAPIKeyHandler) Handle(ctx context.Context, cmd RotateAPIKeyCommand) (CreateAPIKeyResult, error) {
	var result CreateAPIKeyResult
	err := h.apiKeys.WithinTransaction(ctx, func(txCtx context.Context) error {
		oldKey, err := h.apiKeys.FindByID(txCtx, cmd.APIKeyID)
		if err != nil {
			if errors.Is(err, domain.ErrAPIKeyNotFound) {
				return domain.ErrAPIKeyNotFound
			}
			return fmt.Errorf("find api key: %w", err)
		}

		if oldKey.Status() != domain.APIKeyStatusActive {
			return oldKey.ValidateForAuth(time.Now().UTC())
		}

		env, err := domain.ParseAPIKeyEnvironment(cmd.Environment)
		if err != nil {
			if strings.HasPrefix(oldKey.Prefix(), "hal_test_") {
				env = domain.APIKeyEnvTest
			} else {
				env = domain.APIKeyEnvLive
			}
		}

		plaintext, prefix, err := domain.GenerateAPIKeySecret(env)
		if err != nil {
			return fmt.Errorf("generate api key: %w", err)
		}
		hash, err := h.hasher.Hash(txCtx, plaintext)
		if err != nil {
			return fmt.Errorf("hash api key: %w", err)
		}

		newID, err := domain.NewAPIKeyID()
		if err != nil {
			return err
		}
		newKeyID, err := domain.NewKeyID()
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		newKey, err := domain.NewAPIKey(
			newID, newKeyID, oldKey.ServiceAccountID(), oldKey.Name()+" (rotated)",
			prefix, hash, oldKey.Scopes(), cmd.RotatedBy, now, oldKey.ExpiresAt(),
		)
		if err != nil {
			return err
		}

		oldKey.MarkRevoked(now)
		if err := h.apiKeys.Update(txCtx, oldKey); err != nil {
			return fmt.Errorf("revoke old api key: %w", err)
		}
		if err := h.apiKeys.Save(txCtx, newKey); err != nil {
			return fmt.Errorf("save rotated api key: %w", err)
		}

		account, err := h.serviceAccounts.FindByID(txCtx, oldKey.ServiceAccountID())
		if err != nil {
			return fmt.Errorf("find service account: %w", err)
		}
		orgID := account.OrganizationID().String()

		revokedRecord, err := infraevents.APIKeyRevokedRecord(oldKey, orgID, now)
		if err != nil {
			return err
		}
		if err := h.events.Publish(txCtx, revokedRecord); err != nil {
			return fmt.Errorf("publish revoked event: %w", err)
		}

		rotatedRecord, err := infraevents.APIKeyRotatedRecord(oldKey, newKey, orgID, now)
		if err != nil {
			return err
		}
		if err := h.events.Publish(txCtx, rotatedRecord); err != nil {
			return fmt.Errorf("publish rotated event: %w", err)
		}

		result = CreateAPIKeyResult{APIKey: toAPIKeyMetadata(newKey), Plaintext: plaintext}
		return nil
	})
	if err != nil {
		return CreateAPIKeyResult{}, err
	}

	if h.metrics != nil {
		h.metrics.RecordAPIKeyRotation()
	}
	slog.Info("api_key_rotated", "api_key_id", result.APIKey.ID, "prefix", result.APIKey.Prefix)
	return result, nil
}
