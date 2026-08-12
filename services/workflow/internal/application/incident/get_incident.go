package incident

import (
	"context"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// GetIncidentInput identifies an incident to load.
type GetIncidentInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	IncidentID     domain.IncidentID
}

// GetIncidentUseCase loads a single incident.
type GetIncidentUseCase struct {
	incidents ports.IncidentRepository
}

// NewGetIncidentUseCase creates the use case.
func NewGetIncidentUseCase(incidents ports.IncidentRepository) *GetIncidentUseCase {
	return &GetIncidentUseCase{incidents: incidents}
}

// Execute returns an incident scoped to the organization.
func (uc *GetIncidentUseCase) Execute(ctx context.Context, input GetIncidentInput) (*domain.Incident, error) {
	if err := ensureReadAccess(input.Principal); err != nil {
		return nil, err
	}
	return uc.incidents.FindByID(ctx, input.OrganizationID, input.IncidentID)
}
