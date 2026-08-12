package idempotency

import (
	"context"
	"time"
)

// State represents the lifecycle of an idempotency record.
type State string

const (
	StateInProgress State = "IN_PROGRESS"
	StateCompleted  State = "COMPLETED"
)

// Record is the stored idempotency state.
type Record struct {
	State       State     `json:"state"`
	Fingerprint string    `json:"fingerprint"`
	Status      int       `json:"status,omitempty"`
	Body        []byte    `json:"body,omitempty"`
	ContentType string    `json:"content_type,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// ClaimResult describes the outcome of an idempotency claim attempt.
type ClaimResult string

const (
	ClaimAcquired   ClaimResult = "acquired"
	ClaimReplay     ClaimResult = "replay"
	ClaimConflict   ClaimResult = "conflict"
	ClaimInProgress ClaimResult = "in_progress"
)

// ClaimOutcome is returned by Store.Claim.
type ClaimOutcome struct {
	Result ClaimResult
	Record Record
}

// Store persists idempotency records.
type Store interface {
	Claim(ctx context.Context, key string, fingerprint string, inProgressTTL time.Duration) (ClaimOutcome, error)
	Complete(ctx context.Context, key string, fingerprint string, record Record, completedTTL time.Duration) error
	Release(ctx context.Context, key string, fingerprint string) error
}
