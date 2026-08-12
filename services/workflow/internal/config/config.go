package config

import (
	"fmt"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// Config holds runtime configuration.
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
	AuthServiceName     string
	RedisURL            string
	IdempotencyEnabled  bool
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	env := platformconfig.FromEnv()
	logging, err := platformconfig.LoadLogging(env)
	if err != nil {
		return Config{}, err
	}
	http, err := platformconfig.LoadHTTP(env, "WORKFLOW_PORT", 8082)
	if err != nil {
		return Config{}, err
	}
	db, err := platformconfig.LoadDatabase(env, "WORKFLOW_DATABASE_URL", "WORKFLOW_")
	if err != nil {
		return Config{}, err
	}
	idempotency, err := platformconfig.LoadIdempotency(env, "workflow")
	if err != nil {
		return Config{}, err
	}
	jwt := env.Secret("JWT_SECRET")
	if db.URL != "" {
		if err := platformconfig.ValidateSecretMinLength("JWT_SECRET", jwt, 32); err != nil {
			return Config{}, err
		}
	}
	return Config{
		Port:                http.PortString(),
		LogLevel:            logging.Level,
		LogFormat:           logging.Format,
		ServiceName:         env.String("SERVICE_NAME", "workflow"),
		DatabaseURL:         db.URL,
		DatabaseMigrate:     db.Migrate,
		DatabaseMaxConns:    db.MaxConns,
		DatabaseMinConns:    db.MinConns,
		DatabaseConnMaxLife: db.ConnMaxLife,
		DatabaseConnMaxIdle: db.ConnMaxIdle,
		JWTSecret:           jwt.Value(),
		AuthServiceName:     env.String("AUTH_SERVICE_NAME", "auth"),
		RedisURL:            env.String("REDIS_URL", "redis://localhost:6379/0"),
		IdempotencyEnabled:  idempotency.Enabled,
	}, nil
}

func (c Config) validate() error {
	if c.DatabaseMaxConns < c.DatabaseMinConns {
		return fmt.Errorf("WORKFLOW_DATABASE_MAX_CONNS must be >= WORKFLOW_DATABASE_MIN_CONNS")
	}
	return nil
}
