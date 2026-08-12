package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// GatewayMetrics records gateway observability signals.
type GatewayMetrics struct {
	requests         *prometheus.CounterVec
	duration         *prometheus.HistogramVec
	inFlight         *prometheus.GaugeVec
	upstreamErrors   *prometheus.CounterVec
	upstreamLatency  *prometheus.HistogramVec
	rateLimited      *prometheus.CounterVec
		authFailures     *prometheus.CounterVec
		authzFailures    *prometheus.CounterVec
		circuitOpen      *prometheus.CounterVec
		jwksRefresh      prometheus.Counter
		jwksRefreshFails prometheus.Counter
	bodyRejected     prometheus.Counter
}

// NewGatewayMetrics registers gateway Prometheus metrics.
func NewGatewayMetrics() *GatewayMetrics {
	return &GatewayMetrics{
		requests: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Total gateway requests processed.",
		}, []string{"method", "route", "status", "service"}),
		duration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Gateway request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "service"}),
		inFlight: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gateway_requests_in_flight",
			Help: "In-flight gateway upstream requests.",
		}, []string{"service"}),
		upstreamErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_upstream_errors_total",
			Help: "Upstream errors observed by the gateway.",
		}, []string{"service", "reason"}),
		upstreamLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gateway_upstream_latency_seconds",
			Help:    "Upstream latency observed by the gateway.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "service"}),
		rateLimited: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_rate_limited_total",
			Help: "Requests rejected by rate limiting.",
		}, []string{"route"}),
		authFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_authentication_failures_total",
			Help: "Authentication failures at the gateway.",
		}, []string{"reason"}),
		authzFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_authorization_failures_total",
			Help: "Authorization failures at the gateway.",
		}, []string{"reason"}),
		circuitOpen: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_circuit_open_total",
			Help: "Circuit breaker opened events.",
		}, []string{"service"}),
		jwksRefresh: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gateway_jwks_refresh_total",
			Help: "Successful JWKS refresh operations.",
		}),
		jwksRefreshFails: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gateway_jwks_refresh_failures_total",
			Help: "Failed JWKS refresh operations.",
		}),
		bodyRejected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gateway_request_body_rejected_total",
			Help: "Requests rejected for oversized bodies.",
		}),
	}
}

func (m *GatewayMetrics) RecordRequest(method, route, service, status string) {
	m.requests.WithLabelValues(method, route, status, service).Inc()
}

func (m *GatewayMetrics) ObserveDuration(method, route, service string, duration time.Duration) {
	m.duration.WithLabelValues(method, route, service).Observe(duration.Seconds())
	m.upstreamLatency.WithLabelValues(method, route, service).Observe(duration.Seconds())
}

func (m *GatewayMetrics) IncInFlight(service string, delta int) {
	m.inFlight.WithLabelValues(service).Add(float64(delta))
}

func (m *GatewayMetrics) RecordUpstreamError(service, reason string) {
	m.upstreamErrors.WithLabelValues(service, reason).Inc()
}

func (m *GatewayMetrics) RecordRateLimited(routeClass string) {
	m.rateLimited.WithLabelValues(routeClass).Inc()
}

func (m *GatewayMetrics) RecordAuthFailure(reason string) {
	m.authFailures.WithLabelValues(reason).Inc()
}

func (m *GatewayMetrics) RecordCircuitOpen(service string) {
	m.circuitOpen.WithLabelValues(service).Inc()
}

func (m *GatewayMetrics) RecordJWKSRefresh(success bool) {
	if success {
		m.jwksRefresh.Inc()
	} else {
		m.jwksRefreshFails.Inc()
	}
}

func (m *GatewayMetrics) RecordBodyRejected() {
	m.bodyRejected.Inc()
}
