package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/halcyon-io/halcyon/pkg/platform/httperrors"
	"github.com/halcyon-io/halcyon/services/gateway/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/gateway/internal/domain"
)

// BodyLimit rejects oversized request bodies.
func BodyLimit(maxBytes int64, metrics ports.ProxyMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil && r.ContentLength > maxBytes {
				if metrics != nil {
					metrics.RecordBodyRejected()
				}
				httperrors.WriteProblem(w, httperrors.ProblemDetails{
					Type:     "https://halcyon.io/problems/invalid-request",
					Title:    "Payload Too Large",
					Status:   http.StatusRequestEntityTooLarge,
					Detail:   "request body exceeds maximum allowed size",
					Instance: middleware.GetReqID(r.Context()),
				})
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// RecoverBodyLimit converts MaxBytesError into RFC 7807 responses.
func RecoverBodyLimit(metrics ports.ProxyMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if _, ok := rec.(interface{ Error() string }); ok {
						if metrics != nil {
							metrics.RecordBodyRejected()
						}
						httperrors.WriteProblem(w, httperrors.ProblemDetails{
							Type:     "https://halcyon.io/problems/invalid-request",
							Title:    "Payload Too Large",
							Status:   http.StatusRequestEntityTooLarge,
							Detail:   domain.ErrRequestTooLarge.Error(),
							Instance: middleware.GetReqID(r.Context()),
						})
						return
					}
					panic(rec)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
