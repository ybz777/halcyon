package pagination

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics exposes low-cardinality collection query metrics.
type Metrics struct {
	RequestsTotal      *prometheus.CounterVec
	EmptyTotal         *prometheus.CounterVec
	InvalidQueryTotal  *prometheus.CounterVec
}

// NewMetrics registers pagination metrics for a service.
func NewMetrics(service string) *Metrics {
	return &Metrics{
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "api_collection_requests_total",
			Help:        "Collection list requests by route and result.",
			ConstLabels: prometheus.Labels{"service": service},
		}, []string{"route", "result"}),
		EmptyTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "api_collection_empty_total",
			Help:        "Collection list requests that returned zero items.",
			ConstLabels: prometheus.Labels{"service": service},
		}, []string{"route"}),
		InvalidQueryTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "api_invalid_query_total",
			Help:        "Invalid collection query parameters.",
			ConstLabels: prometheus.Labels{"service": service},
		}, []string{"route", "error_type"}),
	}
}

func (m *Metrics) Record(route, result string, empty bool) {
	if m == nil {
		return
	}
	m.RequestsTotal.WithLabelValues(route, result).Inc()
	if empty {
		m.EmptyTotal.WithLabelValues(route).Inc()
	}
}

func (m *Metrics) RecordInvalid(route, errorType string) {
	if m == nil {
		return
	}
	m.InvalidQueryTotal.WithLabelValues(route, errorType).Inc()
}
