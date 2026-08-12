package config

import (
	"fmt"
	"strings"
	"time"
)

// Idempotency holds idempotency middleware settings (PR #25).
type Idempotency struct {
	Enabled          bool
	RedisPrefix      string
	InProgressTTL    time.Duration
	CompletedTTL     time.Duration
	MaxResponseBytes int64
	FailClosed       bool
}

// LoadIdempotency reads idempotency configuration.
func LoadIdempotency(env *Env, serviceName string) (Idempotency, error) {
	enabled, err := env.BoolStrict("IDEMPOTENCY_ENABLED", true)
	if err != nil {
		return Idempotency{}, err
	}
	inProgress, err := env.DurationStrict("IDEMPOTENCY_IN_PROGRESS_TTL", 5*time.Minute)
	if err != nil {
		return Idempotency{}, err
	}
	completed, err := env.DurationStrict("IDEMPOTENCY_COMPLETED_TTL", 24*time.Hour)
	if err != nil {
		return Idempotency{}, err
	}
	maxResp, err := env.IntStrict("IDEMPOTENCY_MAX_RESPONSE_SIZE", 64*1024)
	if err != nil {
		return Idempotency{}, err
	}
	failClosed, err := env.BoolStrict("IDEMPOTENCY_FAIL_CLOSED", true)
	if err != nil {
		return Idempotency{}, err
	}
	cfg := Idempotency{
		Enabled:          enabled,
		RedisPrefix:      env.String("IDEMPOTENCY_REDIS_PREFIX", serviceName),
		InProgressTTL:    inProgress,
		CompletedTTL:     completed,
		MaxResponseBytes: int64(maxResp),
		FailClosed:       failClosed,
	}
	if cfg.InProgressTTL <= 0 {
		return Idempotency{}, fmt.Errorf("IDEMPOTENCY_IN_PROGRESS_TTL must be > 0")
	}
	if cfg.CompletedTTL <= 0 {
		return Idempotency{}, fmt.Errorf("IDEMPOTENCY_COMPLETED_TTL must be > 0")
	}
	if cfg.MaxResponseBytes <= 0 {
		return Idempotency{}, fmt.Errorf("IDEMPOTENCY_MAX_RESPONSE_SIZE must be > 0")
	}
	return cfg, nil
}

// RateLimit holds rate limiting settings (PR #23).
type RateLimit struct {
	Enabled             bool
	RedisURL            string
	RedisPrefix         string
	DefaultRequests     int
	DefaultWindow       time.Duration
	AuthRequests        int
	AuthWindow          time.Duration
	WorkflowRequests    int
	WorkflowWindow      time.Duration
	MaxBodyBytes        int64
	WorkflowConcurrency int
	FailureMode         string
	TrustProxy          bool
	FailClosed          bool
}

// LoadRateLimit reads rate limit configuration.
func LoadRateLimit(env *Env, serviceName string) (RateLimit, error) {
	cfg := RateLimit{
		Enabled:             env.Bool("RATE_LIMIT_ENABLED", false),
		RedisURL:            env.get("REDIS_URL"),
		RedisPrefix:         env.String("RATE_LIMIT_REDIS_PREFIX", serviceName+":ratelimit"),
		DefaultRequests:     env.FirstPositiveInt(300, "RATE_LIMIT_DEFAULT_REQUESTS", "RATE_LIMIT_DEFAULT"),
		DefaultWindow:       env.FirstPositiveDuration(time.Minute, "RATE_LIMIT_DEFAULT_WINDOW", "RATE_LIMIT_WINDOW"),
		AuthRequests:        env.FirstPositiveInt(20, "RATE_LIMIT_AUTH_REQUESTS", "RATE_LIMIT_AUTH"),
		AuthWindow:          env.FirstPositiveDuration(time.Minute, "RATE_LIMIT_AUTH_WINDOW", "RATE_LIMIT_WINDOW"),
		WorkflowRequests:    env.Int("RATE_LIMIT_WORKFLOW_REQUESTS", 30),
		WorkflowWindow:      env.Duration("RATE_LIMIT_WORKFLOW_WINDOW", time.Minute),
		MaxBodyBytes:        int64(env.FirstPositiveInt(1<<20, "REQUEST_MAX_BODY_BYTES", "MAX_REQUEST_BODY_BYTES")),
		WorkflowConcurrency: env.Int("WORKFLOW_CONCURRENCY_LIMIT", 8),
		FailureMode:         strings.ToLower(env.String("RATE_LIMIT_FAILURE_MODE", "closed")),
		TrustProxy:          env.Bool("RATE_LIMIT_TRUST_PROXY", false),
		FailClosed:          env.Bool("RATE_LIMIT_FAIL_CLOSED", true),
	}
	if cfg.DefaultRequests <= 0 || cfg.AuthRequests <= 0 || cfg.WorkflowRequests <= 0 {
		return RateLimit{}, fmt.Errorf("rate limit request counts must be positive")
	}
	if cfg.DefaultWindow <= 0 || cfg.AuthWindow <= 0 || cfg.WorkflowWindow <= 0 {
		return RateLimit{}, fmt.Errorf("rate limit windows must be positive")
	}
	if cfg.MaxBodyBytes <= 0 {
		return RateLimit{}, fmt.Errorf("REQUEST_MAX_BODY_BYTES must be positive")
	}
	if cfg.FailureMode != "open" && cfg.FailureMode != "closed" {
		return RateLimit{}, fmt.Errorf("RATE_LIMIT_FAILURE_MODE must be open or closed")
	}
	if cfg.Enabled && cfg.RedisURL == "" {
		return RateLimit{}, fmt.Errorf("REDIS_URL is required when RATE_LIMIT_ENABLED=true")
	}
	if strings.TrimSpace(cfg.RedisPrefix) == "" {
		return RateLimit{}, fmt.Errorf("RATE_LIMIT_REDIS_PREFIX is required")
	}
	return cfg, nil
}

// Jobs holds background job worker settings (PR #26).
type Jobs struct {
	Enabled              bool
	WorkerCount          int
	BatchSize            int
	LeaseDuration        time.Duration
	LeaseRenewInterval   time.Duration
	PollInterval         time.Duration
	ShutdownTimeout      time.Duration
	MaxConcurrent        int
	MaxAttempts          int
	ExecutionTimeout     time.Duration
	RetryBaseDelay       time.Duration
	RetryMaxDelay        time.Duration
	RetryMultiplier      float64
	RetryJitter          float64
	SuccessRetentionDays int
	FailureRetentionDays int
}

// LoadJobs reads job worker configuration.
func LoadJobs(env *Env) (Jobs, error) {
	cfg := Jobs{
		Enabled:              env.Bool("JOB_WORKER_ENABLED", true),
		WorkerCount:          env.Int("JOB_WORKER_COUNT", 1),
		BatchSize:            env.Int("JOB_BATCH_SIZE", 10),
		LeaseDuration:        env.Duration("JOB_LEASE_DURATION", 30*time.Second),
		LeaseRenewInterval:   env.Duration("JOB_LEASE_RENEW_INTERVAL", 10*time.Second),
		PollInterval:         env.Duration("JOB_POLL_INTERVAL", time.Second),
		ShutdownTimeout:      env.Duration("JOB_SHUTDOWN_TIMEOUT", 30*time.Second),
		MaxConcurrent:        env.Int("JOB_MAX_CONCURRENT", 8),
		MaxAttempts:          env.Int("JOB_MAX_ATTEMPTS", 6),
		ExecutionTimeout:     env.Duration("JOB_EXECUTION_TIMEOUT", 5*time.Minute),
		RetryBaseDelay:       env.Duration("JOB_RETRY_BASE_DELAY", time.Second),
		RetryMaxDelay:        env.Duration("JOB_RETRY_MAX_DELAY", 5*time.Minute),
		RetryMultiplier:      env.Float("JOB_RETRY_MULTIPLIER", 2),
		RetryJitter:          env.Float("JOB_RETRY_JITTER", 0.5),
		SuccessRetentionDays: env.Int("JOB_SUCCESS_RETENTION_DAYS", 7),
		FailureRetentionDays: env.Int("JOB_FAILURE_RETENTION_DAYS", 30),
	}
	return cfg, cfg.Validate()
}

// Validate ensures job configuration relationships are valid.
func (c Jobs) Validate() error {
	if c.WorkerCount < 1 {
		return fmt.Errorf("JOB_WORKER_COUNT must be >= 1")
	}
	if c.BatchSize < 1 || c.BatchSize > 500 {
		return fmt.Errorf("JOB_BATCH_SIZE must be between 1 and 500")
	}
	if c.LeaseDuration < time.Second {
		return fmt.Errorf("JOB_LEASE_DURATION must be >= 1s")
	}
	if c.LeaseRenewInterval <= 0 || c.LeaseRenewInterval >= c.LeaseDuration {
		return fmt.Errorf("JOB_LEASE_RENEW_INTERVAL must be > 0 and < JOB_LEASE_DURATION")
	}
	if c.MaxConcurrent < 1 {
		return fmt.Errorf("JOB_MAX_CONCURRENT must be >= 1")
	}
	if c.MaxAttempts < 1 {
		return fmt.Errorf("JOB_MAX_ATTEMPTS must be >= 1")
	}
	if c.ExecutionTimeout < time.Second {
		return fmt.Errorf("JOB_EXECUTION_TIMEOUT must be >= 1s")
	}
	if c.RetryBaseDelay <= 0 {
		return fmt.Errorf("JOB_RETRY_BASE_DELAY must be > 0")
	}
	if c.RetryMaxDelay < c.RetryBaseDelay {
		return fmt.Errorf("JOB_RETRY_MAX_DELAY must be >= JOB_RETRY_BASE_DELAY")
	}
	if c.RetryMultiplier < 1 {
		return fmt.Errorf("JOB_RETRY_MULTIPLIER must be >= 1")
	}
	if c.RetryJitter < 0 || c.RetryJitter > 1 {
		return fmt.Errorf("JOB_RETRY_JITTER must be between 0 and 1")
	}
	return nil
}

// Observability holds tracing and metrics settings.
type Observability struct {
	Enabled         bool
	ServiceName     string
	ServiceVersion  string
	Environment     string
	InstanceID      string
	OTLPEndpoint    string
	OTLPProtocol    string
	Sampler         string
	SamplerArg      float64
	BatchTimeout    time.Duration
	BatchExportMax  int
	MaxQueueSize    int
	ShutdownTimeout time.Duration
	ResourceAttrs   map[string]string
}

// LoadObservability reads OpenTelemetry configuration.
func LoadObservability(env *Env, serviceName string) (Observability, error) {
	cfg := Observability{
		Enabled:         env.Bool("OTEL_ENABLED", false),
		ServiceName:     env.FirstNonEmpty("OTEL_SERVICE_NAME", serviceName, "halcyon"),
		ServiceVersion:  env.String("SERVICE_VERSION", "dev"),
		Environment:     string(env.Environment()),
		InstanceID:      env.FirstNonEmpty("SERVICE_INSTANCE_ID", "HOSTNAME"),
		OTLPEndpoint:    env.get("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OTLPProtocol:    strings.ToLower(env.String("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")),
		Sampler:         strings.ToLower(env.String("OTEL_TRACES_SAMPLER", "parentbased_traceidratio")),
		BatchTimeout:    env.Duration("OTEL_BSP_SCHEDULE_DELAY", 5*time.Second),
		BatchExportMax:  env.Int("OTEL_BSP_MAX_EXPORT_BATCH_SIZE", 512),
		MaxQueueSize:    env.Int("OTEL_BSP_MAX_QUEUE_SIZE", 2048),
		ShutdownTimeout: env.Duration("OTEL_SHUTDOWN_TIMEOUT", 10*time.Second),
		ResourceAttrs:   parseAttrs(env.get("OTEL_RESOURCE_ATTRIBUTES")),
	}
	samplerArg := env.get("OTEL_TRACES_SAMPLER_ARG")
	if samplerArg != "" {
		f, err := parseFloatStrict("OTEL_TRACES_SAMPLER_ARG", samplerArg)
		if err != nil {
			return Observability{}, err
		}
		cfg.SamplerArg = f
	} else {
		cfg.SamplerArg = 1.0
	}
	return cfg, cfg.Validate()
}

func (c Observability) Validate() error {
	switch c.Sampler {
	case "always_on", "always_off", "traceidratio", "parentbased_traceidratio",
		"parentbased_always_on", "parentbased_always_off":
	default:
		return fmt.Errorf("invalid OTEL_TRACES_SAMPLER")
	}
	if c.SamplerArg < 0 || c.SamplerArg > 1 {
		return fmt.Errorf("OTEL_TRACES_SAMPLER_ARG must be between 0 and 1")
	}
	if c.Enabled {
		switch c.OTLPProtocol {
		case "http/protobuf", "grpc":
		default:
			return fmt.Errorf("invalid OTEL_EXPORTER_OTLP_PROTOCOL")
		}
		if c.OTLPEndpoint == "" {
			return fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required when OTEL_ENABLED=true")
		}
	}
	return nil
}

func parseAttrs(raw string) map[string]string {
	out := make(map[string]string)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			out[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return out
}

func parseFloatStrict(name, raw string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(raw, "%f", &f)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected number", name)
	}
	return f, nil
}

// OutboxRelay holds outbox relay worker settings.
type OutboxRelay struct {
	DatabaseURL  string
	KafkaBrokers string
	ClientID     string
	BatchSize    int
	PollInterval time.Duration
}

// LoadOutboxRelay reads outbox relay configuration.
func LoadOutboxRelay(env *Env) (OutboxRelay, error) {
	url, err := env.RequiredString("DATABASE_URL")
	if err != nil {
		return OutboxRelay{}, err
	}
	brokers, err := env.RequiredString("KAFKA_BROKERS")
	if err != nil {
		return OutboxRelay{}, err
	}
	return OutboxRelay{
		DatabaseURL:  url,
		KafkaBrokers: brokers,
		ClientID:     env.String("KAFKA_CLIENT_ID", "outbox-relay"),
		BatchSize:    env.Int("OUTBOX_BATCH_SIZE", 100),
		PollInterval: env.Duration("OUTBOX_POLL_INTERVAL", time.Second),
	}, nil
}
