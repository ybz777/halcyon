package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// fixedWindowLua atomically increments a counter and sets TTL on first hit.
// Returns: allowed (1/0), remaining, ttl_ms.
var fixedWindowLua = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
local ttl = redis.call('PTTL', KEYS[1])
local limit = tonumber(ARGV[1])
if current > limit then
  return {0, 0, ttl}
end
return {1, limit - current, ttl}
`)

// multiAllowLua checks multiple keys atomically; rolls back on first exceed.
var multiAllowLua = redis.NewScript(`
local limit = tonumber(ARGV[1])
local ttl_ms = tonumber(ARGV[2])
local counts = {}
for i = 1, #KEYS do
  local current = redis.call('INCR', KEYS[i])
  if current == 1 then
    redis.call('PEXPIRE', KEYS[i], ttl_ms)
  end
  counts[i] = current
  if current > limit then
    for j = 1, i do
      redis.call('DECR', KEYS[j])
    end
    local pttl = redis.call('PTTL', KEYS[i])
    return {0, 0, pttl}
  end
end
local min_remaining = limit
for i = 1, #counts do
  local rem = limit - counts[i]
  if rem < min_remaining then
    min_remaining = rem
  end
end
local pttl = redis.call('PTTL', KEYS[1])
return {1, min_remaining, pttl}
`)

// RedisLimiter enforces distributed fixed-window counters.
type RedisLimiter struct {
	client      redis.UniversalClient
	prefix      string
	failureMode FailureMode
	logger      *slog.Logger
	metrics     *Metrics
}

// NewRedisLimiter creates a Redis-backed limiter.
func NewRedisLimiter(client redis.UniversalClient, cfg Config, logger *slog.Logger, metrics *Metrics) *RedisLimiter {
	if logger == nil {
		logger = slog.Default()
	}
	return &RedisLimiter{
		client:      client,
		prefix:      cfg.RedisPrefix,
		failureMode: cfg.FailureMode,
		logger:      logger,
		metrics:     metrics,
	}
}

func (r *RedisLimiter) Allow(ctx context.Context, identity string, policy Policy) (Decision, error) {
	key := RedisKey(r.prefix, policy.Name, identity)
	return r.eval(ctx, key, policy)
}

func (r *RedisLimiter) AllowAll(ctx context.Context, identities []string, policy Policy) (Decision, error) {
	if len(identities) == 0 {
		return Decision{Allowed: true, Remaining: policy.Limit}, nil
	}
	if len(identities) == 1 {
		return r.Allow(ctx, identities[0], policy)
	}
	keys := make([]string, len(identities))
	for i, id := range identities {
		keys[i] = RedisKey(r.prefix, policy.Name, id)
	}
	return r.evalMulti(ctx, keys, policy)
}

func (r *RedisLimiter) eval(ctx context.Context, key string, policy Policy) (Decision, error) {
	res, err := fixedWindowLua.Run(ctx, r.client, []string{key}, policy.Limit, policy.Window.Milliseconds()).Int64Slice()
	if err != nil {
		return r.handleRedisError(ctx, policy, err)
	}
	return decisionFromResult(res, policy), nil
}

func (r *RedisLimiter) evalMulti(ctx context.Context, keys []string, policy Policy) (Decision, error) {
	res, err := multiAllowLua.Run(ctx, r.client, keys, policy.Limit, policy.Window.Milliseconds()).Int64Slice()
	if err != nil {
		return r.handleRedisError(ctx, policy, err)
	}
	return decisionFromResult(res, policy), nil
}

func decisionFromResult(res []int64, policy Policy) Decision {
	if len(res) < 3 {
		return Decision{Allowed: true, Remaining: policy.Limit}
	}
	allowed := res[0] == 1
	remaining := res[1]
	ttlMs := res[2]
	retryAfter := time.Duration(ttlMs) * time.Millisecond
	if retryAfter < 0 {
		retryAfter = 0
	}
	resetAt := time.Now().UTC().Add(retryAfter)
	return Decision{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetAt:    resetAt,
		RetryAfter: retryAfter,
	}
}

func (r *RedisLimiter) handleRedisError(ctx context.Context, policy Policy, err error) (Decision, error) {
	if r.metrics != nil {
		r.metrics.RedisErrors.Inc()
	}
	r.logger.ErrorContext(ctx, "rate limit redis error",
		slog.String("event", "rate_limit.redis_error"),
		slog.String("policy", policy.Name),
		slog.Any("error", err),
	)
	failClosed := policy.FailClosed || r.failureMode == FailureModeClosed
	if failClosed {
		return Decision{
			Allowed:    false,
			Remaining:  0,
			RetryAfter: 30 * time.Second,
		}, fmt.Errorf("%w: %v", ErrRedisUnavailable, err)
	}
	r.logger.WarnContext(ctx, "rate limit fail-open due to redis error",
		slog.String("event", "rate_limit.fail_open"),
		slog.String("policy", policy.Name),
	)
	return Decision{Allowed: true, Remaining: -1}, nil
}

// NewLimiter returns the appropriate limiter for configuration.
func NewLimiter(cfg Config, client redis.UniversalClient, logger *slog.Logger, metrics *Metrics) Limiter {
	if !cfg.Enabled {
		return DisabledLimiter{}
	}
	return NewRedisLimiter(client, cfg, logger, metrics)
}
