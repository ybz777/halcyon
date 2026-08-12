package messaging

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	processedStatusProcessing = "processing"
	processedStatusCompleted  = "completed"
)

// IdempotentProcessor coordinates inbox deduplication with business processing.
type IdempotentProcessor struct {
	pool *pgxpool.Pool
}

// NewIdempotentProcessor creates an idempotent processor backed by PostgreSQL.
func NewIdempotentProcessor(pool *pgxpool.Pool) *IdempotentProcessor {
	return &IdempotentProcessor{pool: pool}
}

// ProcessResult describes idempotency check outcome.
type ProcessResult struct {
	Duplicate bool
}

// ProcessOnce runs fn when the event has not been processed by the consumer group.
func (p *IdempotentProcessor) ProcessOnce(
	ctx context.Context,
	eventID, consumerGroup, organizationID string,
	fn func(ctx context.Context) error,
) (ProcessResult, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return ProcessResult{}, fmt.Errorf("begin idempotency tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO processed_events (event_id, consumer_group, organization_id, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id, consumer_group) DO NOTHING
		RETURNING true
	`, eventID, consumerGroup, organizationID, processedStatusProcessing).Scan(&inserted)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ProcessResult{Duplicate: true}, nil
		}
		return ProcessResult{}, fmt.Errorf("claim processed event: %w", err)
	}

	if err := fn(ctx); err != nil {
		return ProcessResult{}, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE processed_events
		SET status = $3, processed_at = now()
		WHERE event_id = $1 AND consumer_group = $2
	`, eventID, consumerGroup, processedStatusCompleted)
	if err != nil {
		return ProcessResult{}, fmt.Errorf("mark processed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ProcessResult{}, fmt.Errorf("commit idempotency tx: %w", err)
	}
	return ProcessResult{}, nil
}
