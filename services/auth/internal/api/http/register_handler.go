package httpapi

import (
	"net/http"
	"strings"

	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	"github.com/halcyon-io/halcyon/services/auth/internal/api/http/dto"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
)

// RegisterHandler serves the user registration endpoint.
type RegisterHandler struct {
	register *appauth.RegisterUserHandler
}

// NewRegisterHandler creates a RegisterHandler.
func NewRegisterHandler(register *appauth.RegisterUserHandler) *RegisterHandler {
	return &RegisterHandler{register: register}
}

// Register creates a new user account.
func (h *RegisterHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest

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

	userID, err := h.register.Handle(r.Context(), appauth.RegisterUserCommand{
		Email:    strings.TrimSpace(req.Email),
		Password: req.Password,
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, map[string]string{
		"user_id": userID.String(),
	})
}