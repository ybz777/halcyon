package worker

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/engine"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// Config configures the workflow worker.
type Config struct {
	WorkerID       string
	PollInterval   time.Duration
	LeaseDuration  time.Duration
	MaxConcurrent  int
	ClaimBatchSize int
}

// DefaultConfig returns worker defaults.
func DefaultConfig() Config {
	hostname, _ := os.Hostname()
	return Config{
		WorkerID:       hostname,
		PollInterval:   time.Second,
		LeaseDuration:  30 * time.Second,
		MaxConcurrent:  4,
		ClaimBatchSize: 5,
	}
}

// Worker polls and executes workflow jobs.
type Worker struct {
	jobs         ports.JobRepository
	orchestrator *engine.Orchestrator
	metrics      ports.WorkflowEngineMetrics
	cfg          Config
	log          *slog.Logger
}

func NewWorker(jobs ports.JobRepository, orchestrator *engine.Orchestrator, metrics ports.WorkflowEngineMetrics, cfg Config, log *slog.Logger) *Worker {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 1
	}
	if log == nil {
		log = slog.Default()
	}
	return &Worker{jobs: jobs, orchestrator: orchestrator, metrics: metrics, cfg: cfg, log: log}
}

// Run polls until context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	if _, err := w.orchestrator.RecoverLeases(ctx); err != nil {
		w.log.Warn("lease recovery failed", slog.String("error", err.Error()))
	}

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	sem := make(chan struct{}, w.cfg.MaxConcurrent)
	var wg sync.WaitGroup

	for {
		if err := w.refreshQueueDepth(ctx); err != nil {
			w.log.Warn("queue depth refresh failed", slog.String("error", err.Error()))
		}

		jobs, err := w.jobs.ClaimJobs(ctx, w.cfg.WorkerID, w.cfg.ClaimBatchSize, w.cfg.LeaseDuration)
		if err != nil {
			if ctx.Err() != nil {
				wg.Wait()
				return ctx.Err()
			}
			w.log.Error("claim jobs failed", slog.String("error", err.Error()))
		}

		for _, job := range jobs {
			if w.metrics != nil {
				w.metrics.RecordJobClaim()
			}
			sem <- struct{}{}
			wg.Add(1)
			go func(j domain.Job) {
				defer func() {
					<-sem
					wg.Done()
				}()
				if err := w.orchestrator.ProcessJob(ctx, j, w.cfg.WorkerID); err != nil {
					w.log.Error("job processing failed",
						slog.String("job_id", j.ID.String()),
						slog.String("execution_id", j.ExecutionID.String()),
						slog.String("error", err.Error()),
					)
				}
			}(job)
		}

		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) refreshQueueDepth(ctx context.Context) error {
	if w.metrics == nil {
		return nil
	}
	count, err := w.jobs.CountReady(ctx)
	if err != nil {
		return err
	}
	w.metrics.SetJobQueueDepth(count)
	return nil
}
