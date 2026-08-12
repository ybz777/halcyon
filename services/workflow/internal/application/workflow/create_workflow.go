package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// CreateWorkflowInput carries workflow creation data.
type CreateWorkflowInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	Name           string
	Steps          []domain.StepInput
}

// CreateWorkflowUseCase creates and publishes a workflow definition.
type CreateWorkflowUseCase struct {
	workflows ports.WorkflowRepository
	tx        ports.TransactionManager
	limits    domain.WorkflowLimits
}

func NewCreateWorkflowUseCase(workflows ports.WorkflowRepository, tx ports.TransactionManager, limits domain.WorkflowLimits) *CreateWorkflowUseCase {
	return &CreateWorkflowUseCase{workflows: workflows, tx: tx, limits: limits}
}

func (uc *CreateWorkflowUseCase) Execute(ctx context.Context, input CreateWorkflowInput) (*domain.Workflow, *domain.WorkflowDefinition, error) {
	if err := ensureWrite(input.Principal); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateCreateWorkflow(domain.CreateWorkflowInput{Name: input.Name, Steps: input.Steps}, uc.limits); err != nil {
		return nil, nil, err
	}

	now := time.Now().UTC()
	workflowID, err := domain.NewWorkflowID()
	if err != nil {
		return nil, nil, err
	}
	versionID, err := domain.NewWorkflowVersionID()
	if err != nil {
		return nil, nil, err
	}

	workflow := &domain.Workflow{
		ID: workflowID, OrganizationID: input.OrganizationID,
		Name: input.Name, LatestVersion: 1, CreatedAt: now, UpdatedAt: now,
	}

	steps := make([]domain.WorkflowStepDefinition, 0, len(input.Steps))
	for i, s := range input.Steps {
		stepID := fmt.Sprintf("%s_%s", versionID.String(), s.StepKey)
		timeout := uc.limits.DefaultStepTimeout
		if s.Timeout != "" {
			timeout, _ = time.ParseDuration(s.Timeout)
		}
		policy := domain.DefaultRetryPolicy()
		if s.RetryPolicy != nil {
			policy = *s.RetryPolicy
		}
		cfg := s.Configuration
		if cfg == nil {
			cfg = json.RawMessage(`{}`)
		}
		steps = append(steps, domain.WorkflowStepDefinition{
			ID: stepID, StepKey: s.StepKey, Name: s.Name,
			Type: domain.StepType(s.Type), Configuration: cfg,
			Timeout: timeout, RetryPolicy: policy, Dependencies: s.Dependencies, Order: i,
		})
	}

	definition := &domain.WorkflowDefinition{
		ID: versionID, WorkflowID: workflowID, OrganizationID: input.OrganizationID,
		Version: 1, Status: domain.DefinitionPublished, Steps: steps, CreatedAt: now,
	}

	if err := uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return uc.workflows.Create(txCtx, workflow, definition)
	}); err != nil {
		return nil, nil, err
	}
	return workflow, definition, nil
}
