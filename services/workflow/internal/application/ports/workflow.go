package ports

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// WorkflowRepository persists workflow definitions.
type WorkflowRepository interface {
	Create(ctx context.Context, workflow *domain.Workflow, definition *domain.WorkflowDefinition) error
	FindByID(ctx context.Context, orgID domain.OrganizationID, id domain.WorkflowID) (*domain.Workflow, error)
	FindDefinition(ctx context.Context, orgID domain.OrganizationID, workflowID domain.WorkflowID, version int) (*domain.WorkflowDefinition, error)
	FindDefinitionByVersionID(ctx context.Context, orgID domain.OrganizationID, versionID domain.WorkflowVersionID) (*domain.WorkflowDefinition, error)
	FindLatestPublished(ctx context.Context, orgID domain.OrganizationID, workflowID domain.WorkflowID) (*domain.WorkflowDefinition, error)
	List(ctx context.Context, orgID domain.OrganizationID, limit int, cursor string) ([]domain.Workflow, string, error)
}

// ExecutionRepository persists workflow executions and step executions.
type ExecutionRepository interface {
	CreateExecution(ctx context.Context, execution *domain.WorkflowExecution, steps []domain.WorkflowStepExecution) error
	FindExecution(ctx context.Context, orgID domain.OrganizationID, id domain.ExecutionID) (*domain.WorkflowExecution, error)
	UpdateExecution(ctx context.Context, execution *domain.WorkflowExecution) error
	ListStepExecutions(ctx context.Context, orgID domain.OrganizationID, executionID domain.ExecutionID) ([]domain.WorkflowStepExecution, error)
	UpdateStepExecution(ctx context.Context, step *domain.WorkflowStepExecution) error
	FindStepExecution(ctx context.Context, orgID domain.OrganizationID, executionID domain.ExecutionID, stepKey string) (*domain.WorkflowStepExecution, error)
}

// JobRepository manages durable workflow jobs.
type JobRepository interface {
	CreateJob(ctx context.Context, job *domain.Job) error
	ClaimJobs(ctx context.Context, workerID string, limit int, lease time.Duration) ([]domain.Job, error)
	UpdateJob(ctx context.Context, job *domain.Job) error
	RecoverExpiredLeases(ctx context.Context, lease time.Duration) (int64, error)
	CountReady(ctx context.Context) (int64, error)
	CreateAttempt(ctx context.Context, attempt *domain.ExecutionAttempt) error
	UpdateAttempt(ctx context.Context, attempt *domain.ExecutionAttempt) error
}

// IdempotencyRepository stores execution idempotency keys.
type IdempotencyRepository interface {
	Find(ctx context.Context, orgID domain.OrganizationID, key string) (domain.ExecutionID, bool, error)
	Store(ctx context.Context, orgID domain.OrganizationID, key string, executionID domain.ExecutionID) error
}

// WorkflowEngineMetrics records workflow engine observability.
type WorkflowEngineMetrics interface {
	RecordExecution(status string)
	RecordExecutionFailure(failureCategory string)
	ObserveExecutionDuration(duration time.Duration)
	RecordStepExecution(stepType, status string)
	RecordStepFailure(stepType, failureCategory string)
	RecordJobClaim()
	RecordJobRetry(stepType string)
	RecordJobRecovery()
	SetJobQueueDepth(depth int64)
	ObserveJobDuration(stepType string, duration time.Duration)
	RecordCancellation()
	RecordIdempotencyHit()
}

const (
	PermissionWorkflowRead    = "workflows.read"
	PermissionWorkflowWrite   = "workflows.write"
	PermissionWorkflowExecute = "workflows.execute"
	PermissionWorkflowCancel  = "workflows.cancel"
)
