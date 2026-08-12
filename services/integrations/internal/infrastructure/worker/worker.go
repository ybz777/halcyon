package worker

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"

	platformwebhook "github.com/halcyon-io/halcyon/pkg/platform/webhook"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/crypto"
	"github.com/halcyon-io/halcyon/services/integrations/internal/infrastructure/postgres"
)

type Config struct {
	WorkerID       string
	PollInterval   time.Duration
	LeaseDuration  time.Duration
	MaxConcurrent  int
	ClaimBatchSize int
	MaxAttempts    int
	FailureThreshold int
	AllowHTTP      bool
	Timeout        time.Duration
}

type Worker struct {
	deliveries *postgres.DeliveryRepository
	endpoints  *postgres.EndpointRepository
	cipher     *crypto.SecretCipher
	client     *platformwebhook.Client
	cfg        Config
	log        *slog.Logger
}

func NewWorker(deliveries *postgres.DeliveryRepository, endpoints *postgres.EndpointRepository, cipher *crypto.SecretCipher, cfg Config, log *slog.Logger) *Worker {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 4
	}
	if log == nil {
		log = slog.Default()
	}
	return &Worker{
		deliveries: deliveries,
		endpoints:  endpoints,
		cipher:     cipher,
		client:     platformwebhook.NewClient(platformwebhook.ClientConfig{Timeout: cfg.Timeout, AllowHTTP: cfg.AllowHTTP}),
		cfg:        cfg,
		log:        log,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if _, err := w.deliveries.RecoverLeases(ctx, w.cfg.LeaseDuration); err != nil {
		w.log.Warn("lease recovery failed", slog.String("error", err.Error()))
	}
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	sem := make(chan struct{}, w.cfg.MaxConcurrent)
	var wg sync.WaitGroup

	for {
		deliveries, err := w.deliveries.Claim(ctx, w.cfg.WorkerID, w.cfg.ClaimBatchSize, w.cfg.LeaseDuration)
		if err != nil && ctx.Err() == nil {
			w.log.Error("claim deliveries failed", slog.String("error", err.Error()))
		}
		for _, delivery := range deliveries {
			sem <- struct{}{}
			wg.Add(1)
			go func(d domain.WebhookDelivery) {
				defer func() { <-sem; wg.Done() }()
				if err := w.process(ctx, d); err != nil {
					w.log.Error("delivery failed", slog.String("delivery_id", d.ID.String()), slog.String("error", err.Error()))
				}
			}(delivery)
		}
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) process(ctx context.Context, delivery domain.WebhookDelivery) error {
	endpoint, ciphertext, err := w.endpoints.FindByIDWithSecretByID(ctx, delivery.EndpointID)
	if err != nil {
		return err
	}
	if !endpoint.IsActive() {
		now := time.Now().UTC()
		delivery.Status = domain.DeliveryDisabled
		delivery.CompletedAt = &now
		return w.deliveries.Complete(ctx, delivery)
	}
	secret, err := w.cipher.Decrypt(ciphertext)
	if err != nil {
		return err
	}

	result := w.client.Deliver(ctx, endpoint.URL, secret, delivery.EventID, delivery.EventType, delivery.Payload)
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
	} else if delivery.AttemptCount >= w.cfg.MaxAttempts || !result.Retryable {
		delivery.Status = domain.DeliveryFailed
		delivery.CompletedAt = &now
		endpoint.FailureCount++
	} else {
		delivery.Status = domain.DeliveryRetryWait
		next := now.Add(backoff(delivery.AttemptCount))
		delivery.NextAttemptAt = &next
		endpoint.FailureCount++
	}

	if endpoint.FailureCount >= w.cfg.FailureThreshold {
		endpoint.Status = domain.EndpointStatusDisabled
		endpoint.DisabledAt = &now
	}
	endpoint.UpdatedAt = now
	_ = w.endpoints.Update(ctx, endpoint)
	return w.deliveries.Complete(ctx, delivery)
}

func backoff(attempt int) time.Duration {
	base := time.Second
	for i := 1; i < attempt; i++ {
		base *= 2
	}
	if base > 5*time.Minute {
		base = 5 * time.Minute
	}
	jitter := time.Duration(rand.Int63n(int64(base / 2)))
	return base + jitter
}

func DefaultWorkerID() string {
	host, _ := os.Hostname()
	if host == "" {
		return "webhook-worker-1"
	}
	return "webhook-worker-" + host
}
