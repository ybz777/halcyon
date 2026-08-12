package postgres

import (
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func toDomainIncident(
	externalID, organizationID, title, description, severity, priority, status, source string,
	createdAt, updatedAt time.Time,
	resolvedAt *time.Time,
	version int,
) (*domain.Incident, error) {
	id, err := domain.ParseIncidentID(externalID)
	if err != nil {
		return nil, err
	}
	orgID, err := domain.ParseOrganizationID(organizationID)
	if err != nil {
		return nil, err
	}
	titleVal, err := domain.ParseTitle(title)
	if err != nil {
		return nil, err
	}
	desc, err := domain.ParseDescription(description)
	if err != nil {
		return nil, err
	}
	sev, err := domain.ParseSeverity(severity)
	if err != nil {
		return nil, err
	}
	priorityVal, err := domain.ParsePriority(priority)
	if err != nil {
		return nil, err
	}
	statusVal, err := domain.ParseStatus(status)
	if err != nil {
		return nil, err
	}
	sourceVal, err := domain.ParseSource(source)
	if err != nil {
		return nil, err
	}
	return domain.ReconstituteIncident(
		id, orgID, titleVal, desc, sev, priorityVal, statusVal, sourceVal,
		createdAt, updatedAt, resolvedAt, version,
	)
}

func wrapError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}
