package authinfra

import (
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics records authentication observability signals.
type Metrics struct {
	loginAttempts        prometheus.Counter
	loginFailures        *prometheus.CounterVec
	tokenValidations     *prometheus.CounterVec
	tokenRefreshes       *prometheus.CounterVec
	refreshSuccess       prometheus.Counter
	refreshRotation      prometheus.Counter
	refreshReplay        prometheus.Counter
	refreshFamilyRevoked prometheus.Counter
	apiKeyAuthSuccess    prometheus.Counter
	apiKeyAuthFailure    *prometheus.CounterVec
	apiKeyRotation       prometheus.Counter
	apiKeyCreation       prometheus.Counter
	apiKeyRevoked        prometheus.Counter
	authLatency          *prometheus.HistogramVec
}

// NewMetrics registers authentication metrics.
func NewMetrics(service string) *Metrics {
	return &Metrics{
		loginAttempts: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_auth_login_attempts_total",
			Help: "Total login attempts.",
		}),
		loginFailures: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_auth_login_failures_total",
			Help: "Total failed login attempts.",
		}, []string{"reason"}),
		tokenValidations: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_auth_token_validations_total",
			Help: "Total access token validations.",
		}, []string{"result"}),
		tokenRefreshes: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_auth_token_refreshes_total",
			Help: "Total refresh token operations.",
		}, []string{"result"}),
		refreshSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_refresh_success_total",
			Help: "Total successful refresh operations.",
		}),
		refreshRotation: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_refresh_rotation_total",
			Help: "Total refresh token rotations.",
		}),
		refreshReplay: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_refresh_replay_total",
			Help: "Total detected refresh token replay attempts.",
		}),
		refreshFamilyRevoked: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_refresh_family_revoked_total",
			Help: "Total refresh token families revoked.",
		}),
		apiKeyAuthSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_api_key_auth_success_total",
			Help: "Total successful API key authentications.",
		}),
		apiKeyAuthFailure: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_api_key_auth_failure_total",
			Help: "Total failed API key authentications.",
		}, []string{"reason"}),
		apiKeyRotation: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_api_key_rotation_total",
			Help: "Total API key rotations.",
		}),
		apiKeyCreation: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_api_key_creation_total",
			Help: "Total API keys created.",
		}),
		apiKeyRevoked: promauto.NewCounter(prometheus.CounterOpts{
			Name: "halcyon_" + service + "_api_key_revoked_total",
			Help: "Total API keys revoked.",
		}),
		authLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "halcyon_" + service + "_auth_latency_seconds",
			Help:    "Authentication operation latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, []string{"operation"}),
	}
}

func (m *Metrics) RecordLoginAttempt() {
	if m == nil {
		return
	}
	m.loginAttempts.Inc()
}

func (m *Metrics) RecordLoginFailure(reason string) {
	if m == nil {
		return
	}
	m.loginFailures.WithLabelValues(reason).Inc()
	slog.Warn("login failed", "reason", reason)
}

func (m *Metrics) RecordTokenValidation(success bool) {
	if m == nil {
		return
	}
	result := "failure"
	if success {
		result = "success"
	}
	m.tokenValidations.WithLabelValues(result).Inc()
}

func (m *Metrics) RecordTokenRefresh(success bool) {
	if m == nil {
		return
	}
	result := "failure"
	if success {
		result = "success"
	}
	m.tokenRefreshes.WithLabelValues(result).Inc()
}

func (m *Metrics) RecordRefreshSuccess() {
	if m == nil {
		return
	}
	m.refreshSuccess.Inc()
}

func (m *Metrics) RecordRefreshRotation() {
	if m == nil {
		return
	}
	m.refreshRotation.Inc()
}

func (m *Metrics) RecordRefreshReplay() {
	if m == nil {
		return
	}
	m.refreshReplay.Inc()
}

func (m *Metrics) RecordFamilyRevoked() {
	if m == nil {
		return
	}
	m.refreshFamilyRevoked.Inc()
}

func (m *Metrics) RecordAPIKeyCreation() {
	if m == nil {
		return
	}
	m.apiKeyCreation.Inc()
}

func (m *Metrics) RecordAPIKeyAuthSuccess() {
	if m == nil {
		return
	}
	m.apiKeyAuthSuccess.Inc()
}

func (m *Metrics) RecordAPIKeyAuthFailure(reason string) {
	if m == nil {
		return
	}
	m.apiKeyAuthFailure.WithLabelValues(reason).Inc()
}

func (m *Metrics) RecordAPIKeyRotation() {
	if m == nil {
		return
	}
	m.apiKeyRotation.Inc()
}

func (m *Metrics) RecordAPIKeyRevoked() {
	if m == nil {
		return
	}
	m.apiKeyRevoked.Inc()
}

func (m *Metrics) ObserveAuthLatency(operation string, start time.Time) {
	if m == nil {
		return
	}
	m.authLatency.WithLabelValues(operation).Observe(time.Since(start).Seconds())
}
