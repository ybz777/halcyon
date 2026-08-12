package jobs

import (
	"context"
	"time"
)

// CreateInput carries job creation parameters.
type CreateInput struct {
	OrganizationID string
	Type           Type
	Payload        []byte
	Priority       Priority
	MaxAttempts    int
	IdempotencyKey string
	AvailableAt    time.Time
}

// Repository persists and claims durable jobs.
type Repository interface {
	Create(ctx context.Context, input CreateInput) (Job, bool, error)
	Get(ctx context.Context, id ID) (Job, error)
	GetForOrg(ctx context.Context, orgID string, id ID) (Job, error)
	List(ctx context.Context, orgID string, limit int, cursor string) ([]Job, string, error)
	ListPlatform(ctx context.Context, limit int, cursor string) ([]Job, string, error)
	Claim(ctx context.Context, workerID string, types []Type, limit int, lease time.Duration) ([]Job, error)
	Complete(ctx context.Context, workerID string, id ID, at time.Time) error
	Retry(ctx context.Context, workerID string, id ID, next time.Time, lastError string, category ErrorCategory) error
	Fail(ctx context.Context, workerID string, id ID, lastError string, category ErrorCategory, at time.Time, deadLetter bool) error
	Cancel(ctx context.Context, id ID, at time.Time) error
	RecoverLeases(ctx context.Context, lease time.Duration, cfg RetryConfig) (map[Type]int64, error)
	RenewLease(ctx context.Context, workerID string, id ID, lease time.Duration) error
	CountReady(ctx context.Context, jobType Type) (int64, error)
}

// MetricsCollector records bounded job metrics.
type MetricsCollector interface {
	IncCreated(jobType Type)
	IncClaimed(jobType Type)
	IncSucceeded(jobType Type)
	IncFailed(jobType Type, category ErrorCategory)
	IncRetried(jobType Type)
	IncCancelled(jobType Type)
	IncReclaimed(jobType Type)
	IncDeadLettered(jobType Type)
	IncLeaseExpired(jobType Type)
	IncPanics(jobType Type)
	ObserveDuration(jobType Type, d time.Duration)
	ObserveQueueDelay(jobType Type, d time.Duration)
	SetActive(jobType Type, count float64)
}

type noopMetrics struct{}

func (noopMetrics) IncCreated(Type)                       {}
func (noopMetrics) IncClaimed(Type)                       {}
func (noopMetrics) IncSucceeded(Type)                     {}
func (noopMetrics) IncFailed(Type, ErrorCategory)         {}
func (noopMetrics) IncRetried(Type)                       {}
func (noopMetrics) IncCancelled(Type)                     {}
func (noopMetrics) IncReclaimed(Type)                     {}
func (noopMetrics) IncDeadLettered(Type)                  {}
func (noopMetrics) IncLeaseExpired(Type)                  {}
func (noopMetrics) IncPanics(Type)                        {}
func (noopMetrics) ObserveDuration(Type, time.Duration)   {}
func (noopMetrics) ObserveQueueDelay(Type, time.Duration) {}
func (noopMetrics) SetActive(Type, float64)               {}

// NoopMetrics returns a metrics collector that discards events.
func NoopMetrics() MetricsCollector { return noopMetrics{} }
