package incident

import (
	"context"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// UpdateSeverityInput carries severity update data.
type UpdateSeverityInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	IncidentID     domain.IncidentID
	Severity       string
}

// UpdateSeverityUseCase updates incident severity.
type UpdateSeverityUseCase struct {
	incidents ports.IncidentRepository
	history   ports.HistoryRepository
	publisher ports.EventPublisher
	tx        ports.TransactionManager
	metrics   ports.WorkflowMetrics
}

// NewUpdateSeverityUseCase creates the use case.
func NewUpdateSeverityUseCase(
	incidents ports.IncidentRepository,
	history ports.HistoryRepository,
	publisher ports.EventPublisher,
	tx ports.TransactionManager,
	metrics ports.WorkflowMetrics,
) *UpdateSeverityUseCase {
	return &UpdateSeverityUseCase{incidents: incidents, history: history, publisher: publisher, tx: tx, metrics: metrics}
}

// Execute updates severity and emits escalation events when applicable.
func (uc *UpdateSeverityUseCase) Execute(ctx context.Context, input UpdateSeverityInput) (*domain.Incident, error) {
	if err := ensureWriteAccess(input.Principal); err != nil {
		return nil, err
	}

	severity, err := domain.ParseSeverity(input.Severity)
	if err != nil {
		return nil, err
	}

	incident, err := uc.incidents.FindByID(ctx, input.OrganizationID, input.IncidentID)
	if err != nil {
		return nil, err
	}

	prev := incident.Severity()
	if err := incident.UpdateSeverity(severity, input.Principal.UserID, nowUTC()); err != nil {
		return nil, err
	}

	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return persistIncident(txCtx, uc.incidents, uc.history, uc.publisher, incident, false)
	})
	if err != nil {
		return nil, err
	}

	if uc.metrics != nil && severity.IsHigherThan(prev) {
		uc.metrics.RecordTransition(prev.String(), severity.String())
	}
	return incident, nil
}
