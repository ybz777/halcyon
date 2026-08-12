package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ExecutionRepository implements ports.ExecutionRepository.
type ExecutionRepository struct {
	pool *pgxpool.Pool
}

func NewExecutionRepository(pool *pgxpool.Pool) *ExecutionRepository {
	return &ExecutionRepository{pool: pool}
}

func (r *ExecutionRepository) exec(ctx context.Context) dbExecutor {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *ExecutionRepository) CreateExecution(ctx context.Context, execution *domain.WorkflowExecution, steps []domain.WorkflowStepExecution) error {
	q := r.exec(ctx)
	_, err := q.Exec(ctx, `
		INSERT INTO workflow_executions (
			id, workflow_id, workflow_version_id, organization_id, triggered_by,
			status, correlation_id, trigger_depth, started_at, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, execution.ID.String(), execution.WorkflowID.String(), execution.WorkflowVersionID.String(),
		execution.OrganizationID.String(), execution.TriggeredBy.String(), string(execution.Status),
		execution.CorrelationID, execution.TriggerDepth, execution.StartedAt, execution.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert execution: %w", err)
	}

	for _, step := range steps {
		if err := r.insertStep(ctx, q, &step); err != nil {
			return err
		}
	}
	return nil
}

func (r *ExecutionRepository) insertStep(ctx context.Context, q dbExecutor, step *domain.WorkflowStepExecution) error {
	outJSON, _ := json.Marshal(step.OutputReference)
	_, err := q.Exec(ctx, `
		INSERT INTO workflow_step_executions (
			id, execution_id, organization_id, step_key, status, attempt_count,
			started_at, completed_at, last_error, output_reference
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, step.ID.String(), step.ExecutionID.String(), step.OrganizationID.String(),
		step.StepKey, string(step.Status), step.AttemptCount,
		step.StartedAt, step.CompletedAt, step.LastError, outJSON)
	if err != nil {
		return fmt.Errorf("insert step execution: %w", err)
	}
	return nil
}

func (r *ExecutionRepository) FindExecution(ctx context.Context, orgID domain.OrganizationID, id domain.ExecutionID) (*domain.WorkflowExecution, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, workflow_id, workflow_version_id, organization_id, triggered_by,
			status, correlation_id, trigger_depth, started_at, completed_at, created_at
		FROM workflow_executions WHERE id = $1 AND organization_id = $2
	`, id.String(), orgID.String())
	return scanExecution(row)
}

func (r *ExecutionRepository) UpdateExecution(ctx context.Context, execution *domain.WorkflowExecution) error {
	_, err := r.exec(ctx).Exec(ctx, `
		UPDATE workflow_executions
		SET status = $1, started_at = $2, completed_at = $3
		WHERE id = $4 AND organization_id = $5
	`, string(execution.Status), execution.StartedAt, execution.CompletedAt,
		execution.ID.String(), execution.OrganizationID.String())
	return err
}

func (r *ExecutionRepository) ListStepExecutions(ctx context.Context, orgID domain.OrganizationID, executionID domain.ExecutionID) ([]domain.WorkflowStepExecution, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, execution_id, organization_id, step_key, status, attempt_count,
			started_at, completed_at, last_error, output_reference
		FROM workflow_step_executions
		WHERE execution_id = $1 AND organization_id = $2
		ORDER BY step_key
	`, executionID.String(), orgID.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStepRows(rows)
}

func (r *ExecutionRepository) UpdateStepExecution(ctx context.Context, step *domain.WorkflowStepExecution) error {
	outJSON, _ := json.Marshal(step.OutputReference)
	_, err := r.exec(ctx).Exec(ctx, `
		UPDATE workflow_step_executions
		SET status = $1, attempt_count = $2, started_at = $3, completed_at = $4,
			last_error = $5, output_reference = $6
		WHERE id = $7 AND organization_id = $8
	`, string(step.Status), step.AttemptCount, step.StartedAt, step.CompletedAt,
		step.LastError, outJSON, step.ID.String(), step.OrganizationID.String())
	return err
}

func (r *ExecutionRepository) FindStepExecution(ctx context.Context, orgID domain.OrganizationID, executionID domain.ExecutionID, stepKey string) (*domain.WorkflowStepExecution, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, execution_id, organization_id, step_key, status, attempt_count,
			started_at, completed_at, last_error, output_reference
		FROM workflow_step_executions
		WHERE execution_id = $1 AND organization_id = $2 AND step_key = $3
	`, executionID.String(), orgID.String(), stepKey)
	return scanStep(row)
}

func scanExecution(row pgx.Row) (*domain.WorkflowExecution, error) {
	var ex domain.WorkflowExecution
	var id, workflowID, versionID, orgID, triggeredBy, status string
	if err := row.Scan(&id, &workflowID, &versionID, &orgID, &triggeredBy, &status,
		&ex.CorrelationID, &ex.TriggerDepth, &ex.StartedAt, &ex.CompletedAt, &ex.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrExecutionNotFound
		}
		return nil, err
	}
	ex.ID, _ = domain.ParseExecutionID(id)
	ex.WorkflowID, _ = domain.ParseWorkflowID(workflowID)
	ex.WorkflowVersionID, _ = domain.ParseWorkflowVersionID(versionID)
	ex.OrganizationID, _ = domain.ParseOrganizationID(orgID)
	ex.TriggeredBy, _ = domain.ParseUserID(triggeredBy)
	ex.Status = domain.ExecutionStatus(status)
	return &ex, nil
}

func scanStep(row pgx.Row) (*domain.WorkflowStepExecution, error) {
	var step domain.WorkflowStepExecution
	var id, executionID, orgID, stepKey, status string
	var outJSON []byte
	if err := row.Scan(&id, &executionID, &orgID, &stepKey, &status, &step.AttemptCount,
		&step.StartedAt, &step.CompletedAt, &step.LastError, &outJSON); err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrInvalidStepExecution
		}
		return nil, err
	}
	step.ID, _ = domain.ParseStepExecutionID(id)
	step.ExecutionID, _ = domain.ParseExecutionID(executionID)
	step.OrganizationID, _ = domain.ParseOrganizationID(orgID)
	step.StepKey = stepKey
	step.Status = domain.StepStatus(status)
	if len(outJSON) > 0 {
		var out domain.OutputReference
		_ = json.Unmarshal(outJSON, &out)
		step.OutputReference = &out
	}
	return &step, nil
}

func scanStepRows(rows pgx.Rows) ([]domain.WorkflowStepExecution, error) {
	var items []domain.WorkflowStepExecution
	for rows.Next() {
		var step domain.WorkflowStepExecution
		var id, executionID, orgID, stepKey, status string
		var outJSON []byte
		if err := rows.Scan(&id, &executionID, &orgID, &stepKey, &status, &step.AttemptCount,
			&step.StartedAt, &step.CompletedAt, &step.LastError, &outJSON); err != nil {
			return nil, err
		}
		step.ID, _ = domain.ParseStepExecutionID(id)
		step.ExecutionID, _ = domain.ParseExecutionID(executionID)
		step.OrganizationID, _ = domain.ParseOrganizationID(orgID)
		step.StepKey = stepKey
		step.Status = domain.StepStatus(status)
		if len(outJSON) > 0 {
			var out domain.OutputReference
			_ = json.Unmarshal(outJSON, &out)
			step.OutputReference = &out
		}
		items = append(items, step)
	}
	return items, rows.Err()
}

var _ ports.ExecutionRepository = (*ExecutionRepository)(nil)
