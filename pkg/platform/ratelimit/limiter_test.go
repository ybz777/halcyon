package ratelimit

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMemoryLimiter_AllowAndExpire(t *testing.T) {
	limiter := NewMemoryLimiter()
	policy := Policy{Name: "test", Limit: 2, Window: 50 * time.Millisecond}

	d1, err := limiter.Allow(context.Background(), "ip:1.1.1.1", policy)
	require.NoError(t, err)
	require.True(t, d1.Allowed)
	require.Equal(t, int64(1), d1.Remaining)

	d2, err := limiter.Allow(context.Background(), "ip:1.1.1.1", policy)
	require.NoError(t, err)
	require.True(t, d2.Allowed)

	d3, err := limiter.Allow(context.Background(), "ip:1.1.1.1", policy)
	require.NoError(t, err)
	require.False(t, d3.Allowed)
	require.GreaterOrEqual(t, d3.RetryAfter, time.Duration(0))

	time.Sleep(60 * time.Millisecond)
	d4, err := limiter.Allow(context.Background(), "ip:1.1.1.1", policy)
	require.NoError(t, err)
	require.True(t, d4.Allowed)
}

func TestMemoryLimiter_Concurrent(t *testing.T) {
	limiter := NewMemoryLimiter()
	policy := Policy{Name: "test", Limit: 10, Window: time.Second}
	const workers = 32
	var wg sync.WaitGroup
	allowed := make(chan bool, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, err := limiter.Allow(context.Background(), "ip:concurrent", policy)
			require.NoError(t, err)
			allowed <- d.Allowed
		}()
	}
	wg.Wait()
	close(allowed)
	count := 0
	for ok := range allowed {
		if ok {
			count++
		}
	}
	require.Equal(t, 10, count)
}

func TestDisabledLimiter(t *testing.T) {
	var limiter DisabledLimiter
	d, err := limiter.Allow(context.Background(), "ip:1.1.1.1", Policy{Limit: 1, Window: time.Second})
	require.NoError(t, err)
	require.True(t, d.Allowed)
}

func TestLoadConfigValidation(t *testing.T) {
	t.Setenv("RATE_LIMIT_ENABLED", "true")
	t.Setenv("REDIS_URL", "")
	_, err := LoadConfig("gateway")
	require.Error(t, err)
}

func TestNormalizeIP(t *testing.T) {
	require.Equal(t, "127.0.0.1", NormalizeIP("::ffff:127.0.0.1"))
	require.Equal(t, "203.0.113.1", NormalizeIP("203.0.113.1"))
}
