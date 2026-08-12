package postgres

import (
	"context"
	"fmt"
	"strconv"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AttemptRepository implements platformjobs.AttemptRepository.
type AttemptRepository struct {
	pool *pgxpool.Pool
}

func NewAttemptRepository(pool *pgxpool.Pool) *AttemptRepository {
	return &AttemptRepository{pool: pool}
}

func (r *AttemptRepository) Create(ctx context.Context, attempt platformjobs.Attempt) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO job_attempts (
			id, job_id, attempt_number, worker_id, status, error_category, error_message, started_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`, attempt.ID.String(), attempt.JobID.String(), attempt.AttemptNumber, attempt.WorkerID,
		string(attempt.Status), string(attempt.ErrorCategory), attempt.ErrorMessage, attempt.StartedAt.UTC())
	return err
}

func (r *AttemptRepository) Complete(ctx context.Context, attempt platformjobs.Attempt) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE job_attempts
		SET status=$2, error_category=$3, error_message=$4, completed_at=$5, duration_ms=$6
		WHERE id=$1
	`, attempt.ID.String(), string(attempt.Status), string(attempt.ErrorCategory),
		attempt.ErrorMessage, attempt.CompletedAt, attempt.DurationMs)
	return err
}

func (r *AttemptRepository) ListByJob(ctx context.Context, jobID platformjobs.ID, limit int) ([]platformjobs.Attempt, error) {
	attempts, _, err := r.ListByJobPaginated(ctx, jobID, limit, "")
	return attempts, err
}

func (r *AttemptRepository) ListByJobPaginated(ctx context.Context, jobID platformjobs.ID, limit int, cursor string) ([]platformjobs.Attempt, string, error) {
	limit = platformpagination.NormalizePageSize(limit, 50, 100)
	args := []any{jobID.String()}
	where := "job_id=$1"
	argPos := 2
	if cursor != "" {
		keyset, err := platformpagination.DecodeKeysetCursor(cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		attemptNo, err := strconv.Atoi(keyset.Primary)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor")
		}
		where += fmt.Sprintf(" AND (attempt_number < $%d OR (attempt_number = $%d AND id < $%d))", argPos, argPos, argPos+1)
		args = append(args, attemptNo, keyset.ID)
		argPos += 2
	}
	args = append(args, limit+1)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, job_id, attempt_number, worker_id, status, error_category, error_message,
			started_at, completed_at, duration_ms
		FROM job_attempts
		WHERE %s
		ORDER BY attempt_number DESC, id DESC
		LIMIT $%d
	`, where, argPos), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	attempts := make([]platformjobs.Attempt, 0, limit)
	for rows.Next() {
		attempt, err := scanAttempt(rows)
		if err != nil {
			return nil, "", err
		}
		attempts = append(attempts, attempt)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(attempts) > limit {
		last := attempts[limit-1]
		next = platformpagination.EncodeKeysetCursor("attempt_number", "desc", strconv.Itoa(last.AttemptNumber), last.ID.String())
		attempts = attempts[:limit]
	}
	if attempts == nil {
		attempts = []platformjobs.Attempt{}
	}
	return attempts, next, nil
}

func scanAttempt(rows pgx.Rows) (platformjobs.Attempt, error) {
	var (
		id, jobIDRaw, workerID, status, category, message string
		attemptNumber, durationMs                         int
		startedAt                                         time.Time
		completedAt                                       *time.Time
	)
	if err := rows.Scan(&id, &jobIDRaw, &attemptNumber, &workerID, &status, &category, &message,
		&startedAt, &completedAt, &durationMs); err != nil {
		return platformjobs.Attempt{}, err
	}
	parsedID, _ := platformjobs.ParseAttemptID(id)
	parsedJobID, _ := platformjobs.ParseID(jobIDRaw)
	return platformjobs.Attempt{
		ID: parsedID, JobID: parsedJobID, AttemptNumber: attemptNumber, WorkerID: workerID,
		Status: platformjobs.AttemptStatus(status), ErrorCategory: platformjobs.ErrorCategory(category),
		ErrorMessage: message, StartedAt: startedAt.UTC(), CompletedAt: completedAt, DurationMs: durationMs,
	}, nil
}

// DeleteOlderThan removes attempt history past retention.
func (r *AttemptRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM job_attempts WHERE started_at < $1`, cutoff.UTC())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteCompletedJobs removes succeeded jobs older than cutoff.
func (r *Repository) DeleteCompletedOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM jobs WHERE status='SUCCEEDED' AND completed_at IS NOT NULL AND completed_at < $1
	`, cutoff.UTC())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DeleteFailedOlderThan removes failed/dead-lettered jobs older than cutoff.
func (r *Repository) DeleteFailedOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM jobs WHERE status IN ('FAILED','DEAD_LETTERED') AND failed_at IS NOT NULL AND failed_at < $1
	`, cutoff.UTC())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// Ping verifies database connectivity.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("database pool is nil")
	}
	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	return nil
}

var _ platformjobs.AttemptRepository = (*AttemptRepository)(nil)

// rowScanner is satisfied by pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// Ensure pgx.ErrNoRows is referenced for scan helpers.
var _ = pgx.ErrNoRows
