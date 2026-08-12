package jobhandler

import (
	"context"
	"fmt"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/email"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/postgres"
)

// RegisterEmailDelivery wires the email handler into a job registry.
// Returns a cleanup function that closes the notifications database pool.
func RegisterEmailDelivery(
	ctx context.Context,
	registry *platformjobs.Registry,
	databaseURL, emailProvider string,
	cfg EmailConfig,
) (func(), error) {
	if databaseURL == "" {
		return func() {}, nil
	}
	pool, err := postgres.NewPool(ctx, databaseURL, 10, 2)
	if err != nil {
		return nil, fmt.Errorf("notifications database: %w", err)
	}
	sender := email.NewSender(emailProvider)
	handler := NewEmailDeliveryHandler(
		postgres.NewEmailDeliveryRepository(pool),
		postgres.NewNotificationRepository(pool),
		sender,
		cfg,
	)
	if err := registry.Register(platformjobs.TypeEmailDelivery, handler); err != nil {
		pool.Close()
		return nil, err
	}
	return pool.Close, nil
}

// DefaultEmailConfig returns email settings from environment-style defaults.
func DefaultEmailConfig(maxAttempts int, fromName, fromAddress string, timeout time.Duration) EmailConfig {
	return EmailConfig{
		MaxAttempts: maxAttempts,
		FromName:    fromName,
		FromAddress: fromAddress,
		Timeout:     timeout,
	}
}
