package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/halcyon-io/halcyon/pkg/platform/httputil"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// IntrospectAPIKeyHandler validates API keys for gateway and service consumers.
type IntrospectAPIKeyHandler struct {
	authenticator *appauth.AuthenticateAPIKeyHandler
}

func NewIntrospectAPIKeyHandler(authenticator *appauth.AuthenticateAPIKeyHandler) *IntrospectAPIKeyHandler {
	return &IntrospectAPIKeyHandler{authenticator: authenticator}
}

func (h *IntrospectAPIKeyHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.APIKey == "" {
		if key := r.Header.Get("X-API-Key"); key != "" {
			req.APIKey = key
		} else {
			writeAuthError(w, r, domain.ErrUnauthorized)
			return
		}
	}
	principal, err := h.authenticator.Handle(r.Context(), appauth.AuthenticateAPIKeyQuery{APIKey: req.APIKey})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	scopes := make([]string, 0, len(principal.Scopes))
	for _, scope := range principal.Scopes {
		scopes = append(scopes, scope.String())
	}
	perms := make([]string, 0, len(principal.Permissions))
	for _, perm := range principal.Permissions {
		perms = append(perms, perm.String())
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"principal_type":     "API_KEY",
		"api_key_id":         principal.APIKeyID.String(),
		"organization_id":    principal.OrganizationID.String(),
		"service_account_id": principal.ServiceAccountID.String(),
		"scopes":             scopes,
		"permissions":        perms,
	})
}
