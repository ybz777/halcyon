//go:build integration

package idempotency_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRedisStoreClaimAndReplay(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		url = "redis://localhost:6379/0"
	}
	opt, err := redis.ParseURL(url)
	require.NoError(t, err)
	client := redis.NewClient(opt)
	ctx := context.Background()
	require.NoError(t, client.Ping(ctx).Err())
	defer client.Close()

	store := idempotency.NewRedisStore(client)
	key := "test:idempotency:" + time.Now().Format("150405.000000000")
	fp := "fingerprint-1"

	out, err := store.Claim(ctx, key, fp, time.Minute)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimAcquired, out.Result)

	require.NoError(t, store.Complete(ctx, key, fp, idempotency.Record{
		Status: 201, Body: []byte(`{"ok":true}`), ContentType: "application/json",
	}, time.Minute))

	out, err = store.Claim(ctx, key, fp, time.Minute)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimReplay, out.Result)
	require.Equal(t, 201, out.Record.Status)

	_ = client.Del(ctx, key)
}
