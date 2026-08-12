package config

import (
	"fmt"
	"strings"
)

// ValidateProduction enforces required secrets and safe defaults in production.
func ValidateProduction(env *Env) error {
	if env == nil {
		env = FromEnv()
	}
	if env.Environment() != EnvProduction {
		return nil
	}

	if err := ValidateSecretMinLength("JWT_SECRET", env.Secret("JWT_SECRET"), 32); err != nil {
		return err
	}

	publicKey := env.Secret("SERVICE_TOKEN_PUBLIC_KEY")
	privateKey := env.Secret("SERVICE_PRIVATE_KEY")
	if !publicKey.IsSet() && !privateKey.IsSet() {
		return fmt.Errorf("SERVICE_TOKEN_PUBLIC_KEY or SERVICE_PRIVATE_KEY is required in production")
	}
	if publicKey.IsSet() {
		if err := ValidateSecretMinLength("SERVICE_TOKEN_PUBLIC_KEY", publicKey, 32); err != nil {
			return err
		}
	}

	jwt := strings.ToLower(env.Secret("JWT_SECRET").Value())
	for _, weak := range []string{"change-me", "secret", "password", "halcyon"} {
		if strings.Contains(jwt, weak) {
			return fmt.Errorf("JWT_SECRET uses a weak or placeholder value")
		}
	}

	if enc := env.Secret("INTEGRATIONS_ENCRYPTION_KEY"); enc.IsSet() {
		if len(enc.Value()) < 32 {
			return fmt.Errorf("INTEGRATIONS_ENCRYPTION_KEY must be at least 32 bytes in production")
		}
	}

	if env.Bool("RATE_LIMIT_ENABLED", false) && env.String("REDIS_URL", "") == "" {
		return fmt.Errorf("REDIS_URL is required when RATE_LIMIT_ENABLED=true in production")
	}

	if env.Bool("IDEMPOTENCY_ENABLED", true) && env.String("REDIS_URL", "") == "" {
		return fmt.Errorf("REDIS_URL is required when IDEMPOTENCY_ENABLED=true in production")
	}

	return nil
}
