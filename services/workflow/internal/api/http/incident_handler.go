package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	appincident "github.com/halcyon-io/halcyon/services/workflow/internal/application/incident"
	httpmiddleware "github.com/halcyon-io/halcyon/services/workflow/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// IncidentHandler serves incident HTTP endpoints.
type IncidentHandler struct {
	create     *appincident.CreateIncidentUseCase
	get        *appincident.GetIncidentUseCase
	list       *appincident.ListIncidentsUseCase
	search     *appincident.SearchIncidentsUseCase
	transition *appincident.TransitionIncidentUseCase
	severity   *appincident.UpdateSeverityUseCase
	resolve    *appincident.ResolveIncidentUseCase
	close      *appincident.CloseIncidentUseCase
}

// NewIncidentHandler creates an IncidentHandler.
func NewIncidentHandler(
	create *appincident.CreateIncidentUseCase,
	get *appincident.GetIncidentUseCase,
	list *appincident.ListIncidentsUseCase,
	search *appincident.SearchIncidentsUseCase,
	transition *appincident.TransitionIncidentUseCase,
	severity *appincident.UpdateSeverityUseCase,
	resolve *appincident.ResolveIncidentUseCase,
	closeUC *appincident.CloseIncidentUseCase,
) *IncidentHandler {
	return &IncidentHandler{
		create: create, get: get, list: list, search: search,
		transition: transition, severity: severity, resolve: resolve, close: closeUC,
	}
}

type createIncidentRequest struct {
	OrganizationID string `json:"organization_id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	Severity       string `json:"severity"`
	Priority       string `json:"priority"`
	Source         string `json:"source"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type updateSeverityRequest struct {
	Severity string `json:"severity"`
}

type incidentResponse struct {
	ID             string  `json:"id"`
	OrganizationID string  `json:"organization_id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Severity       string  `json:"severity"`
	Priority       string  `json:"priority"`
	Status         string  `json:"status"`
	Source         string  `json:"source"`
	Version        int     `json:"version"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
	ResolvedAt     *string `json:"resolved_at,omitempty"`
}

type listIncidentsResponse = platformpagination.Collection[incidentResponse]

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

// CreateIncident handles POST /api/v1/incidents.
func (h *IncidentHandler) CreateIncident(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}

	var req createIncidentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	orgID, err := organizationIDFromRequest(r, req.OrganizationID)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid organization id")
		return
	}

	incident, err := h.create.Execute(r.Context(), appincident.CreateIncidentInput{
		Principal:      principal,
		OrganizationID: orgID,
		Title:          req.Title,
		Description:    req.Description,
		Severity:       req.Severity,
		Priority:       req.Priority,
		Source:         req.Source,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toIncidentResponse(incident))
}

// ListIncidents handles GET /api/v1/incidents.
func (h *IncidentHandler) ListIncidents(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}

	orgID, err := organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}

	paging, err := parseListQuery(r)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", err.Error())
		return
	}
	query := r.URL.Query().Get("q")
	var result appincident.ListIncidentsResult
	if query != "" {
		result, err = h.search.Execute(r.Context(), appincident.SearchIncidentsInput{
			Principal:      principal,
			OrganizationID: orgID,
			Query:          query,
			Status:         r.URL.Query().Get("status"),
			Severity:       r.URL.Query().Get("severity"),
			Limit:          paging.Limit,
			Cursor:         paging.Cursor,
			SortBy:         r.URL.Query().Get("sort_by"),
			SortOrder:      r.URL.Query().Get("sort_order"),
		})
	} else {
		result, err = h.list.Execute(r.Context(), appincident.ListIncidentsInput{
			Principal:      principal,
			OrganizationID: orgID,
			Status:         r.URL.Query().Get("status"),
			Severity:       r.URL.Query().Get("severity"),
			Limit:          paging.Limit,
			Cursor:         paging.Cursor,
			SortBy:         r.URL.Query().Get("sort_by"),
			SortOrder:      r.URL.Query().Get("sort_order"),
		})
	}
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	items := make([]incidentResponse, 0, len(result.Items))
	for _, incident := range result.Items {
		items = append(items, toIncidentResponse(incident))
	}
	writeJSON(w, http.StatusOK, collectionResponse(items, result.NextCursor))
}

// GetIncident handles GET /api/v1/incidents/{id}.
func (h *IncidentHandler) GetIncident(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}

	incidentID, err := domain.ParseIncidentID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid incident id")
		return
	}
	orgID, err := organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}

	incident, err := h.get.Execute(r.Context(), appincident.GetIncidentInput{
		Principal:      principal,
		OrganizationID: orgID,
		IncidentID:     incidentID,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toIncidentResponse(incident))
}

// UpdateStatus handles PATCH /api/v1/incidents/{id}/status.
func (h *IncidentHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	h.transitionStatus(w, r)
}

// UpdateSeverity handles PATCH /api/v1/incidents/{id}/severity.
func (h *IncidentHandler) UpdateSeverity(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}

	incidentID, err := domain.ParseIncidentID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid incident id")
		return
	}
	orgID, err := organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}

	var req updateSeverityRequest
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	incident, err := h.severity.Execute(r.Context(), appincident.UpdateSeverityInput{
		Principal:      principal,
		OrganizationID: orgID,
		IncidentID:     incidentID,
		Severity:       req.Severity,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toIncidentResponse(incident))
}

// ResolveIncident handles POST /api/v1/incidents/{id}/resolve.
func (h *IncidentHandler) ResolveIncident(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}

	incidentID, err := domain.ParseIncidentID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid incident id")
		return
	}
	orgID, err := organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}

	incident, err := h.resolve.Execute(r.Context(), appincident.ResolveIncidentInput{
		Principal:      principal,
		OrganizationID: orgID,
		IncidentID:     incidentID,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toIncidentResponse(incident))
}

// CloseIncident handles POST /api/v1/incidents/{id}/close.
func (h *IncidentHandler) CloseIncident(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}

	incidentID, err := domain.ParseIncidentID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid incident id")
		return
	}
	orgID, err := organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}

	incident, err := h.close.Execute(r.Context(), appincident.CloseIncidentInput{
		Principal:      principal,
		OrganizationID: orgID,
		IncidentID:     incidentID,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toIncidentResponse(incident))
}

func (h *IncidentHandler) transitionStatus(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}

	incidentID, err := domain.ParseIncidentID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid incident id")
		return
	}
	orgID, err := organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}

	var req updateStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}

	incident, err := h.transition.Execute(r.Context(), appincident.TransitionIncidentInput{
		Principal:      principal,
		OrganizationID: orgID,
		IncidentID:     incidentID,
		Status:         req.Status,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toIncidentResponse(incident))
}

func organizationIDFromRequest(r *http.Request, bodyValue string) (domain.OrganizationID, error) {
	if header := r.Header.Get("X-Organization-ID"); header != "" {
		return domain.ParseOrganizationID(header)
	}
	if bodyValue != "" {
		return domain.ParseOrganizationID(bodyValue)
	}
	return domain.OrganizationID{}, domain.ErrInvalidOrganizationID
}

func toIncidentResponse(incident *domain.Incident) incidentResponse {
	var resolvedAt *string
	if incident.ResolvedAt() != nil {
		formatted := incident.ResolvedAt().Format(timeRFC3339)
		resolvedAt = &formatted
	}
	return incidentResponse{
		ID:             incident.ID().String(),
		OrganizationID: incident.OrganizationID().String(),
		Title:          incident.Title().String(),
		Description:    incident.Description().String(),
		Severity:       incident.Severity().String(),
		Priority:       incident.Priority().String(),
		Status:         incident.Status().String(),
		Source:         incident.Source().String(),
		Version:        incident.Version(),
		CreatedAt:      incident.CreatedAt().Format(timeRFC3339),
		UpdatedAt:      incident.UpdatedAt().Format(timeRFC3339),
		ResolvedAt:     resolvedAt,
	}
}
