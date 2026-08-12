package domain

import (
	"fmt"
	"strings"
)

const userIDPrefix = "usr_"

// UserID identifies an authenticated actor.
type UserID struct {
	value string
}

// ParseUserID validates a user ID from JWT claims.
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
