package middleware

import (
	"net/http"
	"strings"

	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// APIKeyAuthMiddleware validates API keys from Authorization or X-API-Key headers.
func APIKeyAuthMiddleware(authenticator *appauth.AuthenticateAPIKeyHandler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey, ok := extractAPIKey(r)
			if !ok {
				writeUnauthorized(w, r, domain.ErrUnauthorized)
				return
			}

			principal, err := authenticator.Handle(r.Context(), appauth.AuthenticateAPIKeyQuery{
				APIKey: rawKey,
			})
			if err != nil {
				writeUnauthorized(w, r, err)
				return
			}

			ctx := WithPrincipal(r.Context(), principal)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractAPIKey(r *http.Request) (string, bool) {
	if header := strings.TrimSpace(r.Header.Get("X-API-Key")); header != "" {
		return header, isAPIKeyFormat(header)
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if isAPIKeyFormat(token) {
			return token, true
		}
	}
	return "", false
}

func isAPIKeyFormat(value string) bool {
	return strings.HasPrefix(value, "hal_live_") || strings.HasPrefix(value, "hal_test_")
}

// RequireScope enforces API scope checks for API key principals.
func RequireScope(scope domain.APIScope) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok || principal.AuthMethod != appauth.AuthMethodAPIKey {
				writeForbidden(w, r)
				return
			}
			if !principal.HasScope(scope) {
				writeForbidden(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
