package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkflowRepository implements ports.WorkflowRepository.
type WorkflowRepository struct {
	pool *pgxpool.Pool
}

func NewWorkflowRepository(pool *pgxpool.Pool) *WorkflowRepository {
	return &WorkflowRepository{pool: pool}
}

func (r *WorkflowRepository) exec(ctx context.Context) dbExecutor {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *WorkflowRepository) Create(ctx context.Context, workflow *domain.Workflow, definition *domain.WorkflowDefinition) error {
	q := r.exec(ctx)
	_, err := q.Exec(ctx, `
		INSERT INTO workflows (id, organization_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, workflow.ID.String(), workflow.OrganizationID.String(), workflow.Name, workflow.CreatedAt, workflow.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert workflow: %w", err)
	}

	_, err = q.Exec(ctx, `
		INSERT INTO workflow_versions (id, workflow_id, organization_id, version, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, definition.ID.String(), workflow.ID.String(), workflow.OrganizationID.String(),
		definition.Version, string(definition.Status), definition.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert workflow version: %w", err)
	}

	for _, step := range definition.Steps {
		retryJSON, _ := json.Marshal(step.RetryPolicy)
		depsJSON, _ := json.Marshal(step.Dependencies)
		_, err = q.Exec(ctx, `
			INSERT INTO workflow_steps (
				id, workflow_version_id, organization_id, step_key, name, step_type,
				configuration, timeout_seconds, retry_policy, dependencies, step_order
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		`, step.ID, definition.ID.String(), workflow.OrganizationID.String(), step.StepKey, step.Name,
			string(step.Type), step.Configuration, int(step.Timeout.Seconds()), retryJSON, depsJSON, step.Order)
		if err != nil {
			return fmt.Errorf("insert workflow step: %w", err)
		}
	}
	return nil
}

func (r *WorkflowRepository) FindByID(ctx context.Context, orgID domain.OrganizationID, id domain.WorkflowID) (*domain.Workflow, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT w.id, w.organization_id, w.name, COALESCE(MAX(v.version), 0), w.created_at, w.updated_at
		FROM workflows w
		LEFT JOIN workflow_versions v ON v.workflow_id = w.id
		WHERE w.id = $1 AND w.organization_id = $2
		GROUP BY w.id
	`, id.String(), orgID.String())

	var wf domain.Workflow
	var wfID, org, name string
	if err := row.Scan(&wfID, &org, &name, &wf.LatestVersion, &wf.CreatedAt, &wf.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWorkflowNotFound
		}
		return nil, err
	}
	wf.ID, _ = domain.ParseWorkflowID(wfID)
	wf.OrganizationID, _ = domain.ParseOrganizationID(org)
	wf.Name = name
	return &wf, nil
}

func (r *WorkflowRepository) FindDefinitionByVersionID(ctx context.Context, orgID domain.OrganizationID, versionID domain.WorkflowVersionID) (*domain.WorkflowDefinition, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, workflow_id, organization_id, version, status, created_at
		FROM workflow_versions
		WHERE id = $1 AND organization_id = $2
	`, versionID.String(), orgID.String())
	return scanDefinition(ctx, r.pool, row)
}

func (r *WorkflowRepository) FindDefinition(ctx context.Context, orgID domain.OrganizationID, workflowID domain.WorkflowID, version int) (*domain.WorkflowDefinition, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, workflow_id, organization_id, version, status, created_at
		FROM workflow_versions
		WHERE workflow_id = $1 AND organization_id = $2 AND version = $3
	`, workflowID.String(), orgID.String(), version)
	return scanDefinition(ctx, r.pool, row)
}

func (r *WorkflowRepository) FindLatestPublished(ctx context.Context, orgID domain.OrganizationID, workflowID domain.WorkflowID) (*domain.WorkflowDefinition, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, workflow_id, organization_id, version, status, created_at
		FROM workflow_versions
		WHERE workflow_id = $1 AND organization_id = $2 AND status = 'published'
		ORDER BY version DESC LIMIT 1
	`, workflowID.String(), orgID.String())
	return scanDefinition(ctx, r.pool, row)
}

func (r *WorkflowRepository) List(ctx context.Context, orgID domain.OrganizationID, limit int, cursor string) ([]domain.Workflow, string, error) {
	limit = platformpagination.NormalizePageSize(limit, 50, 100)
	args := []any{orgID.String()}
	where := "w.organization_id = $1"
	argPos := 2
	if cursor != "" {
		cursorTime, cursorID, err := platformpagination.DecodeTimeIDCursor(cursor)
		if err != nil {
			return nil, "", domain.ErrInvalidCursor
		}
		where += fmt.Sprintf(" AND (w.created_at < $%d OR (w.created_at = $%d AND w.id < $%d))", argPos, argPos, argPos+1)
		args = append(args, cursorTime, cursorID)
		argPos++
	}
	args = append(args, limit+1)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT w.id, w.organization_id, w.name, COALESCE(MAX(v.version),0), w.created_at, w.updated_at
		FROM workflows w
		LEFT JOIN workflow_versions v ON v.workflow_id = w.id
		WHERE %s
		GROUP BY w.id
		ORDER BY w.created_at DESC, w.id DESC
		LIMIT $%d
	`, where, argPos), args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var items []domain.Workflow
	for rows.Next() {
		var wf domain.Workflow
		var wfID, org, name string
		if err := rows.Scan(&wfID, &org, &name, &wf.LatestVersion, &wf.CreatedAt, &wf.UpdatedAt); err != nil {
			return nil, "", err
		}
		wf.ID, _ = domain.ParseWorkflowID(wfID)
		wf.OrganizationID, _ = domain.ParseOrganizationID(org)
		wf.Name = name
		items = append(items, wf)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(items) > limit {
		last := items[limit-1]
		next = platformpagination.EncodeTimeIDCursor(last.CreatedAt, last.ID.String())
		items = items[:limit]
	}
	return items, next, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDefinition(ctx context.Context, pool *pgxpool.Pool, row scannable) (*domain.WorkflowDefinition, error) {
	var def domain.WorkflowDefinition
	var id, workflowID, orgID, status string
	if err := row.Scan(&id, &workflowID, &orgID, &def.Version, &status, &def.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrWorkflowVersionNotFound
		}
		return nil, err
	}
	def.ID, _ = domain.ParseWorkflowVersionID(id)
	def.WorkflowID, _ = domain.ParseWorkflowID(workflowID)
	def.OrganizationID, _ = domain.ParseOrganizationID(orgID)
	def.Status = domain.DefinitionStatus(status)

	rows, err := pool.Query(ctx, `
		SELECT id, step_key, name, step_type, configuration, timeout_seconds, retry_policy, dependencies, step_order
		FROM workflow_steps WHERE workflow_version_id = $1 ORDER BY step_order ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var step domain.WorkflowStepDefinition
		var stepID, stepType string
		var retryJSON, depsJSON []byte
		var timeoutSec int
		if err := rows.Scan(&stepID, &step.StepKey, &step.Name, &stepType, &step.Configuration, &timeoutSec, &retryJSON, &depsJSON, &step.Order); err != nil {
			return nil, err
		}
		step.ID = stepID
		step.Type = domain.StepType(stepType)
		step.Timeout = time.Duration(timeoutSec) * time.Second
		_ = json.Unmarshal(retryJSON, &step.RetryPolicy)
		_ = json.Unmarshal(depsJSON, &step.Dependencies)
		def.Steps = append(def.Steps, step)
	}
	return &def, rows.Err()
}

var _ ports.WorkflowRepository = (*WorkflowRepository)(nil)

type dbExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}
