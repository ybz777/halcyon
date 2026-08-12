package domain

import "time"

// DefinitionStatus represents workflow definition lifecycle.
type DefinitionStatus string

const (
	DefinitionDraft     DefinitionStatus = "draft"
	DefinitionPublished DefinitionStatus = "published"
)

// ExecutionStatus represents workflow execution lifecycle.
type ExecutionStatus string

const (
	ExecutionPending   ExecutionStatus = "PENDING"
	ExecutionRunning   ExecutionStatus = "RUNNING"
	ExecutionSucceeded ExecutionStatus = "SUCCEEDED"
	ExecutionFailed    ExecutionStatus = "FAILED"
	ExecutionCancelled ExecutionStatus = "CANCELLED"
)

// StepStatus represents step execution lifecycle.
type StepStatus string

const (
	StepPending   StepStatus = "PENDING"
	StepRunning   StepStatus = "RUNNING"
	StepSucceeded StepStatus = "SUCCEEDED"
	StepFailed    StepStatus = "FAILED"
	StepRetryWait StepStatus = "RETRY_WAIT"
	StepCancelled StepStatus = "CANCELLED"
)

// JobStatus represents durable job lifecycle.
type JobStatus string

const (
	JobReady      JobStatus = "READY"
	JobRunning    JobStatus = "RUNNING"
	JobSucceeded  JobStatus = "SUCCEEDED"
	JobFailed     JobStatus = "FAILED"
	JobRetryWait  JobStatus = "RETRY_WAIT"
	JobCancelled  JobStatus = "CANCELLED"
)

// StepType identifies executor implementation.
type StepType string

const (
	StepTypeHTTPAction StepType = "HTTP_ACTION"
	StepTypeEventEmit  StepType = "EVENT_EMIT"
	StepTypeDelay      StepType = "DELAY"
)

// RetryPolicy configures step retry behavior.
type RetryPolicy struct {
	MaxAttempts       int           `json:"max_attempts"`
	InitialBackoff    time.Duration `json:"initial_backoff"`
	MaxBackoff        time.Duration `json:"max_backoff"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
}

// DefaultRetryPolicy returns safe retry defaults.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts:       4,
		InitialBackoff:    time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2,
	}
}

// OutputReference points to stored step output.
type OutputReference struct {
	Type        string `json:"type"`
	Location    string `json:"location"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	Inline      string `json:"inline,omitempty"`
}

// WorkflowStepDefinition describes an immutable step in a version.
type WorkflowStepDefinition struct {
	ID           string
	StepKey      string
	Name         string
	Type         StepType
	Configuration []byte
	Timeout      time.Duration
	RetryPolicy  RetryPolicy
	Dependencies []string
	Order        int
}

// WorkflowDefinition is an immutable published workflow version.
type WorkflowDefinition struct {
	ID             WorkflowVersionID
	WorkflowID     WorkflowID
	OrganizationID OrganizationID
	Version        int
	Status         DefinitionStatus
	Steps          []WorkflowStepDefinition
	CreatedAt      time.Time
}

// Workflow is the durable workflow identity.
type Workflow struct {
	ID             WorkflowID
	OrganizationID OrganizationID
	Name           string
	LatestVersion  int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// WorkflowExecution is one concrete workflow run.
type WorkflowExecution struct {
	ID                ExecutionID
	WorkflowID        WorkflowID
	WorkflowVersionID WorkflowVersionID
	OrganizationID    OrganizationID
	TriggeredBy       UserID
	Status            ExecutionStatus
	CorrelationID     string
	TriggerDepth      int
	StartedAt         *time.Time
	CompletedAt       *time.Time
	CreatedAt         time.Time
}

// WorkflowStepExecution tracks per-step execution state.
type WorkflowStepExecution struct {
	ID              StepExecutionID
	ExecutionID     ExecutionID
	OrganizationID  OrganizationID
	StepKey         string
	Status          StepStatus
	AttemptCount    int
	StartedAt       *time.Time
	CompletedAt     *time.Time
	LastError       string
	OutputReference *OutputReference
}

// Job represents durable executable work.
type Job struct {
	ID              JobID
	OrganizationID  OrganizationID
	ExecutionID     ExecutionID
	StepExecutionID StepExecutionID
	StepKey         string
	Status          JobStatus
	Attempt         int
	AvailableAt     time.Time
	LockedAt        *time.Time
	LockedBy        string
	CreatedAt       time.Time
	CompletedAt     *time.Time
}

// ExecutionAttempt records a single job attempt.
type ExecutionAttempt struct {
	ID              AttemptID
	JobID           JobID
	OrganizationID  OrganizationID
	Attempt         int
	Status          string
	StartedAt       time.Time
	CompletedAt     *time.Time
	ErrorMessage    string
	OutputReference *OutputReference
}
