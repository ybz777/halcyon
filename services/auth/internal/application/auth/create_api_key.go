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

// CreateAPIKeyCommand carries API key creation input.
type CreateAPIKeyCommand struct {
	Name             string
	ServiceAccountID string
	OrganizationID   string
	Scopes           []string
	Environment      string
	ExpiresAt        *time.Time
	CreatedBy        domain.UserID
}

// CreateAPIKeyResult returns the plaintext key once.
type CreateAPIKeyResult struct {
	APIKey     APIKeyMetadata
	Plaintext  string
}

// APIKeyMetadata is safe API key metadata without secrets.
type APIKeyMetadata struct {
	ID               string
	KeyID            string
	ServiceAccountID string
	Name             string
	Prefix           string
	Scopes           []string
	Status           string
	CreatedAt        time.Time
	ExpiresAt        *time.Time
	LastUsedAt       *time.Time
	RevokedAt        *time.Time
	CreatedBy        string
}

// CreateAPIKeyHandler creates API keys for service accounts.
type CreateAPIKeyHandler struct {
	apiKeys         ports.APIKeyRepository
	serviceAccounts ports.ServiceAccountRepository
	hasher          ports.APIKeyHasher
	metrics         ports.AuthMetrics
	tx              ports.TransactionManager
	events          ports.EventPublisher
}

// NewCreateAPIKeyHandler creates a CreateAPIKeyHandler.
func NewCreateAPIKeyHandler(
	apiKeys ports.APIKeyRepository,
	serviceAccounts ports.ServiceAccountRepository,
	hasher ports.APIKeyHasher,
	metrics ports.AuthMetrics,
	tx ports.TransactionManager,
	events ports.EventPublisher,
) *CreateAPIKeyHandler {
	if tx == nil {
		tx = ports.NoopTransactionManager{}
	}
	if events == nil {
		events = ports.NoopEventPublisher{}
	}
	return &CreateAPIKeyHandler{
		apiKeys:         apiKeys,
		serviceAccounts: serviceAccounts,
		hasher:          hasher,
		metrics:         metrics,
		tx:              tx,
		events:          events,
	}
}

// Handle creates a new API key and returns the plaintext secret once.
func (h *CreateAPIKeyHandler) Handle(ctx context.Context, cmd CreateAPIKeyCommand) (CreateAPIKeyResult, error) {
	serviceAccountID, err := domain.ParseServiceAccountID(cmd.ServiceAccountID)
	if err != nil {
		return CreateAPIKeyResult{}, err
	}

	account, err := h.serviceAccounts.FindByID(ctx, serviceAccountID)
	if err != nil {
		if errors.Is(err, domain.ErrServiceAccountNotFound) {
			return CreateAPIKeyResult{}, domain.ErrInvalidServiceAccount
		}
		return CreateAPIKeyResult{}, fmt.Errorf("find service account: %w", err)
	}
	if err := account.EnsureActive(); err != nil {
		return CreateAPIKeyResult{}, err
	}

	if cmd.OrganizationID != "" {
		orgID, err := domain.ParseOrganizationID(cmd.OrganizationID)
		if err != nil {
			return CreateAPIKeyResult{}, err
		}
		if !account.OrganizationID().Equal(orgID) {
			return CreateAPIKeyResult{}, domain.ErrForbidden
		}
	}

	scopes, err := domain.ParseAPIScopes(cmd.Scopes)
	if err != nil {
		return CreateAPIKeyResult{}, err
	}

	env, err := domain.ParseAPIKeyEnvironment(cmd.Environment)
	if err != nil {
		env = domain.APIKeyEnvLive
	}

	plaintext, prefix, err := domain.GenerateAPIKeySecret(env)
	if err != nil {
		return CreateAPIKeyResult{}, fmt.Errorf("generate api key: %w", err)
	}

	hash, err := h.hasher.Hash(ctx, plaintext)
	if err != nil {
		return CreateAPIKeyResult{}, fmt.Errorf("hash api key: %w", err)
	}

	id, err := domain.NewAPIKeyID()
	if err != nil {
		return CreateAPIKeyResult{}, err
	}
	keyID, err := domain.NewKeyID()
	if err != nil {
		return CreateAPIKeyResult{}, err
	}

	now := time.Now().UTC()
	key, err := domain.NewAPIKey(
		id, keyID, serviceAccountID, cmd.Name, prefix, hash, scopes, cmd.CreatedBy, now, cmd.ExpiresAt,
	)
	if err != nil {
		return CreateAPIKeyResult{}, err
	}

	if err := h.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := h.apiKeys.Save(txCtx, key); err != nil {
			return fmt.Errorf("save api key: %w", err)
		}
		record, err := infraevents.APIKeyCreatedRecord(key, account.OrganizationID().String())
		if err != nil {
			return err
		}
		return h.events.Publish(txCtx, record)
	}); err != nil {
		return CreateAPIKeyResult{}, err
	}

	if h.metrics != nil {
		h.metrics.RecordAPIKeyCreation()
	}
	slog.Info("api_key_created",
		"api_key_id", key.ID().String(),
		"service_account_id", serviceAccountID.String(),
		"prefix", key.RedactedPrefix(),
	)

	return CreateAPIKeyResult{
		APIKey:    toAPIKeyMetadata(key),
		Plaintext: plaintext,
	}, nil
}

func toAPIKeyMetadata(key *domain.APIKey) APIKeyMetadata {
	scopes := make([]string, 0, len(key.Scopes()))
	for _, scope := range key.Scopes() {
		scopes = append(scopes, scope.String())
	}
	return APIKeyMetadata{
		ID:               key.ID().String(),
		KeyID:            key.KeyID().String(),
		ServiceAccountID: key.ServiceAccountID().String(),
		Name:             key.Name(),
		Prefix:           key.RedactedPrefix(),
		Scopes:           scopes,
		Status:           string(key.Status()),
		CreatedAt:        key.CreatedAt(),
		ExpiresAt:        key.ExpiresAt(),
		LastUsedAt:       key.LastUsedAt(),
		RevokedAt:        key.RevokedAt(),
		CreatedBy:        key.CreatedBy().String(),
	}
}
