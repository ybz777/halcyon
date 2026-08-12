package domain

import (
	"strings"
	"time"
)

// MembershipStatus represents membership lifecycle state.
type MembershipStatus string

const (
	MembershipStatusActive    MembershipStatus = "active"
	MembershipStatusSuspended MembershipStatus = "suspended"
	MembershipStatusRemoved   MembershipStatus = "removed"
)

func isValidMembershipStatus(status MembershipStatus) bool {
	switch status {
	case MembershipStatusActive, MembershipStatusSuspended, MembershipStatusRemoved:
		return true
	default:
		return false
	}
}

// Membership links a user to an organization with a role.
type Membership struct {
	id             MembershipID
	organizationID OrganizationID
	userID         UserID
	role           MemberRole
	status         MembershipStatus
	joinedAt       time.Time
	updatedAt      time.Time
	events         []Event
}

// NewMembership creates an active membership aggregate.
func NewMembership(
	id MembershipID,
	organizationID OrganizationID,
	userID UserID,
	role MemberRole,
	joinedAt time.Time,
) (*Membership, error) {
	if id.String() == "" || organizationID.IsZero() || userID.IsZero() || joinedAt.IsZero() {
		return nil, ErrInvalidTimestamp
	}
	now := joinedAt.UTC()
	m := &Membership{
		id:             id,
		organizationID: organizationID,
		userID:         userID,
		role:           role,
		status:         MembershipStatusActive,
		joinedAt:       now,
		updatedAt:      now,
	}
	m.record(NewMembershipCreated(organizationID, userID, role, now))
	return m, nil
}

// ReconstituteMembership rebuilds a membership from persistence.
func ReconstituteMembership(
	id MembershipID,
	organizationID OrganizationID,
	userID UserID,
	role MemberRole,
	status MembershipStatus,
	joinedAt, updatedAt time.Time,
) (*Membership, error) {
	if id.String() == "" || organizationID.IsZero() || userID.IsZero() || joinedAt.IsZero() {
		return nil, ErrInvalidTimestamp
	}
	if !isValidMembershipStatus(status) {
		return nil, ErrInvalidMembershipStatus
	}
	return &Membership{
		id:             id,
		organizationID: organizationID,
		userID:         userID,
		role:           role,
		status:         status,
		joinedAt:       joinedAt.UTC(),
		updatedAt:      updatedAt.UTC(),
	}, nil
}

// ChangeRole updates the member role.
func (m *Membership) ChangeRole(newRole MemberRole, now time.Time) error {
	if m.status != MembershipStatusActive {
		return ErrMembershipSuspended
	}
	if m.role == newRole {
		return nil
	}
	oldRole := m.role
	m.role = newRole
	m.updatedAt = now.UTC()
	m.record(NewMembershipRoleChanged(m.organizationID, m.userID, oldRole, newRole, now.UTC()))
	return nil
}

// Suspend marks the membership suspended.
func (m *Membership) Suspend(now time.Time) error {
	if m.status == MembershipStatusRemoved {
		return ErrMembershipRemoved
	}
	if m.status == MembershipStatusSuspended {
		return nil
	}
	m.status = MembershipStatusSuspended
	m.updatedAt = now.UTC()
	m.record(NewMembershipSuspended(m.organizationID, m.userID, now.UTC()))
	return nil
}

// Remove soft-removes the membership.
func (m *Membership) Remove(now time.Time) error {
	if m.status == MembershipStatusRemoved {
		return nil
	}
	m.status = MembershipStatusRemoved
	m.updatedAt = now.UTC()
	m.record(NewMembershipRemoved(m.organizationID, m.userID, now.UTC()))
	return nil
}

// EnsureActive returns an error when membership cannot perform operations.
func (m *Membership) EnsureActive() error {
	switch m.status {
	case MembershipStatusActive:
		return nil
	case MembershipStatusSuspended:
		return ErrMembershipSuspended
	case MembershipStatusRemoved:
		return ErrMembershipRemoved
	default:
		return ErrMembershipSuspended
	}
}

func (m *Membership) record(event Event) { m.events = append(m.events, event) }

func (m *Membership) ID() MembershipID               { return m.id }
func (m *Membership) OrganizationID() OrganizationID { return m.organizationID }
func (m *Membership) UserID() UserID                   { return m.userID }
func (m *Membership) Role() MemberRole                 { return m.role }
func (m *Membership) Status() MembershipStatus         { return m.status }
func (m *Membership) JoinedAt() time.Time              { return m.joinedAt }
func (m *Membership) UpdatedAt() time.Time             { return m.updatedAt }
func (m *Membership) UncommittedEvents() []Event {
	if len(m.events) == 0 {
		return nil
	}
	out := make([]Event, len(m.events))
	copy(out, m.events)
	return out
}
func (m *Membership) ClearUncommittedEvents() { m.events = nil }

// InvitationStatus tracks invitation lifecycle.
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusExpired  InvitationStatus = "expired"
	InvitationStatusRevoked  InvitationStatus = "revoked"
)

// Email is a normalized invitation email address.
type Email struct {
	value string
}

// ParseEmail normalizes and validates an email address.
func ParseEmail(raw string) (Email, error) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" || !strings.Contains(raw, "@") {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: raw}, nil
}

func (e Email) String() string { return e.value }
func (e Email) IsZero() bool   { return e.value == "" }
func (e Email) Equal(other Email) bool { return e.value == other.value }

func normalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// Invitation represents a pending organization membership invite.
type Invitation struct {
	id             InvitationID
	organizationID OrganizationID
	email          Email
	userID         UserID
	role           MemberRole
	tokenHash      string
	invitedBy      UserID
	status         InvitationStatus
	expiresAt      time.Time
	acceptedAt     *time.Time
	revokedAt      *time.Time
	createdAt      time.Time
	events         []Event
}

// NewInvitation creates a pending email invitation.
func NewInvitation(
	id InvitationID,
	organizationID OrganizationID,
	email Email,
	role MemberRole,
	tokenHash string,
	invitedBy UserID,
	expiresAt, createdAt time.Time,
) (*Invitation, error) {
	if id.String() == "" || organizationID.IsZero() || email.IsZero() || tokenHash == "" || invitedBy.IsZero() {
		return nil, ErrInvalidInvitationID
	}
	now := createdAt.UTC()
	inv := &Invitation{
		id:             id,
		organizationID: organizationID,
		email:          email,
		role:           role,
		tokenHash:      tokenHash,
		invitedBy:      invitedBy,
		status:         InvitationStatusPending,
		expiresAt:      expiresAt.UTC(),
		createdAt:      now,
	}
	inv.record(NewInvitationCreated(organizationID, email, role, invitedBy, now))
	return inv, nil
}

// ReconstituteInvitation rebuilds an invitation from persistence.
func ReconstituteInvitation(
	id InvitationID,
	organizationID OrganizationID,
	email Email,
	userID UserID,
	role MemberRole,
	tokenHash string,
	invitedBy UserID,
	status InvitationStatus,
	expiresAt time.Time,
	acceptedAt, revokedAt *time.Time,
	createdAt time.Time,
) (*Invitation, error) {
	if id.String() == "" || organizationID.IsZero() || tokenHash == "" {
		return nil, ErrInvalidInvitationID
	}
	return &Invitation{
		id:             id,
		organizationID: organizationID,
		email:          email,
		userID:         userID,
		role:           role,
		tokenHash:      tokenHash,
		invitedBy:      invitedBy,
		status:         status,
		expiresAt:      expiresAt.UTC(),
		acceptedAt:     acceptedAt,
		revokedAt:      revokedAt,
		createdAt:      createdAt.UTC(),
	}, nil
}

// Accept marks the invitation accepted for a user.
func (i *Invitation) Accept(userID UserID, now time.Time) error {
	if i.status == InvitationStatusAccepted {
		if !i.userID.IsZero() && i.userID.Equal(userID) {
			return nil
		}
		return ErrInvitationUsed
	}
	if i.status == InvitationStatusRevoked {
		return ErrInvitationRevoked
	}
	if i.status == InvitationStatusExpired || now.UTC().After(i.expiresAt) {
		i.status = InvitationStatusExpired
		return ErrInvitationExpired
	}
	if i.status != InvitationStatusPending {
		return ErrInvitationUsed
	}
	accepted := now.UTC()
	i.status = InvitationStatusAccepted
	i.userID = userID
	i.acceptedAt = &accepted
	i.record(NewInvitationAccepted(i.organizationID, userID, i.role, accepted))
	return nil
}

// Revoke marks the invitation revoked.
func (i *Invitation) Revoke(now time.Time) error {
	if i.status == InvitationStatusAccepted {
		return ErrInvitationUsed
	}
	if i.status == InvitationStatusRevoked {
		return nil
	}
	revoked := now.UTC()
	i.status = InvitationStatusRevoked
	i.revokedAt = &revoked
	i.record(NewInvitationRevoked(i.organizationID, i.email, revoked))
	return nil
}

func (i *Invitation) record(event Event) { i.events = append(i.events, event) }

func (i *Invitation) ID() InvitationID             { return i.id }
func (i *Invitation) OrganizationID() OrganizationID { return i.organizationID }
func (i *Invitation) Email() Email                   { return i.email }
func (i *Invitation) UserID() UserID                 { return i.userID }
func (i *Invitation) Role() MemberRole               { return i.role }
func (i *Invitation) TokenHash() string              { return i.tokenHash }
func (i *Invitation) InvitedBy() UserID              { return i.invitedBy }
func (i *Invitation) Status() InvitationStatus       { return i.status }
func (i *Invitation) ExpiresAt() time.Time           { return i.expiresAt }
func (i *Invitation) AcceptedAt() *time.Time         { return i.acceptedAt }
func (i *Invitation) RevokedAt() *time.Time          { return i.revokedAt }
func (i *Invitation) CreatedAt() time.Time           { return i.createdAt }
func (i *Invitation) UncommittedEvents() []Event {
	if len(i.events) == 0 {
		return nil
	}
	out := make([]Event, len(i.events))
	copy(out, i.events)
	return out
}
func (i *Invitation) ClearUncommittedEvents() { i.events = nil }
