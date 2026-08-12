package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	appworkflow "github.com/halcyon-io/halcyon/services/workflow/internal/application/workflow"
	httpmiddleware "github.com/halcyon-io/halcyon/services/workflow/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// WorkflowHandler serves workflow HTTP endpoints.
type WorkflowHandler struct {
	create   *appworkflow.CreateWorkflowUseCase
	get      *appworkflow.GetWorkflowUseCase
	list     *appworkflow.ListWorkflowsUseCase
	start    *appworkflow.StartExecutionUseCase
	getExec  *appworkflow.GetExecutionUseCase
	cancel   *appworkflow.CancelExecutionUseCase
}

func NewWorkflowHandler(
	create *appworkflow.CreateWorkflowUseCase,
	get *appworkflow.GetWorkflowUseCase,
	list *appworkflow.ListWorkflowsUseCase,
	start *appworkflow.StartExecutionUseCase,
	getExec *appworkflow.GetExecutionUseCase,
	cancel *appworkflow.CancelExecutionUseCase,
) *WorkflowHandler {
	return &WorkflowHandler{create: create, get: get, list: list, start: start, getExec: getExec, cancel: cancel}
}

type createWorkflowRequest struct {
	OrganizationID string              `json:"organization_id"`
	Name           string              `json:"name"`
	Steps          []domain.StepInput  `json:"steps"`
}

type workflowResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Version        int    `json:"version"`
	CreatedAt      string `json:"created_at"`
}

type executionResponse struct {
	ID             string `json:"id"`
	WorkflowID     string `json:"workflow_id"`
	OrganizationID string `json:"organization_id"`
	Status         string `json:"status"`
	CorrelationID  string `json:"correlation_id"`
	StartedAt      string `json:"started_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func (h *WorkflowHandler) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}
	var req createWorkflowRequest
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	orgID, err := organizationIDFromRequest(r, req.OrganizationID)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid organization id")
		return
	}
	wf, def, err := h.create.Execute(r.Context(), appworkflow.CreateWorkflowInput{
		Principal: principal, OrganizationID: orgID, Name: req.Name, Steps: req.Steps,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, workflowResponse{
		ID: wf.ID.String(), OrganizationID: wf.OrganizationID.String(),
		Name: wf.Name, Version: def.Version, CreatedAt: wf.CreatedAt.Format(timeRFC3339),
	})
}

func (h *WorkflowHandler) ListWorkflows(w http.ResponseWriter, r *http.Request) {
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
	items, cursor, err := h.list.Execute(r.Context(), appworkflow.ListWorkflowsInput{
		Principal: principal, OrganizationID: orgID, Limit: paging.Limit, Cursor: paging.Cursor,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	resp := make([]workflowResponse, 0, len(items))
	for _, wf := range items {
		resp = append(resp, workflowResponse{
			ID: wf.ID.String(), OrganizationID: wf.OrganizationID.String(),
			Name: wf.Name, Version: wf.LatestVersion, CreatedAt: wf.CreatedAt.Format(timeRFC3339),
		})
	}
	writeJSON(w, http.StatusOK, collectionResponse(resp, cursor))
}

func (h *WorkflowHandler) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}
	wfID, err := domain.ParseWorkflowID(chi.URLParam(r, "workflow_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid workflow id")
		return
	}
	orgID, err := organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}
	version, _ := strconv.Atoi(r.URL.Query().Get("version"))
	wf, def, err := h.get.Execute(r.Context(), appworkflow.GetWorkflowInput{
		Principal: principal, OrganizationID: orgID, WorkflowID: wfID, Version: version,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"workflow": workflowResponse{
			ID: wf.ID.String(), OrganizationID: wf.OrganizationID.String(),
			Name: wf.Name, Version: def.Version, CreatedAt: wf.CreatedAt.Format(timeRFC3339),
		},
		"steps": def.Steps,
	})
}

func (h *WorkflowHandler) StartExecution(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}
	wfID, err := domain.ParseWorkflowID(chi.URLParam(r, "workflow_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid workflow id")
		return
	}
	var req struct {
		OrganizationID string `json:"organization_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request body")
		return
	}
	orgID, err := organizationIDFromRequest(r, req.OrganizationID)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid organization id")
		return
	}
	execution, err := h.start.Execute(r.Context(), appworkflow.StartExecutionInput{
		Principal: principal, OrganizationID: orgID, WorkflowID: wfID,
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, toExecutionResponse(execution))
}

func (h *WorkflowHandler) GetExecution(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}
	execID, err := domain.ParseExecutionID(chi.URLParam(r, "execution_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid execution id")
		return
	}
	orgID, err := organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}
	execution, steps, err := h.getExec.Execute(r.Context(), appworkflow.GetExecutionInput{
		Principal: principal, OrganizationID: orgID, ExecutionID: execID,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"execution": toExecutionResponse(execution),
		"steps":     steps,
	})
}

func (h *WorkflowHandler) CancelExecution(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
		return
	}
	execID, err := domain.ParseExecutionID(chi.URLParam(r, "execution_id"))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid execution id")
		return
	}
	var req struct {
		OrganizationID string `json:"organization_id"`
	}
	_ = decodeJSON(r, &req)
	orgID, err := organizationIDFromRequest(r, req.OrganizationID)
	if err != nil {
		orgID, err = organizationIDFromRequest(r, r.URL.Query().Get("organization_id"))
	}
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "organization_id is required")
		return
	}
	execution, err := h.cancel.Execute(r.Context(), appworkflow.CancelExecutionInput{
		Principal: principal, OrganizationID: orgID, ExecutionID: execID,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toExecutionResponse(execution))
}

func toExecutionResponse(execution *domain.WorkflowExecution) executionResponse {
	resp := executionResponse{
		ID: execution.ID.String(), WorkflowID: execution.WorkflowID.String(),
		OrganizationID: execution.OrganizationID.String(), Status: string(execution.Status),
		CorrelationID: execution.CorrelationID, CreatedAt: execution.CreatedAt.Format(timeRFC3339),
	}
	if execution.StartedAt != nil {
		resp.StartedAt = execution.StartedAt.Format(timeRFC3339)
	}
	if execution.CompletedAt != nil {
		resp.CompletedAt = execution.CompletedAt.Format(timeRFC3339)
	}
	return resp
}
