package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ProcessedEventStore tracks consumed events for idempotency.
type ProcessedEventStore interface {
	MarkProcessed(ctx context.Context, eventID, consumerGroup string) error
	IsProcessed(ctx context.Context, eventID, consumerGroup string) (bool, error)
}

// PostgresProcessedStore implements ProcessedEventStore.
type PostgresProcessedStore struct {
	pool *pgxpool.Pool
}

// NewPostgresProcessedStore creates a PostgreSQL processed-events store.
func NewPostgresProcessedStore(pool *pgxpool.Pool) *PostgresProcessedStore {
	return &PostgresProcessedStore{pool: pool}
}

// MarkProcessed records that an event was handled by a consumer group.
func (s *PostgresProcessedStore) MarkProcessed(ctx context.Context, eventID, consumerGroup string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO processed_events (event_id, consumer_group, status)
		VALUES ($1, $2, 'completed')
		ON CONFLICT (event_id, consumer_group) DO NOTHING
	`, eventID, consumerGroup)
	if err != nil {
		return fmt.Errorf("mark processed event: %w", err)
	}
	return nil
}

// IsProcessed reports whether an event was already handled.
func (s *PostgresProcessedStore) IsProcessed(ctx context.Context, eventID, consumerGroup string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM processed_events
			WHERE event_id = $1 AND consumer_group = $2 AND status = 'completed'
		)
	`, eventID, consumerGroup).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check processed event: %w", err)
	}
	return exists, nil
}

// RetryConfig configures consumer retry behavior.
type RetryConfig struct {
	MaxAttempts int
	Initial     time.Duration
	MaxBackoff  time.Duration
	Multiplier  float64
}

// DefaultRetryConfig returns exponential backoff defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 5,
		Initial:     100 * time.Millisecond,
		MaxBackoff:  5 * time.Second,
		Multiplier:  2,
	}
}

// Backoff calculates exponential backoff for an attempt (1-based).
func Backoff(cfg RetryConfig, attempt int) time.Duration {
	if attempt <= 1 {
		return cfg.Initial
	}
	delay := cfg.Initial
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxBackoff {
			return cfg.MaxBackoff
		}
	}
	return delay
}
