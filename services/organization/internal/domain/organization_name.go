package domain

import (
	"fmt"
	"strings"
)

// OrganizationName is a validated organization display name.
type OrganizationName struct {
	value string
}

// ParseOrganizationName validates an organization name.
func ParseOrganizationName(raw string) (OrganizationName, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return OrganizationName{}, fmt.Errorf("%w: empty", ErrInvalidOrganizationName)
	}
	if len(raw) > 120 {
		return OrganizationName{}, fmt.Errorf("%w: too long", ErrInvalidOrganizationName)
	}
	return OrganizationName{value: raw}, nil
}

func (n OrganizationName) String() string { return n.value }
func (n OrganizationName) IsZero() bool   { return n.value == "" }
