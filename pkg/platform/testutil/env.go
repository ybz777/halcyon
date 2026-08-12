package testutil

import platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"

// TestJWTSecret is a deterministic secret for unit tests only.
const TestJWTSecret = "test-jwt-secret-at-least-32-bytes-long"

// TestEnv returns deterministic configuration for tests and a cleanup function.
func TestEnv(overrides map[string]string) (*platformconfig.Env, func()) {
	values := map[string]string{
		"APP_ENV":                 "test",
		"JWT_SECRET":              TestJWTSecret,
		"JWT_ISSUER":              "auth",
		"JWT_AUDIENCE":            "",
		"SERVICE_TOKEN_ISSUER":    "gateway",
		"SERVICE_TOKEN_AUDIENCE":  "halcyon-internal",
		"REDIS_URL":               "redis://localhost:6379/15",
		"DATABASE_URL":            "postgres://halcyon:halcyon@localhost:5432/halcyon_test?sslmode=disable",
		"RATE_LIMIT_ENABLED":      "false",
		"IDEMPOTENCY_ENABLED":     "false",
		"OTEL_ENABLED":            "false",
	}
	for k, v := range overrides {
		values[k] = v
	}
	cleanup := platformconfig.SetTestEnv(values)
	return platformconfig.FromEnv(), cleanup
}
