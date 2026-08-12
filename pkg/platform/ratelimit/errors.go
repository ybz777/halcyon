package ratelimit

import "errors"

var (
	// ErrRedisUnavailable indicates the backing store is unavailable.
	ErrRedisUnavailable = errors.New("rate limit redis unavailable")
	// ErrInvalidConfig indicates invalid limiter configuration.
	ErrInvalidConfig = errors.New("invalid rate limit configuration")
)

// FailureMode controls behavior when Redis is unavailable.
type FailureMode string

const (
	FailureModeOpen  FailureMode = "open"
	FailureModeClosed FailureMode = "closed"
)
