package workflow

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

type GetExecutionInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	ExecutionID    domain.ExecutionID
}

type GetExecutionUseCase struct {
	executions ports.ExecutionRepository
}

func NewGetExecutionUseCase(executions ports.ExecutionRepository) *GetExecutionUseCase {
	return &GetExecutionUseCase{executions: executions}
}

func (uc *GetExecutionUseCase) Execute(ctx context.Context, input GetExecutionInput) (*domain.WorkflowExecution, []domain.WorkflowStepExecution, error) {
	if err := ensureRead(input.Principal); err != nil {
		return nil, nil, err
	}
	execution, err := uc.executions.FindExecution(ctx, input.OrganizationID, input.ExecutionID)
	if err != nil {
		return nil, nil, err
	}
	steps, err := uc.executions.ListStepExecutions(ctx, input.OrganizationID, input.ExecutionID)
	return execution, steps, err
}

type CancelExecutionInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	ExecutionID    domain.ExecutionID
}

type CancelExecutionUseCase struct {
	executions ports.ExecutionRepository
	publisher  ports.EventPublisher
	tx         ports.TransactionManager
	metrics    ports.WorkflowEngineMetrics
}

func NewCancelExecutionUseCase(
	executions ports.ExecutionRepository,
	publisher ports.EventPublisher,
	tx ports.TransactionManager,
	metrics ports.WorkflowEngineMetrics,
) *CancelExecutionUseCase {
	return &CancelExecutionUseCase{executions: executions, publisher: publisher, tx: tx, metrics: metrics}
}

func (uc *CancelExecutionUseCase) Execute(ctx context.Context, input CancelExecutionInput) (*domain.WorkflowExecution, error) {
	if err := ensureCancel(input.Principal); err != nil {
		return nil, err
	}
	execution, err := uc.executions.FindExecution(ctx, input.OrganizationID, input.ExecutionID)
	if err != nil {
		return nil, err
	}
	if execution.Status == domain.ExecutionSucceeded || execution.Status == domain.ExecutionFailed || execution.Status == domain.ExecutionCancelled {
		return execution, nil
	}

	now := time.Now().UTC()
	if err := domain.TransitionExecution(execution, domain.ExecutionCancelled, now); err != nil {
		return nil, err
	}

	if err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := uc.executions.UpdateExecution(txCtx, execution); err != nil {
			return err
		}
		event := domain.NewWorkflowExecutionCancelled(input.OrganizationID, execution.ID, execution.WorkflowID, now)
		return uc.publisher.Publish(txCtx, event)
	}); err != nil {
		return nil, err
	}

	if uc.metrics != nil {
		uc.metrics.RecordCancellation()
	}
	return execution, nil
}
