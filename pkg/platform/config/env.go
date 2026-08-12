package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment identifies the deployment environment.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvTest        Environment = "test"
	EnvProduction  Environment = "production"
)

// Env reads process environment variables. This is the only package that
// should call os.Getenv directly.
type Env struct {
	deprecations []string
}

// FromEnv creates an environment reader from the process environment.
func FromEnv() *Env {
	return &Env{}
}

// SetTestEnv replaces the process environment for tests. Returns a cleanup func.
func SetTestEnv(values map[string]string) func() {
	prev := testOverride
	testOverride = func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	}
	return func() { testOverride = prev }
}

var testOverride func(key string) (string, bool)

func (e *Env) get(key string) string {
	if testOverride != nil {
		if v, ok := testOverride(key); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	return strings.TrimSpace(os.Getenv(key))
}

// Deprecations returns safe deprecation warnings collected during loading.
func (e *Env) Deprecations() []string {
	return append([]string(nil), e.deprecations...)
}

func (e *Env) warnDeprecated(oldKey, newKey string) {
	e.deprecations = append(e.deprecations, fmt.Sprintf("%s is deprecated; use %s", oldKey, newKey))
}

// String returns a non-secret string with optional default.
func (e *Env) String(key, fallback string) string {
	if v := e.get(key); v != "" {
		return v
	}
	return fallback
}

// RequiredString returns an error if the variable is unset.
func (e *Env) RequiredString(key string) (string, error) {
	v := e.get(key)
	if v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

// Secret loads a sensitive value. Never log the return value directly.
func (e *Env) Secret(key string) Secret {
	return Secret{value: e.get(key), name: key}
}

// Bool parses a boolean with default on unset or parse failure.
func (e *Env) Bool(key string, fallback bool) bool {
	v := e.get(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

// BoolStrict parses a boolean and returns error on invalid value.
func (e *Env) BoolStrict(key string, fallback bool) (bool, error) {
	v := e.get(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("invalid %s: expected true or false", key)
	}
	return b, nil
}

// Int parses an integer with default.
func (e *Env) Int(key string, fallback int) int {
	v := e.get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// IntStrict parses an integer and returns error on invalid value.
func (e *Env) IntStrict(key string, fallback int) (int, error) {
	v := e.get(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected integer", key)
	}
	return n, nil
}

// Int32 parses int32 with default.
func (e *Env) Int32(key string, fallback int32) (int32, error) {
	v := e.get(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected integer", key)
	}
	return int32(n), nil
}

// Uint32 parses uint32 with default.
func (e *Env) Uint32(key string, fallback uint32) (uint32, error) {
	v := e.get(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected positive integer", key)
	}
	return uint32(n), nil
}

// Float parses float64 with default.
func (e *Env) Float(key string, fallback float64) float64 {
	v := e.get(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

// Duration parses a duration with default on unset or parse failure.
func (e *Env) Duration(key string, fallback time.Duration) time.Duration {
	v := e.get(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

// DurationStrict parses duration and errors on invalid value.
func (e *Env) DurationStrict(key string, fallback time.Duration) (time.Duration, error) {
	v := e.get(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected duration (e.g. 30s, 5m)", key)
	}
	return d, nil
}

// FirstNonEmpty returns the first non-empty env value.
func (e *Env) FirstNonEmpty(keys ...string) string {
	for _, key := range keys {
		if v := e.get(key); v != "" {
			return v
		}
	}
	return ""
}

// FirstPositiveInt returns first positive int from keys with fallback.
func (e *Env) FirstPositiveInt(fallback int, keys ...string) int {
	for i, key := range keys {
		v := e.get(key)
		if v == "" {
			continue
		}
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			if i > 0 {
				e.warnDeprecated(key, keys[0])
			}
			return n
		}
	}
	return fallback
}

// FirstPositiveDuration returns first positive duration from keys.
func (e *Env) FirstPositiveDuration(fallback time.Duration, keys ...string) time.Duration {
	for _, key := range keys {
		v := e.get(key)
		if v == "" {
			continue
		}
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			return d
		}
	}
	return fallback
}

// Environment resolves APP_ENV / DEPLOYMENT_ENVIRONMENT.
func (e *Env) Environment() Environment {
	raw := strings.ToLower(e.FirstNonEmpty("APP_ENV", "DEPLOYMENT_ENVIRONMENT"))
	switch raw {
	case "", "dev", "development", "local":
		return EnvDevelopment
	case "test", "testing", "ci":
		return EnvTest
	case "prod", "production":
		return EnvProduction
	default:
		return Environment(raw)
	}
}

// IsProduction reports whether strict production rules apply.
func (e *Env) IsProduction() bool {
	return e.Environment() == EnvProduction
}
