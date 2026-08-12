package jobhandler

import (
	"context"
	"fmt"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/crypto"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/postgres"
)

// RegisterWebhookDelivery wires the webhook handler into a job registry.
// Returns a cleanup function that closes the integrations database pool.
func RegisterWebhookDelivery(
	ctx context.Context,
	registry *platformjobs.Registry,
	databaseURL, encryptionKey string,
	cfg WebhookConfig,
) (func(), error) {
	if databaseURL == "" || encryptionKey == "" {
		return func() {}, nil
	}
	pool, err := postgres.NewPool(ctx, databaseURL, 10, 2)
	if err != nil {
		return nil, fmt.Errorf("integrations database: %w", err)
	}
	cipher, err := crypto.NewSecretCipher(encryptionKey)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("cipher: %w", err)
	}
	handler := NewWebhookDeliveryHandler(
		postgres.NewDeliveryRepository(pool),
		postgres.NewEndpointRepository(pool),
		cipher,
		cfg,
	)
	if err := registry.Register(platformjobs.TypeWebhookDelivery, handler); err != nil {
		pool.Close()
		return nil, err
	}
	return pool.Close, nil
}

// DefaultWebhookConfig returns webhook settings from environment-style defaults.
func DefaultWebhookConfig(maxAttempts, failureThreshold int, allowHTTP bool, timeout time.Duration) WebhookConfig {
	return WebhookConfig{
		MaxAttempts:      maxAttempts,
		FailureThreshold: failureThreshold,
		AllowHTTP:        allowHTTP,
		Timeout:          timeout,
	}
}
