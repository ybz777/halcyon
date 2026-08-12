package auth

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics records low-cardinality authentication and authorization counters.
type Metrics interface {
	RecordAuthSuccess(method, route, authMethod string)
	RecordAuthFailure(method, route, authMethod, category string)
	RecordAuthorizationAllowed(route, permission string)
	RecordAuthorizationDenied(route, permission string)
}

// NoopMetrics discards auth metrics.
type NoopMetrics struct{}

func (NoopMetrics) RecordAuthSuccess(string, string, string)         {}
func (NoopMetrics) RecordAuthFailure(string, string, string, string) {}
func (NoopMetrics) RecordAuthorizationAllowed(string, string)        {}
func (NoopMetrics) RecordAuthorizationDenied(string, string)         {}

// PrometheusMetrics registers auth counters for a service.
type PrometheusMetrics struct {
	attemptsTotal        *prometheus.CounterVec
	failuresTotal        *prometheus.CounterVec
	authorizationAllowed *prometheus.CounterVec
	authorizationDenied  *prometheus.CounterVec
}

// NewPrometheusMetrics creates service-scoped auth metrics.
func NewPrometheusMetrics(service string) *PrometheusMetrics {
	labels := prometheus.Labels{"service": service}
	return &PrometheusMetrics{
		attemptsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "auth_attempts_total",
			Help:        "Authentication attempts by route and method.",
			ConstLabels: labels,
		}, []string{"route", "auth_method"}),
		failuresTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "auth_failures_total",
			Help:        "Authentication failures by route, method, and category.",
			ConstLabels: labels,
		}, []string{"route", "auth_method", "failure_category"}),
		authorizationAllowed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "authorization_allowed_total",
			Help:        "Authorization checks that allowed access.",
			ConstLabels: labels,
		}, []string{"route", "permission"}),
		authorizationDenied: promauto.NewCounterVec(prometheus.CounterOpts{
			Name:        "authorization_denied_total",
			Help:        "Authorization checks that denied access.",
			ConstLabels: labels,
		}, []string{"route", "permission"}),
	}
}

func (m *PrometheusMetrics) RecordAuthSuccess(method, route, authMethod string) {
	m.attemptsTotal.WithLabelValues(route, authMethod).Inc()
}

func (m *PrometheusMetrics) RecordAuthFailure(method, route, authMethod, category string) {
	m.attemptsTotal.WithLabelValues(route, authMethod).Inc()
	m.failuresTotal.WithLabelValues(route, authMethod, category).Inc()
}

func (m *PrometheusMetrics) RecordAuthorizationAllowed(route, permission string) {
	m.authorizationAllowed.WithLabelValues(route, permission).Inc()
}

func (m *PrometheusMetrics) RecordAuthorizationDenied(route, permission string) {
	m.authorizationDenied.WithLabelValues(route, permission).Inc()
}
