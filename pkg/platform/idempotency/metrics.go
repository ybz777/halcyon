package idempotency

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics records idempotency observability signals.
type Metrics struct {
	RequestsTotal    *prometheus.CounterVec
	ReplaysTotal     prometheus.Counter
	ConflictsTotal   prometheus.Counter
	InProgressTotal  prometheus.Counter
	StoreErrorsTotal prometheus.Counter
}

// NewMetrics registers idempotency metrics for a service.
func NewMetrics(service string) *Metrics {
	m := &Metrics{
		RequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "halcyon",
			Subsystem: "idempotency",
			Name:      "requests_total",
			Help:      "Idempotency middleware requests by route, method, policy, and result.",
		}, []string{"route", "method", "policy", "result"}),

		ReplaysTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "halcyon",
			Subsystem: "idempotency",
			Name:      "replays_total",
			Help:      "Idempotent response replays.",
		}),

		ConflictsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "halcyon",
			Subsystem: "idempotency",
			Name:      "conflicts_total",
			Help:      "Idempotency fingerprint conflicts.",
		}),

		InProgressTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "halcyon",
			Subsystem: "idempotency",
			Name:      "in_progress_total",
			Help:      "Concurrent duplicate requests while original is in progress.",
		}),

		StoreErrorsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "halcyon",
			Subsystem: "idempotency",
			Name:      "store_errors_total",
			Help:      "Idempotency store failures.",
		}),
	}

	reg := prometheus.WrapRegistererWith(
		prometheus.Labels{"service": service},
		prometheus.DefaultRegisterer,
	)

	reg.MustRegister(
		m.RequestsTotal,
		m.ReplaysTotal,
		m.ConflictsTotal,
		m.InProgressTotal,
		m.StoreErrorsTotal,
	)

	return m
}

func (m *Metrics) Record(route, method string, policy Policy, result string) {
	if m == nil {
		return
	}

	m.RequestsTotal.
		WithLabelValues(route, method, string(policy), result).
		Inc()
}