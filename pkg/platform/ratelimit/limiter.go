package ratelimit

import (
	"context"
	"time"
)

// Decision describes a rate-limit check outcome.
type Decision struct {
	Allowed    bool
	Remaining  int64
	ResetAt    time.Time
	RetryAfter time.Duration
}

// Policy defines a fixed-window rate limit.
type Policy struct {
	Name       string
	Limit      int64
	Window     time.Duration
	FailClosed bool
}

// Limiter enforces distributed rate limits.
type Limiter interface {
	Allow(ctx context.Context, key string, policy Policy) (Decision, error)
}

// MultiLimiter checks multiple keys; all policies must allow the request.
type MultiLimiter interface {
	AllowAll(ctx context.Context, keys []string, policy Policy) (Decision, error)
}

// Clock abstracts time for tests.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// DisabledLimiter allows all requests when rate limiting is off.
type DisabledLimiter struct{}

func (DisabledLimiter) Allow(context.Context, string, Policy) (Decision, error) {
	return Decision{Allowed: true, Remaining: -1}, nil
}
