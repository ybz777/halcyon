package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	NotificationsCreated   *prometheus.CounterVec
	NotificationsDeduplicated prometheus.Counter
	NotificationsRead      prometheus.Counter
	DispatchFailures       *prometheus.CounterVec
	EmailDeliveries        *prometheus.CounterVec
	EmailSuccess           *prometheus.CounterVec
	EmailFailures          *prometheus.CounterVec
	EmailRetries           prometheus.Counter
	EmailLatency           prometheus.Histogram
	NotificationQueueDepth prometheus.Gauge
	EmailQueueDepth        prometheus.Gauge
}

func New() *Metrics {
	return &Metrics{
		NotificationsCreated: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "notifications_created_total", Help: "Notifications created",
		}, []string{"notification_type", "channel"}),
		NotificationsDeduplicated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "notifications_deduplicated_total", Help: "Duplicate notifications skipped",
		}),
		NotificationsRead: promauto.NewCounter(prometheus.CounterOpts{
			Name: "notifications_read_total", Help: "Notifications marked read",
		}),
		DispatchFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "notification_dispatch_failures_total", Help: "Notification dispatch failures",
		}, []string{"notification_type", "failure_category"}),
		EmailDeliveries: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "email_deliveries_total", Help: "Email delivery attempts",
		}, []string{"notification_type", "status"}),
		EmailSuccess: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "email_delivery_success_total", Help: "Successful email deliveries",
		}, []string{"provider"}),
		EmailFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "email_delivery_failure_total", Help: "Failed email deliveries",
		}, []string{"failure_category"}),
		EmailRetries: promauto.NewCounter(prometheus.CounterOpts{
			Name: "email_delivery_retries_total", Help: "Email delivery retries",
		}),
		EmailLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name: "email_delivery_latency_seconds", Help: "Email delivery latency",
			Buckets: prometheus.DefBuckets,
		}),
		NotificationQueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "notification_queue_depth", Help: "Ready email deliveries",
		}),
		EmailQueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "email_queue_depth", Help: "Ready email deliveries awaiting worker",
		}),
	}
}
