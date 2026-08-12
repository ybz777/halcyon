package idempotency

import (
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// Config holds idempotency runtime settings.
type Config struct {
	Enabled          bool
	RedisPrefix      string
	InProgressTTL    time.Duration
	CompletedTTL     time.Duration
	MaxResponseBytes int64
	MaxBodyBytes     int64
	FailClosed       bool
}

// LoadConfig reads idempotency settings from environment variables.
func LoadConfig(serviceName string) (Config, error) {
	return LoadConfigFrom(platformconfig.FromEnv(), serviceName)
}

// LoadConfigFrom reads idempotency settings from a centralized env reader.
func LoadConfigFrom(env *platformconfig.Env, serviceName string) (Config, error) {
	loaded, err := platformconfig.LoadIdempotency(env, serviceName)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Enabled:          loaded.Enabled,
		RedisPrefix:      loaded.RedisPrefix,
		InProgressTTL:    loaded.InProgressTTL,
		CompletedTTL:     loaded.CompletedTTL,
		MaxResponseBytes: loaded.MaxResponseBytes,
		MaxBodyBytes:     1 << 20,
		FailClosed:       loaded.FailClosed,
	}, nil
}

// DefaultConfig returns production-safe defaults.
func DefaultConfig(serviceName string) Config {
	return Config{
		Enabled:          true,
		RedisPrefix:      serviceName,
		InProgressTTL:    5 * time.Minute,
		CompletedTTL:     24 * time.Hour,
		MaxResponseBytes: 64 * 1024,
		MaxBodyBytes:     1 << 20,
		FailClosed:       true,
	}
}
