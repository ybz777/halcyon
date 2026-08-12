package domain

import (
	"fmt"
	"strings"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const userIDPrefix = "usr_"

// UserID references an auth service user identifier.
type UserID struct {
	value string
}

// ParseUserID validates a user ID from the auth bounded context.
func ParseUserID(raw string) (UserID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, userIDPrefix) {
		return UserID{}, fmt.Errorf("%w: missing %q prefix", ErrInvalidUserID, userIDPrefix)
	}
	suffix := strings.TrimPrefix(raw, userIDPrefix)
	if len(suffix) < 16 {
		return UserID{}, fmt.Errorf("%w: id suffix too short", ErrInvalidUserID)
	}
	return UserID{value: raw}, nil
}

func (id UserID) String() string { return id.value }
func (id UserID) IsZero() bool   { return id.value == "" }
func (id UserID) Equal(other UserID) bool {
	return id.value == other.value
}

const membershipIDPrefix = "mbr_"

// MembershipID identifies a membership row.
type MembershipID struct {
	value string
}

// NewMembershipID generates a membership ID.
func NewMembershipID() (MembershipID, error) {
	raw, err := ids.Generate(membershipIDPrefix)
	if err != nil {
		return MembershipID{}, err
	}
	return ParseMembershipID(raw)
}

// ParseMembershipID validates a membership ID.
func ParseMembershipID(raw string) (MembershipID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, membershipIDPrefix) {
		return MembershipID{}, fmt.Errorf("%w: missing prefix", ErrInvalidMembershipID)
	}
	return MembershipID{value: raw}, nil
}

func (id MembershipID) String() string { return id.value }

const invitationIDPrefix = "inv_"

// InvitationID identifies an invitation.
type InvitationID struct {
	value string
}

// NewInvitationID generates an invitation ID.
func NewInvitationID() (InvitationID, error) {
	raw, err := ids.Generate(invitationIDPrefix)
	if err != nil {
		return InvitationID{}, err
	}
	return ParseInvitationID(raw)
}

// ParseInvitationID validates an invitation ID.
func ParseInvitationID(raw string) (InvitationID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, invitationIDPrefix) {
		return InvitationID{}, fmt.Errorf("%w: missing prefix", ErrInvalidInvitationID)
	}
	return InvitationID{value: raw}, nil
}

func (id InvitationID) String() string { return id.value }
