package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
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
	case errors.Is(err, domain.ErrOrganizationNotFound), errors.Is(err, domain.ErrMembershipNotFound):
		writeProblem(w, r, http.StatusNotFound, "Not Found", "resource not found")
	case errors.Is(err, domain.ErrSlugExists), errors.Is(err, domain.ErrMembershipExists):
		writeProblem(w, r, http.StatusConflict, "Conflict", "resource already exists")
	case errors.Is(err, domain.ErrInvitationExpired), errors.Is(err, domain.ErrInvitationRevoked), errors.Is(err, domain.ErrInvitationUsed):
		writeProblem(w, r, http.StatusUnprocessableEntity, "Unprocessable Entity", "invitation cannot be accepted")
	case errors.Is(err, domain.ErrOrganizationSuspended), errors.Is(err, domain.ErrOrganizationDeleted):
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "organization unavailable")
	case errors.Is(err, domain.ErrMembershipSuspended), errors.Is(err, domain.ErrMembershipRemoved):
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "membership unavailable")
	case errors.Is(err, domain.ErrCrossTenantAccess):
		writeProblem(w, r, http.StatusForbidden, "Forbidden", "insufficient permissions")
	case errors.Is(err, domain.ErrLastOwner):
		writeProblem(w, r, http.StatusConflict, "Conflict", "cannot remove last owner")
	case errors.Is(err, domain.ErrInvalidCursor), errors.Is(err, platformpagination.ErrInvalidLimit):
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
