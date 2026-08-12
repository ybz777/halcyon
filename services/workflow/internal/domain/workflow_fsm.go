package domain

import "time"

// CanTransitionExecution reports whether a workflow execution may change status.
func CanTransitionExecution(from, to ExecutionStatus) bool {
	if from == to {
		return false
	}
	allowed, ok := executionTransitions[from]
	if !ok {
		return false
	}
	for _, next := range allowed {
		if next == to {
			return true
		}
	}
	return false
}

// CanTransitionStep reports whether a step execution may change status.
func CanTransitionStep(from, to StepStatus) bool {
	if from == to {
		return false
	}
	allowed, ok := stepTransitions[from]
	if !ok {
		return false
	}
	for _, next := range allowed {
		if next == to {
			return true
		}
	}
	return false
}

// CanTransitionJob reports whether a job may change status.
func CanTransitionJob(from, to JobStatus) bool {
	if from == to {
		return false
	}
	allowed, ok := jobTransitions[from]
	if !ok {
		return false
	}
	for _, next := range allowed {
		if next == to {
			return true
		}
	}
	return false
}

var executionTransitions = map[ExecutionStatus][]ExecutionStatus{
	ExecutionPending:   {ExecutionRunning, ExecutionCancelled},
	ExecutionRunning:   {ExecutionSucceeded, ExecutionFailed, ExecutionCancelled},
	ExecutionSucceeded: {},
	ExecutionFailed:    {},
	ExecutionCancelled: {},
}

var stepTransitions = map[StepStatus][]StepStatus{
	StepPending:   {StepRunning, StepRetryWait, StepCancelled},
	StepRunning:   {StepSucceeded, StepFailed, StepRetryWait, StepCancelled},
	StepRetryWait: {StepPending, StepRunning, StepCancelled},
	StepSucceeded: {},
	StepFailed:    {},
	StepCancelled: {},
}

var jobTransitions = map[JobStatus][]JobStatus{
	JobReady:      {JobRunning, JobCancelled},
	JobRunning:    {JobSucceeded, JobFailed, JobRetryWait, JobCancelled},
	JobRetryWait:  {JobReady, JobCancelled},
	JobSucceeded:  {},
	JobFailed:     {},
	JobCancelled:  {},
}

// TransitionExecution applies a workflow execution status change.
func TransitionExecution(execution *WorkflowExecution, to ExecutionStatus, now time.Time) error {
	if execution == nil {
		return ErrInvalidExecution
	}
	if !CanTransitionExecution(execution.Status, to) {
		return ErrInvalidTransition
	}
	execution.Status = to
	switch to {
	case ExecutionRunning:
		if execution.StartedAt == nil {
			execution.StartedAt = &now
		}
	case ExecutionSucceeded, ExecutionFailed, ExecutionCancelled:
		execution.CompletedAt = &now
	}
	return nil
}

// TransitionStep applies a step execution status change.
func TransitionStep(step *WorkflowStepExecution, to StepStatus, now time.Time) error {
	if step == nil {
		return ErrInvalidStepExecution
	}
	if !CanTransitionStep(step.Status, to) {
		return ErrInvalidTransition
	}
	step.Status = to
	switch to {
	case StepRunning:
		if step.StartedAt == nil {
			step.StartedAt = &now
		}
	case StepSucceeded, StepFailed, StepCancelled:
		step.CompletedAt = &now
	}
	return nil
}

// TransitionJob applies a job status change.
func TransitionJob(job *Job, to JobStatus, now time.Time) error {
	if job == nil {
		return ErrInvalidJob
	}
	if !CanTransitionJob(job.Status, to) {
		return ErrInvalidTransition
	}
	job.Status = to
	switch to {
	case JobSucceeded, JobFailed, JobCancelled:
		job.CompletedAt = &now
	case JobReady, JobRetryWait:
		job.LockedAt = nil
		job.LockedBy = ""
	}
	return nil
}
