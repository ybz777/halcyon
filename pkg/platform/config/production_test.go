package config

import "testing"

func TestValidateProduction_RejectsWeakJWT(t *testing.T) {
	t.Parallel()
	cleanup := SetTestEnv(map[string]string{
		"APP_ENV":                 "production",
		"JWT_SECRET":              "change-me-to-a-secret-with-at-least-32-characters",
		"SERVICE_TOKEN_PUBLIC_KEY": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAu\n-----END PUBLIC KEY-----",
	})
	defer cleanup()
	if err := ValidateProduction(FromEnv()); err == nil {
		t.Fatal("expected weak JWT rejection")
	}
}

func TestValidateProduction_RequiresServiceToken(t *testing.T) {
	t.Parallel()
	cleanup := SetTestEnv(map[string]string{
		"APP_ENV":    "production",
		"JWT_SECRET": "super-secret-production-jwt-key-32b-min",
	})
	defer cleanup()
	if err := ValidateProduction(FromEnv()); err == nil {
		t.Fatal("expected missing service token rejection")
	}
}

func TestValidateProduction_DevelopmentSkips(t *testing.T) {
	t.Parallel()
	cleanup := SetTestEnv(map[string]string{"APP_ENV": "development"})
	defer cleanup()
	if err := ValidateProduction(FromEnv()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
