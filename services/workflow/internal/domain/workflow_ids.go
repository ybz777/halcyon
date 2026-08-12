package domain

import (
	"fmt"
	"strings"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const (
	workflowIDPrefix         = "wfl_"
	workflowVersionIDPrefix  = "wfv_"
	executionIDPrefix        = "wex_"
	stepExecutionIDPrefix    = "wse_"
	jobIDPrefix              = "wjb_"
	attemptIDPrefix          = "wat_"
)

// WorkflowID identifies a workflow definition lineage.
type WorkflowID struct{ value string }

func NewWorkflowID() (WorkflowID, error) {
	raw, err := ids.Generate(workflowIDPrefix)
	if err != nil {
		return WorkflowID{}, err
	}
	return ParseWorkflowID(raw)
}

func ParseWorkflowID(raw string) (WorkflowID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, workflowIDPrefix) || len(raw) < len(workflowIDPrefix)+8 {
		return WorkflowID{}, fmt.Errorf("%w: invalid workflow id", ErrInvalidWorkflowID)
	}
	return WorkflowID{value: raw}, nil
}

func (id WorkflowID) String() string { return id.value }
func (id WorkflowID) IsZero() bool   { return id.value == "" }

// WorkflowVersionID identifies an immutable workflow version.
type WorkflowVersionID struct{ value string }

func NewWorkflowVersionID() (WorkflowVersionID, error) {
	raw, err := ids.Generate(workflowVersionIDPrefix)
	if err != nil {
		return WorkflowVersionID{}, err
	}
	return ParseWorkflowVersionID(raw)
}

func ParseWorkflowVersionID(raw string) (WorkflowVersionID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, workflowVersionIDPrefix) {
		return WorkflowVersionID{}, fmt.Errorf("%w: invalid workflow version id", ErrInvalidWorkflowVersionID)
	}
	return WorkflowVersionID{value: raw}, nil
}

func (id WorkflowVersionID) String() string { return id.value }
func (id WorkflowVersionID) IsZero() bool   { return id.value == "" }

// ExecutionID identifies a workflow execution run.
type ExecutionID struct{ value string }

func NewExecutionID() (ExecutionID, error) {
	raw, err := ids.Generate(executionIDPrefix)
	if err != nil {
		return ExecutionID{}, err
	}
	return ParseExecutionID(raw)
}

func ParseExecutionID(raw string) (ExecutionID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, executionIDPrefix) {
		return ExecutionID{}, fmt.Errorf("%w: invalid execution id", ErrInvalidExecutionID)
	}
	return ExecutionID{value: raw}, nil
}

func (id ExecutionID) String() string { return id.value }
func (id ExecutionID) IsZero() bool   { return id.value == "" }

// StepExecutionID identifies a step execution record.
type StepExecutionID struct{ value string }

func NewStepExecutionID() (StepExecutionID, error) {
	raw, err := ids.Generate(stepExecutionIDPrefix)
	if err != nil {
		return StepExecutionID{}, err
	}
	return ParseStepExecutionID(raw)
}

func ParseStepExecutionID(raw string) (StepExecutionID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, stepExecutionIDPrefix) {
		return StepExecutionID{}, fmt.Errorf("%w: invalid step execution id", ErrInvalidStepExecutionID)
	}
	return StepExecutionID{value: raw}, nil
}

func (id StepExecutionID) String() string { return id.value }
func (id StepExecutionID) IsZero() bool   { return id.value == "" }

// JobID identifies a durable workflow job.
type JobID struct{ value string }

func NewJobID() (JobID, error) {
	raw, err := ids.Generate(jobIDPrefix)
	if err != nil {
		return JobID{}, err
	}
	return ParseJobID(raw)
}

func ParseJobID(raw string) (JobID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, jobIDPrefix) {
		return JobID{}, fmt.Errorf("%w: invalid job id", ErrInvalidJobID)
	}
	return JobID{value: raw}, nil
}

func (id JobID) String() string { return id.value }
func (id JobID) IsZero() bool   { return id.value == "" }

// AttemptID identifies a job execution attempt.
type AttemptID struct{ value string }

func NewAttemptID() (AttemptID, error) {
	raw, err := ids.Generate(attemptIDPrefix)
	if err != nil {
		return AttemptID{}, err
	}
	return ParseAttemptID(raw)
}

func ParseAttemptID(raw string) (AttemptID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, attemptIDPrefix) {
		return AttemptID{}, fmt.Errorf("%w: invalid attempt id", ErrInvalidAttemptID)
	}
	return AttemptID{value: raw}, nil
}

func (id AttemptID) String() string { return id.value }
func (id AttemptID) IsZero() bool   { return id.value == "" }
