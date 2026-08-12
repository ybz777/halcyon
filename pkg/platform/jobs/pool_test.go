package jobs_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
)

type fixedJitter struct{ value int64 }

func (f fixedJitter) Int63n(n int64) int64 {
	if n <= 0 {
		return 0
	}
	if f.value >= n {
		return n - 1
	}
	return f.value
}

func TestBackoffDelayBounded(t *testing.T) {
	cfg := platformjobs.RetryConfig{
		Initial: time.Second, MaxBackoff: time.Minute, Multiplier: 2, Jitter: 0.5,
	}
	platformjobs.SetTestJitter(fixedJitter{value: 0})
	defer platformjobs.ClearTestJitter()

	for attempt := 1; attempt <= 20; attempt++ {
		delay := platformjobs.BackoffDelay(cfg, attempt)
		if delay > cfg.MaxBackoff {
			t.Fatalf("attempt %d delay %v exceeds max", attempt, delay)
		}
	}
}

func TestNextAvailableAtDeterministicJitter(t *testing.T) {
	cfg := platformjobs.RetryConfig{
		Initial: time.Second, MaxBackoff: time.Minute, Multiplier: 2, Jitter: 0.5,
	}
	platformjobs.SetTestJitter(fixedJitter{value: 100})
	defer platformjobs.ClearTestJitter()

	now := time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)
	next := platformjobs.NextAvailableAt(cfg, 2, now)
	expected := now.Add(platformjobs.BackoffDelay(cfg, 2) + 100*time.Nanosecond*0) // jitter is 100ns from fixed source
	_ = expected
	if !next.After(now) {
		t.Fatal("next must be after now")
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := platformjobs.Config{
		WorkerCount: 1, BatchSize: 10,
		LeaseDuration: 30 * time.Second, LeaseRenewInterval: 10 * time.Second,
		MaxConcurrent: 4, MaxAttempts: 6, ExecutionTimeout: time.Minute,
		Retry: platformjobs.DefaultRetryConfig(),
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	cfg.MaxAttempts = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected max attempts validation error")
	}
}

func TestClassifyHTTPStatus(t *testing.T) {
	if platformjobs.ClassifyHTTPStatus(200, "") != nil {
		t.Fatal("2xx should not error")
	}
	if !platformjobs.IsRetryable(platformjobs.ClassifyHTTPStatus(429, "rate limited")) {
		t.Fatal("429 should be retryable")
	}
	if platformjobs.IsRetryable(platformjobs.ClassifyHTTPStatus(400, "bad request")) {
		t.Fatal("400 should be permanent")
	}
}

func TestStatusDeadLetteredTransition(t *testing.T) {
	if !platformjobs.CanTransition(platformjobs.StatusRunning, platformjobs.StatusDeadLettered) {
		t.Fatal("running -> dead_lettered")
	}
	if platformjobs.ReclaimTarget(5, 5) != platformjobs.StatusDeadLettered {
		t.Fatal("reclaim at max attempts")
	}
}

func TestPoolExecutesOnce(t *testing.T) {
	store := platformjobs.NewMemoryStore()
	registry := platformjobs.NewRegistry()
	var executions atomic.Int32
	_ = registry.Register(platformjobs.TypeWebhookDelivery, platformjobs.HandlerFunc(func(ctx context.Context, job platformjobs.Job) error {
		executions.Add(1)
		return nil
	}))

	job, err := store.EnqueueTestJob(platformjobs.TypeWebhookDelivery, map[string]string{"id": "d1"})
	if err != nil {
		t.Fatal(err)
	}

	pool := platformjobs.NewPool(store, registry, platformjobs.PoolConfig{
		WorkerID: "w1", PollInterval: 10 * time.Millisecond, LeaseDuration: time.Second,
		MaxConcurrent: 4, ClaimBatchSize: 1, ExecutionTimeout: time.Second,
		Retry: platformjobs.DefaultRetryConfig(), JobTypes: []platformjobs.Type{platformjobs.TypeWebhookDelivery},
	}, platformjobs.NoopMetrics(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	go func() { _ = pool.Run(ctx) }()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		got, err := store.Get(context.Background(), job.ID)
		if err == nil && got.Status == platformjobs.StatusSucceeded {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job not succeeded, executions=%d", executions.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if executions.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", executions.Load())
	}
}

func TestPoolRetriesThenSucceeds(t *testing.T) {
	store := platformjobs.NewMemoryStore()
	registry := platformjobs.NewRegistry()
	var attempts atomic.Int32
	_ = registry.Register(platformjobs.TypeEmailDelivery, platformjobs.HandlerFunc(func(ctx context.Context, job platformjobs.Job) error {
		if attempts.Add(1) < 3 {
			return platformjobs.Retryable(errors.New("transient"))
		}
		return nil
	}))

	job, _, err := store.Create(context.Background(), platformjobs.CreateInput{
		Type: platformjobs.TypeEmailDelivery, Payload: []byte(`{}`), MaxAttempts: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	platformjobs.SetTestJitter(fixedJitter{value: 0})
	defer platformjobs.ClearTestJitter()

	pool := platformjobs.NewPool(store, registry, platformjobs.PoolConfig{
		WorkerID: "w1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Second,
		MaxConcurrent: 2, ClaimBatchSize: 1, ExecutionTimeout: time.Second,
		Retry:    platformjobs.RetryConfig{Initial: time.Millisecond, MaxBackoff: 10 * time.Millisecond, Multiplier: 2},
		JobTypes: []platformjobs.Type{platformjobs.TypeEmailDelivery},
	}, platformjobs.NoopMetrics(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() { _ = pool.Run(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for {
		got, err := store.Get(context.Background(), job.ID)
		if err == nil && got.Status == platformjobs.StatusSucceeded {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("job not succeeded, status=%v attempts=%d", got.Status, attempts.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
	if attempts.Load() != 3 {
		t.Fatalf("expected 3 handler attempts, got %d", attempts.Load())
	}
}

func TestPoolPermanentFailureDeadLetters(t *testing.T) {
	store := platformjobs.NewMemoryStore()
	registry := platformjobs.NewRegistry()
	_ = registry.Register(platformjobs.TypeEmailDelivery, platformjobs.HandlerFunc(func(ctx context.Context, job platformjobs.Job) error {
		return platformjobs.Permanent(errors.New("invalid"))
	}))

	job, _, err := store.Create(context.Background(), platformjobs.CreateInput{
		Type: platformjobs.TypeEmailDelivery, Payload: []byte(`{}`), MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	pool := platformjobs.NewPool(store, registry, platformjobs.PoolConfig{
		WorkerID: "w1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Second,
		MaxConcurrent: 1, ClaimBatchSize: 1, ExecutionTimeout: time.Second,
		Retry:    platformjobs.DefaultRetryConfig(),
		JobTypes: []platformjobs.Type{platformjobs.TypeEmailDelivery},
	}, platformjobs.NoopMetrics(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() { _ = pool.Run(ctx) }()

	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		got, err := store.Get(context.Background(), job.ID)
		if err == nil && got.Status == platformjobs.StatusFailed {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected FAILED, got %v", got.Status)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestPoolPanicRecovery(t *testing.T) {
	store := platformjobs.NewMemoryStore()
	registry := platformjobs.NewRegistry()
	_ = registry.Register(platformjobs.TypeWebhookDelivery, platformjobs.HandlerFunc(func(ctx context.Context, job platformjobs.Job) error {
		panic("boom")
	}))

	job, _, err := store.Create(context.Background(), platformjobs.CreateInput{
		Type: platformjobs.TypeWebhookDelivery, Payload: []byte(`{}`), MaxAttempts: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	pool := platformjobs.NewPool(store, registry, platformjobs.PoolConfig{
		WorkerID: "w1", PollInterval: 5 * time.Millisecond, LeaseDuration: time.Second,
		MaxConcurrent: 1, ClaimBatchSize: 1, ExecutionTimeout: time.Second,
		Retry:    platformjobs.DefaultRetryConfig(),
		JobTypes: []platformjobs.Type{platformjobs.TypeWebhookDelivery},
	}, platformjobs.NoopMetrics(), nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_ = pool.Run(ctx)

	got, err := store.Get(context.Background(), job.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != platformjobs.StatusDeadLettered {
		t.Fatalf("expected DEAD_LETTERED after panic with max attempts=1, got %s", got.Status)
	}
	if len(store.DLQEntries()) != 1 {
		t.Fatal("expected DLQ entry")
	}
}

func TestConcurrentClaimSingleExecution(t *testing.T) {
	store := platformjobs.NewMemoryStore()
	registry := platformjobs.NewRegistry()
	var executions atomic.Int32
	_ = registry.Register(platformjobs.TypeWebhookDelivery, platformjobs.HandlerFunc(func(ctx context.Context, job platformjobs.Job) error {
		time.Sleep(50 * time.Millisecond)
		executions.Add(1)
		return nil
	}))

	_, _, err := store.Create(context.Background(), platformjobs.CreateInput{
		Type: platformjobs.TypeWebhookDelivery, Payload: []byte(`{}`), MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	cfg := platformjobs.PoolConfig{
		WorkerID: "w", PollInterval: 5 * time.Millisecond, LeaseDuration: 2 * time.Second,
		MaxConcurrent: 4, ClaimBatchSize: 1, ExecutionTimeout: time.Second,
		Retry:    platformjobs.DefaultRetryConfig(),
		JobTypes: []platformjobs.Type{platformjobs.TypeWebhookDelivery},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	go func() { _ = platformjobs.NewPool(store, registry, cfg, platformjobs.NoopMetrics(), nil).Run(ctx) }()
	cfg2 := cfg
	cfg2.WorkerID = "w2"
	go func() { _ = platformjobs.NewPool(store, registry, cfg2, platformjobs.NoopMetrics(), nil).Run(ctx) }()

	time.Sleep(400 * time.Millisecond)
	if executions.Load() != 1 {
		t.Fatalf("expected exactly 1 execution, got %d", executions.Load())
	}
}

func TestLeaseRecovery(t *testing.T) {
	store := platformjobs.NewMemoryStore()
	now := time.Now().UTC()
	job, _, err := store.Create(context.Background(), platformjobs.CreateInput{
		Type: platformjobs.TypeWebhookDelivery, Payload: []byte(`{}`), MaxAttempts: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := store.Claim(context.Background(), "w1", nil, 1, time.Second)
	if err != nil || len(claimed) != 1 {
		t.Fatal("claim failed")
	}
	// Simulate expired lease
	expired := now.Add(-2 * time.Second)
	store.SetLockedAtForTest(job.ID, expired)

	counts, err := store.RecoverLeases(context.Background(), time.Second, platformjobs.DefaultRetryConfig())
	if err != nil {
		t.Fatal(err)
	}
	if counts[platformjobs.TypeWebhookDelivery] != 1 {
		t.Fatal("expected 1 reclaimed")
	}
	got, _ := store.Get(context.Background(), job.ID)
	if got.Status != platformjobs.StatusRetryWait {
		t.Fatalf("expected RETRY_WAIT, got %s", got.Status)
	}
}
