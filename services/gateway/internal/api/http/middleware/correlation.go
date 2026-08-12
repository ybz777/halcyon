package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/halcyon-io/halcyon/pkg/platform/contextx"
)

const maxCorrelationIDLen = 128

// Correlation propagates correlation identifiers.
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
		if correlationID == "" {
			correlationID = middleware.GetReqID(r.Context())
		}
		if correlationID == "" {
			correlationID = "corr_" + uuid.NewString()
		}
		if len(correlationID) > maxCorrelationIDLen {
			correlationID = correlationID[:maxCorrelationIDLen]
		}
		ctx := contextx.WithCorrelationID(r.Context(), correlationID)
		w.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
