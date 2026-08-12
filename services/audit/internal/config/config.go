package config

import (
	"fmt"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// Config holds audit service runtime configuration.
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
	AuditRetentionDays  int
	ServiceAuth         platformconfig.ServiceAuth
	Environment         platformconfig.Environment
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	env := platformconfig.FromEnv()
	logging, err := platformconfig.LoadLogging(env)
	if err != nil {
		return Config{}, err
	}
	http, err := platformconfig.LoadHTTP(env, "AUDIT_PORT", 8083)
	if err != nil {
		return Config{}, err
	}
	db, err := platformconfig.LoadDatabase(env, "AUDIT_DATABASE_URL", "AUDIT_")
	if err != nil {
		return Config{}, err
	}
	jwt := env.Secret("JWT_SECRET")
	if err := platformconfig.ValidateProduction(env); err != nil {
		return Config{}, err
	}
	serviceAuth, err := platformconfig.LoadServiceAuth(env)
	if err != nil {
		return Config{}, err
	}
	if db.URL != "" {
		if err := platformconfig.ValidateSecretMinLength("JWT_SECRET", jwt, 32); err != nil {
			return Config{}, err
		}
	}
	return Config{
		Port:                http.PortString(),
		LogLevel:            logging.Level,
		LogFormat:           logging.Format,
		ServiceName:         env.String("SERVICE_NAME", "audit"),
		DatabaseURL:         db.URL,
		DatabaseMigrate:     db.Migrate,
		DatabaseMaxConns:    db.MaxConns,
		DatabaseMinConns:    db.MinConns,
		DatabaseConnMaxLife: db.ConnMaxLife,
		DatabaseConnMaxIdle: db.ConnMaxIdle,
		JWTSecret:           jwt.Value(),
		AuthServiceName:     env.String("AUTH_SERVICE_NAME", "auth"),
		AuditRetentionDays:  env.Int("AUDIT_RETENTION_DAYS", 365),
		ServiceAuth:         serviceAuth,
		Environment:         env.Environment(),
	}, nil
}

func (c Config) validate() error {
	if c.DatabaseMaxConns < c.DatabaseMinConns {
		return fmt.Errorf("AUDIT_DATABASE_MAX_CONNS must be >= AUDIT_DATABASE_MIN_CONNS")
	}
	if c.AuditRetentionDays < 1 {
		return fmt.Errorf("AUDIT_RETENTION_DAYS must be positive")
	}
	return nil
}
