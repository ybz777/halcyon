package jobs

import (
	"fmt"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const jobIDPrefix = "job_"

// Status represents the job lifecycle state.
type Status string

const (
	StatusPending      Status = "PENDING"
	StatusRunning      Status = "RUNNING"
	StatusRetryWait    Status = "RETRY_WAIT"
	StatusSucceeded    Status = "SUCCEEDED"
	StatusFailed       Status = "FAILED"
	StatusDeadLettered Status = "DEAD_LETTERED"
	StatusCancelled    Status = "CANCELLED"
)

// IsTerminal reports whether the job has reached a final state.
func (s Status) IsTerminal() bool {
	switch s {
	case StatusSucceeded, StatusFailed, StatusDeadLettered, StatusCancelled:
		return true
	default:
		return false
	}
}

// Priority is a bounded job priority.
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityNormal Priority = "NORMAL"
	PriorityHigh   Priority = "HIGH"
)

// Type identifies a registered job handler.
type Type string

const (
	TypeWebhookDelivery      Type = "webhook.delivery"
	TypeEmailDelivery        Type = "email.delivery"
	TypeNotificationDispatch Type = "notification.dispatch"
	TypeWorkflowExecution    Type = "workflow.execution"
	TypeAuditPersistence     Type = "audit.persistence"
)

var validTypes = map[Type]struct{}{
	TypeWebhookDelivery: {}, TypeEmailDelivery: {}, TypeNotificationDispatch: {},
	TypeWorkflowExecution: {}, TypeAuditPersistence: {},
}

// ValidateType reports whether a job type is registered.
func ValidateType(t Type) error {
	if _, ok := validTypes[t]; ok {
		return nil
	}
	return fmt.Errorf("%w: %q", ErrInvalidJobType, t)
}

// ID is a typed job identifier.
type ID struct{ value string }

func NewID() (ID, error) {
	raw, err := ids.Generate(jobIDPrefix)
	if err != nil {
		return ID{}, err
	}
	return ID{value: raw}, nil
}

func ParseID(raw string) (ID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, jobIDPrefix) {
		return ID{}, fmt.Errorf("%w: invalid id", ErrInvalidJobID)
	}
	return ID{value: raw}, nil
}

func (id ID) String() string { return id.value }

// Job is a durable background work item.
type Job struct {
	ID             ID
	OrganizationID string
	Type           Type
	Status         Status
	Payload        []byte
	Priority       Priority
	AttemptCount   int
	MaxAttempts    int
	IdempotencyKey string
	AvailableAt    time.Time
	LockedAt       *time.Time
	LockedBy       string
	LastError      string
	CreatedAt      time.Time
	StartedAt      *time.Time
	CompletedAt    *time.Time
	FailedAt       *time.Time
}

const MaxPayloadBytes = 64 * 1024
