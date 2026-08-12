package config

import (
	"fmt"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

type Config struct {
	Port                     string
	LogLevel                 string
	LogFormat                string
	ServiceName              string
	DatabaseURL              string
	DatabaseMigrate          bool
	DatabaseMaxConns         int32
	DatabaseMinConns         int32
	JWTSecret                string
	AuthServiceName          string
	EmailProvider            string
	EmailFromAddress         string
	EmailFromName            string
	EmailTimeout             time.Duration
	MaxRetryAttempts         int
	RetentionDays            int
	MaxNotificationsPerEvent int
	ServiceAuth              platformconfig.ServiceAuth
	Environment              platformconfig.Environment
}

func Load() (Config, error) {
	env := platformconfig.FromEnv()
	logging, err := platformconfig.LoadLogging(env)
	if err != nil {
		return Config{}, err
	}
	http, err := platformconfig.LoadHTTP(env, "NOTIFICATIONS_PORT", 8085)
	if err != nil {
		return Config{}, err
	}
	db, err := platformconfig.LoadDatabase(env, "NOTIFICATIONS_DATABASE_URL", "NOTIFICATIONS_")
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
		Port:                     http.PortString(),
		LogLevel:                 logging.Level,
		LogFormat:                logging.Format,
		ServiceName:              env.String("SERVICE_NAME", "notifications"),
		DatabaseURL:              db.URL,
		DatabaseMigrate:          db.Migrate,
		DatabaseMaxConns:         db.MaxConns,
		DatabaseMinConns:         db.MinConns,
		JWTSecret:                jwt.Value(),
		AuthServiceName:          env.String("AUTH_SERVICE_NAME", "auth"),
		EmailProvider:            env.String("EMAIL_PROVIDER", "log"),
		EmailFromAddress:         env.String("EMAIL_FROM_ADDRESS", "noreply@halcyon.local"),
		EmailFromName:            env.String("EMAIL_FROM_NAME", "Halcyon"),
		EmailTimeout:             env.Duration("EMAIL_TIMEOUT", 10*time.Second),
		MaxRetryAttempts:         env.Int("EMAIL_MAX_RETRY_ATTEMPTS", 6),
		RetentionDays:            env.Int("NOTIFICATION_RETENTION_DAYS", 90),
		MaxNotificationsPerEvent: env.Int("MAX_NOTIFICATIONS_PER_EVENT", 50),
		ServiceAuth:              serviceAuth,
		Environment:              env.Environment(),
	}, nil
}

func (c Config) validate() error {
	if c.DatabaseMaxConns < c.DatabaseMinConns {
		return fmt.Errorf("NOTIFICATIONS_DATABASE_MAX_CONNS must be >= NOTIFICATIONS_DATABASE_MIN_CONNS")
	}
	return nil
}
