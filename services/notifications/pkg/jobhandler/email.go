package jobhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/halcyon-io/halcyon/services/notifications/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/email"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/postgres"
)

// EmailDeliveryHandler executes email.delivery jobs.
type EmailDeliveryHandler struct {
	deliveries    *postgres.EmailDeliveryRepository
	notifications *postgres.NotificationRepository
	sender        ports.EmailSender
	fromName      string
	fromAddress   string
	maxAttempts   int
	timeout       time.Duration
}

// EmailConfig configures email job execution.
type EmailConfig struct {
	MaxAttempts int
	FromName    string
	FromAddress string
	Timeout     time.Duration
}

// NewEmailDeliveryHandler creates a handler for email.delivery jobs.
func NewEmailDeliveryHandler(deliveries *postgres.EmailDeliveryRepository, notifications *postgres.NotificationRepository, sender ports.EmailSender, cfg EmailConfig) *EmailDeliveryHandler {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 6
	}
	return &EmailDeliveryHandler{
		deliveries: deliveries, notifications: notifications, sender: sender,
		fromName: cfg.FromName, fromAddress: cfg.FromAddress, maxAttempts: cfg.MaxAttempts, timeout: cfg.Timeout,
	}
}

func (h *EmailDeliveryHandler) Handle(ctx context.Context, job platformjobs.Job) error {
	if job.Status == platformjobs.StatusCancelled {
		return platformjobs.Cancelled()
	}
	var payload platformjobs.EmailDeliveryPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil || payload.DeliveryID == "" {
		return platformjobs.Permanent(err)
	}
	deliveryID, err := domain.ParseEmailDeliveryID(payload.DeliveryID)
	if err != nil {
		return platformjobs.Permanent(err)
	}
	delivery, err := h.deliveries.FindByIDOnly(ctx, deliveryID)
	if err != nil {
		return platformjobs.Permanent(err)
	}
	notification, err := h.notifications.FindByIDOnly(ctx, delivery.NotificationID)
	if err != nil {
		return platformjobs.Retryable(err)
	}

	msg := email.Render(h.fromName, h.fromAddress, notification, delivery.Email)
	sendCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()
	result, err := h.sender.Send(sendCtx, msg)
	now := time.Now().UTC()
	delivery.LastAttemptAt = &now
	delivery.AttemptCount++
	delivery.Provider = h.sender.ProviderName()

	if err == nil && result.ErrorSummary == "" {
		delivery.Status = domain.EmailSent
		delivery.ProviderMessageID = result.ProviderMessageID
		delivery.CompletedAt = &now
	} else if delivery.AttemptCount >= h.maxAttempts || (err == nil && !result.Retryable) {
		delivery.Status = domain.EmailFailed
		delivery.CompletedAt = &now
		if err != nil {
			delivery.LastError = truncate(err.Error(), 200)
		} else {
			delivery.LastError = truncate(result.ErrorSummary, 200)
		}
	} else {
		delivery.Status = domain.EmailRetryWait
		next := now.Add(platformjobs.NextAvailableAt(platformjobs.DefaultRetryConfig(), delivery.AttemptCount, now).Sub(now))
		delivery.NextAttemptAt = &next
		if err != nil {
			delivery.LastError = truncate(err.Error(), 200)
		} else {
			delivery.LastError = truncate(result.ErrorSummary, 200)
		}
	}
	if err := h.deliveries.Complete(ctx, delivery); err != nil {
		return platformjobs.Retryable(err)
	}
	if delivery.Status == domain.EmailSent {
		return nil
	}
	if delivery.Status == domain.EmailFailed {
		if err != nil {
			return platformjobs.Permanent(err)
		}
		return platformjobs.Permanent(fmt.Errorf("email delivery failed: %s", delivery.LastError))
	}
	if err != nil {
		return platformjobs.Retryable(err)
	}
	return platformjobs.Retryable(fmt.Errorf("email delivery retry: %s", delivery.LastError))
}

func truncate(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}

var _ platformjobs.Handler = (*EmailDeliveryHandler)(nil)
