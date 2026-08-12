package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/contextx"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// StartExecutionInput starts a workflow execution.
type StartExecutionInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	WorkflowID     domain.WorkflowID
	IdempotencyKey string
	CorrelationID  string
	TriggerDepth   int
}

// StartExecutionUseCase creates an execution and initial job.
type StartExecutionUseCase struct {
	workflows    ports.WorkflowRepository
	executions   ports.ExecutionRepository
	jobs         ports.JobRepository
	idempotency  ports.IdempotencyRepository
	publisher    ports.EventPublisher
	tx           ports.TransactionManager
	metrics      ports.WorkflowEngineMetrics
	limits       domain.WorkflowLimits
}

func NewStartExecutionUseCase(
	workflows ports.WorkflowRepository,
	executions ports.ExecutionRepository,
	jobs ports.JobRepository,
	idempotency ports.IdempotencyRepository,
	publisher ports.EventPublisher,
	tx ports.TransactionManager,
	metrics ports.WorkflowEngineMetrics,
	limits domain.WorkflowLimits,
) *StartExecutionUseCase {
	return &StartExecutionUseCase{
		workflows: workflows, executions: executions, jobs: jobs,
		idempotency: idempotency, publisher: publisher, tx: tx, metrics: metrics, limits: limits,
	}
}

func (uc *StartExecutionUseCase) Execute(ctx context.Context, input StartExecutionInput) (*domain.WorkflowExecution, error) {
	if err := ensureExecute(input.Principal); err != nil {
		return nil, err
	}
	if input.TriggerDepth > uc.limits.MaxTriggerDepth {
		return nil, domain.ErrRecursionDepthExceeded
	}

	if input.IdempotencyKey != "" {
		if existing, ok, err := uc.idempotency.Find(ctx, input.OrganizationID, input.IdempotencyKey); err != nil {
			return nil, err
		} else if ok {
			if uc.metrics != nil {
				uc.metrics.RecordIdempotencyHit()
			}
			return uc.executions.FindExecution(ctx, input.OrganizationID, existing)
		}
	}

	def, err := uc.workflows.FindLatestPublished(ctx, input.OrganizationID, input.WorkflowID)
	if err != nil {
		return nil, err
	}
	if len(def.Steps) == 0 {
		return nil, domain.ErrInvalidWorkflowDefinition
	}

	now := time.Now().UTC()
	executionID, err := domain.NewExecutionID()
	if err != nil {
		return nil, err
	}
	correlationID := input.CorrelationID
	if correlationID == "" {
		correlationID = executionID.String()
	}
	if cid := contextx.CorrelationIDFromContext(ctx); cid != "" {
		correlationID = cid
	}

	execution := &domain.WorkflowExecution{
		ID: executionID, WorkflowID: input.WorkflowID, WorkflowVersionID: def.ID,
		OrganizationID: input.OrganizationID, TriggeredBy: input.Principal.UserID,
		Status: domain.ExecutionPending, CorrelationID: correlationID,
		TriggerDepth: input.TriggerDepth, CreatedAt: now,
	}
	_ = domain.TransitionExecution(execution, domain.ExecutionRunning, now)

	stepExecs := make([]domain.WorkflowStepExecution, 0, len(def.Steps))
	for _, step := range def.Steps {
		stepID, err := domain.NewStepExecutionID()
		if err != nil {
			return nil, err
		}
		stepExecs = append(stepExecs, domain.WorkflowStepExecution{
			ID: stepID, ExecutionID: executionID, OrganizationID: input.OrganizationID,
			StepKey: step.StepKey, Status: domain.StepPending,
		})
	}

	firstStep := def.Steps[0]
	var firstJob *domain.Job

	if err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.executions.CreateExecution(txCtx, execution, stepExecs); err != nil {
			return err
		}
		if input.IdempotencyKey != "" {
			if err := uc.idempotency.Store(txCtx, input.OrganizationID, input.IdempotencyKey, executionID); err != nil {
				return err
			}
		}
		jobID, err := domain.NewJobID()
		if err != nil {
			return err
		}
		var stepExecID domain.StepExecutionID
		for _, se := range stepExecs {
			if se.StepKey == firstStep.StepKey {
				stepExecID = se.ID
				break
			}
		}
		firstJob = &domain.Job{
			ID: jobID, OrganizationID: input.OrganizationID, ExecutionID: executionID,
			StepExecutionID: stepExecID, StepKey: firstStep.StepKey,
			Status: domain.JobReady, AvailableAt: now, CreatedAt: now,
		}
		if err := uc.jobs.CreateJob(txCtx, firstJob); err != nil {
			return err
		}
		started := domain.NewWorkflowExecutionStarted(input.OrganizationID, executionID, input.WorkflowID, correlationID, input.Principal.UserID.String(), now)
		if err := uc.publisher.Publish(txCtx, started); err != nil {
			return err
		}
		stepStarted := domain.NewWorkflowStepStarted(input.OrganizationID, executionID, input.WorkflowID, firstStep.StepKey, 1, now)
		return uc.publisher.Publish(txCtx, stepStarted)
	}); err != nil {
		if input.IdempotencyKey != "" && errors.Is(err, domain.ErrDuplicateIdempotencyKey) {
			if uc.metrics != nil {
				uc.metrics.RecordIdempotencyHit()
			}
			existingID, ok, findErr := uc.idempotency.Find(ctx, input.OrganizationID, input.IdempotencyKey)
			if findErr != nil || !ok {
				return nil, err
			}
			return uc.executions.FindExecution(ctx, input.OrganizationID, existingID)
		}
		return nil, err
	}

	if uc.metrics != nil {
		uc.metrics.RecordExecution(string(domain.ExecutionRunning))
	}
	return execution, nil
}
