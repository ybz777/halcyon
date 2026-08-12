package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const userIDPrefix = "usr_"

// UserID is an immutable identifier for a User aggregate.
type UserID struct {
	value string
}

// ParseUserID validates and constructs a UserID from a persisted value.
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

// NewUserID generates a new unique UserID.
func NewUserID() (UserID, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return UserID{}, fmt.Errorf("generate user id: %w", err)
	}

	return UserID{value: userIDPrefix + base64.RawURLEncoding.EncodeToString(b)}, nil
}

// String returns the canonical string representation.
func (id UserID) String() string {
	return id.value
}

// IsZero reports whether the UserID is unset.
func (id UserID) IsZero() bool {
	return id.value == ""
}

// Equal reports whether two UserIDs refer to the same value.
func (id UserID) Equal(other UserID) bool {
	return id.value == other.value
}
