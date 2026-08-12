package httpapi

import (
	"net/http"

	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/jwks"
)

// JWKSHandler serves JSON Web Key Sets for token verification.
type JWKSHandler struct {
	provider *jwks.Provider
}

// NewJWKSHandler creates a JWKSHandler.
func NewJWKSHandler(provider *jwks.Provider) *JWKSHandler {
	return &JWKSHandler{provider: provider}
}

// ServeHTTP returns the JWKS document.
func (h *JWKSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil {
		http.Error(w, "jwks unavailable", http.StatusServiceUnavailable)
		return
	}
	body, err := h.provider.Marshal()
	if err != nil {
		http.Error(w, "jwks unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
