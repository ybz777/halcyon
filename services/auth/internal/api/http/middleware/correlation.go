package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/halcyon-io/halcyon/pkg/platform/contextx"
)

// Correlation propagates correlation and causation identifiers through requests.
func Correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = middleware.GetReqID(r.Context())
		}
		if correlationID == "" {
			correlationID = "corr_" + uuid.NewString()
		}

		causationID := r.Header.Get("X-Causation-ID")

		ctx := contextx.WithCorrelationID(r.Context(), correlationID)
		if causationID != "" {
			ctx = contextx.WithCausationID(ctx, causationID)
		}

		w.Header().Set("X-Correlation-ID", correlationID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
