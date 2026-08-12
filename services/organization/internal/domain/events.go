package domain

import (
	"strings"
	"time"
)

// Event is a domain event emitted by organization aggregates.
type Event interface {
	EventName() string
	OccurredAt() time.Time
}

type baseEvent struct {
	name string
	at   time.Time
}

func (e baseEvent) EventName() string     { return e.name }
func (e baseEvent) OccurredAt() time.Time { return e.at }

// OrganizationCreated is emitted when an organization is created.
type OrganizationCreated struct {
	baseEvent
	OrganizationID OrganizationID
	Name           OrganizationName
	Slug           Slug
}

func NewOrganizationCreated(id OrganizationID, name OrganizationName, slug Slug, at time.Time) OrganizationCreated {
	return OrganizationCreated{
		baseEvent:      baseEvent{name: "organizations.organization.created", at: at.UTC()},
		OrganizationID: id,
		Name:           name,
		Slug:           slug,
	}
}

// OrganizationUpdated is emitted when an organization is updated.
type OrganizationUpdated struct {
	baseEvent
	OrganizationID OrganizationID
	Name           OrganizationName
}

func NewOrganizationUpdated(id OrganizationID, name OrganizationName, at time.Time) OrganizationUpdated {
	return OrganizationUpdated{
		baseEvent:      baseEvent{name: "organizations.organization.updated", at: at.UTC()},
		OrganizationID: id,
		Name:           name,
	}
}

// OrganizationDeleted is emitted when an organization is soft-deleted.
type OrganizationDeleted struct {
	baseEvent
	OrganizationID OrganizationID
}

func NewOrganizationDeleted(id OrganizationID, at time.Time) OrganizationDeleted {
	return OrganizationDeleted{
		baseEvent:      baseEvent{name: "organizations.organization.deleted", at: at.UTC()},
		OrganizationID: id,
	}
}

// MembershipCreated is emitted when a membership is created.
type MembershipCreated struct {
	baseEvent
	OrganizationID OrganizationID
	UserID         UserID
	Role           MemberRole
}

func NewMembershipCreated(orgID OrganizationID, userID UserID, role MemberRole, at time.Time) MembershipCreated {
	return MembershipCreated{
		baseEvent:      baseEvent{name: "organizations.membership.created", at: at.UTC()},
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
	}
}

// MembershipRoleChanged is emitted when a member role changes.
type MembershipRoleChanged struct {
	baseEvent
	OrganizationID OrganizationID
	UserID         UserID
	FromRole       MemberRole
	ToRole         MemberRole
}

func NewMembershipRoleChanged(orgID OrganizationID, userID UserID, from, to MemberRole, at time.Time) MembershipRoleChanged {
	return MembershipRoleChanged{
		baseEvent:      baseEvent{name: "organizations.membership.role_changed", at: at.UTC()},
		OrganizationID: orgID,
		UserID:         userID,
		FromRole:       from,
		ToRole:         to,
	}
}

// MembershipSuspended is emitted when a membership is suspended.
type MembershipSuspended struct {
	baseEvent
	OrganizationID OrganizationID
	UserID         UserID
}

func NewMembershipSuspended(orgID OrganizationID, userID UserID, at time.Time) MembershipSuspended {
	return MembershipSuspended{
		baseEvent:      baseEvent{name: "organizations.membership.suspended", at: at.UTC()},
		OrganizationID: orgID,
		UserID:         userID,
	}
}

// MemberRemoved is emitted when a member is removed.
type MembershipRemoved struct {
	baseEvent
	OrganizationID OrganizationID
	UserID         UserID
}

func NewMembershipRemoved(orgID OrganizationID, userID UserID, at time.Time) MembershipRemoved {
	return MembershipRemoved{
		baseEvent:      baseEvent{name: "organizations.membership.removed", at: at.UTC()},
		OrganizationID: orgID,
		UserID:         userID,
	}
}

// MemberInvited is emitted when a membership invitation is created.
type MemberInvited struct {
	baseEvent
	OrganizationID OrganizationID
	Email          Email
	Role           MemberRole
}

func NewMemberInvited(orgID OrganizationID, email Email, role MemberRole, at time.Time) MemberInvited {
	return MemberInvited{
		baseEvent:      baseEvent{name: "organizations.member.invited", at: at.UTC()},
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
	}
}

// InvitationCreated is emitted when an invitation aggregate is created.
type InvitationCreated struct {
	baseEvent
	OrganizationID OrganizationID
	Email          Email
	Role           MemberRole
	InvitedBy      UserID
}

func NewInvitationCreated(orgID OrganizationID, email Email, role MemberRole, invitedBy UserID, at time.Time) InvitationCreated {
	return InvitationCreated{
		baseEvent:      baseEvent{name: "organizations.invitation.created", at: at.UTC()},
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		InvitedBy:      invitedBy,
	}
}

// InvitationAccepted is emitted when an invitation is accepted.
type InvitationAccepted struct {
	baseEvent
	OrganizationID OrganizationID
	UserID         UserID
	Role           MemberRole
}

func NewInvitationAccepted(orgID OrganizationID, userID UserID, role MemberRole, at time.Time) InvitationAccepted {
	return InvitationAccepted{
		baseEvent:      baseEvent{name: "organizations.invitation.accepted", at: at.UTC()},
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
	}
}

// InvitationRevoked is emitted when an invitation is revoked.
type InvitationRevoked struct {
	baseEvent
	OrganizationID OrganizationID
	Email          Email
}

func NewInvitationRevoked(orgID OrganizationID, email Email, at time.Time) InvitationRevoked {
	return InvitationRevoked{
		baseEvent:      baseEvent{name: "organizations.invitation.revoked", at: at.UTC()},
		OrganizationID: orgID,
		Email:          email,
	}
}

// MemberJoined is emitted when a user accepts an invitation.
type MemberJoined struct {
	baseEvent
	OrganizationID OrganizationID
	UserID         UserID
	Role           MemberRole
}

func NewMemberJoined(orgID OrganizationID, userID UserID, role MemberRole, at time.Time) MemberJoined {
	return MemberJoined{
		baseEvent:      baseEvent{name: "organizations.member.joined", at: at.UTC()},
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
	}
}

// NormalizeEmail trims and lowercases email addresses.
func NormalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}
