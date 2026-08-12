package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// concurrencyLua acquires a slot or rejects when at capacity.
var concurrencyLua = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[2])
end
local limit = tonumber(ARGV[1])
if current > limit then
  redis.call('DECR', KEYS[1])
  local ttl = redis.call('PTTL', KEYS[1])
  return {0, ttl}
end
return {1, 0}
`)

// releaseConcurrencyLua releases a slot.
var releaseConcurrencyLua = redis.NewScript(`
local current = redis.call('DECR', KEYS[1])
if current <= 0 then
  redis.call('DEL', KEYS[1])
end
return current
`)

// ConcurrencyLimiter bounds concurrent expensive operations.
type ConcurrencyLimiter struct {
	client  redis.UniversalClient
	prefix  string
	limit   int
	ttl     time.Duration
	logger  *slog.Logger
	metrics *Metrics
}

// NewConcurrencyLimiter creates a distributed semaphore.
func NewConcurrencyLimiter(client redis.UniversalClient, prefix string, limit int, ttl time.Duration, logger *slog.Logger, metrics *Metrics) *ConcurrencyLimiter {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &ConcurrencyLimiter{client: client, prefix: prefix, limit: limit, ttl: ttl, logger: logger, metrics: metrics}
}

// Acquire attempts to take a concurrency slot.
func (c *ConcurrencyLimiter) Acquire(ctx context.Context, identity string) (bool, time.Duration, error) {
	if c == nil || c.client == nil || c.limit <= 0 {
		return true, 0, nil
	}
	key := RedisKey(c.prefix, "concurrency", identity)
	res, err := concurrencyLua.Run(ctx, c.client, []string{key}, c.limit, c.ttl.Milliseconds()).Int64Slice()
	if err != nil {
		if c.metrics != nil {
			c.metrics.RedisErrors.Inc()
		}
		return true, 0, fmt.Errorf("%w: %v", ErrRedisUnavailable, err)
	}
	if len(res) < 2 {
		return true, 0, nil
	}
	if res[0] == 0 {
		if c.metrics != nil {
			c.metrics.ConcurrencyRejected.Inc()
		}
		retry := time.Duration(res[1]) * time.Millisecond
		if retry < 0 {
			retry = time.Second
		}
		return false, retry, nil
	}
	return true, 0, nil
}

// Release returns a concurrency slot.
func (c *ConcurrencyLimiter) Release(ctx context.Context, identity string) error {
	if c == nil || c.client == nil || c.limit <= 0 {
		return nil
	}
	key := RedisKey(c.prefix, "concurrency", identity)
	_, err := releaseConcurrencyLua.Run(ctx, c.client, []string{key}).Result()
	return err
}
