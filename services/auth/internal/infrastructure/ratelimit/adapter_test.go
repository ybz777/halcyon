package ratelimit_test

import (
	"context"
	"testing"
	"time"

	platformratelimit "github.com/halcyon-io/halcyon/pkg/platform/ratelimit"
	authratelimit "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/ratelimit"
	"github.com/stretchr/testify/require"
)

func TestLoginAdapter_AllowsUntilLimit(t *testing.T) {
	limiter := platformratelimit.NewMemoryLimiter()
	policy := platformratelimit.Policy{Name: platformratelimit.PolicyAuth, Limit: 1, Window: time.Minute}
	adapter := authratelimit.NewLoginAdapter(limiter, policy)

	allowed, err := adapter.Allow(context.Background(), "user@example.com")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = adapter.Allow(context.Background(), "user@example.com")
	require.NoError(t, err)
	require.False(t, allowed)
}
