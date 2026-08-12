//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentConfigured(t *testing.T) {
	if os.Getenv("APP_ENV") == "" {
		t.Skip("APP_ENV not set")
	}
	require.Equal(t, "test", os.Getenv("APP_ENV"))
}

func TestPostgresReachable(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, cleanup, err := setupPostgres(ctx, dsn)
	require.NoError(t, err)
	defer cleanup()
	require.NoError(t, pool.Ping(ctx))
}

func TestRedisReachable(t *testing.T) {
	url := os.Getenv("REDIS_URL")
	if url == "" {
		t.Skip("REDIS_URL not set")
	}
	require.NoError(t, pingRedis(url))
}

func TestKafkaBrokerConfigured(t *testing.T) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		t.Skip("KAFKA_BROKERS not set")
	}
	require.Contains(t, brokers, ":")
}
