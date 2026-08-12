package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const auditIDPrefix = "aud_"

// ActorType identifies who performed an action.
type ActorType string

const (
	ActorUser      ActorType = "USER"
	ActorService   ActorType = "SERVICE"
	ActorAPIKey    ActorType = "API_KEY"
	ActorSystem    ActorType = "SYSTEM"
	ActorAnonymous ActorType = "ANONYMOUS"
)

// TargetType identifies the resource affected.
type TargetType string

const (
	TargetUser              TargetType = "USER"
	TargetOrganization      TargetType = "ORGANIZATION"
	TargetMembership        TargetType = "MEMBERSHIP"
	TargetInvitation        TargetType = "INVITATION"
	TargetAPIKey            TargetType = "API_KEY"
	TargetWorkflow          TargetType = "WORKFLOW"
	TargetWorkflowExecution TargetType = "WORKFLOW_EXECUTION"
	TargetService           TargetType = "SERVICE"
	TargetAuthSession       TargetType = "AUTH_SESSION"
	TargetSystem            TargetType = "SYSTEM"
)

// Result describes the outcome of an audited action.
type Result string

const (
	ResultSuccess Result = "SUCCESS"
	ResultFailure Result = "FAILURE"
	ResultDenied  Result = "DENIED"
)

// AuditEventID is a globally unique audit record identifier.
type AuditEventID struct{ value string }

func NewAuditEventID() (AuditEventID, error) {
	raw, err := ids.Generate(auditIDPrefix)
	if err != nil {
		return AuditEventID{}, err
	}
	return ParseAuditEventID(raw)
}

func ParseAuditEventID(raw string) (AuditEventID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, auditIDPrefix) {
		return AuditEventID{}, fmt.Errorf("%w: invalid audit id", ErrInvalidAuditEventID)
	}
	return AuditEventID{value: raw}, nil
}

func (id AuditEventID) String() string { return id.value }
func (id AuditEventID) IsZero() bool   { return id.value == "" }

// AuditEvent is an immutable audit log record.
type AuditEvent struct {
	ID             AuditEventID
	OrganizationID string
	EventType      string
	Action         string
	ActorType      ActorType
	ActorID        string
	TargetType     TargetType
	TargetID       string
	Result         Result
	OccurredAt     time.Time
	RequestID      string
	CorrelationID  string
	IPAddress      string
	UserAgent      string
	SourceService  string
	Metadata       map[string]string
	CreatedAt      time.Time
}

// MetadataJSON returns sanitized metadata as JSON bytes.
func (e AuditEvent) MetadataJSON() ([]byte, error) {
	if e.Metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(e.Metadata)
}

// IsSecurityEvent reports whether the event requires elevated audit permission.
func (e AuditEvent) IsSecurityEvent() bool {
	switch {
	case strings.HasPrefix(e.EventType, "security."):
		return true
	case strings.HasPrefix(e.EventType, "auth.login.failed"):
		return true
	case strings.HasPrefix(e.EventType, "authz."):
		return true
	default:
		return false
	}
}
