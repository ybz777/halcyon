package config

import (
	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// Config holds worker + DB config from env.
type Config struct {
	Port                 string
	LogLevel             string
	LogFormat            string
	ServiceName          string
	DatabaseURL          string
	DatabaseMigrate      bool
	DatabaseMaxConns     int32
	DatabaseMinConns     int32
	JWTSecret            string
	AuthServiceName      string
	IntegrationsDBURL    string
	NotificationsDBURL   string
	IntegrationsEncKey   string
	Worker               platformconfig.Jobs
	SuccessRetentionDays int
	FailureRetentionDays int
	ServiceAuth          platformconfig.ServiceAuth
	Environment          platformconfig.Environment
}

// Load reads jobs service configuration.
func Load() (Config, error) {
	env := platformconfig.FromEnv()
	logging, err := platformconfig.LoadLogging(env)
	if err != nil {
		return Config{}, err
	}
	http, err := platformconfig.LoadHTTP(env, "JOBS_PORT", 8086)
	if err != nil {
		return Config{}, err
	}
	db, err := platformconfig.LoadDatabase(env, "JOBS_DATABASE_URL", "JOBS_")
	if err != nil {
		return Config{}, err
	}
	worker, err := platformconfig.LoadJobs(env)
	if err != nil {
		return Config{}, err
	}
	if err := platformconfig.ValidateProduction(env); err != nil {
		return Config{}, err
	}
	serviceAuth, err := platformconfig.LoadServiceAuth(env)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Port:                 http.PortString(),
		LogLevel:             logging.Level,
		LogFormat:            logging.Format,
		ServiceName:          env.String("SERVICE_NAME", "jobs"),
		DatabaseURL:          db.URL,
		DatabaseMigrate:      db.Migrate,
		DatabaseMaxConns:     db.MaxConns,
		DatabaseMinConns:     db.MinConns,
		JWTSecret:            env.Secret("JWT_SECRET").Value(),
		AuthServiceName:      env.String("AUTH_SERVICE_NAME", "auth"),
		IntegrationsDBURL:    env.String("INTEGRATIONS_DATABASE_URL", ""),
		NotificationsDBURL:   env.String("NOTIFICATIONS_DATABASE_URL", ""),
		IntegrationsEncKey:   env.Secret("INTEGRATIONS_ENCRYPTION_KEY").Value(),
		Worker:               worker,
		SuccessRetentionDays: worker.SuccessRetentionDays,
		FailureRetentionDays: worker.FailureRetentionDays,
		ServiceAuth:          serviceAuth,
		Environment:          env.Environment(),
	}, nil
}
