package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/go-chi/chi/v5"
	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	appquery "github.com/halcyon-io/halcyon/services/jobs/internal/application/query"
	httpmiddleware "github.com/halcyon-io/halcyon/services/jobs/internal/api/http/middleware"
	"github.com/halcyon-io/halcyon/services/jobs/internal/domain"
)

// JobHandler serves operational job APIs.
type JobHandler struct {
	query *appquery.UseCase
}

func NewJobHandler(query *appquery.UseCase) *JobHandler {
	return &JobHandler{query: query}
}

type jobResponse struct {
	ID             string         `json:"id"`
	OrganizationID string         `json:"organization_id,omitempty"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	Priority       string         `json:"priority"`
	AttemptCount   int            `json:"attempt_count"`
	MaxAttempts    int            `json:"max_attempts"`
	LastError      string         `json:"last_error,omitempty"`
	PayloadSummary map[string]any `json:"payload_summary,omitempty"`
	CreatedAt      string         `json:"created_at"`
	StartedAt      string         `json:"started_at,omitempty"`
	CompletedAt    string         `json:"completed_at,omitempty"`
	FailedAt       string         `json:"failed_at,omitempty"`
}

type listJobsResponse = platformpagination.Collection[jobResponse]

var jobsQueryParser = platformpagination.NewParser(50, 200)

type attemptResponse struct {
	ID            string `json:"id"`
	AttemptNumber int    `json:"attempt_number"`
	WorkerID      string `json:"worker_id"`
	Status        string `json:"status"`
	ErrorCategory string `json:"error_category,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
	StartedAt     string `json:"started_at"`
	CompletedAt   string `json:"completed_at,omitempty"`
	DurationMs    int    `json:"duration_ms"`
}

func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return
	}
	paging, err := jobsQueryParser.ParsePagination(r.URL.Query())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	jobs, cursor, err := h.query.List(r.Context(), appquery.ListInput{
		Principal: principal,
		OrgID:     httpmiddleware.OrganizationFromRequest(r),
		Limit:     paging.Limit,
		Cursor:    paging.Cursor,
	})
	if err != nil {
		writeProblem(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	items := make([]jobResponse, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, toJobResponse(job))
	}
	writeJSON(w, http.StatusOK, listJobsResponse{
		Items: items, NextCursor: cursor, HasMore: cursor != "",
	})
}

func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := platformjobs.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid job id")
		return
	}
	job, err := h.query.Get(r.Context(), principal, httpmiddleware.OrganizationFromRequest(r), id)
	if err != nil {
		if errors.Is(err, platformjobs.ErrJobNotFound) {
			writeProblem(w, http.StatusNotFound, "job not found")
			return
		}
		writeProblem(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	writeJSON(w, http.StatusOK, toJobResponse(job))
}

func (h *JobHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := platformjobs.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid job id")
		return
	}
	if err := h.query.Cancel(r.Context(), principal, httpmiddleware.OrganizationFromRequest(r), id); err != nil {
		if errors.Is(err, platformjobs.ErrJobNotFound) {
			writeProblem(w, http.StatusNotFound, "job not found")
			return
		}
		writeProblem(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *JobHandler) ListAttempts(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeProblem(w, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := platformjobs.ParseID(chi.URLParam(r, "id"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid job id")
		return
	}
	paging, err := jobsQueryParser.ParsePagination(r.URL.Query())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, err.Error())
		return
	}
	attempts, cursor, err := h.query.ListAttempts(r.Context(), principal, httpmiddleware.OrganizationFromRequest(r), id, paging.Limit, paging.Cursor)
	if err != nil {
		if errors.Is(err, platformjobs.ErrJobNotFound) {
			writeProblem(w, http.StatusNotFound, "job not found")
			return
		}
		writeProblem(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	items := make([]attemptResponse, 0, len(attempts))
	for _, attempt := range attempts {
		item := attemptResponse{
			ID: attempt.ID.String(), AttemptNumber: attempt.AttemptNumber, WorkerID: attempt.WorkerID,
			Status: string(attempt.Status), ErrorCategory: string(attempt.ErrorCategory),
			ErrorMessage: attempt.ErrorMessage, StartedAt: attempt.StartedAt.UTC().Format(time.RFC3339Nano),
			DurationMs: attempt.DurationMs,
		}
		if attempt.CompletedAt != nil {
			item.CompletedAt = attempt.CompletedAt.UTC().Format(time.RFC3339Nano)
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, platformpagination.Collection[attemptResponse]{
		Items: items, NextCursor: cursor, HasMore: cursor != "",
	})
}

func toJobResponse(job platformjobs.Job) jobResponse {
	resp := jobResponse{
		ID: job.ID.String(), OrganizationID: job.OrganizationID, Type: string(job.Type),
		Status: string(job.Status), Priority: string(job.Priority), AttemptCount: job.AttemptCount,
		MaxAttempts: job.MaxAttempts, LastError: job.LastError,
		PayloadSummary: domain.PayloadSummary(job.Payload),
		CreatedAt: job.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if job.StartedAt != nil {
		resp.StartedAt = job.StartedAt.UTC().Format(time.RFC3339Nano)
	}
	if job.CompletedAt != nil {
		resp.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339Nano)
	}
	if job.FailedAt != nil {
		resp.FailedAt = job.FailedAt.UTC().Format(time.RFC3339Nano)
	}
	return resp
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{Title: http.StatusText(status), Status: status, Detail: detail})
}
