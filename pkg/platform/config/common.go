package config

import (
	"fmt"
	"strconv"
	"time"
)

// App holds core application metadata.
type App struct {
	Environment Environment
	ServiceName string
	Version     string
}

// LoadApp reads application metadata from environment.
func LoadApp(env *Env, defaultService string) App {
	return App{
		Environment: env.Environment(),
		ServiceName: env.String("SERVICE_NAME", defaultService),
		Version:     env.String("SERVICE_VERSION", "dev"),
	}
}

// Logging holds log configuration.
type Logging struct {
	Level  string
	Format string
}

// LoadLogging reads logging configuration.
func LoadLogging(env *Env) (Logging, error) {
	cfg := Logging{
		Level:  env.String("LOG_LEVEL", "info"),
		Format: env.String("LOG_FORMAT", "json"),
	}
	switch cfg.Level {
	case "debug", "info", "warn", "error":
	default:
		return Logging{}, fmt.Errorf("invalid LOG_LEVEL: expected debug, info, warn, or error")
	}
	switch cfg.Format {
	case "json", "text":
	default:
		return Logging{}, fmt.Errorf("invalid LOG_FORMAT: expected json or text")
	}
	return cfg, nil
}

// HTTP holds HTTP server settings.
type HTTP struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxRequestBody  int64
}

// LoadHTTP reads HTTP server configuration.
func LoadHTTP(env *Env, portKey string, defaultPort int) (HTTP, error) {
	portStr := env.String(portKey, strconv.Itoa(defaultPort))
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return HTTP{}, fmt.Errorf("invalid %s: must be 1-65535", portKey)
	}
	return HTTP{
		Host:            env.String("HTTP_HOST", ""),
		Port:            port,
		ReadTimeout:     env.Duration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    env.Duration("HTTP_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     env.Duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: env.Duration("HTTP_SHUTDOWN_TIMEOUT", 30*time.Second),
		MaxRequestBody: int64(env.FirstPositiveInt(1<<20,
			"REQUEST_MAX_BODY_BYTES", "MAX_REQUEST_BODY_BYTES")),
	}, nil
}

// PortString returns the listen port as string.
func (h HTTP) PortString() string {
	return strconv.Itoa(h.Port)
}

// Database holds PostgreSQL connection settings.
type Database struct {
	URL         string
	Migrate     bool
	MaxConns    int32
	MinConns    int32
	ConnMaxLife time.Duration
	ConnMaxIdle time.Duration
}

// LoadDatabase reads database configuration with a service-specific URL key.
func LoadDatabase(env *Env, urlKey string, prefix string) (Database, error) {
	maxKey := prefix + "DATABASE_MAX_CONNS"
	minKey := prefix + "DATABASE_MIN_CONNS"
	if prefix == "" {
		maxKey = "DATABASE_MAX_CONNS"
		minKey = "DATABASE_MIN_CONNS"
	}
	migrateKey := prefix + "DATABASE_MIGRATE"
	lifeKey := prefix + "DATABASE_CONN_MAX_LIFETIME"
	idleKey := prefix + "DATABASE_CONN_MAX_IDLE_TIME"
	if prefix == "" {
		migrateKey = "DATABASE_MIGRATE"
		lifeKey = "DATABASE_CONN_MAX_LIFETIME"
		idleKey = "DATABASE_CONN_MAX_IDLE_TIME"
	}

	maxConns, err := env.Int32(maxKey, 25)
	if err != nil {
		return Database{}, err
	}
	minConns, err := env.Int32(minKey, 2)
	if err != nil {
		return Database{}, err
	}
	cfg := Database{
		URL:         env.get(urlKey),
		Migrate:     env.Bool(migrateKey, true),
		MaxConns:    maxConns,
		MinConns:    minConns,
		ConnMaxLife: env.Duration(lifeKey, time.Hour),
		ConnMaxIdle: env.Duration(idleKey, 30*time.Minute),
	}
	if cfg.MaxConns < cfg.MinConns {
		return Database{}, fmt.Errorf("%s must be >= %s", maxKey, minKey)
	}
	return cfg, nil
}

// Redis holds Redis client settings.
type Redis struct {
	URL string
}

// LoadRedis reads Redis configuration.
func LoadRedis(env *Env) Redis {
	return Redis{URL: env.get("REDIS_URL")}
}

// Kafka holds Kafka client settings.
type Kafka struct {
	Brokers       string
	ClientID      string
	ConsumerGroup string
	DLQEnabled    bool
	MaxRetries    int
}

// LoadKafka reads Kafka configuration.
func LoadKafka(env *Env) Kafka {
	return Kafka{
		Brokers:       env.get("KAFKA_BROKERS"),
		ClientID:      env.String("KAFKA_CLIENT_ID", "halcyon"),
		ConsumerGroup: env.String("KAFKA_CONSUMER_GROUP", ""),
		DLQEnabled:    env.Bool("EVENT_DLQ_ENABLED", true),
		MaxRetries:    env.Int("EVENT_MAX_RETRIES", 4),
	}
}

// Auth holds JWT and password hashing settings.
type Auth struct {
	JWTSecret    Secret
	JWTKeyID     string
	JWTIssuer    string
	JWTAudience  string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	Argon2Memory uint32
	Argon2Iter   uint32
	Argon2Par    uint8
	Argon2Salt   uint32
	Argon2KeyLen uint32
}

// ServiceAuth holds gateway service-token trust settings for downstream services.
type ServiceAuth struct {
	PublicKeyPEM Secret
	Issuer       string
	Audience     string
}

// LoadAuth reads authentication configuration.
func LoadAuth(env *Env) (Auth, error) {
	mem, err := env.Uint32("ARGON2_MEMORY", 64*1024)
	if err != nil {
		return Auth{}, err
	}
	iter, err := env.Uint32("ARGON2_ITERATIONS", 3)
	if err != nil {
		return Auth{}, err
	}
	par, err := env.Uint32("ARGON2_PARALLELISM", 4)
	if err != nil {
		return Auth{}, err
	}
	salt, err := env.Uint32("ARGON2_SALT_LENGTH", 16)
	if err != nil {
		return Auth{}, err
	}
	keyLen, err := env.Uint32("ARGON2_KEY_LENGTH", 32)
	if err != nil {
		return Auth{}, err
	}
	return Auth{
		JWTSecret:    env.Secret("JWT_SECRET"),
		JWTKeyID:     env.String("JWT_KEY_ID", "halcyon-auth-1"),
		JWTIssuer:    env.String("JWT_ISSUER", "auth"),
		JWTAudience:  env.String("JWT_AUDIENCE", ""),
		AccessTTL:    env.Duration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:   env.Duration("JWT_REFRESH_TTL", 7*24*time.Hour),
		Argon2Memory: mem,
		Argon2Iter:   iter,
		Argon2Par:    uint8(par),
		Argon2Salt:   salt,
		Argon2KeyLen: keyLen,
	}, nil
}

// ValidateAuth validates auth settings when database is enabled.
func ValidateAuth(cfg Auth, dbConfigured bool, env *Env) error {
	if cfg.AccessTTL <= 0 {
		return fmt.Errorf("JWT_ACCESS_TTL must be positive")
	}
	if cfg.RefreshTTL <= 0 {
		return fmt.Errorf("JWT_REFRESH_TTL must be positive")
	}
	if dbConfigured {
		minLen := 32
		if env.IsProduction() {
			minLen = 32
		}
		if err := ValidateSecretMinLength("JWT_SECRET", cfg.JWTSecret, minLen); err != nil {
			return err
		}
	}
	return nil
}
