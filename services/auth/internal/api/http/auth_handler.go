package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	httpmiddleware "github.com/halcyon-io/halcyon/services/auth/internal/api/http/middleware"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// AuthHandler serves authenticated session endpoints.
type AuthHandler struct {
	logout *appauth.LogoutUseCase
}

// NewAuthHandler creates an AuthHandler for logout and profile endpoints.
func NewAuthHandler(logout *appauth.LogoutUseCase) *AuthHandler {
	return &AuthHandler{logout: logout}
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type meResponse struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// Logout revokes a refresh token.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req logoutRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeBadRequest(w, r, "malformed JSON request body")
		return
	}

	if err := h.logout.Execute(r.Context(), req.RefreshToken); err != nil {
		writeAuthError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Me returns the authenticated user's profile.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
	if !ok {
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/unauthorized",
			Title:    "Unauthorized",
			Status:   http.StatusUnauthorized,
			Detail:   "authentication required",
			Instance: middleware.GetReqID(r.Context()),
		})
		return
	}

	roles := make([]string, 0, len(principal.Roles))
	for _, role := range principal.Roles {
		roles = append(roles, role.String())
	}

	permissions := make([]string, 0, len(principal.Permissions))
	for _, perm := range principal.Permissions {
		permissions = append(permissions, perm.String())
	}

	httputil.WriteJSON(w, http.StatusOK, meResponse{
		ID:          principal.UserID.String(),
		Email:       principal.Email.String(),
		Roles:       roles,
		Permissions: permissions,
	})
}

func writeBadRequest(w http.ResponseWriter, r *http.Request, detail string) {
	httperrors.WriteProblem(w, httperrors.ProblemDetails{
		Type:     "https://halcyon.io/problems/invalid-request",
		Title:    "Bad Request",
		Status:   http.StatusBadRequest,
		Detail:   detail,
		Instance: middleware.GetReqID(r.Context()),
	})
}

func writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/unauthorized",
			Title:    "Unauthorized",
			Status:   http.StatusUnauthorized,
			Detail:   "invalid email or password",
			Instance: middleware.GetReqID(r.Context()),
		})
	case errors.Is(err, domain.ErrUserDisabled):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/forbidden",
			Title:    "Forbidden",
			Status:   http.StatusForbidden,
			Detail:   "account is disabled",
			Instance: middleware.GetReqID(r.Context()),
		})
	case errors.Is(err, domain.ErrForbidden):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/forbidden",
			Title:    "Forbidden",
			Status:   http.StatusForbidden,
			Detail:   "request forbidden",
			Instance: middleware.GetReqID(r.Context()),
		})
	case errors.Is(err, domain.ErrRefreshTokenReused):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/unauthorized",
			Title:    "Refresh Token Reuse Detected",
			Status:   http.StatusUnauthorized,
			Detail:   "refresh token reuse detected; token family revoked",
			Instance: middleware.GetReqID(r.Context()),
		})
	case errors.Is(err, domain.ErrTokenExpired),
		errors.Is(err, domain.ErrInvalidAccessToken),
		errors.Is(err, domain.ErrInvalidRefreshToken),
		errors.Is(err, domain.ErrRefreshTokenRevoked):
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/unauthorized",
			Title:    "Unauthorized",
			Status:   http.StatusUnauthorized,
			Detail:   "invalid or expired token",
			Instance: middleware.GetReqID(r.Context()),
		})
	default:
		httperrors.WriteProblem(w, httperrors.ProblemDetails{
			Type:     "https://halcyon.io/problems/internal-error",
			Title:    "Internal Server Error",
			Status:   http.StatusInternalServerError,
			Detail:   "an unexpected error occurred",
			Instance: middleware.GetReqID(r.Context()),
		})
	}
}
