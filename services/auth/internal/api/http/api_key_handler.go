package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	"github.com/halcyon-io/halcyon/services/auth/internal/api/http/dto"
	httpmiddleware "github.com/halcyon-io/halcyon/services/auth/internal/api/http/middleware"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// APIKeyHandler serves API key management endpoints.
type APIKeyHandler struct {
	create *appauth.CreateAPIKeyHandler
	list   *appauth.ListAPIKeysHandler
	get    *appauth.GetAPIKeyHandler
	revoke *appauth.RevokeAPIKeyHandler
	rotate *appauth.RotateAPIKeyHandler
}

var apiKeyQueryParser = platformpagination.NewParser(platformpagination.DefaultPageSize, platformpagination.MaxPageSize)

// NewAPIKeyHandler creates an APIKeyHandler.
func NewAPIKeyHandler(
	create *appauth.CreateAPIKeyHandler,
	list *appauth.ListAPIKeysHandler,
	get *appauth.GetAPIKeyHandler,
	revoke *appauth.RevokeAPIKeyHandler,
	rotate *appauth.RotateAPIKeyHandler,
) *APIKeyHandler {
	return &APIKeyHandler{create: create, list: list, get: get, revoke: revoke, rotate: rotate}
}

// Create issues a new API key and returns the plaintext once.
func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateAPIKeyRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeBadRequest(w, r, "malformed JSON request body")
		return
	}

	if fieldErrors := validateRequired(map[string]string{
		"name":             req.Name,
		"serviceAccountId": req.ServiceAccountID,
		"environment":      req.Environment,
	}); len(fieldErrors) > 0 {
		writeValidationProblem(w, r, "request validation failed", fieldErrors)
		return
	}
	if len(req.Scopes) == 0 {
		writeValidationProblem(w, r, "request validation failed", []httperrors.FieldError{{
			Field: "scopes", Code: "required", Message: "scopes is required",
		}})
		return
	}

	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeAuthError(w, r, domain.ErrUnauthorized)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ExpiresAt))
		if err != nil {
			writeBadRequest(w, r, "invalid expiresAt format")
			return
		}
		expiresAt = &parsed
	}

	result, err := h.create.Handle(r.Context(), appauth.CreateAPIKeyCommand{
		Name:             req.Name,
		ServiceAccountID: req.ServiceAccountID,
		OrganizationID:   req.OrganizationID,
		Scopes:           req.Scopes,
		Environment:      req.Environment,
		ExpiresAt:        expiresAt,
		CreatedBy:        principal.UserID,
	})
	if err != nil {
		writeAPIKeyError(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, dto.ToCreateAPIKeyResponse(result))
}

// List returns API key metadata for a service account.
func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	serviceAccountID := strings.TrimSpace(r.URL.Query().Get("serviceAccountId"))
	if serviceAccountID == "" {
		writeValidationProblem(w, r, "request validation failed", []httperrors.FieldError{{
			Field: "serviceAccountId", Code: "required", Message: "serviceAccountId is required",
		}})
		return
	}

	saID, err := domain.ParseServiceAccountID(serviceAccountID)
	if err != nil {
		writeAPIKeyError(w, r, err)
		return
	}

	paging, err := apiKeyQueryParser.ParsePagination(r.URL.Query())
	if err != nil {
		writeBadRequest(w, r, err.Error())
		return
	}

	keys, cursor, err := h.list.Handle(r.Context(), appauth.ListAPIKeysQuery{
		ServiceAccountID: saID, Limit: paging.Limit, Cursor: paging.Cursor,
	})
	if err != nil {
		writeAPIKeyError(w, r, err)
		return
	}

	resp := make([]dto.APIKeyResponse, 0, len(keys))
	for _, key := range keys {
		resp = append(resp, dto.ToAPIKeyResponse(key))
	}
	httputil.WriteJSON(w, http.StatusOK, platformpagination.Collection[dto.APIKeyResponse]{
		Items: resp, NextCursor: cursor, HasMore: cursor != "",
	})
}

// Get returns API key metadata by ID.
func (h *APIKeyHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseAPIKeyID(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIKeyError(w, r, err)
		return
	}
	key, err := h.get.Handle(r.Context(), appauth.GetAPIKeyQuery{APIKeyID: id})
	if err != nil {
		writeAPIKeyError(w, r, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, dto.ToAPIKeyResponse(key))
}

// Revoke revokes an API key.
func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseAPIKeyID(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIKeyError(w, r, err)
		return
	}

	if err := h.revoke.Handle(r.Context(), appauth.RevokeAPIKeyCommand{APIKeyID: id}); err != nil {
		writeAPIKeyError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RevokePost is an alias for Revoke using POST semantics.
func (h *APIKeyHandler) RevokePost(w http.ResponseWriter, r *http.Request) {
	h.Revoke(w, r)
}

// Rotate rotates an API key and returns the new plaintext once.
func (h *APIKeyHandler) Rotate(w http.ResponseWriter, r *http.Request) {
	id, err := domain.ParseAPIKeyID(chi.URLParam(r, "id"))
	if err != nil {
		writeAPIKeyError(w, r, err)
		return
	}

	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		writeAuthError(w, r, domain.ErrUnauthorized)
		return
	}

	result, err := h.rotate.Handle(r.Context(), appauth.RotateAPIKeyCommand{
		APIKeyID:    id,
		Environment: strings.TrimSpace(r.URL.Query().Get("environment")),
		RotatedBy:   principal.UserID,
	})
	if err != nil {
		writeAPIKeyError(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToCreateAPIKeyResponse(result))
}

func writeAPIKeyError(w http.ResponseWriter, r *http.Request, err error) {
	instance := middleware.GetReqID(r.Context())

	switch {
	case errors.Is(err, domain.ErrInvalidCursor), errors.Is(err, platformpagination.ErrInvalidLimit):
		writeBadRequest(w, r, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/unauthorized",
			Title:    "Unauthorized",
			Status:   http.StatusUnauthorized,
			Detail:   "authentication required",
			Instance: instance,
		})
	case errors.Is(err, domain.ErrForbidden):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/forbidden",
			Title:    "Forbidden",
			Status:   http.StatusForbidden,
			Detail:   "request forbidden",
			Instance: instance,
		})
	case errors.Is(err, domain.ErrAPIKeyNotFound), errors.Is(err, domain.ErrServiceAccountNotFound):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/not-found",
			Title:    "Not Found",
			Status:   http.StatusNotFound,
			Detail:   err.Error(),
			Instance: instance,
		})
	case errors.Is(err, domain.ErrInvalidAPIKey),
		errors.Is(err, domain.ErrInvalidAPIScope),
		errors.Is(err, domain.ErrInvalidServiceAccount),
		errors.Is(err, domain.ErrServiceAccountDisabled):
		writeBadRequest(w, r, err.Error())
	case errors.Is(err, domain.ErrAPIKeyRevoked),
		errors.Is(err, domain.ErrAPIKeyDisabled),
		errors.Is(err, domain.ErrAPIKeyExpired):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/conflict",
			Title:    "Conflict",
			Status:   http.StatusConflict,
			Detail:   err.Error(),
			Instance: instance,
		})
	default:
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/internal-error",
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "an unexpected error occurred",
			Instance: instance,
		})
	}
}
