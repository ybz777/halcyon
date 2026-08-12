package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	appwebhook "github.com/halcyon-io/halcyon/services/integrations/internal/application/webhook"
	httpmiddleware "github.com/halcyon-io/halcyon/services/integrations/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
)

type WebhookHandler struct {
	service *appwebhook.Service
}

func NewWebhookHandler(service *appwebhook.Service) *WebhookHandler {
	return &WebhookHandler{service: service}
}

func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	principal, orgID, ok := requireContext(w, r)
	if !ok {
		return
	}
	var req struct {
		URL         string   `json:"url"`
		Description string   `json:"description"`
		Events      []string `json:"subscribed_events"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.service.CreateEndpoint(r.Context(), appwebhook.CreateEndpointInput{
		Principal: principal, OrganizationID: orgID, URL: req.URL,
		Description: req.Description, Events: req.Events,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, map[string]any{
		"id": result.Endpoint.ID.String(), "url": result.Endpoint.URL,
		"description": result.Endpoint.Description, "status": result.Endpoint.Status,
		"subscribed_events": result.Endpoint.SubscribedEvents,
		"secret": result.PlainSecret, "secret_prefix": result.Endpoint.SecretPrefix,
	})
}

var integrationsQueryParser = platformpagination.NewParser(50, platformpagination.MaxPageSize)

func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, orgID, ok := requireContext(w, r)
	if !ok {
		return
	}
	paging, err := integrationsQueryParser.ParsePagination(r.URL.Query())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	items, cursor, err := h.service.ListEndpoints(r.Context(), principal, orgID, paging.Limit, paging.Cursor)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	page := platformpagination.Collection[map[string]any]{
		Items: toEndpointResponses(items), NextCursor: cursor, HasMore: cursor != "",
	}
	httputil.WriteJSON(w, http.StatusOK, page)
}

func (h *WebhookHandler) Get(w http.ResponseWriter, r *http.Request) {
	principal, orgID, ok := requireContext(w, r)
	if !ok {
		return
	}
	id, err := domain.ParseEndpointID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid endpoint id")
		return
	}
	endpoint, err := h.service.GetEndpoint(r.Context(), principal, orgID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toEndpointResponse(endpoint))
}

func (h *WebhookHandler) Patch(w http.ResponseWriter, r *http.Request) {
	principal, orgID, ok := requireContext(w, r)
	if !ok {
		return
	}
	id, err := domain.ParseEndpointID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid endpoint id")
		return
	}
	var req struct {
		URL         string   `json:"url"`
		Description string   `json:"description"`
		Events      []string `json:"subscribed_events"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid request body")
		return
	}
	endpoint, err := h.service.PatchEndpoint(r.Context(), principal, orgID, id, req.URL, req.Description, req.Events)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toEndpointResponse(endpoint))
}

func (h *WebhookHandler) Enable(w http.ResponseWriter, r *http.Request)  { h.setStatus(w, r, true) }
func (h *WebhookHandler) Disable(w http.ResponseWriter, r *http.Request) { h.setStatus(w, r, false) }

func (h *WebhookHandler) setStatus(w http.ResponseWriter, r *http.Request, active bool) {
	principal, orgID, ok := requireContext(w, r)
	if !ok {
		return
	}
	id, err := domain.ParseEndpointID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid endpoint id")
		return
	}
	endpoint, err := h.service.SetEndpointStatus(r.Context(), principal, orgID, id, active)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toEndpointResponse(endpoint))
}

func (h *WebhookHandler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	principal, orgID, ok := requireContext(w, r)
	if !ok {
		return
	}
	endpointID, err := domain.ParseEndpointID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid endpoint id")
		return
	}
	paging, err := integrationsQueryParser.ParsePagination(r.URL.Query())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	items, cursor, err := h.service.ListDeliveries(r.Context(), principal, orgID, endpointID, paging.Limit, paging.Cursor)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	page := platformpagination.Collection[map[string]any]{
		Items: toDeliveryResponses(items), NextCursor: cursor, HasMore: cursor != "",
	}
	httputil.WriteJSON(w, http.StatusOK, page)
}

func (h *WebhookHandler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	principal, orgID, ok := requireContext(w, r)
	if !ok {
		return
	}
	id, err := domain.ParseDeliveryID(chi.URLParam(r, "delivery_id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid delivery id")
		return
	}
	delivery, err := h.service.GetDelivery(r.Context(), principal, orgID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toDeliveryResponse(delivery))
}

func (h *WebhookHandler) Test(w http.ResponseWriter, r *http.Request) {
	principal, orgID, ok := requireContext(w, r)
	if !ok {
		return
	}
	id, err := domain.ParseEndpointID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid endpoint id")
		return
	}
	result, err := h.service.TestEndpoint(r.Context(), principal, orgID, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"event_id": result.EventID, "event_type": result.EventType,
		"status_code": result.StatusCode, "response_time_ms": result.ResponseTimeMs,
		"success": result.Success, "error": result.ErrorSummary,
	})
}

func requireContext(w http.ResponseWriter, r *http.Request) (appwebhook.Principal, string, bool) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return appwebhook.Principal{}, "", false
	}
	orgID := httpmiddleware.OrganizationFromRequest(r)
	if orgID == "" {
		writeProblem(w, http.StatusBadRequest, "organization context required")
		return appwebhook.Principal{}, "", false
	}
	return principal, orgID, true
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: http.StatusText(status), Status: status, Detail: detail})
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		writeProblem(w, http.StatusUnauthorized, "authentication required")
	case errors.Is(err, domain.ErrForbidden):
		writeProblem(w, http.StatusForbidden, "insufficient permissions")
	case errors.Is(err, domain.ErrEndpointNotFound), errors.Is(err, domain.ErrDeliveryNotFound):
		writeProblem(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrInvalidURL), errors.Is(err, domain.ErrInvalidSubscription):
		writeProblem(w, http.StatusBadRequest, err.Error())
	default:
		writeProblem(w, http.StatusInternalServerError, "unexpected error")
	}
}

func toEndpointResponses(items []domain.WebhookEndpoint) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, toEndpointResponse(item))
	}
	return out
}

func toEndpointResponse(item domain.WebhookEndpoint) map[string]any {
	return map[string]any{
		"id": item.ID.String(), "organization_id": item.OrganizationID, "url": item.URL,
		"description": item.Description, "status": item.Status,
		"subscribed_events": item.SubscribedEvents, "secret_prefix": item.SecretPrefix,
		"failure_count": item.FailureCount, "created_at": item.CreatedAt,
	}
}

func toDeliveryResponses(items []domain.WebhookDelivery) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, toDeliveryResponse(item))
	}
	return out
}

func toDeliveryResponse(item domain.WebhookDelivery) map[string]any {
	resp := map[string]any{
		"id": item.ID.String(), "webhook_endpoint_id": item.EndpointID.String(),
		"event_id": item.EventID, "event_type": item.EventType, "status": item.Status,
		"attempt_count": item.AttemptCount, "last_error": item.LastError,
		"created_at": item.CreatedAt,
	}
	if item.ResponseStatus != nil {
		resp["response_status"] = *item.ResponseStatus
	}
	if item.ResponseTimeMs != nil {
		resp["response_time_ms"] = *item.ResponseTimeMs
	}
	return resp
}
