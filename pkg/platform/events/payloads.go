package events

// UserRegisteredPayload is the v1 payload for auth.user.registered.
type UserRegisteredPayload struct {
	UserID string
	Email  string
}

// UserLoggedInPayload is the v1 payload for auth.user.logged_in.
type UserLoggedInPayload struct {
	UserID string
	Email  string
}

// UserLoginFailedPayload is the v1 payload for auth.user.login_failed.
type UserLoginFailedPayload struct {
	UserID string
	Reason string
}

// OrganizationCreatedPayload is the v1 payload for organizations.organization.created.
type OrganizationCreatedPayload struct {
	OrganizationID string
	Name           string
	Slug           string
}

// MemberInvitedPayload is the v1 payload for organizations.member.invited.
type MemberInvitedPayload struct {
	OrganizationID string
	UserID         string
	Role           string
}

// MemberJoinedPayload is the v1 payload for organizations.member.joined.
type MemberJoinedPayload struct {
	OrganizationID string
	UserID         string
	Role           string
}

// OrganizationUpdatedPayload is the v1 payload for organizations.organization.updated.
type OrganizationUpdatedPayload struct {
	OrganizationID string
	Name           string
}

// OrganizationDeletedPayload is the v1 payload for organizations.organization.deleted.
type OrganizationDeletedPayload struct {
	OrganizationID string
}

// MembershipCreatedPayload is the v1 payload for organizations.membership.created.
type MembershipCreatedPayload struct {
	OrganizationID string
	UserID         string
	Role           string
}

// MembershipRoleChangedPayload is the v1 payload for organizations.membership.role_changed.
type MembershipRoleChangedPayload struct {
	OrganizationID string
	UserID         string
	FromRole       string
	ToRole         string
}

// MembershipSuspendedPayload is the v1 payload for organizations.membership.suspended.
type MembershipSuspendedPayload struct {
	OrganizationID string
	UserID         string
}

// MembershipRemovedPayload is the v1 payload for organizations.membership.removed.
type MembershipRemovedPayload struct {
	OrganizationID string
	UserID         string
}

// InvitationCreatedPayload is the v1 payload for organizations.invitation.created.
type InvitationCreatedPayload struct {
	OrganizationID string
	Email          string
	Role           string
}

// InvitationAcceptedPayload is the v1 payload for organizations.invitation.accepted.
type InvitationAcceptedPayload struct {
	OrganizationID string
	UserID         string
	Role           string
}

// InvitationRevokedPayload is the v1 payload for organizations.invitation.revoked.
type InvitationRevokedPayload struct {
	OrganizationID string
	Email          string
}

// APIKeyCreatedPayload is the v1 payload for auth.api_key.created.
type APIKeyCreatedPayload struct {
	APIKeyID         string
	KeyID            string
	ServiceAccountID string
	OrganizationID   string
	Name             string
}

// APIKeyRevokedPayload is the v1 payload for auth.api_key.revoked.
type APIKeyRevokedPayload struct {
	APIKeyID         string
	ServiceAccountID string
	OrganizationID   string
}

// APIKeyRotatedPayload is the v1 payload for auth.api_key.rotated.
type APIKeyRotatedPayload struct {
	OldAPIKeyID      string
	NewAPIKeyID      string
	ServiceAccountID string
	OrganizationID   string
}

// APIKeyAuthenticationFailedPayload is the v1 payload for auth.api_key.authentication_failed.
type APIKeyAuthenticationFailedPayload struct {
	APIKeyID       string
	KeyPrefix      string
	OrganizationID string
	Reason         string
}

// SecurityInvalidTokenPayload is the v1 payload for security.invalid_token.
type SecurityInvalidTokenPayload struct {
	Reason    string
	IPAddress string
	UserAgent string
}

// SecurityRateLimitedPayload is the v1 payload for security.rate_limited.
type SecurityRateLimitedPayload struct {
	RouteClass string
	IPAddress  string
}

// SecurityAuthzDeniedPayload is the v1 payload for security.authz.denied.
type SecurityAuthzDeniedPayload struct {
	Resource   string
	Permission string
}

// SecuritySuspiciousRequestPayload is the v1 payload for security.suspicious_request.
type SecuritySuspiciousRequestPayload struct {
	Reason    string
	IPAddress string
}
