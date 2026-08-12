package domain

// DefaultPreference returns whether a channel is enabled by default for a notification type.
func DefaultPreference(notificationType NotificationType, channel Channel) bool {
	if IsMandatorySecurityType(notificationType) {
		return true
	}
	switch notificationType {
	case TypeWorkflowExecutionFailed, TypeIntegrationWebhookFailed:
		return channel == ChannelInApp || channel == ChannelEmail
	case TypeWorkflowExecutionCompleted, TypeOrganizationMemberAdded,
		TypeOrganizationRoleChanged, TypeOrganizationMemberRemoved:
		return channel == ChannelInApp
	case TypeSystemAnnouncement:
		return channel == ChannelInApp
	default:
		return channel == ChannelInApp
	}
}

// IsChannelEnabled resolves explicit preference or default.
func IsChannelEnabled(notificationType NotificationType, channel Channel, explicit map[Channel]bool) bool {
	if IsMandatorySecurityType(notificationType) && channel == ChannelEmail {
		return true
	}
	if val, ok := explicit[channel]; ok {
		return val
	}
	return DefaultPreference(notificationType, channel)
}
