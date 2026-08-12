//go:build integration

package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestRedisLimiter_AtomicAcrossClients(t *testing.T) {
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	addr := host + ":" + port.Port()
	clientA := redis.NewClient(&redis.Options{Addr: addr})
	clientB := redis.NewClient(&redis.Options{Addr: addr})
	cfg := Config{Enabled: true, RedisPrefix: "integration", FailureMode: FailureModeClosed}
	limiterA := NewRedisLimiter(clientA, cfg, nil, nil)
	limiterB := NewRedisLimiter(clientB, cfg, nil, nil)
	policy := Policy{Name: "api", Limit: 5, Window: 2 * time.Second}

	const workers = 20
	var wg sync.WaitGroup
	allowed := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			limiter := limiterA
			if i%2 == 0 {
				limiter = limiterB
			}
			d, err := limiter.Allow(ctx, IdentityKey(IdentityIP, "203.0.113.42"), policy)
			require.NoError(t, err)
			allowed <- d.Allowed
		}(i)
	}
	wg.Wait()
	close(allowed)
	count := 0
	for ok := range allowed {
		if ok {
			count++
		}
	}
	require.Equal(t, 5, count)
}

func TestRedisLimiter_FailClosed(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond, ReadTimeout: 50 * time.Millisecond})
	cfg := Config{Enabled: true, RedisPrefix: "fail", FailureMode: FailureModeClosed}
	limiter := NewRedisLimiter(client, cfg, nil, nil)
	policy := Policy{Name: PolicyAuth, Limit: 10, Window: time.Minute, FailClosed: true}
	d, err := limiter.Allow(context.Background(), IdentityKey(IdentityIP, "1.2.3.4"), policy)
	require.Error(t, err)
	require.False(t, d.Allowed)
}

func TestRedisLimiter_FailOpen(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 50 * time.Millisecond, ReadTimeout: 50 * time.Millisecond})
	cfg := Config{Enabled: true, RedisPrefix: "fail", FailureMode: FailureModeOpen}
	limiter := NewRedisLimiter(client, cfg, nil, nil)
	policy := Policy{Name: PolicyAPI, Limit: 10, Window: time.Minute, FailClosed: false}
	d, err := limiter.Allow(context.Background(), IdentityKey(IdentityIP, "1.2.3.4"), policy)
	require.NoError(t, err)
	require.True(t, d.Allowed)
}
