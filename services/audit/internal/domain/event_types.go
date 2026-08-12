package domain

import "errors"

var (
	ErrInvalidAuditEventID = errors.New("invalid audit event id")
	ErrAuditEventNotFound  = errors.New("audit event not found")
	ErrForbidden           = errors.New("forbidden")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrInvalidFilter       = errors.New("invalid audit filter")
	ErrInvalidCursor       = errors.New("invalid cursor")
)

// Canonical audit event type constants.
const (
	EventAuthLoginSucceeded = "auth.login.succeeded"
	EventAuthLoginFailed    = "auth.login.failed"
	EventAuthLogout         = "auth.logout"
	EventAuthTokenRevoked   = "auth.token.revoked"

	EventAuthzDenied             = "authz.denied"
	EventAuthzPermissionChanged  = "authz.permission_changed"
	EventAuthzRoleChanged        = "authz.role_changed"

	EventOrganizationCreated = "organization.created"
	EventOrganizationUpdated = "organization.updated"
	EventOrganizationDeleted = "organization.deleted"

	EventMembershipCreated    = "membership.created"
	EventMembershipRemoved    = "membership.removed"
	EventMembershipRoleChanged = "membership.role_changed"

	EventInvitationCreated  = "invitation.created"
	EventInvitationAccepted = "invitation.accepted"
	EventInvitationRevoked  = "invitation.revoked"
	EventInvitationExpired  = "invitation.expired"

	EventAPIKeyCreated = "api_key.created"
	EventAPIKeyRevoked = "api_key.revoked"
	EventAPIKeyRotated = "api_key.rotated"
	EventAPIKeyAuthenticationFailed = "api_key.authentication_failed"

	EventWorkflowCreated            = "workflow.created"
	EventWorkflowUpdated            = "workflow.updated"
	EventWorkflowExecutionStarted   = "workflow.execution.started"
	EventWorkflowExecutionSucceeded = "workflow.execution.succeeded"
	EventWorkflowExecutionFailed    = "workflow.execution.failed"
	EventWorkflowExecutionCancelled = "workflow.execution.cancelled"

	EventSecurityRateLimited       = "security.rate_limited"
	EventSecurityInvalidToken      = "security.invalid_token"
	EventSecuritySuspiciousRequest = "security.suspicious_request"
)

// SystemOrganizationID is used for platform-scoped auth/security events.
const SystemOrganizationID = "org_system0000000001"
