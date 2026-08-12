package httpapi

import (
	"net/http"
	"strings"

	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	"github.com/halcyon-io/halcyon/services/auth/internal/api/http/dto"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
)

// LoginHandler serves the login endpoint.
type LoginHandler struct {
	login *appauth.LoginUseCase
}

// NewLoginHandler creates a LoginHandler.
func NewLoginHandler(login *appauth.LoginUseCase) *LoginHandler {
	return &LoginHandler{login: login}
}

// Login authenticates a user and returns tokens.
func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeBadRequest(w, r, "malformed JSON request body")
		return
	}

	if fieldErrors := validateRequired(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}); len(fieldErrors) > 0 {
		writeValidationProblem(w, r, "request validation failed", fieldErrors)
		return
	}

	result, err := h.login.Execute(r.Context(), appauth.LoginInput{
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, dto.ToTokenResponse(result))
}
