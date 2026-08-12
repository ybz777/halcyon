package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/events"
)

// AuthenticateAPIKeyQuery carries a presented API key.
type AuthenticateAPIKeyQuery struct {
	APIKey string
}

// AuthenticateAPIKeyHandler validates API keys and returns a principal.
type AuthenticateAPIKeyHandler struct {
	apiKeys         ports.APIKeyRepository
	serviceAccounts ports.ServiceAccountRepository
	hasher          ports.APIKeyHasher
	rateLimiter     ports.APIKeyRateLimiter
	metrics         ports.AuthMetrics
	events          ports.EventPublisher
}

// NewAuthenticateAPIKeyHandler creates an AuthenticateAPIKeyHandler.
func NewAuthenticateAPIKeyHandler(
	apiKeys ports.APIKeyRepository,
	serviceAccounts ports.ServiceAccountRepository,
	hasher ports.APIKeyHasher,
	rateLimiter ports.APIKeyRateLimiter,
	metrics ports.AuthMetrics,
	eventPublisher ports.EventPublisher,
) *AuthenticateAPIKeyHandler {
	if rateLimiter == nil {
		rateLimiter = ports.NoopAPIKeyRateLimiter{}
	}
	if eventPublisher == nil {
		eventPublisher = ports.NoopEventPublisher{}
	}
	return &AuthenticateAPIKeyHandler{
		apiKeys:         apiKeys,
		serviceAccounts: serviceAccounts,
		hasher:          hasher,
		rateLimiter:     rateLimiter,
		metrics:       metrics,
		events:          eventPublisher,
	}
}

// Handle authenticates an API key and returns the service account principal.
func (h *AuthenticateAPIKeyHandler) Handle(ctx context.Context, query AuthenticateAPIKeyQuery) (Principal, error) {
	start := time.Now()
	if h.metrics != nil {
		defer h.metrics.ObserveAuthLatency("api_key_auth", start)
	}

	prefix, err := domain.ExtractAPIKeyPrefix(query.APIKey)
	if err != nil {
		h.recordFailure("invalid_format", "", prefix, "")
		return Principal{}, domain.ErrInvalidAPIKey
	}

	allowed, err := h.rateLimiter.Allow(ctx, prefix)
	if err != nil {
		return Principal{}, fmt.Errorf("rate limit check: %w", err)
	}
	if !allowed {
		h.recordFailure("rate_limited", "", prefix, "")
		return Principal{}, domain.ErrForbidden
	}

	key, err := h.apiKeys.FindByPrefix(ctx, prefix)
	if err != nil {
		h.recordFailure("not_found", "", prefix, "")
		if errors.Is(err, domain.ErrAPIKeyNotFound) {
			return Principal{}, domain.ErrInvalidAPIKey
		}
		return Principal{}, fmt.Errorf("find api key: %w", err)
	}

	ok, err := h.hasher.Verify(ctx, query.APIKey, key.HashedSecret())
	if err != nil {
		return Principal{}, fmt.Errorf("verify api key: %w", err)
	}
	if !ok {
		h.recordFailure("invalid_secret", key.ID().String(), key.RedactedPrefix(), "")
		return Principal{}, domain.ErrInvalidAPIKey
	}

	now := time.Now().UTC()
	if err := key.ValidateForAuth(now); err != nil {
		orgID := ""
		if account, acctErr := h.serviceAccounts.FindByID(ctx, key.ServiceAccountID()); acctErr == nil {
			orgID = account.OrganizationID().String()
		}
		h.recordFailure("invalid_status", key.ID().String(), key.RedactedPrefix(), orgID)
		return Principal{}, err
	}

	account, err := h.serviceAccounts.FindByID(ctx, key.ServiceAccountID())
	if err != nil {
		h.recordFailure("service_account", key.ID().String(), key.RedactedPrefix(), "")
		return Principal{}, domain.ErrInvalidAPIKey
	}
	if err := account.EnsureActive(); err != nil {
		h.recordFailure("service_account_disabled", key.ID().String(), key.RedactedPrefix(), account.OrganizationID().String())
		return Principal{}, err
	}

	key.MarkUsed(now)
	if err := h.apiKeys.Update(ctx, key); err != nil {
		return Principal{}, fmt.Errorf("update api key usage: %w", err)
	}

	if h.metrics != nil {
		h.metrics.RecordAPIKeyAuthSuccess()
	}
	slog.Info("api_key_used",
		"api_key_id", key.ID().String(),
		"service_account_id", account.ID().String(),
		"prefix", key.RedactedPrefix(),
	)

	return Principal{
		AuthMethod:       AuthMethodAPIKey,
		ServiceAccountID: account.ID(),
		APIKeyID:         key.ID(),
		OrganizationID:   account.OrganizationID(),
		Scopes:           key.Scopes(),
	}, nil
}

func (h *AuthenticateAPIKeyHandler) recordFailure(reason, apiKeyID, keyPrefix, organizationID string) {
	if h.metrics != nil {
		h.metrics.RecordAPIKeyAuthFailure(reason)
	}
	if h.events == nil {
		return
	}
	record, err := events.APIKeyAuthenticationFailedRecord(apiKeyID, keyPrefix, organizationID, reason, time.Now().UTC())
	if err != nil {
		slog.Warn("build api key auth failure event", "error", err)
		return
	}
	if err := h.events.Publish(context.Background(), record); err != nil {
		slog.Warn("publish api key auth failure event", "error", err)
	}
}
