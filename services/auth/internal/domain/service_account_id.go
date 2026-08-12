package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const serviceAccountIDPrefix = "svc_"
const organizationIDPrefix = "org_"

// ServiceAccountID identifies a service account aggregate.
type ServiceAccountID struct {
	value string
}

// NewServiceAccountID generates a new service account identifier.
func NewServiceAccountID() (ServiceAccountID, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ServiceAccountID{}, fmt.Errorf("generate service account id: %w", err)
	}
	return ParseServiceAccountID(serviceAccountIDPrefix + base64.RawURLEncoding.EncodeToString(b))
}

// ParseServiceAccountID validates a service account identifier.
func ParseServiceAccountID(raw string) (ServiceAccountID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, serviceAccountIDPrefix) {
		return ServiceAccountID{}, fmt.Errorf("%w: missing %q prefix", ErrInvalidServiceAccount, serviceAccountIDPrefix)
	}
	if len(strings.TrimPrefix(raw, serviceAccountIDPrefix)) < 16 {
		return ServiceAccountID{}, fmt.Errorf("%w: id suffix too short", ErrInvalidServiceAccount)
	}
	return ServiceAccountID{value: raw}, nil
}

func (id ServiceAccountID) String() string { return id.value }
func (id ServiceAccountID) IsZero() bool  { return id.value == "" }

// OrganizationID identifies an organization tenant.
type OrganizationID struct {
	value string
}

// ParseOrganizationID validates an organization identifier.
func ParseOrganizationID(raw string) (OrganizationID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, organizationIDPrefix) {
		return OrganizationID{}, fmt.Errorf("%w: missing %q prefix", ErrInvalidOrganizationID, organizationIDPrefix)
	}
	if len(strings.TrimPrefix(raw, organizationIDPrefix)) < 8 {
		return OrganizationID{}, fmt.Errorf("%w: id suffix too short", ErrInvalidOrganizationID)
	}
	return OrganizationID{value: raw}, nil
}

func (id OrganizationID) String() string { return id.value }
func (id OrganizationID) IsZero() bool   { return id.value == "" }
func (id OrganizationID) Equal(other OrganizationID) bool { return id.value == other.value }
