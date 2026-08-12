package config

import (
	"fmt"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

type Config struct {
	Port                string
	LogLevel            string
	LogFormat           string
	ServiceName         string
	DatabaseURL         string
	DatabaseMigrate     bool
	DatabaseMaxConns    int32
	DatabaseMinConns    int32
	DatabaseConnMaxLife time.Duration
	DatabaseConnMaxIdle time.Duration
	JWTSecret           string
	JWTIssuer           string
	JWTAudience         string
	AuthServiceName     string
	ServiceAuth         platformconfig.ServiceAuth
	Environment         platformconfig.Environment
	EncryptionKey       string
	AllowHTTPWebhooks   bool
	WebhookTimeout      time.Duration
	MaxRetryAttempts    int
	FailureThreshold    int
	RedisURL            string
	IdempotencyEnabled  bool
}

func Load() (Config, error) {
	env := platformconfig.FromEnv()
	logging, err := platformconfig.LoadLogging(env)
	if err != nil {
		return Config{}, err
	}
	http, err := platformconfig.LoadHTTP(env, "INTEGRATIONS_PORT", 8084)
	if err != nil {
		return Config{}, err
	}
	db, err := platformconfig.LoadDatabase(env, "INTEGRATIONS_DATABASE_URL", "INTEGRATIONS_")
	if err != nil {
		return Config{}, err
	}
	idempotency, err := platformconfig.LoadIdempotency(env, "integrations")
	if err != nil {
		return Config{}, err
	}
	serviceAuth, err := platformconfig.LoadServiceAuth(env)
	if err != nil {
		return Config{}, err
	}
	jwt := env.Secret("JWT_SECRET")
	if err := platformconfig.ValidateProduction(env); err != nil {
		return Config{}, err
	}
	if db.URL != "" {
		if err := platformconfig.ValidateSecretMinLength("JWT_SECRET", jwt, 32); err != nil {
			return Config{}, err
		}
	}
	cfg := Config{
		Port:                http.PortString(),
		LogLevel:            logging.Level,
		LogFormat:           logging.Format,
		ServiceName:         env.String("SERVICE_NAME", "integrations"),
		DatabaseURL:         db.URL,
		DatabaseMigrate:     db.Migrate,
		DatabaseMaxConns:    db.MaxConns,
		DatabaseMinConns:    db.MinConns,
		DatabaseConnMaxLife: db.ConnMaxLife,
		DatabaseConnMaxIdle: db.ConnMaxIdle,
		JWTSecret:           jwt.Value(),
		JWTIssuer:           env.String("JWT_ISSUER", env.String("AUTH_SERVICE_NAME", "auth")),
		JWTAudience:         env.String("JWT_AUDIENCE", ""),
		AuthServiceName:     env.String("AUTH_SERVICE_NAME", "auth"),
		ServiceAuth:         serviceAuth,
		EncryptionKey:       env.Secret("INTEGRATIONS_ENCRYPTION_KEY").Value(),
		AllowHTTPWebhooks:   env.Bool("WEBHOOK_ALLOW_HTTP", false),
		WebhookTimeout:      env.Duration("WEBHOOK_TIMEOUT", 10*time.Second),
		MaxRetryAttempts:    env.Int("WEBHOOK_MAX_RETRY_ATTEMPTS", 6),
		FailureThreshold:    env.Int("WEBHOOK_FAILURE_THRESHOLD", 10),
		RedisURL:            env.String("REDIS_URL", "redis://localhost:6379/0"),
		IdempotencyEnabled:  idempotency.Enabled,
		Environment:         env.Environment(),
	}
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.DatabaseMaxConns < c.DatabaseMinConns {
		return fmt.Errorf("INTEGRATIONS_DATABASE_MAX_CONNS must be >= INTEGRATIONS_DATABASE_MIN_CONNS")
	}
	return nil
}
