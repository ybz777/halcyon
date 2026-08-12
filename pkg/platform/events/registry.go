package events

import (
	"fmt"
	"sync"
)

// Descriptor describes a registered event type.
type Descriptor struct {
	Type          string
	Version       string
	AggregateType string
}

// Registry maps event types to descriptors for deserialization.
type Registry struct {
	mu    sync.RWMutex
	types map[string]Descriptor
}

// NewRegistry creates an empty event registry.
func NewRegistry() *Registry {
	return &Registry{types: make(map[string]Descriptor)}
}

// Register adds an event type. Future events plug in via Register.
func (r *Registry) Register(desc Descriptor) error {
	if desc.Type == "" || desc.Version == "" {
		return fmt.Errorf("event type and version are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types[desc.Type] = desc
	return nil
}

// Lookup returns a descriptor for an event type.
func (r *Registry) Lookup(eventType string) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	desc, ok := r.types[eventType]
	return desc, ok
}

// DeserializePayload unmarshals payload bytes for a registered event type.
func (r *Registry) DeserializePayload(eventType string, payload []byte) (any, error) {
	if _, ok := r.Lookup(eventType); !ok {
		return nil, fmt.Errorf("unknown event type %q", eventType)
	}
	msg, err := UnmarshalPayload(eventType, payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	return msg, nil
}

// DefaultRegistry contains initial Halcyon platform events.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	_ = r.Register(Descriptor{Type: TypeUserRegistered, Version: VersionV1, AggregateType: "user"})
	_ = r.Register(Descriptor{Type: TypeUserLoggedIn, Version: VersionV1, AggregateType: "user"})
	_ = r.Register(Descriptor{Type: TypeUserLoginFailed, Version: VersionV1, AggregateType: "user"})
	_ = r.Register(Descriptor{Type: TypeOrganizationCreated, Version: VersionV1, AggregateType: "organization"})
	_ = r.Register(Descriptor{Type: TypeOrganizationUpdated, Version: VersionV1, AggregateType: "organization"})
	_ = r.Register(Descriptor{Type: TypeOrganizationDeleted, Version: VersionV1, AggregateType: "organization"})
	_ = r.Register(Descriptor{Type: TypeMembershipCreated, Version: VersionV1, AggregateType: "membership"})
	_ = r.Register(Descriptor{Type: TypeMembershipRoleChanged, Version: VersionV1, AggregateType: "membership"})
	_ = r.Register(Descriptor{Type: TypeMembershipSuspended, Version: VersionV1, AggregateType: "membership"})
	_ = r.Register(Descriptor{Type: TypeMembershipRemoved, Version: VersionV1, AggregateType: "membership"})
	_ = r.Register(Descriptor{Type: TypeMemberInvited, Version: VersionV1, AggregateType: "membership"})
	_ = r.Register(Descriptor{Type: TypeMemberJoined, Version: VersionV1, AggregateType: "membership"})
	_ = r.Register(Descriptor{Type: TypeInvitationCreated, Version: VersionV1, AggregateType: "invitation"})
	_ = r.Register(Descriptor{Type: TypeInvitationAccepted, Version: VersionV1, AggregateType: "invitation"})
	_ = r.Register(Descriptor{Type: TypeInvitationRevoked, Version: VersionV1, AggregateType: "invitation"})
	_ = r.Register(Descriptor{Type: TypeAPIKeyCreated, Version: VersionV1, AggregateType: "api_key"})
	_ = r.Register(Descriptor{Type: TypeAPIKeyRevoked, Version: VersionV1, AggregateType: "api_key"})
	_ = r.Register(Descriptor{Type: TypeAPIKeyRotated, Version: VersionV1, AggregateType: "api_key"})
	_ = r.Register(Descriptor{Type: TypeAPIKeyAuthenticationFailed, Version: VersionV1, AggregateType: "api_key"})
	_ = r.Register(Descriptor{Type: TypeWorkflowExecutionStarted, Version: VersionV1, AggregateType: "workflow_execution"})
	_ = r.Register(Descriptor{Type: TypeWorkflowExecutionSucceeded, Version: VersionV1, AggregateType: "workflow_execution"})
	_ = r.Register(Descriptor{Type: TypeWorkflowExecutionFailed, Version: VersionV1, AggregateType: "workflow_execution"})
	_ = r.Register(Descriptor{Type: TypeWorkflowExecutionCancelled, Version: VersionV1, AggregateType: "workflow_execution"})
	_ = r.Register(Descriptor{Type: TypeWorkflowStepStarted, Version: VersionV1, AggregateType: "workflow_execution"})
	_ = r.Register(Descriptor{Type: TypeWorkflowStepSucceeded, Version: VersionV1, AggregateType: "workflow_execution"})
	_ = r.Register(Descriptor{Type: TypeWorkflowStepFailed, Version: VersionV1, AggregateType: "workflow_execution"})
	_ = r.Register(Descriptor{Type: TypeSecurityInvalidToken, Version: VersionV1, AggregateType: "security"})
	_ = r.Register(Descriptor{Type: TypeSecurityRateLimited, Version: VersionV1, AggregateType: "security"})
	_ = r.Register(Descriptor{Type: TypeSecurityAuthzDenied, Version: VersionV1, AggregateType: "security"})
	_ = r.Register(Descriptor{Type: TypeSecuritySuspiciousRequest, Version: VersionV1, AggregateType: "security"})
	return r
}

const VersionV1 = "1.0"

const (
	TypeUserRegistered      = "auth.user.registered"
	TypeUserLoggedIn        = "auth.user.logged_in"
	TypeUserLoginFailed     = "auth.user.login_failed"
	TypeAPIKeyCreated       = "auth.api_key.created"
	TypeAPIKeyRevoked       = "auth.api_key.revoked"
	TypeAPIKeyRotated       = "auth.api_key.rotated"
	TypeAPIKeyAuthenticationFailed = "auth.api_key.authentication_failed"
	TypeOrganizationCreated      = "organizations.organization.created"
	TypeOrganizationUpdated      = "organizations.organization.updated"
	TypeOrganizationDeleted      = "organizations.organization.deleted"
	TypeMembershipCreated        = "organizations.membership.created"
	TypeMembershipRoleChanged    = "organizations.membership.role_changed"
	TypeMembershipSuspended      = "organizations.membership.suspended"
	TypeMembershipRemoved        = "organizations.membership.removed"
	TypeMemberInvited            = "organizations.member.invited"
	TypeMemberJoined             = "organizations.member.joined"
	TypeInvitationCreated        = "organizations.invitation.created"
	TypeInvitationAccepted       = "organizations.invitation.accepted"
	TypeInvitationRevoked        = "organizations.invitation.revoked"
	TypeWorkflowExecutionStarted   = "workflow.execution.started"
	TypeWorkflowExecutionSucceeded = "workflow.execution.succeeded"
	TypeWorkflowExecutionFailed    = "workflow.execution.failed"
	TypeWorkflowExecutionCancelled = "workflow.execution.cancelled"
	TypeWorkflowStepStarted        = "workflow.step.started"
	TypeWorkflowStepSucceeded      = "workflow.step.succeeded"
	TypeWorkflowStepFailed         = "workflow.step.failed"
	TypeSecurityInvalidToken       = "security.invalid_token"
	TypeSecurityRateLimited        = "security.rate_limited"
	TypeSecurityAuthzDenied        = "security.authz.denied"
	TypeSecuritySuspiciousRequest  = "security.suspicious_request"
)

// SystemOrganizationID is used when no tenant context exists yet.
const SystemOrganizationID = "org_system0000000001"
