package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// JobRepository implements ports.JobRepository.
type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (r *JobRepository) CreateJob(ctx context.Context, job *domain.Job) error {
	_, err := r.exec(ctx).Exec(ctx, `
		INSERT INTO workflow_jobs (
			id, organization_id, execution_id, step_execution_id, step_key,
			status, attempt, available_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, job.ID.String(), job.OrganizationID.String(), job.ExecutionID.String(),
		job.StepExecutionID.String(), job.StepKey, string(job.Status), job.Attempt,
		job.AvailableAt, job.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert job: %w", err)
	}
	return nil
}

func (r *JobRepository) ClaimJobs(ctx context.Context, workerID string, limit int, lease time.Duration) ([]domain.Job, error) {
	if limit <= 0 {
		limit = 1
	}
	now := time.Now().UTC()
	leaseUntil := now.Add(lease)

	rows, err := r.pool.Query(ctx, `
		UPDATE workflow_jobs
		SET status = 'RUNNING', locked_at = $1, locked_by = $2, attempt = attempt + 1
		WHERE id IN (
			SELECT id FROM workflow_jobs
			WHERE status IN ('READY', 'RETRY_WAIT')
				AND available_at <= $3
			ORDER BY available_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $4
		)
		RETURNING id, organization_id, execution_id, step_execution_id, step_key,
			status, attempt, available_at, locked_at, locked_by, created_at, completed_at
	`, now, workerID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("claim jobs: %w", err)
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		job, err := scanJob(rows.Scan)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	_ = leaseUntil // lease expiry checked in RecoverExpiredLeases
	return jobs, rows.Err()
}

func (r *JobRepository) UpdateJob(ctx context.Context, job *domain.Job) error {
	_, err := r.exec(ctx).Exec(ctx, `
		UPDATE workflow_jobs
		SET status = $1, attempt = $2, available_at = $3, locked_at = $4,
			locked_by = $5, completed_at = $6
		WHERE id = $7 AND organization_id = $8
	`, string(job.Status), job.Attempt, job.AvailableAt, job.LockedAt,
		job.LockedBy, job.CompletedAt, job.ID.String(), job.OrganizationID.String())
	return err
}

func (r *JobRepository) RecoverExpiredLeases(ctx context.Context, lease time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-lease)
	tag, err := r.pool.Exec(ctx, `
		UPDATE workflow_jobs
		SET status = 'READY', locked_at = NULL, locked_by = NULL
		WHERE status = 'RUNNING' AND locked_at IS NOT NULL AND locked_at < $1
	`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *JobRepository) CountReady(ctx context.Context) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM workflow_jobs WHERE status IN ('READY', 'RETRY_WAIT') AND available_at <= now()
	`).Scan(&count)
	return count, err
}

func (r *JobRepository) CreateAttempt(ctx context.Context, attempt *domain.ExecutionAttempt) error {
	outJSON, _ := json.Marshal(attempt.OutputReference)
	_, err := r.exec(ctx).Exec(ctx, `
		INSERT INTO execution_attempts (
			id, job_id, organization_id, attempt, status, started_at, completed_at, error_message, output_reference
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`, attempt.ID.String(), attempt.JobID.String(), attempt.OrganizationID.String(),
		attempt.Attempt, attempt.Status, attempt.StartedAt, attempt.CompletedAt,
		attempt.ErrorMessage, outJSON)
	return err
}

func (r *JobRepository) UpdateAttempt(ctx context.Context, attempt *domain.ExecutionAttempt) error {
	outJSON, _ := json.Marshal(attempt.OutputReference)
	_, err := r.exec(ctx).Exec(ctx, `
		UPDATE execution_attempts
		SET status = $1, completed_at = $2, error_message = $3, output_reference = $4
		WHERE id = $5
	`, attempt.Status, attempt.CompletedAt, attempt.ErrorMessage, outJSON, attempt.ID.String())
	return err
}

func (r *JobRepository) exec(ctx context.Context) dbExecutor {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func scanJob(scan func(dest ...any) error) (domain.Job, error) {
	var job domain.Job
	var id, orgID, executionID, stepExecID, stepKey, status string
	if err := scan(&id, &orgID, &executionID, &stepExecID, &stepKey, &status,
		&job.Attempt, &job.AvailableAt, &job.LockedAt, &job.LockedBy, &job.CreatedAt, &job.CompletedAt); err != nil {
		return domain.Job{}, err
	}
	job.ID, _ = domain.ParseJobID(id)
	job.OrganizationID, _ = domain.ParseOrganizationID(orgID)
	job.ExecutionID, _ = domain.ParseExecutionID(executionID)
	job.StepExecutionID, _ = domain.ParseStepExecutionID(stepExecID)
	job.StepKey = stepKey
	job.Status = domain.JobStatus(status)
	return job, nil
}

// IdempotencyRepository implements ports.IdempotencyRepository.
type IdempotencyRepository struct {
	pool *pgxpool.Pool
}

func NewIdempotencyRepository(pool *pgxpool.Pool) *IdempotencyRepository {
	return &IdempotencyRepository{pool: pool}
}

func (r *IdempotencyRepository) Find(ctx context.Context, orgID domain.OrganizationID, key string) (domain.ExecutionID, bool, error) {
	var executionID string
	err := r.pool.QueryRow(ctx, `
		SELECT execution_id FROM workflow_idempotency_keys
		WHERE organization_id = $1 AND idempotency_key = $2
	`, orgID.String(), key).Scan(&executionID)
	if err == pgx.ErrNoRows {
		return domain.ExecutionID{}, false, nil
	}
	if err != nil {
		return domain.ExecutionID{}, false, err
	}
	id, err := domain.ParseExecutionID(executionID)
	return id, true, err
}

func (r *IdempotencyRepository) Store(ctx context.Context, orgID domain.OrganizationID, key string, executionID domain.ExecutionID) error {
	tag, err := r.exec(ctx).Exec(ctx, `
		INSERT INTO workflow_idempotency_keys (organization_id, idempotency_key, execution_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, idempotency_key) DO NOTHING
	`, orgID.String(), key, executionID.String())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrDuplicateIdempotencyKey
	}
	return nil
}

func (r *IdempotencyRepository) exec(ctx context.Context) dbExecutor {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

var (
	_ ports.JobRepository         = (*JobRepository)(nil)
	_ ports.IdempotencyRepository = (*IdempotencyRepository)(nil)
)
