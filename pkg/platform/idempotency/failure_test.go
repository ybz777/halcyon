package idempotency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/idempotency"
	"github.com/stretchr/testify/require"
)

type failingStore struct {
	idempotency.Store
	failClaim bool
}

func (f *failingStore) Claim(ctx context.Context, key, fingerprint string, ttl time.Duration) (idempotency.ClaimOutcome, error) {
	if f.failClaim {
		return idempotency.ClaimOutcome{}, errors.New("redis unavailable")
	}
	return f.Store.Claim(ctx, key, fingerprint, ttl)
}

func TestFailure_RedisUnavailableFailsClaim(t *testing.T) {
	t.Parallel()
	base := idempotency.NewMemoryStore()
	store := &failingStore{Store: base, failClaim: true}
	_, err := store.Claim(context.Background(), "k1", "fp", time.Minute)
	require.Error(t, err)
}

func TestFailure_ReleaseAllowsRetryAfterBusinessFailure(t *testing.T) {
	t.Parallel()
	store := idempotency.NewMemoryStore()
	ctx := context.Background()
	key := "k-release"
	fp := "fp1"
	_, err := store.Claim(ctx, key, fp, time.Minute)
	require.NoError(t, err)
	require.NoError(t, store.Release(ctx, key, fp))

	out, err := store.Claim(ctx, key, fp, time.Minute)
	require.NoError(t, err)
	require.Equal(t, idempotency.ClaimAcquired, out.Result)
}
