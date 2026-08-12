package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTPMetrics records request count and duration for a service.
type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// NewHTTPMetrics registers Prometheus metrics for the given service name.
func NewHTTPMetrics(service string) *HTTPMetrics {
	return &HTTPMetrics{
		requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_http_requests_total",
			Help: "Total HTTP requests processed.",
		}, []string{"method", "path", "status"}),
		duration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "halcyon_" + service + "_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
	}
}

// Middleware records request metrics using a route pattern for the path label.
func (m *HTTPMetrics) Middleware(routePattern string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			status := strconv.Itoa(rec.status)
			m.requests.WithLabelValues(r.Method, routePattern, status).Inc()
			m.duration.WithLabelValues(r.Method, routePattern).Observe(time.Since(start).Seconds())
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}
