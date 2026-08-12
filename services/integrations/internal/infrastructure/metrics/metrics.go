package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// IntegrationsMetrics exposes bounded Prometheus metrics for webhooks.
type IntegrationsMetrics struct {
	EndpointsCreated   prometheus.Counter
	DeliveriesTotal    *prometheus.CounterVec
	DeliveryFailures   *prometheus.CounterVec
	DeliveryRetries    prometheus.Counter
	DeliveryLatency    prometheus.Histogram
	EndpointDisabled   prometheus.Counter
	SSRFRejections     prometheus.Counter
	QueueDepth         prometheus.Gauge
}

func NewIntegrationsMetrics() *IntegrationsMetrics {
	return &IntegrationsMetrics{
		EndpointsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "webhook_endpoints_created_total",
			Help: "Total webhook endpoints created",
		}),
		DeliveriesTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "webhook_deliveries_total",
			Help: "Total webhook delivery attempts",
		}, []string{"event_type", "status"}),
		DeliveryFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "webhook_delivery_failures_total",
			Help: "Total webhook delivery failures",
		}, []string{"event_type", "failure_category"}),
		DeliveryRetries: promauto.NewCounter(prometheus.CounterOpts{
			Name: "webhook_delivery_retries_total",
			Help: "Total webhook delivery retries",
		}),
		DeliveryLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "webhook_delivery_latency_seconds",
			Help:    "Webhook delivery latency in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		EndpointDisabled: promauto.NewCounter(prometheus.CounterOpts{
			Name: "webhook_endpoint_disabled_total",
			Help: "Total webhook endpoints auto-disabled",
		}),
		SSRFRejections: promauto.NewCounter(prometheus.CounterOpts{
			Name: "webhook_ssrf_rejections_total",
			Help: "Total webhook SSRF rejections",
		}),
		QueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "webhook_queue_depth",
			Help: "Ready webhook deliveries awaiting processing",
		}),
	}
}
