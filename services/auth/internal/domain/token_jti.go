package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// JTI is a unique refresh token identifier.
type JTI struct {
	value string
}

// NewJTI generates a new token identifier.
func NewJTI() (JTI, error) {
	return ParseJTI(uuid.NewString())
}

// ParseJTI validates a JTI string.
func ParseJTI(raw string) (JTI, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return JTI{}, fmt.Errorf("%w: empty jti", ErrInvalidRefreshToken)
	}
	if _, err := uuid.Parse(raw); err != nil {
		return JTI{}, fmt.Errorf("%w: invalid jti", ErrInvalidRefreshToken)
	}
	return JTI{value: raw}, nil
}

// String returns the JTI value.
func (j JTI) String() string {
	return j.value
}

// IsZero reports whether the JTI is unset.
func (j JTI) IsZero() bool {
	return j.value == ""
}

// Equal reports whether two JTIs match.
func (j JTI) Equal(other JTI) bool {
	return j.value == other.value
}

// FamilyID groups refresh tokens in a rotation chain.
type FamilyID struct {
	value string
}

// NewFamilyID generates a new token family identifier.
func NewFamilyID() (FamilyID, error) {
	return ParseFamilyID(uuid.NewString())
}

// ParseFamilyID validates a family identifier.
func ParseFamilyID(raw string) (FamilyID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return FamilyID{}, fmt.Errorf("%w: empty family id", ErrInvalidRefreshToken)
	}
	if _, err := uuid.Parse(raw); err != nil {
		return FamilyID{}, fmt.Errorf("%w: invalid family id", ErrInvalidRefreshToken)
	}
	return FamilyID{value: raw}, nil
}

// String returns the family identifier.
func (f FamilyID) String() string {
	return f.value
}

// IsZero reports whether the family identifier is unset.
func (f FamilyID) IsZero() bool {
	return f.value == ""
}

// Equal reports whether two family identifiers match.
func (f FamilyID) Equal(other FamilyID) bool {
	return f.value == other.value
}
