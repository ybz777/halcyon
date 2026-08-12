package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Summary is a safe startup configuration summary.
type Summary struct {
	Environment     string            `json:"environment"`
	ServiceName     string            `json:"service_name"`
	Version         string            `json:"version"`
	HTTPPort        string            `json:"http_port,omitempty"`
	DatabaseURL     string            `json:"database_url,omitempty"`
	RedisConfigured bool              `json:"redis_configured"`
	KafkaConfigured bool              `json:"kafka_configured"`
	RateLimit       bool              `json:"rate_limit_enabled"`
	Idempotency     bool              `json:"idempotency_enabled"`
	JobsEnabled     bool              `json:"jobs_enabled"`
	Tracing         bool              `json:"tracing_enabled"`
	Features        map[string]string `json:"features,omitempty"`
}

// NewSummary builds a redacted startup summary.
func NewSummary(app App, opts ...SummaryOption) Summary {
	s := Summary{
		Environment: string(app.Environment),
		ServiceName: app.ServiceName,
		Version:     app.Version,
		Features:    make(map[string]string),
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// SummaryOption configures summary fields.
type SummaryOption func(*Summary)

func WithHTTPPort(port string) SummaryOption {
	return func(s *Summary) { s.HTTPPort = port }
}

func WithDatabaseURL(url string) SummaryOption {
	return func(s *Summary) {
		if url != "" {
			s.DatabaseURL = RedactURL(url)
		}
	}
}

func WithRedis(url string) SummaryOption {
	return func(s *Summary) { s.RedisConfigured = strings.TrimSpace(url) != "" }
}

func WithKafka(brokers string) SummaryOption {
	return func(s *Summary) { s.KafkaConfigured = strings.TrimSpace(brokers) != "" }
}

func WithRateLimit(enabled bool) SummaryOption {
	return func(s *Summary) { s.RateLimit = enabled }
}

func WithIdempotency(enabled bool) SummaryOption {
	return func(s *Summary) { s.Idempotency = enabled }
}

func WithJobs(enabled bool) SummaryOption {
	return func(s *Summary) { s.JobsEnabled = enabled }
}

func WithTracing(enabled bool) SummaryOption {
	return func(s *Summary) { s.Tracing = enabled }
}

func WithFeature(key, value string) SummaryOption {
	return func(s *Summary) { s.Features[key] = value }
}

// String returns JSON representation safe for logging.
func (s Summary) String() string {
	b, err := json.Marshal(s)
	if err != nil {
		return fmt.Sprintf(`{"service_name":%q,"error":"summary marshal failed"}`, s.ServiceName)
	}
	return string(b)
}

// MarshalJSON implements json.Marshaler.
func (s Summary) MarshalJSON() ([]byte, error) {
	type alias Summary
	return json.Marshal(alias(s))
}
