package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	httpapi "github.com/halcyon-io/halcyon/services/workflow/internal/api/http"
	httpmiddleware "github.com/halcyon-io/halcyon/services/workflow/internal/api/http/middleware"
	appincident "github.com/halcyon-io/halcyon/services/workflow/internal/application/incident"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/events"
)

func TestCreateIncidentHandler(t *testing.T) {
	t.Parallel()

	repo := newMemoryIncidentRepo()
	history := &memoryHistoryRepo{}
	publisher := events.NoopPublisher{}
	tx := &noopTxManager{}
	metrics := &noopMetrics{}

	handler := httpapi.NewIncidentHandler(
		appincident.NewCreateIncidentUseCase(repo, history, publisher, tx, metrics),
		appincident.NewGetIncidentUseCase(repo),
		appincident.NewListIncidentsUseCase(repo),
		appincident.NewSearchIncidentsUseCase(repo),
		appincident.NewTransitionIncidentUseCase(repo, history, publisher, tx, metrics),
		appincident.NewUpdateSeverityUseCase(repo, history, publisher, tx, metrics),
		appincident.NewResolveIncidentUseCase(appincident.NewTransitionIncidentUseCase(repo, history, publisher, tx, metrics)),
		appincident.NewCloseIncidentUseCase(appincident.NewTransitionIncidentUseCase(repo, history, publisher, tx, metrics)),
	)

	router := chi.NewRouter()
	router.Post("/api/v1/incidents", func(w http.ResponseWriter, r *http.Request) {
		principal, _ := stubValidator{}.Validate(r.Context(), "")
		r = r.WithContext(httpmiddleware.WithPrincipal(r.Context(), principal))
		handler.CreateIncident(w, r)
	})

	body := `{"organization_id":"org_test123456789012","title":"API outage","description":"5xx errors","severity":"HIGH","priority":"HIGH","source":"monitoring"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "OPEN" {
		t.Fatalf("status = %v", resp["status"])
	}
}

type memoryIncidentRepo struct {
	items map[string]*domain.Incident
}

func newMemoryIncidentRepo() *memoryIncidentRepo {
	return &memoryIncidentRepo{items: make(map[string]*domain.Incident)}
}

func (m *memoryIncidentRepo) Create(_ context.Context, incident *domain.Incident) error {
	m.items[incident.ID().String()] = incident
	return nil
}

func (m *memoryIncidentRepo) Update(_ context.Context, incident *domain.Incident) error {
	if _, ok := m.items[incident.ID().String()]; !ok {
		return domain.ErrIncidentNotFound
	}
	m.items[incident.ID().String()] = incident
	return nil
}

func (m *memoryIncidentRepo) FindByID(_ context.Context, organizationID domain.OrganizationID, id domain.IncidentID) (*domain.Incident, error) {
	incident, ok := m.items[id.String()]
	if !ok || incident.OrganizationID().String() != organizationID.String() {
		return nil, domain.ErrIncidentNotFound
	}
	return incident, nil
}

func (m *memoryIncidentRepo) Search(_ context.Context, filter ports.IncidentSearchFilter) ([]*domain.Incident, string, error) {
	var items []*domain.Incident
	for _, incident := range m.items {
		if incident.OrganizationID().String() == filter.OrganizationID.String() {
			items = append(items, incident)
		}
	}
	return items, "", nil
}

type memoryHistoryRepo struct{}

func (m *memoryHistoryRepo) Append(context.Context, []domain.TransitionRecord) error { return nil }

type noopTxManager struct{}

func (n *noopTxManager) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type noopMetrics struct{}

func (noopMetrics) RecordIncidentCreated()                              {}
func (noopMetrics) RecordTransition(string, string)                     {}
func (noopMetrics) ObserveResolutionDuration(time.Duration)             {}
func (noopMetrics) ObserveStateDuration(string, time.Duration)          {}

type stubValidator struct{}

func (stubValidator) Validate(context.Context, string) (ports.Principal, error) {
	userID, _ := domain.ParseUserID("usr_actor12345678901")
	return ports.Principal{
		UserID: userID,
		Permissions: map[string]struct{}{
			ports.PermissionIncidentRead:  {},
			ports.PermissionIncidentWrite: {},
		},
	}, nil
}
