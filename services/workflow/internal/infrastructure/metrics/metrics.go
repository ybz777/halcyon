package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
)

// Metrics records workflow service metrics.
type Metrics struct {
	created     prometheus.Counter
	transitions *prometheus.CounterVec
	resolution  prometheus.Histogram
	stateDur    *prometheus.HistogramVec
}

// New creates workflow metrics.
func New() *Metrics {
	return &Metrics{
		created: promauto.NewCounter(prometheus.CounterOpts{
			Name: "incident_created_total",
			Help: "Total incidents created.",
		}),
		transitions: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "incident_transition_total",
			Help: "Total incident workflow transitions.",
		}, []string{"from_status", "to_status"}),
		resolution: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "incident_resolution_time_seconds",
			Help:    "Time from incident creation to resolution in seconds.",
			Buckets: prometheus.DefBuckets,
		}),
		stateDur: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "incident_state_duration_seconds",
			Help:    "Time spent in an incident state before transition.",
			Buckets: prometheus.DefBuckets,
		}, []string{"status"}),
	}
}

func (m *Metrics) RecordIncidentCreated() { m.created.Inc() }

func (m *Metrics) RecordTransition(from, to string) {
	m.transitions.WithLabelValues(from, to).Inc()
}

func (m *Metrics) ObserveResolutionDuration(duration time.Duration) {
	m.resolution.Observe(duration.Seconds())
}

func (m *Metrics) ObserveStateDuration(status string, duration time.Duration) {
	m.stateDur.WithLabelValues(status).Observe(duration.Seconds())
}

var _ ports.WorkflowMetrics = (*Metrics)(nil)
