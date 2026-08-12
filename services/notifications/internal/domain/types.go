package domain

// Channel identifies a notification delivery channel.
type Channel string

const (
	ChannelInApp Channel = "IN_APP"
	ChannelEmail Channel = "EMAIL"
)

// NotificationType is the canonical notification category.
type NotificationType string

const (
	TypeWorkflowExecutionCompleted NotificationType = "workflow.execution.completed"
	TypeWorkflowExecutionFailed    NotificationType = "workflow.execution.failed"
	TypeOrganizationMemberAdded    NotificationType = "organization.member_added"
	TypeOrganizationMemberRemoved  NotificationType = "organization.member_removed"
	TypeOrganizationRoleChanged    NotificationType = "organization.role_changed"
	TypeSecurityLoginAlert         NotificationType = "security.login_alert"
	TypeSecurityAPIKeyCreated      NotificationType = "security.api_key_created"
	TypeSecurityAPIKeyRevoked      NotificationType = "security.api_key_revoked"
	TypeIntegrationWebhookFailed   NotificationType = "integration.webhook_failed"
	TypeSystemAnnouncement         NotificationType = "system.announcement"
)

// EmailDeliveryStatus tracks outbound email state.
type EmailDeliveryStatus string

const (
	EmailPending    EmailDeliveryStatus = "PENDING"
	EmailDelivering EmailDeliveryStatus = "DELIVERING"
	EmailSent       EmailDeliveryStatus = "SENT"
	EmailFailed     EmailDeliveryStatus = "FAILED"
	EmailRetryWait  EmailDeliveryStatus = "RETRY_WAIT"
)

// Security notification types cannot be disabled by users.
func IsMandatorySecurityType(t NotificationType) bool {
	switch t {
	case TypeSecurityLoginAlert, TypeSecurityAPIKeyCreated, TypeSecurityAPIKeyRevoked:
		return true
	default:
		return false
	}
}

const (
	MaxPageSize              = 100
	DefaultPageSize          = 50
	MaxNotificationsPerEvent = 50
	MaxEmailPerOrgPerHour    = 500
	RetentionDaysDefault     = 90
)
