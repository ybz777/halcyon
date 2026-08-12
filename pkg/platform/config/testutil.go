package config

import (
	"fmt"
	"testing"
)

// ForEnv creates an Env backed by test values with automatic cleanup.
func ForEnv(t *testing.T, values map[string]string) *Env {
	t.Helper()
	cleanup := SetTestEnv(values)
	t.Cleanup(cleanup)
	return FromEnv()
}

// MustLoad is a test helper that fails the test on config error.
func MustLoad[T any](t *testing.T, load func(*Env) (T, error)) T {
	t.Helper()
	v, err := load(FromEnv())
	if err != nil {
		t.Fatalf("config load failed: %v", err)
	}
	return v
}

// AssertNoSecretLeak verifies serialized output does not contain raw secrets.
func AssertNoSecretLeak(t *testing.T, output string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		if contains(output, secret) {
			t.Fatalf("secret leak detected in output")
		}
	}
}

func contains(haystack, needle string) bool {
	return needle != "" && len(haystack) >= len(needle) &&
		(indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// RequireProduction validates production-specific rules.
func RequireProduction(env *Env, checks ...func(*Env) error) error {
	if !env.IsProduction() {
		return nil
	}
	for _, check := range checks {
		if err := check(env); err != nil {
			return fmt.Errorf("production validation: %w", err)
		}
	}
	return nil
}
