package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const attemptIDPrefix = "jat_"

// AttemptStatus records the outcome of a single execution try.
type AttemptStatus string

const (
	AttemptRunning   AttemptStatus = "RUNNING"
	AttemptSucceeded AttemptStatus = "SUCCEEDED"
	AttemptFailed    AttemptStatus = "FAILED"
)

// AttemptID is a typed attempt identifier.
type AttemptID struct{ value string }

func NewAttemptID() (AttemptID, error) {
	raw, err := ids.Generate(attemptIDPrefix)
	if err != nil {
		return AttemptID{}, err
	}
	return AttemptID{value: raw}, nil
}

func ParseAttemptID(raw string) (AttemptID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, attemptIDPrefix) {
		return AttemptID{}, fmt.Errorf("invalid attempt id")
	}
	return AttemptID{value: raw}, nil
}

func (id AttemptID) String() string { return id.value }

// Attempt records one job execution try.
type Attempt struct {
	ID            AttemptID
	JobID         ID
	AttemptNumber int
	WorkerID      string
	Status        AttemptStatus
	ErrorCategory ErrorCategory
	ErrorMessage  string
	StartedAt     time.Time
	CompletedAt   *time.Time
	DurationMs    int
}

// AttemptRepository stores bounded attempt history.
type AttemptRepository interface {
	Create(ctx context.Context, attempt Attempt) error
	Complete(ctx context.Context, attempt Attempt) error
	ListByJob(ctx context.Context, jobID ID, limit int) ([]Attempt, error)
}
