package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics exposes low-cardinality Prometheus counters for rate limiting.
type Metrics struct {
	RequestsTotal         *prometheus.CounterVec
	RejectedTotal         *prometheus.CounterVec
	RedisErrors           prometheus.Counter
	BodyRejectedTotal     prometheus.Counter
	ConcurrencyRejected   prometheus.Counter
}

// NewMetrics registers rate-limit metrics for a service.
func NewMetrics(service string) *Metrics {
	return &Metrics{
		RequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rate_limit_requests_total",
			Help: "Rate limit checks by route, method, policy, and result.",
			ConstLabels: prometheus.Labels{"service": service},
		}, []string{"route", "method", "policy", "result"}),
		RejectedTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "rate_limit_rejected_total",
			Help: "Rate limit rejections by route, method, and policy.",
			ConstLabels: prometheus.Labels{"service": service},
		}, []string{"route", "method", "policy"}),
		RedisErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "rate_limit_redis_errors_total",
			Help:        "Redis errors during rate limiting.",
			ConstLabels: prometheus.Labels{"service": service},
		}),
		BodyRejectedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "request_body_rejected_total",
			Help:        "Requests rejected due to body size limits.",
			ConstLabels: prometheus.Labels{"service": service},
		}),
		ConcurrencyRejected: promauto.NewCounter(prometheus.CounterOpts{
			Name:        "concurrency_limit_rejected_total",
			Help:        "Requests rejected due to concurrency limits.",
			ConstLabels: prometheus.Labels{"service": service},
		}),
	}
}

func (m *Metrics) Record(route, method, policy string, allowed bool) {
	if m == nil {
		return
	}
	result := "allowed"
	if !allowed {
		result = "rejected"
		m.RejectedTotal.WithLabelValues(route, method, policy).Inc()
	}
	m.RequestsTotal.WithLabelValues(route, method, policy, result).Inc()
}
