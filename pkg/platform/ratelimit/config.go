package ratelimit

import (
	"strings"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

// Config holds rate-limit configuration from environment variables.
type Config struct {
	Enabled             bool
	RedisURL            string
	RedisPrefix         string
	DefaultRequests     int
	DefaultWindow       time.Duration
	AuthRequests        int
	AuthWindow          time.Duration
	WorkflowRequests    int
	WorkflowWindow      time.Duration
	MaxBodyBytes        int64
	WorkflowConcurrency int
	FailureMode         FailureMode
	TrustProxy          bool
}

// LoadConfig reads rate limit settings from the environment.
func LoadConfig(serviceName string) (Config, error) {
	return LoadConfigFrom(platformconfig.FromEnv(), serviceName)
}

// LoadConfigFrom reads rate limit settings from a centralized env reader.
func LoadConfigFrom(env *platformconfig.Env, serviceName string) (Config, error) {
	loaded, err := platformconfig.LoadRateLimit(env, serviceName)
	if err != nil {
		return Config{}, err
	}
	return Config{
		Enabled:             loaded.Enabled,
		RedisURL:            loaded.RedisURL,
		RedisPrefix:         loaded.RedisPrefix,
		DefaultRequests:     loaded.DefaultRequests,
		DefaultWindow:       loaded.DefaultWindow,
		AuthRequests:        loaded.AuthRequests,
		AuthWindow:          loaded.AuthWindow,
		WorkflowRequests:    loaded.WorkflowRequests,
		WorkflowWindow:      loaded.WorkflowWindow,
		MaxBodyBytes:        loaded.MaxBodyBytes,
		WorkflowConcurrency: loaded.WorkflowConcurrency,
		FailureMode:         FailureMode(strings.ToLower(loaded.FailureMode)),
		TrustProxy:          loaded.TrustProxy,
	}, nil
}
