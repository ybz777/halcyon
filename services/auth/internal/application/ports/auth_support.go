package ports

import (
	"context"
	"time"
)

// LoginRateLimiter gates authentication attempts.
type LoginRateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// AuthMetrics records authentication observability signals.
type AuthMetrics interface {
	RecordLoginAttempt()
	RecordLoginFailure(reason string)
	RecordTokenValidation(success bool)
	RecordTokenRefresh(success bool)
	RecordRefreshSuccess()
	RecordRefreshRotation()
	RecordRefreshReplay()
	RecordFamilyRevoked()
	RecordAPIKeyCreation()
	RecordAPIKeyAuthSuccess()
	RecordAPIKeyAuthFailure(reason string)
	RecordAPIKeyRotation()
	RecordAPIKeyRevoked()
	ObserveAuthLatency(operation string, start time.Time)
}

// NoopLoginRateLimiter allows all login attempts.
type NoopLoginRateLimiter struct{}

// Allow implements LoginRateLimiter.
func (NoopLoginRateLimiter) Allow(context.Context, string) (bool, error) {
	return true, nil
}
