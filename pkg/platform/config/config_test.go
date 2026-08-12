package config_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"
)

func TestSecretRedaction(t *testing.T) {
	secret := platformconfig.ForEnv(t, map[string]string{
		"JWT_SECRET": "top-secret-value-that-must-not-leak",
	}).Secret("JWT_SECRET")

	if secret.Value() != "top-secret-value-that-must-not-leak" {
		t.Fatal("value mismatch")
	}
	if strings.Contains(secret.String(), "top-secret") {
		t.Fatal("String() leaked secret")
	}
	b, err := json.Marshal(secret)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "top-secret") {
		t.Fatal("MarshalJSON leaked secret")
	}
}

func TestRedactURL(t *testing.T) {
	redacted := platformconfig.RedactURL("postgres://user:pass@localhost:5432/db")
	if strings.Contains(redacted, "pass") {
		t.Fatalf("password leaked: %s", redacted)
	}
	if !strings.Contains(redacted, "localhost") {
		t.Fatal("host should remain")
	}
}

func TestLoadJobsValidation(t *testing.T) {
	env := platformconfig.ForEnv(t, map[string]string{
		"JOB_WORKER_COUNT":         "2",
		"JOB_LEASE_DURATION":       "30s",
		"JOB_LEASE_RENEW_INTERVAL": "10s",
		"JOB_MAX_ATTEMPTS":         "6",
		"JOB_EXECUTION_TIMEOUT":    "1m",
		"JOB_RETRY_BASE_DELAY":     "1s",
		"JOB_RETRY_MAX_DELAY":      "5m",
	})
	jobs, err := platformconfig.LoadJobs(env)
	if err != nil {
		t.Fatal(err)
	}
	if jobs.WorkerCount != 2 {
		t.Fatalf("worker count: %d", jobs.WorkerCount)
	}
}

func TestLoadJobsInvalidRenewInterval(t *testing.T) {
	env := platformconfig.ForEnv(t, map[string]string{
		"JOB_LEASE_DURATION":       "5s",
		"JOB_LEASE_RENEW_INTERVAL": "10s",
	})
	_, err := platformconfig.LoadJobs(env)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadIdempotency(t *testing.T) {
	env := platformconfig.ForEnv(t, map[string]string{
		"IDEMPOTENCY_ENABLED":           "true",
		"IDEMPOTENCY_IN_PROGRESS_TTL":   "5m",
		"IDEMPOTENCY_COMPLETED_TTL":     "24h",
		"IDEMPOTENCY_MAX_RESPONSE_SIZE": "65536",
	})
	cfg, err := platformconfig.LoadIdempotency(env, "auth")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled || cfg.InProgressTTL != 5*time.Minute {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadRateLimitRequiresRedis(t *testing.T) {
	env := platformconfig.ForEnv(t, map[string]string{
		"RATE_LIMIT_ENABLED": "true",
	})
	_, err := platformconfig.LoadRateLimit(env, "auth")
	if err == nil {
		t.Fatal("expected redis required error")
	}
}

func TestDeprecatedAlias(t *testing.T) {
	env := platformconfig.ForEnv(t, map[string]string{
		"RATE_LIMIT_DEFAULT": "100",
	})
	rl, err := platformconfig.LoadRateLimit(env, "gateway")
	if err != nil {
		t.Fatal(err)
	}
	if rl.DefaultRequests != 100 {
		t.Fatalf("expected 100, got %d", rl.DefaultRequests)
	}
	if len(env.Deprecations()) == 0 {
		t.Fatal("expected deprecation warning")
	}
}

func TestSummaryNoSecretLeak(t *testing.T) {
	summary := platformconfig.NewSummary(platformconfig.App{
		Environment: platformconfig.EnvProduction,
		ServiceName: "auth",
		Version:     "1.0",
	},
		platformconfig.WithDatabaseURL("postgres://user:super-secret@localhost/db"),
		platformconfig.WithHTTPPort("8080"),
	)
	out := summary.String()
	platformconfig.AssertNoSecretLeak(t, out, "super-secret")
	if !strings.Contains(out, "postgres://") {
		t.Fatal("expected database url host in summary")
	}
	if strings.Contains(out, "super-secret") {
		t.Fatal("expected redacted database url")
	}
}

func TestAuthValidationWhenDatabaseSet(t *testing.T) {
	env := platformconfig.ForEnv(t, map[string]string{
		"JWT_SECRET": "short",
	})
	auth, err := platformconfig.LoadAuth(env)
	if err != nil {
		t.Fatal(err)
	}
	err = platformconfig.ValidateAuth(auth, true, env)
	if err == nil {
		t.Fatal("expected jwt secret length error")
	}
}

func TestEnvironment(t *testing.T) {
	env := platformconfig.ForEnv(t, map[string]string{"APP_ENV": "production"})
	if !env.IsProduction() {
		t.Fatal("expected production")
	}
}

func TestOutboxRelayRequired(t *testing.T) {
	_, err := platformconfig.LoadOutboxRelay(platformconfig.ForEnv(t, map[string]string{}))
	if err == nil {
		t.Fatal("expected required error")
	}
}
