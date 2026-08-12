package domain

import (
	"fmt"
	"strings"
	"time"
)

// ServiceAccount is the service account aggregate root.
type ServiceAccount struct {
	id             ServiceAccountID
	organizationID OrganizationID
	name           string
	description    string
	status         ServiceAccountStatus
	createdAt      time.Time
}

// NewServiceAccount creates an active service account.
func NewServiceAccount(
	id ServiceAccountID,
	organizationID OrganizationID,
	name, description string,
	createdAt time.Time,
) (*ServiceAccount, error) {
	if id.IsZero() || organizationID.IsZero() || strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: incomplete service account", ErrInvalidServiceAccount)
	}
	return &ServiceAccount{
		id:             id,
		organizationID: organizationID,
		name:           strings.TrimSpace(name),
		description:    strings.TrimSpace(description),
		status:         ServiceAccountStatusActive,
		createdAt:      createdAt.UTC(),
	}, nil
}

// RehydrateServiceAccount restores a persisted service account.
func RehydrateServiceAccount(
	id ServiceAccountID,
	organizationID OrganizationID,
	name, description string,
	status ServiceAccountStatus,
	createdAt time.Time,
) *ServiceAccount {
	return &ServiceAccount{
		id:             id,
		organizationID: organizationID,
		name:           name,
		description:    description,
		status:         status,
		createdAt:      createdAt.UTC(),
	}
}

func (s *ServiceAccount) ID() ServiceAccountID           { return s.id }
func (s *ServiceAccount) OrganizationID() OrganizationID { return s.organizationID }
func (s *ServiceAccount) Name() string                   { return s.name }
func (s *ServiceAccount) Description() string            { return s.description }
func (s *ServiceAccount) Status() ServiceAccountStatus   { return s.status }
func (s *ServiceAccount) CreatedAt() time.Time           { return s.createdAt }

// EnsureActive validates the service account can authenticate API keys.
func (s *ServiceAccount) EnsureActive() error {
	if s.status != ServiceAccountStatusActive {
		return ErrServiceAccountDisabled
	}
	return nil
}
