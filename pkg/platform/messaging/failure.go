package messaging

import (
	"errors"
	"fmt"
)

// FailureCategory classifies consumer/producer failures.
type FailureCategory string

const (
	FailureTransient  FailureCategory = "transient"
	FailurePermanent  FailureCategory = "permanent"
	FailureBusiness   FailureCategory = "business"
	FailureDeserialize FailureCategory = "deserialize"
)

// TransientError marks retryable failures.
type TransientError struct {
	Err error
}

func (e *TransientError) Error() string { return e.Err.Error() }
func (e *TransientError) Unwrap() error { return e.Err }

// PermanentError marks non-retryable failures.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

// BusinessError marks domain rejections that should not retry indefinitely.
type BusinessError struct {
	Err error
}

func (e *BusinessError) Error() string { return e.Err.Error() }
func (e *BusinessError) Unwrap() error { return e.Err }

// Transient wraps an error as retryable.
func Transient(err error) error {
	if err == nil {
		return nil
	}
	return &TransientError{Err: err}
}

// Permanent wraps an error as non-retryable.
func Permanent(err error) error {
	if err == nil {
		return nil
	}
	return &PermanentError{Err: err}
}

// Business wraps a domain rejection.
func Business(err error) error {
	if err == nil {
		return nil
	}
	return &BusinessError{Err: err}
}

// ClassifyFailure maps errors to failure categories.
func ClassifyFailure(err error) FailureCategory {
	if err == nil {
		return FailureTransient
	}
	var transient *TransientError
	if errors.As(err, &transient) {
		return FailureTransient
	}
	var permanent *PermanentError
	if errors.As(err, &permanent) {
		return FailurePermanent
	}
	var business *BusinessError
	if errors.As(err, &business) {
		return FailureBusiness
	}
	return FailureTransient
}

// ShouldRetry reports whether a failure should be retried.
func ShouldRetry(err error) bool {
	switch ClassifyFailure(err) {
	case FailurePermanent, FailureDeserialize, FailureBusiness:
		return false
	default:
		return true
	}
}

// FailureReason returns a safe string reason for DLQ metadata.
func FailureReason(err error) string {
	if err == nil {
		return "unknown"
	}
	return fmt.Sprintf("%T: %s", err, err.Error())
}
