package idempotency

import (
	"context"
	"fmt"
	"net/http"

	"github.com/redis/go-redis/v9"
)

// RedisMiddleware wires a Redis-backed idempotency middleware.
func RedisMiddleware(ctx context.Context, redisURL string, cfg Config, opts HTTPOptions) (func(http.Handler) http.Handler, *redis.Client, error) {
	if !cfg.Enabled {
		return nil, nil, nil
	}
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opt)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("ping redis: %w", err)
	}
	opts.Store = NewRedisStore(client)
	opts.Config = cfg
	if opts.Logger == nil {
		opts.Logger = nil
	}
	return Middleware(opts), client, nil
}
