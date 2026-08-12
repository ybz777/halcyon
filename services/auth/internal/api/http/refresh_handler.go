package httpapi

import (
	"net/http"

	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	"github.com/halcyon-io/halcyon/services/auth/internal/api/http/dto"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
)

// RefreshHandler serves the refresh endpoint.
type RefreshHandler struct {
	refresh *appauth.RefreshTokenUseCase
}

// NewRefreshHandler creates a RefreshHandler.
func NewRefreshHandler(refresh *appauth.RefreshTokenUseCase) *RefreshHandler {
	return &RefreshHandler{refresh: refresh}
}

// Refresh rotates refresh tokens and returns a new token pair.
func (h *RefreshHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeBadRequest(w, r, "malformed JSON request body")
		return
	}

	if fieldErrors := validateRequired(map[string]string{
		"refreshToken": req.RefreshToken,
	}); len(fieldErrors) > 0 {
		writeValidationProblem(w, r, "request validation failed", fieldErrors)
		return
	}

	result, err := h.refresh.Execute(r.Context(), appauth.RefreshTokenInput{
		RefreshToken: req.RefreshToken,
		IPAddress:    clientIP(r),
		UserAgent:    r.UserAgent(),
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToTokenResponse(result))
}
