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
	JWTIssuer           string
	JWTAudience         string
	AuthServiceName     string
	RedisURL            string
	IdempotencyEnabled  bool
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	loaded, err := platformconfig.LoadOrganizationService()
	if err != nil {
		return Config{}, err
	}
	return Config{
		Port:                loaded.HTTP.PortString(),
		LogLevel:            loaded.Logging.Level,
		LogFormat:           loaded.Logging.Format,
		ServiceName:         loaded.App.ServiceName,
		DatabaseURL:         loaded.Database.URL,
		DatabaseMigrate:     loaded.Database.Migrate,
		DatabaseMaxConns:    loaded.Database.MaxConns,
		DatabaseMinConns:    loaded.Database.MinConns,
		DatabaseConnMaxLife: loaded.Database.ConnMaxLife,
		DatabaseConnMaxIdle: loaded.Database.ConnMaxIdle,
		JWTSecret:           loaded.JWTSecret.Value(),
		JWTIssuer:           loaded.JWTIssuer,
		JWTAudience:         loaded.JWTAudience,
		AuthServiceName:     loaded.AuthService,
		RedisURL:            loaded.Redis.URL,
		IdempotencyEnabled:  loaded.Idempotency.Enabled,
	}, nil
}

func (c Config) validate() error {
	if c.DatabaseMaxConns < c.DatabaseMinConns {
		return fmt.Errorf("ORGANIZATION_DATABASE_MAX_CONNS must be >= ORGANIZATION_DATABASE_MIN_CONNS")
	}
	return nil
}
