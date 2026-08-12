package jobs

import (
	"context"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// PoolConfig configures the worker pool.
type PoolConfig struct {
	WorkerID          string
	PollInterval      time.Duration
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	ShutdownTimeout   time.Duration
	MaxConcurrent     int
	ClaimBatchSize    int
	ExecutionTimeout  time.Duration
	RecoveryInterval  time.Duration
	Retry             RetryConfig
	JobTypes          []Type
}

// Pool executes jobs using a handler registry.
type Pool struct {
	repo     Repository
	attempts AttemptRepository
	registry *Registry
	policies *PolicyRegistry
	metrics  MetricsCollector
	cfg      PoolConfig
	log      *slog.Logger
	shutdown atomic.Bool
	active   sync.Map // jobType -> *atomic.Int64
}

// PoolOption configures optional pool dependencies.
type PoolOption func(*Pool)

// WithAttemptRepository records bounded attempt history.
func WithAttemptRepository(repo AttemptRepository) PoolOption {
	return func(p *Pool) { p.attempts = repo }
}

// WithPolicyRegistry supplies per-type execution policies.
func WithPolicyRegistry(registry *PolicyRegistry) PoolOption {
	return func(p *Pool) {
		if registry != nil {
			p.policies = registry
		}
	}
}

func NewPool(repo Repository, registry *Registry, cfg PoolConfig, metrics MetricsCollector, log *slog.Logger, opts ...PoolOption) *Pool {
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 4
	}
	if cfg.ClaimBatchSize <= 0 {
		cfg.ClaimBatchSize = 10
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = 30 * time.Second
	}
	if cfg.ExecutionTimeout <= 0 {
		cfg.ExecutionTimeout = 5 * time.Minute
	}
	if cfg.RecoveryInterval <= 0 {
		cfg.RecoveryInterval = cfg.LeaseDuration
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = 30 * time.Second
	}
	if metrics == nil {
		metrics = NoopMetrics()
	}
	if log == nil {
		log = slog.Default()
	}
	p := &Pool{
		repo: repo, registry: registry, policies: NewPolicyRegistry(),
		metrics: metrics, cfg: cfg, log: log,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Run starts the worker pool until context cancellation.
func (p *Pool) Run(ctx context.Context) error {
	p.recoverLeases(ctx)

	recoveryTicker := time.NewTicker(p.cfg.RecoveryInterval)
	defer recoveryTicker.Stop()

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	sem := make(chan struct{}, p.cfg.MaxConcurrent)
	var wg sync.WaitGroup

	for {
		if p.shutdown.Load() {
			break
		}

		if ctx.Err() != nil {
			break
		}

		jobs, err := p.repo.Claim(ctx, p.cfg.WorkerID, p.cfg.JobTypes, p.cfg.ClaimBatchSize, p.cfg.LeaseDuration)
		if err != nil && ctx.Err() == nil {
			p.log.Error("claim jobs failed", slog.String("error", err.Error()))
		}
		for _, job := range jobs {
			p.metrics.IncClaimed(job.Type)
			if job.CreatedAt.Before(time.Now()) {
				p.metrics.ObserveQueueDelay(job.Type, time.Since(job.CreatedAt))
			}
			sem <- struct{}{}
			wg.Add(1)
			go func(j Job) {
				defer func() { <-sem; wg.Done() }()
				p.execute(ctx, j)
			}(job)
		}

		select {
		case <-ctx.Done():
			p.shutdown.Store(true)
			done := make(chan struct{})
			go func() { wg.Wait(); close(done) }()
			select {
			case <-done:
			case <-time.After(p.cfg.ShutdownTimeout):
				p.log.Warn("shutdown timeout reached with active jobs")
			}
			return ctx.Err()
		case <-recoveryTicker.C:
			p.recoverLeases(ctx)
		case <-ticker.C:
		}
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(p.cfg.ShutdownTimeout):
	}
	return ctx.Err()
}

func (p *Pool) recoverLeases(ctx context.Context) {
	reclaimed, err := p.repo.RecoverLeases(ctx, p.cfg.LeaseDuration, p.cfg.Retry)
	if err != nil {
		p.log.Warn("lease recovery failed", slog.String("error", err.Error()))
		return
	}
	for jobType, count := range reclaimed {
		for i := int64(0); i < count; i++ {
			p.metrics.IncReclaimed(jobType)
			p.metrics.IncLeaseExpired(jobType)
		}
	}
}

func (p *Pool) execute(parent context.Context, job Job) {
	if job.Status == StatusCancelled {
		return
	}

	p.incActive(job.Type, 1)
	defer p.incActive(job.Type, -1)

	tracer := otel.Tracer("jobs")
	ctx, span := tracer.Start(parent, "job "+string(job.Type),
		trace.WithAttributes(
			attribute.String("job.type", string(job.Type)),
			attribute.Int("job.attempt", job.AttemptCount),
			attribute.String("job.priority", string(job.Priority)),
			attribute.String("job.state", string(job.Status)),
		),
	)
	defer span.End()

	timeout := p.cfg.ExecutionTimeout
	if policy := p.policies.Lookup(job.Type); policy.ExecutionTimeout > 0 {
		timeout = policy.ExecutionTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if p.cfg.HeartbeatInterval > 0 {
		stop := p.startHeartbeat(ctx, job)
		defer stop()
	}

	handler, ok := p.registry.Get(job.Type)
	if !ok {
		_ = p.repo.Fail(ctx, p.cfg.WorkerID, job.ID, "unknown job type", CategoryValidation, time.Now().UTC(), false)
		p.metrics.IncFailed(job.Type, CategoryValidation)
		return
	}

	attemptNum := job.AttemptCount
	var attemptID AttemptID
	var recordAttempt bool
	if p.attempts != nil {
		if id, err := NewAttemptID(); err == nil {
			attemptID = id
			recordAttempt = true
			_ = p.attempts.Create(ctx, Attempt{
				ID: attemptID, JobID: job.ID, AttemptNumber: attemptNum,
				WorkerID: p.cfg.WorkerID, Status: AttemptRunning, StartedAt: time.Now().UTC(),
			})
		}
	}

	start := time.Now()
	err := p.runHandler(ctx, handler, job)
	duration := time.Since(start)
	p.metrics.ObserveDuration(job.Type, duration)

	now := time.Now().UTC()
	if err == nil {
		p.finishAttempt(ctx, recordAttempt, attemptID, AttemptSucceeded, "", "", duration)
		if completeErr := p.repo.Complete(ctx, p.cfg.WorkerID, job.ID, now); completeErr != nil {
			p.log.Warn("complete job failed", slog.String("job_id", job.ID.String()), slog.String("error", completeErr.Error()))
			return
		}
		p.log.Info("job succeeded",
			slog.String("job_id", job.ID.String()),
			slog.String("job_type", string(job.Type)),
			slog.Int("attempt", job.AttemptCount),
			slog.Duration("duration", duration),
		)
		p.metrics.IncSucceeded(job.Type)
		return
	}

	if errorsIsCancelled(err) {
		_ = p.repo.Cancel(ctx, job.ID, now)
		p.metrics.IncCancelled(job.Type)
		return
	}

	category := CategoryOf(err)
	if category == CategoryUnknown {
		category = ClassifyError(err)
	}
	lastErr := truncate(err.Error(), 500)

	exhausted := job.AttemptCount >= job.MaxAttempts
	if !IsRetryable(err) || exhausted {
		span.SetStatus(codes.Error, string(category))
		p.finishAttempt(ctx, recordAttempt, attemptID, AttemptFailed, category, lastErr, duration)
		deadLetter := exhausted
		_ = p.repo.Fail(ctx, p.cfg.WorkerID, job.ID, lastErr, category, now, deadLetter)
		if deadLetter {
			p.metrics.IncDeadLettered(job.Type)
		} else {
			p.metrics.IncFailed(job.Type, category)
		}
		p.log.Warn("job failed",
			slog.String("job_id", job.ID.String()),
			slog.String("job_type", string(job.Type)),
			slog.Int("attempt", job.AttemptCount),
			slog.String("error_category", string(category)),
			slog.Bool("dead_lettered", deadLetter),
		)
		return
	}

	p.finishAttempt(ctx, recordAttempt, attemptID, AttemptFailed, category, lastErr, duration)
	next := NextAvailableAt(p.cfg.Retry, job.AttemptCount, now)
	if retryErr := p.repo.Retry(ctx, p.cfg.WorkerID, job.ID, next, lastErr, category); retryErr != nil {
		p.log.Warn("retry job failed", slog.String("job_id", job.ID.String()), slog.String("error", retryErr.Error()))
		return
	}
	p.log.Info("job scheduled for retry",
		slog.String("job_id", job.ID.String()),
		slog.String("job_type", string(job.Type)),
		slog.Int("attempt", job.AttemptCount),
		slog.String("error_category", string(category)),
		slog.Time("next_attempt_at", next),
	)
	p.metrics.IncRetried(job.Type)
}

func (p *Pool) runHandler(ctx context.Context, handler Handler, job Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			p.metrics.IncPanics(job.Type)
			p.log.Error("job handler panic",
				slog.String("job_id", job.ID.String()),
				slog.String("job_type", string(job.Type)),
				slog.Int("attempt", job.AttemptCount),
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())),
			)
			err = &HandlerError{
				Category:  CategoryInternal,
				Message:   "handler panic",
				Retryable: true,
			}
		}
	}()
	err = handler.Handle(ctx, job)
	if ctx.Err() == context.DeadlineExceeded {
		return Timeout(ctx.Err())
	}
	return err
}

func (p *Pool) startHeartbeat(ctx context.Context, job Job) func() {
	if p.cfg.HeartbeatInterval <= 0 {
		return func() {}
	}
	ticker := time.NewTicker(p.cfg.HeartbeatInterval)
	done := make(chan struct{})
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = p.repo.RenewLease(ctx, p.cfg.WorkerID, job.ID, p.cfg.LeaseDuration)
			}
		}
	}()
	return func() { close(done) }
}

func (p *Pool) incActive(jobType Type, delta int64) {
	val, _ := p.active.LoadOrStore(jobType, &atomic.Int64{})
	counter := val.(*atomic.Int64)
	current := counter.Add(delta)
	if current < 0 {
		counter.Store(0)
		current = 0
	}
	p.metrics.SetActive(jobType, float64(current))
}

func errorsIsCancelled(err error) bool {
	if he, ok := err.(*HandlerError); ok {
		return he.Category == CategoryCancelled
	}
	return false
}

func truncate(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}

func (p *Pool) finishAttempt(ctx context.Context, enabled bool, id AttemptID, status AttemptStatus, category ErrorCategory, message string, duration time.Duration) {
	if !enabled || p.attempts == nil {
		return
	}
	now := time.Now().UTC()
	_ = p.attempts.Complete(ctx, Attempt{
		ID: id, Status: status, ErrorCategory: category, ErrorMessage: truncate(message, 500),
		CompletedAt: &now, DurationMs: int(duration.Milliseconds()),
	})
}
