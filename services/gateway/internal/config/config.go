package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// Config holds gateway runtime configuration.
type Config struct {
	Port                    string
	LogLevel                string
	LogFormat               string
	ServiceName             string
	AuthServiceURL          string
	OrgServiceURL           string
	AuditServiceURL         string
	IntegrationsServiceURL  string
	NotificationsServiceURL string
	JobsServiceURL          string
	KafkaBrokers            string
	JWKSURL                 string
	JWSCacheTTL             time.Duration
	JWTIssuer               string
	JWTAudience             string
	RedisURL                string
	RateLimitEnabled        bool
	RateLimitFailClosed     bool
	RateLimitTrustProxy     bool
	RateLimitRedisPrefix    string
	RateLimitFailureMode    string
	RateLimitDefault        int
	RateLimitAuth           int
	RateLimitInvitation     int
	RateLimitWorkflow       int
	RateLimitWindow         time.Duration
	RateLimitAuthWindow     time.Duration
	RateLimitWorkflowWindow time.Duration
	WorkflowConcurrency     int
	MaxRequestBodyBytes     int64
	UpstreamTimeout         time.Duration
	UpstreamTLSInsecure     bool
	CircuitBreakerEnabled   bool
	CircuitFailureThreshold int
	CircuitOpenDuration     time.Duration
	ServiceIdentity         string
	ServicePrivateKey       string
	ServiceKeyID            string
	ServiceTokenAudience    string
	ServiceTokenTTL         time.Duration
	SecurityHeaders         SecurityHeadersConfig
}

// SecurityHeadersConfig controls response security headers.
type SecurityHeadersConfig struct {
	ContentTypeOptions string
	FrameOptions       string
	ReferrerPolicy     string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	env := platformconfig.FromEnv()
	cfg := Config{
		Port:                    env.String("GATEWAY_PORT", "8088"),
		LogLevel:                env.String("LOG_LEVEL", "info"),
		LogFormat:               env.String("LOG_FORMAT", "json"),
		ServiceName:             env.String("SERVICE_NAME", "gateway"),
		AuthServiceURL:          env.String("AUTH_SERVICE_URL", ""),
		OrgServiceURL:           env.String("ORG_SERVICE_URL", ""),
		AuditServiceURL:         env.String("AUDIT_SERVICE_URL", ""),
		IntegrationsServiceURL:  env.String("INTEGRATIONS_SERVICE_URL", ""),
		NotificationsServiceURL: env.String("NOTIFICATIONS_SERVICE_URL", ""),
		JobsServiceURL:          env.String("JOBS_SERVICE_URL", ""),
		KafkaBrokers:            env.String("KAFKA_BROKERS", ""),
		JWKSURL:                 env.String("JWKS_URL", ""),
		JWSCacheTTL:             env.Duration("JWKS_CACHE_TTL", 5*time.Minute),
		JWTIssuer:               env.String("JWT_ISSUER", ""),
		JWTAudience:             env.String("JWT_AUDIENCE", ""),
		RedisURL:                env.String("REDIS_URL", ""),
		RateLimitEnabled:        env.Bool("RATE_LIMIT_ENABLED", true),
		RateLimitFailClosed:     env.Bool("RATE_LIMIT_FAIL_CLOSED", true),
		RateLimitTrustProxy:     env.Bool("RATE_LIMIT_TRUST_PROXY", false),
		RateLimitRedisPrefix:    env.String("RATE_LIMIT_REDIS_PREFIX", "gateway:ratelimit"),
		RateLimitFailureMode:    env.String("RATE_LIMIT_FAILURE_MODE", "closed"),
		RateLimitDefault:        env.FirstPositiveInt(300, "RATE_LIMIT_DEFAULT_REQUESTS", "RATE_LIMIT_DEFAULT"),
		RateLimitAuth:           env.FirstPositiveInt(20, "RATE_LIMIT_AUTH_REQUESTS", "RATE_LIMIT_AUTH"),
		RateLimitInvitation:     env.Int("RATE_LIMIT_INVITATION", 30),
		RateLimitWorkflow:       env.Int("RATE_LIMIT_WORKFLOW_REQUESTS", 30),
		RateLimitWindow:         env.FirstPositiveDuration(time.Minute, "RATE_LIMIT_DEFAULT_WINDOW", "RATE_LIMIT_WINDOW"),
		RateLimitAuthWindow:     env.FirstPositiveDuration(time.Minute, "RATE_LIMIT_AUTH_WINDOW", "RATE_LIMIT_WINDOW"),
		RateLimitWorkflowWindow: env.Duration("RATE_LIMIT_WORKFLOW_WINDOW", time.Minute),
		WorkflowConcurrency:     env.Int("WORKFLOW_CONCURRENCY_LIMIT", 8),
		MaxRequestBodyBytes:     int64(env.FirstPositiveInt(1<<20, "REQUEST_MAX_BODY_BYTES", "MAX_REQUEST_BODY_BYTES")),
		UpstreamTimeout:         env.Duration("UPSTREAM_TIMEOUT", 30*time.Second),
		UpstreamTLSInsecure:     env.Bool("UPSTREAM_TLS_INSECURE", false),
		CircuitBreakerEnabled:   env.Bool("CIRCUIT_BREAKER_ENABLED", true),
		CircuitFailureThreshold: env.Int("CIRCUIT_FAILURE_THRESHOLD", 5),
		CircuitOpenDuration:     env.Duration("CIRCUIT_OPEN_DURATION", 30*time.Second),
		ServiceIdentity:         env.String("SERVICE_IDENTITY", "gateway"),
		ServicePrivateKey:       env.Secret("SERVICE_PRIVATE_KEY").Value(),
		ServiceKeyID:            env.String("SERVICE_KEY_ID", "gateway-1"),
		ServiceTokenAudience:    env.String("SERVICE_TOKEN_AUDIENCE", "halcyon-internal"),
		ServiceTokenTTL:         env.Duration("SERVICE_TOKEN_TTL", 5*time.Minute),
		SecurityHeaders: SecurityHeadersConfig{
			ContentTypeOptions: env.String("SECURITY_HEADER_CONTENT_TYPE_OPTIONS", "nosniff"),
			FrameOptions:       env.String("SECURITY_HEADER_FRAME_OPTIONS", "DENY"),
			ReferrerPolicy:     env.String("SECURITY_HEADER_REFERRER_POLICY", "no-referrer"),
		},
	}
	if cfg.JWKSURL == "" && cfg.AuthServiceURL != "" {
		cfg.JWKSURL = strings.TrimRight(cfg.AuthServiceURL, "/") + "/api/v1/auth/jwks"
	}
	if cfg.JWTIssuer == "" {
		cfg.JWTIssuer = "auth"
	}
	if err := platformconfig.ValidateProduction(env); err != nil {
		return Config{}, err
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	port, err := platformconfig.ParsePort("GATEWAY_PORT", c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid GATEWAY_PORT")
	}
	if c.AuthServiceURL == "" {
		return fmt.Errorf("AUTH_SERVICE_URL is required")
	}
	if c.OrgServiceURL == "" {
		return fmt.Errorf("ORG_SERVICE_URL is required")
	}
	if err := validateServiceURL(c.AuthServiceURL); err != nil {
		return fmt.Errorf("AUTH_SERVICE_URL: %w", err)
	}
	if err := validateServiceURL(c.OrgServiceURL); err != nil {
		return fmt.Errorf("ORG_SERVICE_URL: %w", err)
	}
	if c.JWKSURL == "" {
		return fmt.Errorf("JWKS_URL is required")
	}
	if c.RateLimitEnabled && c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required when RATE_LIMIT_ENABLED=true")
	}
	if c.RateLimitDefault <= 0 || c.RateLimitAuth <= 0 || c.RateLimitWorkflow <= 0 {
		return fmt.Errorf("rate limit request counts must be positive")
	}
	if c.RateLimitWindow <= 0 || c.RateLimitAuthWindow <= 0 || c.RateLimitWorkflowWindow <= 0 {
		return fmt.Errorf("rate limit windows must be positive durations")
	}
	if c.RateLimitFailureMode != "open" && c.RateLimitFailureMode != "closed" {
		return fmt.Errorf("RATE_LIMIT_FAILURE_MODE must be open or closed")
	}
	if c.MaxRequestBodyBytes <= 0 {
		return fmt.Errorf("MAX_REQUEST_BODY_BYTES must be positive")
	}
	if c.UpstreamTimeout <= 0 {
		return fmt.Errorf("UPSTREAM_TIMEOUT must be positive")
	}
	if strings.TrimSpace(c.ServicePrivateKey) == "" {
		return fmt.Errorf("SERVICE_PRIVATE_KEY is required")
	}
	return nil
}

func validateServiceURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}
