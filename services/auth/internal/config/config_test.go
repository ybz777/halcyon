package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", cfg.LogLevel)
	}
	if cfg.ServiceName != "auth" {
		t.Errorf("ServiceName = %q, want auth", cfg.ServiceName)
	}
}

func TestLoadCustomPort(t *testing.T) {
	clearEnv(t)
	t.Setenv("AUTH_PORT", "9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Port)
	}
}

func TestLoadInvalidPort(t *testing.T) {
	clearEnv(t)
	t.Setenv("AUTH_PORT", "70000")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_LEVEL", "verbose")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"AUTH_PORT", "LOG_LEVEL", "LOG_FORMAT", "SERVICE_NAME", "DATABASE_URL",
		"DATABASE_MIGRATE", "DATABASE_MAX_CONNS", "DATABASE_MIN_CONNS",
		"DATABASE_CONN_MAX_LIFETIME", "DATABASE_CONN_MAX_IDLE_TIME",
		"JWT_SECRET", "JWT_ACCESS_TTL", "JWT_REFRESH_TTL",
		"ARGON2_MEMORY", "ARGON2_ITERATIONS", "ARGON2_PARALLELISM",
		"ARGON2_SALT_LENGTH", "ARGON2_KEY_LENGTH",
	} {
		os.Unsetenv(key)
	}
}
