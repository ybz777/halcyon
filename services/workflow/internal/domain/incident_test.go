package domain_test

import (
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func TestIncidentWorkflowTransitions(t *testing.T) {
	t.Parallel()

	inc := newTestIncident(t)
	actor := mustUserID(t, "usr_actor12345678901")
	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)

	steps := []domain.Status{
		domain.StatusAcknowledged,
		domain.StatusInvestigating,
		domain.StatusMitigated,
		domain.StatusResolved,
		domain.StatusClosed,
	}

	for _, target := range steps {
		if err := inc.TransitionTo(target, actor, now); err != nil {
			t.Fatalf("transition to %s: %v", target, err)
		}
		if inc.Status() != target {
			t.Fatalf("status = %s, want %s", inc.Status(), target)
		}
	}

	if err := inc.TransitionTo(domain.StatusOpen, actor, now); err == nil {
		t.Fatal("expected invalid transition from CLOSED")
	}
}

func TestIncidentEscalation(t *testing.T) {
	t.Parallel()

	inc := newTestIncident(t)
	actor := mustUserID(t, "usr_actor12345678901")
	now := time.Now().UTC()

	if err := inc.UpdateSeverity(domain.SeverityCritical, actor, now); err != nil {
		t.Fatalf("update severity: %v", err)
	}

	events := inc.UncommittedEvents()
	found := false
	for _, event := range events {
		if event.EventName() == "incidents.incident.escalated" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected escalation event")
	}
}

func TestCanTransitionMatrix(t *testing.T) {
	t.Parallel()

	if !domain.CanTransition(domain.StatusOpen, domain.StatusAcknowledged) {
		t.Fatal("OPEN -> ACKNOWLEDGED should be allowed")
	}
	if domain.CanTransition(domain.StatusOpen, domain.StatusResolved) {
		t.Fatal("OPEN -> RESOLVED should be rejected")
	}
}

func newTestIncident(t *testing.T) *domain.Incident {
	t.Helper()

	id, err := domain.ParseIncidentID("inc_test123456789012")
	if err != nil {
		t.Fatalf("parse incident id: %v", err)
	}
	orgID, err := domain.ParseOrganizationID("org_test123456789012")
	if err != nil {
		t.Fatalf("parse org id: %v", err)
	}
	title, err := domain.ParseTitle("Database outage")
	if err != nil {
		t.Fatalf("parse title: %v", err)
	}
	desc, err := domain.ParseDescription("Primary DB unreachable")
	if err != nil {
		t.Fatalf("parse description: %v", err)
	}
	source, err := domain.ParseSource("monitoring")
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}
	actor := mustUserID(t, "usr_actor12345678901")
	now := time.Now().UTC()

	inc, err := domain.CreateIncident(
		id, orgID, title, desc,
		domain.SeverityHigh, domain.PriorityHigh, source,
		actor, now,
	)
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}
	inc.ClearUncommittedEvents()
	inc.ClearPendingHistory()
	return inc
}

func mustUserID(t *testing.T, raw string) domain.UserID {
	t.Helper()
	id, err := domain.ParseUserID(raw)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}
	return id
}
