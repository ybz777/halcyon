package jobhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	platformwebhook "github.com/halcyon-io/halcyon/pkg/platform/webhook"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/crypto"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/postgres"
)

// WebhookDeliveryHandler executes webhook.delivery jobs.
type WebhookDeliveryHandler struct {
	deliveries       *postgres.DeliveryRepository
	endpoints        *postgres.EndpointRepository
	cipher           *crypto.SecretCipher
	client           *platformwebhook.Client
	maxAttempts      int
	failureThreshold int
}

// WebhookConfig configures webhook job execution.
type WebhookConfig struct {
	MaxAttempts      int
	FailureThreshold int
	AllowHTTP        bool
	Timeout          time.Duration
}

// NewWebhookDeliveryHandler creates a handler for webhook.delivery jobs.
func NewWebhookDeliveryHandler(deliveries *postgres.DeliveryRepository, endpoints *postgres.EndpointRepository, cipher *crypto.SecretCipher, cfg WebhookConfig) *WebhookDeliveryHandler {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 6
	}
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 10
	}
	return &WebhookDeliveryHandler{
		deliveries: deliveries, endpoints: endpoints, cipher: cipher,
		client: platformwebhook.NewClient(platformwebhook.ClientConfig{Timeout: cfg.Timeout, AllowHTTP: cfg.AllowHTTP}),
		maxAttempts: cfg.MaxAttempts, failureThreshold: cfg.FailureThreshold,
	}
}

func (h *WebhookDeliveryHandler) Handle(ctx context.Context, job platformjobs.Job) error {
	if job.Status == platformjobs.StatusCancelled {
		return platformjobs.Cancelled()
	}
	var payload platformjobs.WebhookDeliveryPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil || payload.DeliveryID == "" {
		return platformjobs.Permanent(err)
	}
	deliveryID, err := domain.ParseDeliveryID(payload.DeliveryID)
	if err != nil {
		return platformjobs.Permanent(err)
	}
	delivery, err := h.deliveries.FindByIDOnly(ctx, deliveryID)
	if err != nil {
		if err == domain.ErrDeliveryNotFound {
			return platformjobs.Permanent(err)
		}
		return platformjobs.Retryable(err)
	}

	endpoint, ciphertext, err := h.endpoints.FindByIDWithSecretByID(ctx, delivery.EndpointID)
	if err != nil {
		return platformjobs.Retryable(err)
	}
	if !endpoint.IsActive() {
		now := time.Now().UTC()
		delivery.Status = domain.DeliveryDisabled
		delivery.CompletedAt = &now
		_ = h.deliveries.Complete(ctx, delivery)
		return platformjobs.Permanent(fmt.Errorf("endpoint disabled"))
	}
	secret, err := h.cipher.Decrypt(ciphertext)
	if err != nil {
		return platformjobs.Permanent(err)
	}

	result := h.client.Deliver(ctx, endpoint.URL, secret, delivery.EventID, delivery.EventType, delivery.Payload)
	now := time.Now().UTC()
	delivery.LastAttemptAt = &now
	delivery.AttemptCount++

	statusCode := result.StatusCode
	delivery.ResponseStatus = &statusCode
	rt := int(result.ResponseTimeMs)
	delivery.ResponseTimeMs = &rt
	delivery.LastError = result.ErrorSummary

	if statusCode >= 200 && statusCode < 300 {
		delivery.Status = domain.DeliverySucceeded
		delivery.CompletedAt = &now
		endpoint.FailureCount = 0
	} else if delivery.AttemptCount >= h.maxAttempts || !result.Retryable {
		delivery.Status = domain.DeliveryFailed
		delivery.CompletedAt = &now
		endpoint.FailureCount++
	} else {
		delivery.Status = domain.DeliveryRetryWait
		next := now.Add(platformjobs.NextAvailableAt(platformjobs.DefaultRetryConfig(), delivery.AttemptCount, now).Sub(now))
		delivery.NextAttemptAt = &next
		endpoint.FailureCount++
	}

	if endpoint.FailureCount >= h.failureThreshold {
		endpoint.Status = domain.EndpointStatusDisabled
		endpoint.DisabledAt = &now
	}
	endpoint.UpdatedAt = now
	_ = h.endpoints.Update(ctx, endpoint)
	if err := h.deliveries.Complete(ctx, delivery); err != nil {
		return platformjobs.Retryable(err)
	}

	if delivery.Status == domain.DeliverySucceeded {
		return nil
	}
	if delivery.Status == domain.DeliveryFailed || delivery.Status == domain.DeliveryDisabled {
		return platformjobs.Permanent(fmt.Errorf("webhook delivery failed: %s", delivery.LastError))
	}
	return platformjobs.Retryable(fmt.Errorf("webhook delivery retry: %s", delivery.LastError))
}

var _ platformjobs.Handler = (*WebhookDeliveryHandler)(nil)
