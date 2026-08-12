package middleware

import (
	"net/http"

	"github.com/halcyon-io/halcyon/services/gateway/internal/config"
)

// SecurityHeaders adds gateway security headers.
func SecurityHeaders(cfg config.SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.ContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", cfg.ContentTypeOptions)
			}
			if cfg.FrameOptions != "" {
				w.Header().Set("X-Frame-Options", cfg.FrameOptions)
			}
			if cfg.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}
			next.ServeHTTP(w, r)
		})
	}
}
