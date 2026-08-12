package jobs

import (
	"errors"
	"net/http"
	"strings"
)

var (
	ErrInvalidJobID      = errors.New("invalid job id")
	ErrInvalidJobType    = errors.New("invalid job type")
	ErrJobNotFound       = errors.New("job not found")
	ErrStaleLease        = errors.New("stale worker lease")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrPayloadTooLarge   = errors.New("job payload too large")
	ErrCancelled         = errors.New("job cancelled")
)

// ErrorCategory classifies handler failures for retry policy.
type ErrorCategory string

const (
	CategoryValidation    ErrorCategory = "VALIDATION"
	CategoryAuthorization ErrorCategory = "AUTHORIZATION"
	CategoryNotFound      ErrorCategory = "NOT_FOUND"
	CategoryConflict      ErrorCategory = "CONFLICT"
	CategoryTransient     ErrorCategory = "TRANSIENT"
	CategoryDependency    ErrorCategory = "DEPENDENCY"
	CategoryRateLimited   ErrorCategory = "RATE_LIMITED"
	CategoryTimeout       ErrorCategory = "TIMEOUT"
	CategoryInternal      ErrorCategory = "INTERNAL"
	CategoryCancelled     ErrorCategory = "CANCELLED"
	CategoryUnknown       ErrorCategory = "UNKNOWN"
)

// HandlerError is returned by job handlers to control retry behavior.
type HandlerError struct {
	Category  ErrorCategory
	Message   string
	Retryable bool
}

func (e *HandlerError) Error() string {
	if e.Message == "" {
		return string(e.Category)
	}
	return e.Message
}

func Retryable(err error) *HandlerError {
	return &HandlerError{Category: CategoryTransient, Message: err.Error(), Retryable: true}
}

func RateLimited(err error) *HandlerError {
	return &HandlerError{Category: CategoryRateLimited, Message: err.Error(), Retryable: true}
}

func Timeout(err error) *HandlerError {
	return &HandlerError{Category: CategoryTimeout, Message: err.Error(), Retryable: true}
}

func Dependency(err error) *HandlerError {
	return &HandlerError{Category: CategoryDependency, Message: err.Error(), Retryable: true}
}

func Permanent(err error) *HandlerError {
	return &HandlerError{Category: CategoryValidation, Message: err.Error(), Retryable: false}
}

func Cancelled() *HandlerError {
	return &HandlerError{Category: CategoryCancelled, Message: "cancelled", Retryable: false}
}

func IsRetryable(err error) bool {
	if he, ok := err.(*HandlerError); ok {
		return he.Retryable
	}
	return true
}

func CategoryOf(err error) ErrorCategory {
	if he, ok := err.(*HandlerError); ok {
		if he.Category != "" {
			return he.Category
		}
	}
	return CategoryInternal
}

// ClassifyHTTPStatus maps HTTP response codes to handler errors.
func ClassifyHTTPStatus(statusCode int, message string) *HandlerError {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return nil
	case statusCode == http.StatusTooManyRequests:
		return RateLimited(errors.New(message))
	case statusCode == http.StatusRequestTimeout, statusCode == http.StatusGatewayTimeout:
		return Timeout(errors.New(message))
	case statusCode == http.StatusBadGateway, statusCode == http.StatusServiceUnavailable:
		return Dependency(errors.New(message))
	case statusCode >= 400 && statusCode < 500:
		return Permanent(errors.New(message))
	default:
		return Retryable(errors.New(message))
	}
}

// ClassifyError maps common infrastructure errors to categories.
func ClassifyError(err error) ErrorCategory {
	if err == nil {
		return ""
	}
	if he, ok := err.(*HandlerError); ok && he.Category != "" {
		return he.Category
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline exceeded"):
		return CategoryTimeout
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "temporary"):
		return CategoryTransient
	case strings.Contains(msg, "unauthorized"), strings.Contains(msg, "forbidden"):
		return CategoryAuthorization
	case strings.Contains(msg, "not found"):
		return CategoryNotFound
	case strings.Contains(msg, "invalid"), strings.Contains(msg, "validation"):
		return CategoryValidation
	default:
		return CategoryUnknown
	}
}
