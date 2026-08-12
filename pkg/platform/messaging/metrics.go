package messaging

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics exposes platform event metrics.
type Metrics struct {
	outboxCreated   *prometheus.CounterVec
	outboxPublished *prometheus.CounterVec
	outboxFailures  *prometheus.CounterVec
	outboxPending   prometheus.Gauge
	outboxLatency   *prometheus.HistogramVec
	consumed        *prometheus.CounterVec
	consumerErrors  *prometheus.CounterVec
	processingDur   *prometheus.HistogramVec
	dlq             *prometheus.CounterVec
	published       *prometheus.CounterVec
	duplicates      *prometheus.CounterVec
	retries         *prometheus.CounterVec
	consumeFail     *prometheus.CounterVec
	eventDLQ        *prometheus.CounterVec
	eventConsumed   *prometheus.CounterVec
}

// NewMetrics registers event metrics.
func NewMetrics() *Metrics {
	return &Metrics{
		outboxCreated: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "outbox_events_created_total",
			Help: "Total outbox events created.",
		}, []string{"event_type"}),
		outboxPublished: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "outbox_events_published_total",
			Help: "Total outbox events published to Kafka.",
		}, []string{"event_type", "topic"}),
		outboxFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "outbox_publish_failures_total",
			Help: "Total outbox publish failures.",
		}, []string{"event_type"}),
		outboxPending: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "outbox_pending_events",
			Help: "Current number of pending outbox entries.",
		}),
		outboxLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "outbox_publish_latency_seconds",
			Help:    "Outbox relay publish latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"event_type"}),
		consumed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total Kafka messages consumed.",
		}, []string{"event_type", "topic"}),
		consumerErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "kafka_consumer_errors_total",
			Help: "Total Kafka consumer errors.",
		}, []string{"event_type", "stage"}),
		processingDur: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "consumer_processing_duration_seconds",
			Help:    "Event handler processing duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"event_type"}),
		dlq: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "dead_letter_events_total",
			Help: "Total events sent to dead letter queues.",
		}, []string{"topic", "event_type"}),
		published: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "events_published_total",
			Help: "Total events published to Kafka.",
		}, []string{"event_type", "topic"}),
		duplicates: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "events_duplicate_total",
			Help: "Duplicate events skipped by consumers.",
		}, []string{"event_type", "consumer_group"}),
		retries: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "events_retry_total",
			Help: "Consumer handler retries.",
		}, []string{"event_type", "consumer_group"}),
		consumeFail: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "events_consume_failures_total",
			Help: "Consumer processing failures.",
		}, []string{"event_type", "failure_category"}),
		eventDLQ: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "events_dlq_total",
			Help: "Events routed to DLQ by consumers.",
		}, []string{"topic", "event_type", "failure_category"}),
		eventConsumed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "events_consumed_total",
			Help: "Successfully consumed events.",
		}, []string{"event_type", "topic"}),
	}
}

func (m *Metrics) RecordOutboxCreated(eventType string) {
	m.outboxCreated.WithLabelValues(eventType).Inc()
}

func (m *Metrics) RecordOutboxPublished(eventType, topic string) {
	m.outboxPublished.WithLabelValues(eventType, topic).Inc()
	m.published.WithLabelValues(eventType, topic).Inc()
}

func (m *Metrics) RecordOutboxPublishFailure(eventType string) {
	m.outboxFailures.WithLabelValues(eventType).Inc()
}

func (m *Metrics) SetOutboxPending(size int64) {
	m.outboxPending.Set(float64(size))
}

func (m *Metrics) ObserveOutboxPublishLatency(eventType string, d time.Duration) {
	m.outboxLatency.WithLabelValues(eventType).Observe(d.Seconds())
}

func (m *Metrics) RecordConsumed(eventType, topic string) {
	m.consumed.WithLabelValues(eventType, topic).Inc()
}

func (m *Metrics) RecordEventConsumed(eventType, topic string) {
	m.eventConsumed.WithLabelValues(eventType, topic).Inc()
}

func (m *Metrics) ObserveLatency(eventType string, d time.Duration) {
	m.processingDur.WithLabelValues(eventType).Observe(d.Seconds())
}

func (m *Metrics) RecordFailure(eventType, stage string) {
	m.consumerErrors.WithLabelValues(eventType, stage).Inc()
}

func (m *Metrics) RecordConsumeFailure(eventType, category string) {
	m.consumeFail.WithLabelValues(eventType, category).Inc()
}

func (m *Metrics) RecordDeadLetter(topic, eventType string) {
	m.dlq.WithLabelValues(topic, eventType).Inc()
}

func (m *Metrics) RecordDuplicate(eventType, consumerGroup string) {
	m.duplicates.WithLabelValues(eventType, consumerGroup).Inc()
}

func (m *Metrics) RecordRetry(eventType, consumerGroup string) {
	m.retries.WithLabelValues(eventType, consumerGroup).Inc()
}

func (m *Metrics) RecordEventDLQ(topic, eventType, category string) {
	m.eventDLQ.WithLabelValues(topic, eventType, category).Inc()
}

func (m *Metrics) RecordPublished(eventType, topic string) {
	m.RecordOutboxPublished(eventType, topic)
}

func (m *Metrics) SetOutboxSize(size int64) {
	m.SetOutboxPending(size)
}

func (m *Metrics) RecordDLQ(topic, eventType string) {
	m.RecordDeadLetter(topic, eventType)
}
