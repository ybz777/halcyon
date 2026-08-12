package postgres

import (
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// NewMetrics registers database metrics for the auth service.
func NewMetrics(service string, acquiredConns func() int32) *Metrics {
	return &Metrics{
		queryDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "halcyon_" + service + "_db_query_duration_seconds",
			Help:    "Database query duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
		queryErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_db_query_errors_total",
			Help: "Total database query errors.",
		}, []string{"operation"}),
		poolConns: promauto.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "halcyon_" + service + "_db_pool_acquired_conns",
			Help: "Number of acquired database pool connections.",
		}, func() float64 {
			if acquiredConns == nil {
				return 0
			}
			return float64(acquiredConns())
		}),
	}
}

// Metrics records database observability signals.
type Metrics struct {
	queryDuration *prometheus.HistogramVec
	queryErrors   *prometheus.CounterVec
	poolConns     prometheus.GaugeFunc
}

func (m *Metrics) observe(operation string, start time.Time, err error) {
	if m == nil {
		return
	}

	m.queryDuration.WithLabelValues(operation).Observe(time.Since(start).Seconds())
	if err != nil {
		m.queryErrors.WithLabelValues(operation).Inc()
		slog.Error("database query failed",
			"operation", operation,
			"error", err,
		)
	}
}
