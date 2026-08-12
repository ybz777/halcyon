//go:build integration

package integration_test

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func pingRedis(url string) error {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return err
	}
	client := redis.NewClient(opt)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return client.Ping(ctx).Err()
}
