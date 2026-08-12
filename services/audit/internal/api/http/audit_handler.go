package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	appaudit "github.com/halcyon-io/halcyon/services/audit/internal/application/audit"
	httpmiddleware "github.com/halcyon-io/halcyon/services/audit/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
	auditmetrics "github.com/halcyon-io/halcyon/services/audit/internal/infrastructure/metrics"
)

// AuditHandler serves audit query endpoints.
type AuditHandler struct {
	query   *appaudit.QueryUseCase
	metrics *auditmetrics.Collector
}

// NewAuditHandler creates an AuditHandler.
func NewAuditHandler(query *appaudit.QueryUseCase, metrics *auditmetrics.Collector) *AuditHandler {
	return &AuditHandler{query: query, metrics: metrics}
}

type actorResponse struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
}

type targetResponse struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
}

type auditEventResponse struct {
	ID            string            `json:"id"`
	EventType     string            `json:"event_type"`
	Action        string            `json:"action"`
	Actor         actorResponse     `json:"actor"`
	Target        targetResponse    `json:"target"`
	Result        string            `json:"result"`
	OccurredAt    string            `json:"occurred_at"`
	RequestID     string            `json:"request_id,omitempty"`
	CorrelationID string            `json:"correlation_id,omitempty"`
	SourceService string            `json:"source_service,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type listResponse = platformpagination.Collection[auditEventResponse]

var auditQueryParser = platformpagination.NewParser(domain.DefaultPageSize, domain.MaxPageSize)

func (h *AuditHandler) List(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return
	}

	filter, err := parseListFilter(r)
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordQueryError()
		}
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.query.List(r.Context(), appaudit.ListInput{
		Principal:      principal,
		TrustedOrgID:   httpmiddleware.OrganizationFromRequest(r),
		RequestedOrgID: r.URL.Query().Get("organization_id"),
		Filter:         filter,
	})
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordQueryError()
		}
		writeDomainError(w, r, err)
		return
	}

	items := make([]auditEventResponse, 0, len(result.Events))
	for _, event := range result.Events {
		items = append(items, toResponse(event))
	}
	if items == nil {
		items = []auditEventResponse{}
	}
	page := listResponse{
		Items:      items,
		NextCursor: result.NextCursor,
		HasMore:    result.NextCursor != "",
	}
	if h.metrics != nil {
		h.metrics.ObserveQuery("list", start)
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *AuditHandler) Get(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return
	}

	id, err := domain.ParseAuditEventID(chi.URLParam(r, "audit_event_id"))
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordQueryError()
		}
		writeProblem(w, http.StatusBadRequest, "invalid audit event id")
		return
	}

	event, err := h.query.Get(r.Context(), appaudit.GetInput{
		Principal:      principal,
		TrustedOrgID:   httpmiddleware.OrganizationFromRequest(r),
		RequestedOrgID: r.URL.Query().Get("organization_id"),
		ID:             id,
	})
	if err != nil {
		if h.metrics != nil {
			h.metrics.RecordQueryError()
		}
		writeDomainError(w, r, err)
		return
	}
	if h.metrics != nil {
		h.metrics.ObserveQuery("get", start)
	}
	writeJSON(w, http.StatusOK, toResponse(event))
}

func parseListFilter(r *http.Request) (domain.ListFilter, error) {
	q := r.URL.Query()
	paging, err := auditQueryParser.ParsePagination(q)
	if err != nil {
		return domain.ListFilter{}, err
	}
	filter := domain.ListFilter{
		EventType:  q.Get("event_type"),
		ActorID:    q.Get("actor_id"),
		TargetID:   q.Get("target_id"),
		Cursor:     paging.Cursor,
		PageSize:   paging.Limit,
		ActorType:  domain.ActorType(q.Get("actor_type")),
		TargetType: domain.TargetType(q.Get("target_type")),
		Result:     domain.Result(q.Get("result")),
	}

	start, err := platformpagination.ParseRFC3339Time(q.Get("start_time"))
	if err != nil {
		return domain.ListFilter{}, err
	}
	filter.StartTime = start
	end, err := platformpagination.ParseRFC3339Time(q.Get("end_time"))
	if err != nil {
		return domain.ListFilter{}, err
	}
	filter.EndTime = end
	return filter, nil
}

func toResponse(event domain.AuditEvent) auditEventResponse {
	resp := auditEventResponse{
		ID:         event.ID.String(),
		EventType:  event.EventType,
		Action:     event.Action,
		Actor:      actorResponse{Type: string(event.ActorType), ID: event.ActorID},
		Target:     targetResponse{Type: string(event.TargetType), ID: event.TargetID},
		Result:     string(event.Result),
		OccurredAt: event.OccurredAt.Format(time.RFC3339Nano),
		RequestID:  event.RequestID,
		CorrelationID: event.CorrelationID,
		SourceService: event.SourceService,
	}
	if len(event.Metadata) > 0 {
		resp.Metadata = event.Metadata
	}
	return resp
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	})
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		writeProblem(w, http.StatusUnauthorized, "authentication required")
	case errors.Is(err, domain.ErrForbidden):
		writeProblem(w, http.StatusForbidden, "insufficient permissions")
	case errors.Is(err, domain.ErrAuditEventNotFound):
		writeProblem(w, http.StatusNotFound, "audit event not found")
	case errors.Is(err, domain.ErrInvalidFilter), errors.Is(err, domain.ErrInvalidCursor), errors.Is(err, platformpagination.ErrInvalidLimit):
		writeProblem(w, http.StatusBadRequest, err.Error())
	default:
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "unexpected error",
			Instance: middleware.GetReqID(r.Context()),
		})
	}
}
