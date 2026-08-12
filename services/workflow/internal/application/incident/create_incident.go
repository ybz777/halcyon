package incident

import (
	"context"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// CreateIncidentInput carries incident creation data.
type CreateIncidentInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	Title          string
	Description    string
	Severity       string
	Priority       string
	Source         string
}

// CreateIncidentUseCase opens a new incident.
type CreateIncidentUseCase struct {
	incidents ports.IncidentRepository
	history   ports.HistoryRepository
	publisher ports.EventPublisher
	tx        ports.TransactionManager
	metrics   ports.WorkflowMetrics
}

// NewCreateIncidentUseCase creates the use case.
func NewCreateIncidentUseCase(
	incidents ports.IncidentRepository,
	history ports.HistoryRepository,
	publisher ports.EventPublisher,
	tx ports.TransactionManager,
	metrics ports.WorkflowMetrics,
) *CreateIncidentUseCase {
	return &CreateIncidentUseCase{incidents: incidents, history: history, publisher: publisher, tx: tx, metrics: metrics}
}

// Execute creates an incident for the organization.
func (uc *CreateIncidentUseCase) Execute(ctx context.Context, input CreateIncidentInput) (*domain.Incident, error) {
	if err := ensureWriteAccess(input.Principal); err != nil {
		return nil, err
	}

	title, err := domain.ParseTitle(input.Title)
	if err != nil {
		return nil, err
	}
	description, err := domain.ParseDescription(input.Description)
	if err != nil {
		return nil, err
	}
	severity, err := domain.ParseSeverity(input.Severity)
	if err != nil {
		return nil, err
	}
	priority, err := domain.ParsePriority(input.Priority)
	if err != nil {
		return nil, err
	}
	source, err := domain.ParseSource(input.Source)
	if err != nil {
		return nil, err
	}

	incidentID, err := domain.NewIncidentID()
	if err != nil {
		return nil, err
	}

	now := nowUTC()
	incident, err := domain.CreateIncident(
		incidentID,
		input.OrganizationID,
		title,
		description,
		severity,
		priority,
		source,
		input.Principal.UserID,
		now,
	)
	if err != nil {
		return nil, err
	}

	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return persistIncident(txCtx, uc.incidents, uc.history, uc.publisher, incident, true)
	})
	if err != nil {
		return nil, err
	}

	if uc.metrics != nil {
		uc.metrics.RecordIncidentCreated()
	}
	return incident, nil
}
