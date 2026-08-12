package health

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// RedisChecker verifies Redis connectivity for readiness.
type RedisChecker struct {
	client *redis.Client
}

// NewRedisChecker creates a Redis readiness checker.
func NewRedisChecker(client *redis.Client) *RedisChecker {
	return &RedisChecker{client: client}
}

// Check returns dependency status.
func (c *RedisChecker) Check(ctx context.Context) map[string]string {
	status := map[string]string{"redis": "disabled"}
	if c.client == nil {
		return status
	}
	if err := c.client.Ping(ctx).Err(); err != nil {
		status["redis"] = "unhealthy: " + err.Error()
		return status
	}
	status["redis"] = "healthy"
	return status
}
