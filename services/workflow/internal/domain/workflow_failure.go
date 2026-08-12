package domain

import (
	"math/rand"
	"time"
)

// FailureCategory classifies execution failures.
type FailureCategory string

const (
	FailureTransient FailureCategory = "TRANSIENT"
	FailurePermanent FailureCategory = "PERMANENT"
	FailureTimeout   FailureCategory = "TIMEOUT"
	FailureCancelled FailureCategory = "CANCELLED"
)

// ExecutionError is a structured step execution error.
type ExecutionError struct {
	Category FailureCategory
	Message  string
	Cause    error
}

func (e *ExecutionError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return string(e.Category)
}

func (e *ExecutionError) Unwrap() error { return e.Cause }

// TransientExecutionError marks retryable failures.
func TransientExecutionError(err error) error {
	return &ExecutionError{Category: FailureTransient, Cause: err, Message: err.Error()}
}

// PermanentExecutionError marks non-retryable failures.
func PermanentExecutionError(err error) error {
	return &ExecutionError{Category: FailurePermanent, Cause: err, Message: err.Error()}
}

// TimeoutExecutionError marks timeout failures.
func TimeoutExecutionError(err error) error {
	return &ExecutionError{Category: FailureTimeout, Cause: err, Message: err.Error()}
}

// CancelledExecutionError marks cancellation.
func CancelledExecutionError() error {
	return &ExecutionError{Category: FailureCancelled, Message: "execution cancelled"}
}

// ClassifyExecutionFailure maps errors to categories.
func ClassifyExecutionFailure(err error) FailureCategory {
	if err == nil {
		return FailureTransient
	}
	var execErr *ExecutionError
	if AsExecutionError(err, &execErr) {
		return execErr.Category
	}
	return FailureTransient
}

// ShouldRetryExecution reports whether the engine should retry.
func ShouldRetryExecution(err error) bool {
	switch ClassifyExecutionFailure(err) {
	case FailurePermanent, FailureCancelled:
		return false
	default:
		return true
	}
}

// AsExecutionError is a helper for errors.As without importing errors in callers.
func AsExecutionError(err error, target **ExecutionError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*ExecutionError); ok {
		*target = e
		return true
	}
	return false
}

// ComputeBackoff calculates retry delay with jitter.
func ComputeBackoff(policy RetryPolicy, attempt int) time.Duration {
	if attempt <= 1 {
		return policy.InitialBackoff
	}
	delay := policy.InitialBackoff
	for i := 1; i < attempt; i++ {
		mult := policy.BackoffMultiplier
		if mult <= 0 {
			mult = 2
		}
		delay = time.Duration(float64(delay) * mult)
		if delay > policy.MaxBackoff {
			delay = policy.MaxBackoff
			break
		}
	}
	jitter := time.Duration(rand.Int63n(int64(delay / 5)))
	return delay + jitter
}
