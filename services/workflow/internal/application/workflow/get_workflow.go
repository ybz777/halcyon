package workflow

import (
	"context"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

type GetWorkflowInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	WorkflowID     domain.WorkflowID
	Version        int
}

type GetWorkflowUseCase struct {
	workflows ports.WorkflowRepository
}

func NewGetWorkflowUseCase(workflows ports.WorkflowRepository) *GetWorkflowUseCase {
	return &GetWorkflowUseCase{workflows: workflows}
}

func (uc *GetWorkflowUseCase) Execute(ctx context.Context, input GetWorkflowInput) (*domain.Workflow, *domain.WorkflowDefinition, error) {
	if err := ensureRead(input.Principal); err != nil {
		return nil, nil, err
	}
	wf, err := uc.workflows.FindByID(ctx, input.OrganizationID, input.WorkflowID)
	if err != nil {
		return nil, nil, err
	}
	version := input.Version
	if version <= 0 {
		version = wf.LatestVersion
	}
	def, err := uc.workflows.FindDefinition(ctx, input.OrganizationID, input.WorkflowID, version)
	return wf, def, err
}

type ListWorkflowsInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	Limit          int
	Cursor         string
}

type ListWorkflowsUseCase struct {
	workflows ports.WorkflowRepository
}

func NewListWorkflowsUseCase(workflows ports.WorkflowRepository) *ListWorkflowsUseCase {
	return &ListWorkflowsUseCase{workflows: workflows}
}

func (uc *ListWorkflowsUseCase) Execute(ctx context.Context, input ListWorkflowsInput) ([]domain.Workflow, string, error) {
	if err := ensureRead(input.Principal); err != nil {
		return nil, "", err
	}
	return uc.workflows.List(ctx, input.OrganizationID, input.Limit, input.Cursor)
}
