package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func writeProblem(w http.ResponseWriter, r *http.Request, status int, title, detail string) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Type:     "https://halcyon.io/problems/error",
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: middleware.GetReqID(r.Context()),
	})
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "insufficient permissions")
	case errors.Is(err, domain.ErrUnauthorized):
		writeProblem(w, r, http.StatusUnauthorized, "Unauthorized", "authentication required")
	case errors.Is(err, domain.ErrIncidentNotFound):
		writeProblem(w, r, http.StatusNotFound, "Not Found", "incident not found")
	case errors.Is(err, domain.ErrVersionConflict):
		writeProblem(w, r, http.StatusConflict, "Conflict", "incident was modified concurrently")
	case errors.Is(err, domain.ErrInvalidTransition):
		writeProblem(w, r, http.StatusConflict, "Conflict", "invalid status transition")
	case errors.Is(err, domain.ErrWorkflowNotFound), errors.Is(err, domain.ErrWorkflowVersionNotFound):
		writeProblem(w, r, http.StatusNotFound, "Not Found", "workflow not found")
	case errors.Is(err, domain.ErrExecutionNotFound):
		writeProblem(w, r, http.StatusNotFound, "Not Found", "execution not found")
	case errors.Is(err, domain.ErrSSRFBlocked):
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "url blocked by security policy")
	case errors.Is(err, domain.ErrWorkflowLimitExceeded), errors.Is(err, domain.ErrRecursionDepthExceeded):
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", err.Error())
	case errors.Is(err, domain.ErrInvalidCursor), errors.Is(err, platformpagination.ErrInvalidLimit),
		errors.Is(err, platformpagination.ErrInvalidSort), errors.Is(err, platformpagination.ErrInvalidFilter):
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", err.Error())
	default:
		writeProblem(w, r, http.StatusBadRequest, "Bad Request", "invalid request")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	httputil.WriteJSON(w, status, body)
}
