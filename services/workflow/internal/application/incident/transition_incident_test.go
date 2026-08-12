package incident_test

import (
	"context"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/incident"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func TestTransitionIncidentUseCase_InvalidTransition(t *testing.T) {
	t.Parallel()

	repo := &stubIncidentRepo{incident: openIncident(t)}
	uc := incident.NewTransitionIncidentUseCase(repo, &stubHistoryRepo{}, incidentNoopPublisher{}, &stubTx{}, nil)

	_, err := uc.Execute(context.Background(), incident.TransitionIncidentInput{
		Principal:      writePrincipal(),
		OrganizationID: mustOrgID(t),
		IncidentID:     mustIncidentID(t),
		Status:         "RESOLVED",
	})
	if err == nil {
		t.Fatal("expected invalid transition error")
	}
}

type stubIncidentRepo struct {
	incident *domain.Incident
}

func (s *stubIncidentRepo) Create(context.Context, *domain.Incident) error { return nil }
func (s *stubIncidentRepo) Update(context.Context, *domain.Incident) error { return nil }
func (s *stubIncidentRepo) FindByID(context.Context, domain.OrganizationID, domain.IncidentID) (*domain.Incident, error) {
	return s.incident, nil
}
func (s *stubIncidentRepo) Search(context.Context, ports.IncidentSearchFilter) ([]*domain.Incident, string, error) {
	return nil, "", nil
}

type stubHistoryRepo struct{}

func (stubHistoryRepo) Append(context.Context, []domain.TransitionRecord) error { return nil }

type incidentNoopPublisher struct{}

func (incidentNoopPublisher) Publish(context.Context, domain.Event) error { return nil }

type stubTx struct{}

func (stubTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func writePrincipal() ports.Principal {
	userID, _ := domain.ParseUserID("usr_actor12345678901")
	return ports.Principal{
		UserID: userID,
		Permissions: map[string]struct{}{
			ports.PermissionIncidentWrite: {},
		},
	}
}

func openIncident(t *testing.T) *domain.Incident {
	t.Helper()
	id, _ := domain.ParseIncidentID("inc_test123456789012")
	orgID, _ := domain.ParseOrganizationID("org_test123456789012")
	title, _ := domain.ParseTitle("Test")
	desc, _ := domain.ParseDescription("")
	source, _ := domain.ParseSource("manual")
	actor, _ := domain.ParseUserID("usr_actor12345678901")
	inc, err := domain.CreateIncident(id, orgID, title, desc, domain.SeverityHigh, domain.PriorityHigh, source, actor, time.Now().UTC())
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	inc.ClearUncommittedEvents()
	inc.ClearPendingHistory()
	return inc
}

func mustOrgID(t *testing.T) domain.OrganizationID {
	t.Helper()
	id, err := domain.ParseOrganizationID("org_test123456789012")
	if err != nil {
		t.Fatalf("org id: %v", err)
	}
	return id
}

func mustIncidentID(t *testing.T) domain.IncidentID {
	t.Helper()
	id, err := domain.ParseIncidentID("inc_test123456789012")
	if err != nil {
		t.Fatalf("incident id: %v", err)
	}
	return id
}
