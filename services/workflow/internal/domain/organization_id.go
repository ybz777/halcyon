package domain

import (
	"fmt"
	"strings"
)

const organizationIDPrefix = "org_"

// OrganizationID scopes incidents to a tenant.
type OrganizationID struct {
	value string
}

// ParseOrganizationID validates an organization ID.
func ParseOrganizationID(raw string) (OrganizationID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, organizationIDPrefix) {
		return OrganizationID{}, fmt.Errorf("%w: missing %q prefix", ErrInvalidOrganizationID, organizationIDPrefix)
	}
	suffix := strings.TrimPrefix(raw, organizationIDPrefix)
	if len(suffix) < 16 {
		return OrganizationID{}, fmt.Errorf("%w: id suffix too short", ErrInvalidOrganizationID)
	}
	return OrganizationID{value: raw}, nil
}

func (id OrganizationID) String() string { return id.value }
func (id OrganizationID) IsZero() bool   { return id.value == "" }
