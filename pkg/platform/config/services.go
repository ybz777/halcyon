package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AuthServiceConfig is the shared loader for the auth HTTP service.
type AuthServiceConfig struct {
	App           App
	Logging       Logging
	HTTP          HTTP
	Database      Database
	Redis         Redis
	Auth          Auth
	RateLimit     RateLimit
	Idempotency   Idempotency
	Observability Observability
}

// LoadAuthService loads auth service configuration.
func LoadAuthService() (AuthServiceConfig, error) {
	env := FromEnv()
	logging, err := LoadLogging(env)
	if err != nil {
		return AuthServiceConfig{}, err
	}
	http, err := LoadHTTP(env, "AUTH_PORT", 8080)
	if err != nil {
		return AuthServiceConfig{}, err
	}
	db, err := LoadDatabase(env, "DATABASE_URL", "")
	if err != nil {
		return AuthServiceConfig{}, err
	}
	auth, err := LoadAuth(env)
	if err != nil {
		return AuthServiceConfig{}, err
	}
	rateLimit, err := LoadRateLimit(env, "auth")
	if err != nil {
		return AuthServiceConfig{}, err
	}
	idempotency, err := LoadIdempotency(env, "auth")
	if err != nil {
		return AuthServiceConfig{}, err
	}
	observability, err := LoadObservability(env, "auth")
	if err != nil {
		return AuthServiceConfig{}, err
	}
	cfg := AuthServiceConfig{
		App:           LoadApp(env, "auth"),
		Logging:       logging,
		HTTP:          http,
		Database:      db,
		Redis:         LoadRedis(env),
		Auth:          auth,
		RateLimit:     rateLimit,
		Idempotency:   idempotency,
		Observability: observability,
	}
	if err := ValidateAuth(cfg.Auth, cfg.Database.URL != "", env); err != nil {
		return AuthServiceConfig{}, err
	}
	if err := ValidateProduction(env); err != nil {
		return AuthServiceConfig{}, err
	}
	if cfg.RateLimit.Enabled && cfg.Redis.URL == "" {
		return AuthServiceConfig{}, fmt.Errorf("REDIS_URL is required when RATE_LIMIT_ENABLED=true")
	}
	return cfg, nil
}

// Summary returns a safe startup summary for auth.
func (c AuthServiceConfig) Summary() Summary {
	return NewSummary(c.App,
		WithHTTPPort(c.HTTP.PortString()),
		WithDatabaseURL(c.Database.URL),
		WithRedis(c.Redis.URL),
		WithRateLimit(c.RateLimit.Enabled),
		WithIdempotency(c.Idempotency.Enabled),
		WithTracing(c.Observability.Enabled),
	)
}

// OrganizationServiceConfig is the shared loader for the organization service.
type OrganizationServiceConfig struct {
	App         App
	Logging     Logging
	HTTP        HTTP
	Database    Database
	Redis       Redis
	AuthService string
	JWTSecret   Secret
	JWTIssuer   string
	JWTAudience string
	Idempotency Idempotency
}

// LoadOrganizationService loads organization service configuration.
func LoadOrganizationService() (OrganizationServiceConfig, error) {
	env := FromEnv()
	logging, err := LoadLogging(env)
	if err != nil {
		return OrganizationServiceConfig{}, err
	}
	http, err := LoadHTTP(env, "ORGANIZATION_PORT", 8081)
	if err != nil {
		return OrganizationServiceConfig{}, err
	}
	db, err := LoadDatabase(env, "ORGANIZATION_DATABASE_URL", "ORGANIZATION_")
	if err != nil {
		return OrganizationServiceConfig{}, err
	}
	idempotency, err := LoadIdempotency(env, "organization")
	if err != nil {
		return OrganizationServiceConfig{}, err
	}
	jwt := env.Secret("JWT_SECRET")
	if db.URL != "" {
		if err := ValidateSecretMinLength("JWT_SECRET", jwt, 32); err != nil {
			return OrganizationServiceConfig{}, err
		}
	}
	if err := ValidateProduction(env); err != nil {
		return OrganizationServiceConfig{}, err
	}
	return OrganizationServiceConfig{
		App:         LoadApp(env, "organization"),
		Logging:     logging,
		HTTP:        http,
		Database:    db,
		Redis:       Redis{URL: env.String("REDIS_URL", "redis://localhost:6379/0")},
		AuthService: env.String("AUTH_SERVICE_NAME", "auth"),
		JWTSecret:   jwt,
		JWTIssuer:   env.String("JWT_ISSUER", env.String("AUTH_SERVICE_NAME", "auth")),
		JWTAudience: env.String("JWT_AUDIENCE", ""),
		Idempotency: idempotency,
	}, nil
}

// OutboxRelayConfig alias for backward compatibility.
type OutboxRelayConfig = OutboxRelay

// MigrateConfig holds database migration CLI configuration.
type MigrateConfig struct {
	DatabaseURL string
}

// LoadMigrate loads migration configuration for a given URL key.
func LoadMigrate(urlKey string) (MigrateConfig, error) {
	url, err := FromEnv().RequiredString(urlKey)
	if err != nil {
		return MigrateConfig{}, err
	}
	return MigrateConfig{DatabaseURL: url}, nil
}

// WorkerConfig holds common worker process settings.
type WorkerConfig struct {
	App      App
	Logging  Logging
	Database Database
	Kafka    Kafka
	Jobs     Jobs
}

// LoadConsumerWorker loads settings for Kafka consumer workers.
func LoadConsumerWorker(serviceName, dbURLKey string) (WorkerConfig, error) {
	env := FromEnv()
	logging, err := LoadLogging(env)
	if err != nil {
		return WorkerConfig{}, err
	}
	db, err := LoadDatabase(env, dbURLKey, "")
	if err != nil {
		return WorkerConfig{}, err
	}
	jobs, err := LoadJobs(env)
	if err != nil {
		return WorkerConfig{}, err
	}
	return WorkerConfig{
		App:      LoadApp(env, serviceName),
		Logging:  logging,
		Database: db,
		Kafka:    LoadKafka(env),
		Jobs:     jobs,
	}, nil
}

// DispatcherConfig holds Kafka dispatcher worker settings.
type DispatcherConfig struct {
	DatabaseURL         string
	KafkaBrokers        string
	JobsDatabaseURL     string
	ConsumerConcurrency int
	DLQEnabled          bool
}

// LoadDispatcher reads dispatcher configuration.
func LoadDispatcher(env *Env, databaseURLKey string) (DispatcherConfig, error) {
	url, err := env.RequiredString(databaseURLKey)
	if err != nil {
		return DispatcherConfig{}, err
	}
	brokers, err := env.RequiredString("KAFKA_BROKERS")
	if err != nil {
		return DispatcherConfig{}, err
	}
	return DispatcherConfig{
		DatabaseURL:         url,
		KafkaBrokers:        brokers,
		JobsDatabaseURL:     env.String("JOBS_DATABASE_URL", ""),
		ConsumerConcurrency: env.Int("CONSUMER_CONCURRENCY", 4),
		DLQEnabled:          env.Bool("EVENT_DLQ_ENABLED", true),
	}, nil
}

// WorkflowWorkerConfig holds workflow worker settings.
type WorkflowWorkerConfig struct {
	DatabaseURL     string
	DatabaseMigrate bool
	LeaseDuration   time.Duration
	PollInterval    time.Duration
	MaxConcurrent   int
	ClaimBatchSize  int
	WorkerID        string
}

// LoadWorkflowWorker reads workflow worker configuration.
func LoadWorkflowWorker(env *Env) (WorkflowWorkerConfig, error) {
	url, err := env.RequiredString("WORKFLOW_DATABASE_URL")
	if err != nil {
		return WorkflowWorkerConfig{}, err
	}
	return WorkflowWorkerConfig{
		DatabaseURL:     url,
		DatabaseMigrate: env.Bool("WORKFLOW_DATABASE_MIGRATE", true),
		LeaseDuration:   env.Duration("JOB_LEASE_DURATION", 30*time.Second),
		PollInterval:    env.Duration("WORKER_POLL_INTERVAL", time.Second),
		MaxConcurrent:   env.Int("MAX_CONCURRENT_JOBS", 4),
		ClaimBatchSize:  env.Int("WORKER_CLAIM_BATCH_SIZE", 10),
		WorkerID:        env.String("WORKER_ID", "workflow-worker"),
	}, nil
}

// AuditConsumerConfig holds audit Kafka consumer settings.
type AuditConsumerConfig struct {
	DatabaseURL         string
	KafkaBrokers        string
	ConsumerConcurrency int
	DLQEnabled          bool
	MaxRetries          int
}

// LoadAuditConsumer reads audit consumer configuration.
func LoadAuditConsumer(env *Env) (AuditConsumerConfig, error) {
	dbURL := env.FirstNonEmpty("AUDIT_DATABASE_URL", "DATABASE_URL")
	if dbURL == "" {
		return AuditConsumerConfig{}, fmt.Errorf("AUDIT_DATABASE_URL is required")
	}
	brokers, err := env.RequiredString("KAFKA_BROKERS")
	if err != nil {
		return AuditConsumerConfig{}, err
	}
	kafka := LoadKafka(env)
	return AuditConsumerConfig{
		DatabaseURL:         dbURL,
		KafkaBrokers:        brokers,
		ConsumerConcurrency: env.Int("CONSUMER_CONCURRENCY", 4),
		DLQEnabled:          kafka.DLQEnabled,
		MaxRetries:          kafka.MaxRetries,
	}, nil
}

// ParsePort validates a port string.
func ParsePort(name, raw string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("invalid %s: must be 1-65535", name)
	}
	return port, nil
}

// LogDeprecations writes deprecation warnings once at startup.
func LogDeprecations(env *Env, log func(msg string, args ...any)) {
	for _, d := range env.Deprecations() {
		log("config deprecation: "+d, nil)
	}
}
