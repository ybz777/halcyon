package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository implements platformjobs.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, input platformjobs.CreateInput) (platformjobs.Job, bool, error) {
	id, err := platformjobs.NewID()
	if err != nil {
		return platformjobs.Job{}, false, err
	}
	now := time.Now().UTC()
	orgID := nullableString(input.OrganizationID)
	idemKey := nullableString(input.IdempotencyKey)
	payload := input.Payload
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	if !json.Valid(payload) {
		return platformjobs.Job{}, false, platformjobs.Permanent(fmt.Errorf("invalid json payload"))
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO jobs (
			id, organization_id, job_type, status, payload, priority,
			attempt_count, max_attempts, idempotency_key, available_at, created_at
		) VALUES ($1,$2,$3,'PENDING',$4,$5,0,$6,$7,$8,$9)
		ON CONFLICT (organization_id, job_type, idempotency_key)
			WHERE idempotency_key IS NOT NULL AND idempotency_key <> ''
		DO NOTHING
		RETURNING id
	`, id.String(), orgID, string(input.Type), payload, string(input.Priority), input.MaxAttempts, idemKey, input.AvailableAt.UTC(), now)

	var insertedID string
	if err := row.Scan(&insertedID); err != nil {
		if err == pgx.ErrNoRows {
			existing, err := r.findByIdempotency(ctx, input.OrganizationID, input.Type, input.IdempotencyKey)
			return existing, true, err
		}
		return platformjobs.Job{}, false, fmt.Errorf("insert job: %w", err)
	}
	job, err := r.Get(ctx, id)
	return job, false, err
}

func (r *Repository) Get(ctx context.Context, id platformjobs.ID) (platformjobs.Job, error) {
	return r.scanOne(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id=$1`, id.String())
}

func (r *Repository) GetForOrg(ctx context.Context, orgID string, id platformjobs.ID) (platformjobs.Job, error) {
	return r.scanOne(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id=$1 AND organization_id=$2`, id.String(), orgID)
}

func (r *Repository) List(ctx context.Context, orgID string, limit int, cursor string) ([]platformjobs.Job, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	args := []any{orgID}
	where := "organization_id = $1"
	argPos := 2
	if cursor != "" {
		createdAt, id, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		where += fmt.Sprintf(" AND (created_at < $%d OR (created_at = $%d AND id < $%d))", argPos, argPos, argPos+1)
		args = append(args, createdAt, id)
		argPos += 2
	}
	args = append(args, limit+1)
	query := fmt.Sprintf(`SELECT %s FROM jobs WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d`, jobColumns, where, argPos)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	jobs, err := scanJobs(rows)
	if err != nil {
		return nil, "", err
	}
	var next string
	if len(jobs) > limit {
		last := jobs[limit-1]
		next = encodeCursor(last.CreatedAt, last.ID.String())
		jobs = jobs[:limit]
	}
	return jobs, next, rows.Err()
}

func (r *Repository) ListPlatform(ctx context.Context, limit int, cursor string) ([]platformjobs.Job, string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	args := []any{}
	where := "1=1"
	argPos := 1
	if cursor != "" {
		createdAt, id, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		where += fmt.Sprintf(" AND (created_at < $%d OR (created_at = $%d AND id < $%d))", argPos, argPos, argPos+1)
		args = append(args, createdAt, id)
		argPos += 2
	}
	args = append(args, limit+1)
	query := fmt.Sprintf(`SELECT %s FROM jobs WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d`, jobColumns, where, argPos)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	jobs, err := scanJobs(rows)
	if err != nil {
		return nil, "", err
	}
	var next string
	if len(jobs) > limit {
		last := jobs[limit-1]
		next = encodeCursor(last.CreatedAt, last.ID.String())
		jobs = jobs[:limit]
	}
	return jobs, next, rows.Err()
}

func (r *Repository) Claim(ctx context.Context, workerID string, types []platformjobs.Type, limit int, lease time.Duration) ([]platformjobs.Job, error) {
	if limit <= 0 {
		limit = 1
	}
	now := time.Now().UTC()
	leaseUntil := now.Add(lease)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	typeFilter := ""
	args := []any{now, leaseUntil, workerID, now, limit}
	if len(types) > 0 {
		typeNames := make([]string, len(types))
		for i, t := range types {
			typeNames[i] = string(t)
		}
		typeFilter = " AND job_type = ANY($6)"
		args = append(args, typeNames)
	}

	query := fmt.Sprintf(`
		UPDATE jobs
		SET status = 'RUNNING',
			locked_at = $2,
			locked_by = $3,
			attempt_count = attempt_count + 1,
			started_at = COALESCE(started_at, $2),
			last_error = '',
			last_error_category = ''
		WHERE id IN (
			SELECT id FROM jobs
			WHERE status IN ('PENDING', 'RETRY_WAIT')
				AND available_at <= $4
				AND (locked_at IS NULL OR locked_at < $1)
				%s
			ORDER BY CASE priority WHEN 'HIGH' THEN 3 WHEN 'NORMAL' THEN 2 ELSE 1 END DESC,
				available_at ASC, created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT $5
		)
		RETURNING %s
	`, typeFilter, jobColumns)

	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("claim jobs: %w", err)
	}
	defer rows.Close()

	jobs, err := scanJobs(rows)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (r *Repository) Complete(ctx context.Context, workerID string, id platformjobs.ID, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE jobs
		SET status='SUCCEEDED', completed_at=$4, locked_at=NULL, locked_by='', last_error='', last_error_category=''
		WHERE id=$1 AND status='RUNNING' AND locked_by=$2 AND locked_at >= $3
	`, id.String(), workerID, at.UTC(), at.UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return platformjobs.ErrStaleLease
	}
	return nil
}

func (r *Repository) Retry(ctx context.Context, workerID string, id platformjobs.ID, next time.Time, lastError string, category platformjobs.ErrorCategory) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE jobs
		SET status='RETRY_WAIT', available_at=$3, locked_at=NULL, locked_by='',
			last_error=$4, last_error_category=$5
		WHERE id=$1 AND status='RUNNING' AND locked_by=$2 AND locked_at IS NOT NULL
	`, id.String(), workerID, next.UTC(), truncate(lastError, 500), string(category))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return platformjobs.ErrStaleLease
	}
	return nil
}

func (r *Repository) Fail(ctx context.Context, workerID string, id platformjobs.ID, lastError string, category platformjobs.ErrorCategory, at time.Time, deadLetter bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	targetStatus := "FAILED"
	if deadLetter {
		targetStatus = "DEAD_LETTERED"
	}

	var orgID, jobType *string
	var attemptCount int
	err = tx.QueryRow(ctx, `
		UPDATE jobs
		SET status=$3, failed_at=$4, completed_at=$4, locked_at=NULL, locked_by='',
			last_error=$5, last_error_category=$6
		WHERE id=$1 AND status='RUNNING' AND locked_by=$2 AND locked_at IS NOT NULL
		RETURNING organization_id, job_type, attempt_count
	`, id.String(), workerID, targetStatus, at.UTC(), truncate(lastError, 500), string(category)).Scan(&orgID, &jobType, &attemptCount)
	if err == pgx.ErrNoRows {
		return platformjobs.ErrStaleLease
	}
	if err != nil {
		return err
	}

	if deadLetter {
		dlqID, err := platformjobs.NewID()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO dead_letter_jobs (id, job_id, organization_id, job_type, failure_category, last_error, attempt_count)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, dlqID.String(), id.String(), orgID, deref(jobType), string(category), truncate(lastError, 500), attemptCount)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *Repository) Cancel(ctx context.Context, id platformjobs.ID, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE jobs
		SET status='CANCELLED', completed_at=$2, locked_at=NULL, locked_by=''
		WHERE id=$1 AND status IN ('PENDING','RETRY_WAIT','RUNNING')
	`, id.String(), at.UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return platformjobs.ErrJobNotFound
	}
	return nil
}

func (r *Repository) RecoverLeases(ctx context.Context, lease time.Duration, cfg platformjobs.RetryConfig) (map[platformjobs.Type]int64, error) {
	cutoff := time.Now().UTC().Add(-lease)
	now := time.Now().UTC()
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		WITH expired AS (
			SELECT id, job_type, attempt_count, max_attempts, organization_id, last_error, last_error_category
			FROM jobs
			WHERE status='RUNNING' AND locked_at IS NOT NULL AND locked_at < $1
			FOR UPDATE SKIP LOCKED
		),
		updated AS (
			UPDATE jobs j
			SET status = CASE WHEN e.attempt_count >= e.max_attempts THEN 'DEAD_LETTERED' ELSE 'RETRY_WAIT' END,
				available_at = CASE WHEN e.attempt_count >= e.max_attempts THEN j.available_at ELSE $2 END,
				failed_at = CASE WHEN e.attempt_count >= e.max_attempts THEN $2 ELSE j.failed_at END,
				completed_at = CASE WHEN e.attempt_count >= e.max_attempts THEN $2 ELSE j.completed_at END,
				last_error = CASE WHEN e.attempt_count >= e.max_attempts THEN COALESCE(NULLIF(j.last_error,''), 'lease expired') ELSE j.last_error END,
				last_error_category = CASE WHEN e.attempt_count >= e.max_attempts THEN 'TIMEOUT' ELSE j.last_error_category END,
				locked_at = NULL,
				locked_by = ''
			FROM expired e
			WHERE j.id = e.id
			RETURNING j.id, j.job_type, j.organization_id, j.attempt_count, j.last_error, j.last_error_category,
				CASE WHEN e.attempt_count >= e.max_attempts THEN true ELSE false END AS dead_letter
		)
		SELECT job_type, dead_letter, id, organization_id, attempt_count, last_error, last_error_category
		FROM updated
	`, cutoff, platformjobs.NextAvailableAt(cfg, 1, now))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[platformjobs.Type]int64)
	for rows.Next() {
		var jobType string
		var deadLetter bool
		var jobID string
		var orgID *string
		var attemptCount int
		var lastError, category string
		if err := rows.Scan(&jobType, &deadLetter, &jobID, &orgID, &attemptCount, &lastError, &category); err != nil {
			return nil, err
		}
		counts[platformjobs.Type(jobType)]++
		if deadLetter {
			dlqID, err := platformjobs.NewID()
			if err != nil {
				return nil, err
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO dead_letter_jobs (id, job_id, organization_id, job_type, failure_category, last_error, attempt_count)
				VALUES ($1,$2,$3,$4,$5,$6,$7)
				ON CONFLICT DO NOTHING
			`, dlqID.String(), jobID, orgID, jobType, category, truncate(lastError, 500), attemptCount)
			if err != nil {
				return nil, err
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return counts, nil
}

func (r *Repository) RenewLease(ctx context.Context, workerID string, id platformjobs.ID, lease time.Duration) error {
	leaseUntil := time.Now().UTC().Add(lease)
	tag, err := r.pool.Exec(ctx, `
		UPDATE jobs SET locked_at=$3
		WHERE id=$1 AND status='RUNNING' AND locked_by=$2 AND locked_at IS NOT NULL
	`, id.String(), workerID, leaseUntil)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return platformjobs.ErrStaleLease
	}
	return nil
}

func (r *Repository) CountReady(ctx context.Context, jobType platformjobs.Type) (int64, error) {
	var count int64
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM jobs
		WHERE status IN ('PENDING','RETRY_WAIT') AND available_at <= now()
			AND ($1 = '' OR job_type = $1)
	`, string(jobType)).Scan(&count)
	return count, err
}

func (r *Repository) findByIdempotency(ctx context.Context, orgID string, jobType platformjobs.Type, key string) (platformjobs.Job, error) {
	return r.scanOne(ctx, `
		SELECT `+jobColumns+` FROM jobs
		WHERE organization_id IS NOT DISTINCT FROM $1
			AND job_type=$2 AND idempotency_key=$3
	`, nullableString(orgID), string(jobType), key)
}

func (r *Repository) scanOne(ctx context.Context, query string, args ...any) (platformjobs.Job, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	job, err := scanJob(row.Scan)
	if err == pgx.ErrNoRows {
		return platformjobs.Job{}, platformjobs.ErrJobNotFound
	}
	return job, err
}

const jobColumns = `
	id, organization_id, job_type, status, payload, priority, attempt_count, max_attempts,
	idempotency_key, available_at, locked_at, locked_by, last_error, created_at, started_at, completed_at, failed_at`

func scanJobs(rows pgx.Rows) ([]platformjobs.Job, error) {
	var jobs []platformjobs.Job
	for rows.Next() {
		job, err := scanJob(rows.Scan)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func scanJob(scan func(dest ...any) error) (platformjobs.Job, error) {
	var (
		id, orgID, jobType, status, priority, idemKey, lockedBy, lastError string
		payload                                                            []byte
		attemptCount, maxAttempts                                          int
		availableAt, createdAt                                             time.Time
		lockedAt, startedAt, completedAt, failedAt                         *time.Time
	)
	if err := scan(&id, &orgID, &jobType, &status, &payload, &priority, &attemptCount, &maxAttempts,
		&idemKey, &availableAt, &lockedAt, &lockedBy, &lastError, &createdAt, &startedAt, &completedAt, &failedAt); err != nil {
		return platformjobs.Job{}, err
	}
	parsedID, _ := platformjobs.ParseID(id)
	return platformjobs.Job{
		ID: parsedID, OrganizationID: orgID, Type: platformjobs.Type(jobType),
		Status: platformjobs.Status(status), Payload: payload, Priority: platformjobs.Priority(priority),
		AttemptCount: attemptCount, MaxAttempts: maxAttempts, IdempotencyKey: idemKey,
		AvailableAt: availableAt.UTC(), LockedAt: lockedAt, LockedBy: lockedBy, LastError: lastError,
		CreatedAt: createdAt.UTC(), StartedAt: startedAt, CompletedAt: completedAt, FailedAt: failedAt,
	}, nil
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func truncate(v string, max int) string {
	if len(v) <= max {
		return v
	}
	return v[:max]
}

func deref(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

var _ platformjobs.Repository = (*Repository)(nil)
