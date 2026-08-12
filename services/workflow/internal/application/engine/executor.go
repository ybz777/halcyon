package engine

import (
	"context"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// StepContext carries execution context for a step.
type StepContext struct {
	OrganizationID domain.OrganizationID
	ExecutionID    domain.ExecutionID
	WorkflowID     domain.WorkflowID
	CorrelationID  string
	StepKey        string
	Attempt        int
	TriggerDepth   int
}

// StepResult is the output of a step execution.
type StepResult struct {
	Output *domain.OutputReference
}

// Executor executes a single workflow step type.
type Executor interface {
	Type() domain.StepType
	Execute(ctx context.Context, step domain.WorkflowStepDefinition, execCtx StepContext) (StepResult, error)
}

// Registry maps step types to executors.
type Registry struct {
	executors map[domain.StepType]Executor
}

func NewRegistry(executors ...Executor) *Registry {
	r := &Registry{executors: make(map[domain.StepType]Executor)}
	for _, e := range executors {
		r.executors[e.Type()] = e
	}
	return r
}

func (r *Registry) Get(stepType domain.StepType) (Executor, bool) {
	e, ok := r.executors[stepType]
	return e, ok
}
