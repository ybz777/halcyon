package domain

import (
	"fmt"
	"strings"

	"github.com/halcyon-io/halcyon/pkg/platform/ids"
)

const organizationIDPrefix = "org_"

// OrganizationID is an immutable organization identifier.
type OrganizationID struct {
	value string
}

// NewOrganizationID generates a new organization ID.
func NewOrganizationID() (OrganizationID, error) {
	raw, err := ids.Generate(organizationIDPrefix)
	if err != nil {
		return OrganizationID{}, fmt.Errorf("generate organization id: %w", err)
	}
	return ParseOrganizationID(raw)
}

// ParseOrganizationID validates a persisted organization ID.
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

func (id OrganizationID) String() string  { return id.value }
func (id OrganizationID) IsZero() bool    { return id.value == "" }
func (id OrganizationID) Equal(other OrganizationID) bool {
	return id.value == other.value
}
