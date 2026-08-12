package worker

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/halcyon-io/halcyon/services/notifications/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/email"
	"github.com/halcyon-io/halcyon/services/notifications/internal/infrastructure/postgres"
)

type Config struct {
	WorkerID       string
	PollInterval   time.Duration
	LeaseDuration  time.Duration
	MaxConcurrent  int
	ClaimBatchSize int
	MaxAttempts    int
	FromName       string
	FromAddress    string
	Timeout        time.Duration
}

type Worker struct {
	deliveries    *postgres.EmailDeliveryRepository
	notifications *postgres.NotificationRepository
	sender        ports.EmailSender
	cfg           Config
	log           *slog.Logger
}

func NewWorker(deliveries *postgres.EmailDeliveryRepository, notifications *postgres.NotificationRepository, sender ports.EmailSender, cfg Config, log *slog.Logger) *Worker {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 4
	}
	if log == nil {
		log = slog.Default()
	}
	return &Worker{deliveries: deliveries, notifications: notifications, sender: sender, cfg: cfg, log: log}
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
			w.log.Error("claim email deliveries failed", slog.String("error", err.Error()))
		}
		for _, delivery := range deliveries {
			sem <- struct{}{}
			wg.Add(1)
			go func(d domain.EmailDelivery) {
				defer func() { <-sem; wg.Done() }()
				if err := w.process(ctx, d); err != nil {
					w.log.Error("email delivery failed", slog.String("delivery_id", d.ID.String()), slog.String("error", err.Error()))
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

func (w *Worker) process(ctx context.Context, delivery domain.EmailDelivery) error {
	notification, err := w.notifications.FindByIDOnly(ctx, delivery.NotificationID)
	if err != nil {
		return err
	}
	msg := email.Render(w.cfg.FromName, w.cfg.FromAddress, notification, delivery.Email)
	sendCtx, cancel := context.WithTimeout(ctx, w.cfg.Timeout)
	defer cancel()
	result, err := w.sender.Send(sendCtx, msg)
	now := time.Now().UTC()
	delivery.LastAttemptAt = &now
	delivery.AttemptCount++
	delivery.Provider = w.sender.ProviderName()

	if err == nil && result.ErrorSummary == "" {
		delivery.Status = domain.EmailSent
		delivery.ProviderMessageID = result.ProviderMessageID
		delivery.CompletedAt = &now
	} else if delivery.AttemptCount >= w.cfg.MaxAttempts || (err == nil && !result.Retryable) {
		delivery.Status = domain.EmailFailed
		delivery.CompletedAt = &now
		if err != nil {
			delivery.LastError = truncate(err.Error(), 200)
		} else {
			delivery.LastError = truncate(result.ErrorSummary, 200)
		}
	} else {
		delivery.Status = domain.EmailRetryWait
		next := now.Add(backoff(delivery.AttemptCount))
		delivery.NextAttemptAt = &next
		if err != nil {
			delivery.LastError = truncate(err.Error(), 200)
		} else {
			delivery.LastError = truncate(result.ErrorSummary, 200)
		}
	}
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
	return base + time.Duration(rand.Int63n(int64(base/2)))
}

func truncate(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}

func DefaultWorkerID() string {
	host, _ := os.Hostname()
	if host == "" {
		return "email-worker-1"
	}
	return "email-worker-" + host
}
