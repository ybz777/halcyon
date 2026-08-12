package jobs_test

import (
	"testing"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("JOB_WORKER_COUNT", "2")
	t.Setenv("JOB_MAX_ATTEMPTS", "8")
	cfg, err := platformjobs.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.WorkerCount != 2 {
		t.Fatalf("worker count: %d", cfg.WorkerCount)
	}
	if cfg.MaxAttempts != 8 {
		t.Fatalf("max attempts: %d", cfg.MaxAttempts)
	}
}

func TestLoadConfigRejectsInvalidLeaseRenew(t *testing.T) {
	t.Setenv("JOB_LEASE_DURATION", "5s")
	t.Setenv("JOB_LEASE_RENEW_INTERVAL", "10s")
	_, err := platformjobs.LoadConfig()
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestPolicyRegistry(t *testing.T) {
	reg := platformjobs.NewPolicyRegistry()
	p := reg.Lookup(platformjobs.TypeWebhookDelivery)
	if p.MaxAttempts != 6 {
		t.Fatalf("webhook max attempts: %d", p.MaxAttempts)
	}
	if p.ExecutionTimeout <= 0 {
		t.Fatal("expected execution timeout")
	}
}

func TestTypePolicyDefaults(t *testing.T) {
	policies := platformjobs.DefaultPolicies()
	if len(policies) < 2 {
		t.Fatal("expected default policies")
	}
	for _, p := range policies {
		if p.MaxAttempts < 1 {
			t.Fatal("max attempts must be positive")
		}
		if p.Retry.Initial <= 0 || p.Retry.MaxBackoff < p.Retry.Initial {
			t.Fatal("invalid retry config")
		}
		if p.ExecutionTimeout < time.Second {
			t.Fatal("execution timeout too small")
		}
	}
}
