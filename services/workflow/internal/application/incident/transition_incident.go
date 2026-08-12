package incident

import (
	"context"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// TransitionIncidentInput carries a workflow transition request.
type TransitionIncidentInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	IncidentID     domain.IncidentID
	Status         string
}

// TransitionIncidentUseCase transitions an incident status.
type TransitionIncidentUseCase struct {
	incidents ports.IncidentRepository
	history   ports.HistoryRepository
	publisher ports.EventPublisher
	tx        ports.TransactionManager
	metrics   ports.WorkflowMetrics
}

// NewTransitionIncidentUseCase creates the use case.
func NewTransitionIncidentUseCase(
	incidents ports.IncidentRepository,
	history ports.HistoryRepository,
	publisher ports.EventPublisher,
	tx ports.TransactionManager,
	metrics ports.WorkflowMetrics,
) *TransitionIncidentUseCase {
	return &TransitionIncidentUseCase{incidents: incidents, history: history, publisher: publisher, tx: tx, metrics: metrics}
}

// Execute applies a valid workflow transition.
func (uc *TransitionIncidentUseCase) Execute(ctx context.Context, input TransitionIncidentInput) (*domain.Incident, error) {
	if err := ensureWriteAccess(input.Principal); err != nil {
		return nil, err
	}

	target, err := domain.ParseStatus(input.Status)
	if err != nil {
		return nil, err
	}

	incident, err := uc.incidents.FindByID(ctx, input.OrganizationID, input.IncidentID)
	if err != nil {
		return nil, err
	}

	from := incident.Status()
	if err := incident.TransitionTo(target, input.Principal.UserID, nowUTC()); err != nil {
		return nil, err
	}

	err = uc.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		return persistIncident(txCtx, uc.incidents, uc.history, uc.publisher, incident, false)
	})
	if err != nil {
		return nil, err
	}

	if uc.metrics != nil {
		uc.metrics.RecordTransition(from.String(), target.String())
		if target == domain.StatusResolved && incident.ResolvedAt() != nil {
			uc.metrics.ObserveResolutionDuration(incident.ResolvedAt().Sub(incident.CreatedAt()))
		}
		uc.metrics.ObserveStateDuration(from.String(), nowUTC().Sub(incident.UpdatedAt()))
	}
	return incident, nil
}

// ResolveIncidentInput identifies an incident to resolve.
type ResolveIncidentInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	IncidentID     domain.IncidentID
}

// ResolveIncidentUseCase resolves an incident.
type ResolveIncidentUseCase struct {
	transition *TransitionIncidentUseCase
}

// NewResolveIncidentUseCase creates the use case.
func NewResolveIncidentUseCase(transition *TransitionIncidentUseCase) *ResolveIncidentUseCase {
	return &ResolveIncidentUseCase{transition: transition}
}

// Execute resolves an incident.
func (uc *ResolveIncidentUseCase) Execute(ctx context.Context, input ResolveIncidentInput) (*domain.Incident, error) {
	return uc.transition.Execute(ctx, TransitionIncidentInput{
		Principal:      input.Principal,
		OrganizationID: input.OrganizationID,
		IncidentID:     input.IncidentID,
		Status:         domain.StatusResolved.String(),
	})
}

// CloseIncidentInput identifies an incident to close.
type CloseIncidentInput struct {
	Principal      ports.Principal
	OrganizationID domain.OrganizationID
	IncidentID     domain.IncidentID
}

// CloseIncidentUseCase closes an incident.
type CloseIncidentUseCase struct {
	transition *TransitionIncidentUseCase
}

// NewCloseIncidentUseCase creates the use case.
func NewCloseIncidentUseCase(transition *TransitionIncidentUseCase) *CloseIncidentUseCase {
	return &CloseIncidentUseCase{transition: transition}
}

// Execute closes an incident.
func (uc *CloseIncidentUseCase) Execute(ctx context.Context, input CloseIncidentInput) (*domain.Incident, error) {
	return uc.transition.Execute(ctx, TransitionIncidentInput{
		Principal:      input.Principal,
		OrganizationID: input.OrganizationID,
		IncidentID:     input.IncidentID,
		Status:         domain.StatusClosed.String(),
	})
}
