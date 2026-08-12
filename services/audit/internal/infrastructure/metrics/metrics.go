package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Collector records audit service metrics with bounded labels.
type Collector struct {
	consumed    *prometheus.CounterVec
	persisted   *prometheus.CounterVec
	failed      *prometheus.CounterVec
	duplicate   *prometheus.CounterVec
	persistence *prometheus.HistogramVec
	query       *prometheus.HistogramVec
	queryErrors prometheus.Counter
}

// NewCollector creates audit metrics.
func NewCollector() *Collector {
	return &Collector{
		consumed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_events_consumed_total",
			Help: "Audit events consumed from Kafka.",
		}, []string{"event_type"}),
		persisted: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_events_persisted_total",
			Help: "Audit events persisted to the database.",
		}, []string{"event_type", "source_service", "result"}),
		failed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_events_failed_total",
			Help: "Audit event persistence failures.",
		}, []string{"event_type"}),
		duplicate: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_events_duplicate_total",
			Help: "Duplicate audit events skipped.",
		}, []string{"event_type"}),
		persistence: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "audit_persistence_duration_seconds",
			Help:    "Audit event persistence duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"event_type"}),
		query: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "audit_query_duration_seconds",
			Help:    "Audit query duration.",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
		queryErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "audit_query_errors_total",
			Help: "Audit query errors.",
		}),
	}
}

func (c *Collector) RecordConsumed(eventType string) {
	c.consumed.WithLabelValues(eventType).Inc()
}

func (c *Collector) RecordPersisted(eventType, sourceService, result string) {
	c.persisted.WithLabelValues(eventType, sourceService, result).Inc()
}

func (c *Collector) RecordFailed(eventType string) {
	c.failed.WithLabelValues(eventType).Inc()
}

func (c *Collector) RecordDuplicate(eventType string) {
	c.duplicate.WithLabelValues(eventType).Inc()
}

func (c *Collector) ObservePersistence(eventType string, seconds float64) {
	c.persistence.WithLabelValues(eventType).Observe(seconds)
}

func (c *Collector) ObserveQuery(operation string, start time.Time) {
	c.query.WithLabelValues(operation).Observe(time.Since(start).Seconds())
}

func (c *Collector) RecordQueryError() {
	c.queryErrors.Inc()
}
