package config

import (
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// Config holds runtime configuration loaded from environment variables.
type Config struct {
	Port                string
	LogLevel            string
	LogFormat           string
	ServiceName         string
	DatabaseURL         string
	DatabaseMaxConns    int32
	DatabaseMinConns    int32
	DatabaseMigrate     bool
	DatabaseConnMaxLife time.Duration
	DatabaseConnMaxIdle time.Duration
	JWTSecret           string
	JWTKeyID            string
	JWTAccessTTL        time.Duration
	JWTRefreshTTL       time.Duration
	Argon2Memory        uint32
	Argon2Iterations    uint32
	Argon2Parallelism   uint8
	Argon2SaltLength    uint32
	Argon2KeyLength     uint32
	RedisURL            string
	RateLimitEnabled    bool
	IdempotencyEnabled  bool
}

// Load reads configuration from environment variables with validated defaults.
func Load() (Config, error) {
	loaded, err := platformconfig.LoadAuthService()
	if err != nil {
		return Config{}, err
	}
	return Config{
		Port:                loaded.HTTP.PortString(),
		LogLevel:            loaded.Logging.Level,
		LogFormat:           loaded.Logging.Format,
		ServiceName:         loaded.App.ServiceName,
		DatabaseURL:         loaded.Database.URL,
		DatabaseMaxConns:    loaded.Database.MaxConns,
		DatabaseMinConns:    loaded.Database.MinConns,
		DatabaseMigrate:     loaded.Database.Migrate,
		DatabaseConnMaxLife: loaded.Database.ConnMaxLife,
		DatabaseConnMaxIdle: loaded.Database.ConnMaxIdle,
		JWTSecret:           loaded.Auth.JWTSecret.Value(),
		JWTKeyID:            loaded.Auth.JWTKeyID,
		JWTAccessTTL:        loaded.Auth.AccessTTL,
		JWTRefreshTTL:       loaded.Auth.RefreshTTL,
		Argon2Memory:        loaded.Auth.Argon2Memory,
		Argon2Iterations:    loaded.Auth.Argon2Iter,
		Argon2Parallelism:   loaded.Auth.Argon2Par,
		Argon2SaltLength:    loaded.Auth.Argon2Salt,
		Argon2KeyLength:     loaded.Auth.Argon2KeyLen,
		RedisURL:            loaded.Redis.URL,
		RateLimitEnabled:    loaded.RateLimit.Enabled,
		IdempotencyEnabled:  loaded.Idempotency.Enabled,
	}, nil
}

// Summary returns a safe startup configuration summary.
func (c Config) Summary() platformconfig.Summary {
	loaded, _ := platformconfig.LoadAuthService()
	return loaded.Summary()
}
