package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a PostgreSQL-backed outbox store.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Insert persists a new outbox entry.
func (s *PostgresStore) Insert(ctx context.Context, entry Entry) error {
	id := entry.ID
	if id == "" {
		id = uuid.NewString()
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO outbox (
			id, event_id, event_type, event_version, aggregate_id, aggregate_type,
			organization_id, correlation_id, request_id, actor_id, subject_id, causation_id,
			occurred_at, producer, payload, trace_id, span_id, topic, partition_key, status, next_attempt_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`,
		id, entry.EventID, entry.EventType, entry.EventVersion, entry.AggregateID, entry.AggregateType,
		entry.OrganizationID, entry.CorrelationID, entry.RequestID, entry.ActorID, entry.SubjectID, entry.CausationID,
		entry.OccurredAt, entry.Producer, entry.Payload, entry.TraceID, entry.SpanID, entry.Topic, entry.PartitionKey, string(StatusPending),
		entry.NextAttemptAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox entry: %w", err)
	}
	return nil
}

// RequeueStalePublishing resets entries stuck in publishing state.
func (s *PostgresStore) RequeueStalePublishing(ctx context.Context, olderThan time.Duration) error {
	if olderThan <= 0 {
		olderThan = 5 * time.Minute
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox
		SET status = $1
		WHERE status = $2 AND created_at < NOW() - ($3 * INTERVAL '1 second')
	`, string(StatusPending), string(StatusPublishing), int64(olderThan.Seconds()))
	if err != nil {
		return fmt.Errorf("requeue stale publishing: %w", err)
	}
	return nil
}

// ClaimPending atomically claims pending entries for relay workers.
func (s *PostgresStore) ClaimPending(ctx context.Context, limit int) ([]Entry, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.pool.Query(ctx, `
		UPDATE outbox
		SET status = $1
		WHERE id IN (
			SELECT id FROM outbox
			WHERE status = $2
				AND (next_attempt_at IS NULL OR next_attempt_at <= NOW())
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $3
		)
		RETURNING id, event_id, event_type, event_version, aggregate_id, aggregate_type,
			organization_id, correlation_id, request_id, actor_id, subject_id, causation_id,
			occurred_at, producer, payload, trace_id, span_id, topic, partition_key, status, attempts,
			COALESCE(last_error, ''), next_attempt_at, created_at, published_at
	`, string(StatusPublishing), string(StatusPending), limit)
	if err != nil {
		return nil, fmt.Errorf("claim pending outbox: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

// MarkPublished marks an entry as successfully published.
func (s *PostgresStore) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE outbox
		SET status = $1, published_at = $2, last_error = NULL, next_attempt_at = NULL
		WHERE id = $3 AND status IN ($4, $5)
	`, string(StatusPublished), publishedAt, id, string(StatusPending), string(StatusPublishing))
	if err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("outbox entry %q not found or not pending", id)
	}
	return nil
}

// MarkFailed records a failed publish attempt and schedules retry.
func (s *PostgresStore) MarkFailed(ctx context.Context, id string, attempts int, lastError string, nextAttemptAt *time.Time) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox
		SET attempts = $1, last_error = $2, next_attempt_at = $3, status = $4
		WHERE id = $5 AND status IN ($6, $7)
	`, attempts, lastError, nextAttemptAt, string(StatusPending), id, string(StatusPending), string(StatusPublishing))
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return nil
}

// MarkDeadLetter marks an entry as permanently failed.
func (s *PostgresStore) MarkDeadLetter(ctx context.Context, id string, lastError string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox
		SET status = $1, last_error = $2, next_attempt_at = NULL
		WHERE id = $3
	`, string(StatusFailed), lastError, id)
	if err != nil {
		return fmt.Errorf("mark outbox dead letter: %w", err)
	}
	return nil
}

// PendingCount returns the number of pending outbox entries.
func (s *PostgresStore) PendingCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox WHERE status = $1`, string(StatusPending)).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count pending outbox: %w", err)
	}
	return count, nil
}

// TxStore wraps a pgx.Tx for transactional outbox inserts.
type TxStore struct {
	tx pgx.Tx
}

// NewTxStore creates a store bound to an active transaction.
func NewTxStore(tx pgx.Tx) *TxStore {
	return &TxStore{tx: tx}
}

// Insert persists a new outbox entry within the transaction.
func (s *TxStore) Insert(ctx context.Context, entry Entry) error {
	id := entry.ID
	if id == "" {
		id = uuid.NewString()
	}

	_, err := s.tx.Exec(ctx, `
		INSERT INTO outbox (
			id, event_id, event_type, event_version, aggregate_id, aggregate_type,
			organization_id, correlation_id, request_id, actor_id, subject_id, causation_id,
			occurred_at, producer, payload, trace_id, span_id, topic, partition_key, status, next_attempt_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19, $20, $21
		)
	`,
		id, entry.EventID, entry.EventType, entry.EventVersion, entry.AggregateID, entry.AggregateType,
		entry.OrganizationID, entry.CorrelationID, entry.RequestID, entry.ActorID, entry.SubjectID, entry.CausationID,
		entry.OccurredAt, entry.Producer, entry.Payload, entry.TraceID, entry.SpanID, entry.Topic, entry.PartitionKey, string(StatusPending),
		entry.NextAttemptAt,
	)
	if err != nil {
		return fmt.Errorf("insert outbox entry in tx: %w", err)
	}
	return nil
}

func (s *TxStore) RequeueStalePublishing(context.Context, time.Duration) error {
	return fmt.Errorf("RequeueStalePublishing not supported on transaction store")
}

func (s *TxStore) ClaimPending(context.Context, int) ([]Entry, error) {
	return nil, fmt.Errorf("ClaimPending not supported on transaction store")
}

func (s *TxStore) MarkPublished(context.Context, string, time.Time) error {
	return fmt.Errorf("MarkPublished not supported on transaction store")
}

func (s *TxStore) MarkFailed(context.Context, string, int, string, *time.Time) error {
	return fmt.Errorf("MarkFailed not supported on transaction store")
}

func (s *TxStore) MarkDeadLetter(context.Context, string, string) error {
	return fmt.Errorf("MarkDeadLetter not supported on transaction store")
}

func (s *TxStore) PendingCount(context.Context) (int64, error) {
	return 0, fmt.Errorf("PendingCount not supported on transaction store")
}

type rowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanEntries(rows rowScanner) ([]Entry, error) {
	var entries []Entry
	for rows.Next() {
		e, err := scanEntry(rows.Scan)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outbox rows: %w", err)
	}
	return entries, nil
}

func scanEntry(scan func(dest ...any) error) (Entry, error) {
	var e Entry
	var status string
	var publishedAt *time.Time
	if err := scan(
		&e.ID, &e.EventID, &e.EventType, &e.EventVersion, &e.AggregateID, &e.AggregateType,
		&e.OrganizationID, &e.CorrelationID, &e.RequestID, &e.ActorID, &e.SubjectID, &e.CausationID,
		&e.OccurredAt, &e.Producer, &e.Payload, &e.TraceID, &e.SpanID, &e.Topic, &e.PartitionKey, &status, &e.Attempts,
		&e.LastError, &e.NextAttemptAt, &e.CreatedAt, &publishedAt,
	); err != nil {
		return Entry{}, fmt.Errorf("scan outbox row: %w", err)
	}
	e.Status = Status(status)
	e.PublishedAt = publishedAt
	return e, nil
}
