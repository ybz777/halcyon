package domain

import "time"

// Workflow domain events for outbox publication.

const (
	EventWorkflowExecutionStarted   = "workflow.execution.started"
	EventWorkflowExecutionSucceeded = "workflow.execution.succeeded"
	EventWorkflowExecutionFailed    = "workflow.execution.failed"
	EventWorkflowExecutionCancelled = "workflow.execution.cancelled"
	EventWorkflowStepStarted        = "workflow.step.started"
	EventWorkflowStepSucceeded      = "workflow.step.succeeded"
	EventWorkflowStepFailed         = "workflow.step.failed"
)

// WorkflowEvent is a workflow domain event marker.
type WorkflowEvent interface {
	Event
	ExecutionID() ExecutionID
}

type executionEvent struct {
	name           string
	occurredAt     time.Time
	organizationID OrganizationID
	executionID    ExecutionID
	workflowID     WorkflowID
	stepKey        string
	reason         string
}

func (e executionEvent) EventName() string     { return e.name }
func (e executionEvent) OccurredAt() time.Time { return e.occurredAt }
func (e executionEvent) ExecutionID() ExecutionID { return e.executionID }
func (e executionEvent) OrganizationID() OrganizationID { return e.organizationID }
func (e executionEvent) WorkflowID() WorkflowID { return e.workflowID }
func (e executionEvent) StepKey() string { return e.stepKey }

// WorkflowExecutionStarted is emitted when an execution begins.
type WorkflowExecutionStarted struct {
	executionEvent
	CorrelationID string
	TriggeredBy   string
}

func NewWorkflowExecutionStarted(org OrganizationID, execution ExecutionID, workflow WorkflowID, correlation, triggeredBy string, at time.Time) WorkflowExecutionStarted {
	return WorkflowExecutionStarted{
		executionEvent: executionEvent{
			name: EventWorkflowExecutionStarted, occurredAt: at,
			organizationID: org, executionID: execution, workflowID: workflow,
		},
		CorrelationID: correlation,
		TriggeredBy:   triggeredBy,
	}
}

// WorkflowExecutionSucceeded is emitted when an execution completes successfully.
type WorkflowExecutionSucceeded struct{ executionEvent }

func NewWorkflowExecutionSucceeded(org OrganizationID, execution ExecutionID, workflow WorkflowID, at time.Time) WorkflowExecutionSucceeded {
	return WorkflowExecutionSucceeded{executionEvent{
		name: EventWorkflowExecutionSucceeded, occurredAt: at,
		organizationID: org, executionID: execution, workflowID: workflow,
	}}
}

// WorkflowExecutionFailed is emitted when an execution fails.
type WorkflowExecutionFailed struct {
	executionEvent
	Reason string
}

func NewWorkflowExecutionFailed(org OrganizationID, execution ExecutionID, workflow WorkflowID, reason string, at time.Time) WorkflowExecutionFailed {
	return WorkflowExecutionFailed{
		executionEvent: executionEvent{
			name: EventWorkflowExecutionFailed, occurredAt: at,
			organizationID: org, executionID: execution, workflowID: workflow, reason: reason,
		},
		Reason: reason,
	}
}

// WorkflowExecutionCancelled is emitted when an execution is cancelled.
type WorkflowExecutionCancelled struct{ executionEvent }

func NewWorkflowExecutionCancelled(org OrganizationID, execution ExecutionID, workflow WorkflowID, at time.Time) WorkflowExecutionCancelled {
	return WorkflowExecutionCancelled{executionEvent{
		name: EventWorkflowExecutionCancelled, occurredAt: at,
		organizationID: org, executionID: execution, workflowID: workflow,
	}}
}

// WorkflowStepStarted is emitted when a step begins.
type WorkflowStepStarted struct {
	executionEvent
	Attempt int
}

func NewWorkflowStepStarted(org OrganizationID, execution ExecutionID, workflow WorkflowID, stepKey string, attempt int, at time.Time) WorkflowStepStarted {
	return WorkflowStepStarted{
		executionEvent: executionEvent{
			name: EventWorkflowStepStarted, occurredAt: at,
			organizationID: org, executionID: execution, workflowID: workflow, stepKey: stepKey,
		},
		Attempt: attempt,
	}
}

// WorkflowStepSucceeded is emitted when a step succeeds.
type WorkflowStepSucceeded struct{ executionEvent }

func NewWorkflowStepSucceeded(org OrganizationID, execution ExecutionID, workflow WorkflowID, stepKey string, at time.Time) WorkflowStepSucceeded {
	return WorkflowStepSucceeded{executionEvent{
		name: EventWorkflowStepSucceeded, occurredAt: at,
		organizationID: org, executionID: execution, workflowID: workflow, stepKey: stepKey,
	}}
}

// WorkflowStepFailed is emitted when a step fails permanently.
type WorkflowStepFailed struct {
	executionEvent
	Reason string
}

func NewWorkflowStepFailed(org OrganizationID, execution ExecutionID, workflow WorkflowID, stepKey, reason string, at time.Time) WorkflowStepFailed {
	return WorkflowStepFailed{
		executionEvent: executionEvent{
			name: EventWorkflowStepFailed, occurredAt: at,
			organizationID: org, executionID: execution, workflowID: workflow, stepKey: stepKey, reason: reason,
		},
		Reason: reason,
	}
}
