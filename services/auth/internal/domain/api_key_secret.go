package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	apiKeyPrefixLive = "hal_live_"
	apiKeyPrefixTest = "hal_test_"
	apiKeySecretLen  = 32
	apiKeyPrefixLen  = 17
)

// APIKeyEnvironment identifies live or test keys.
type APIKeyEnvironment string

const (
	APIKeyEnvLive APIKeyEnvironment = "live"
	APIKeyEnvTest APIKeyEnvironment = "test"
)

// ParseAPIKeyEnvironment validates an API key environment.
func ParseAPIKeyEnvironment(raw string) (APIKeyEnvironment, error) {
	env := APIKeyEnvironment(strings.TrimSpace(strings.ToLower(raw)))
	switch env {
	case APIKeyEnvLive, APIKeyEnvTest:
		return env, nil
	default:
		return "", fmt.Errorf("%w: invalid environment", ErrInvalidAPIKey)
	}
}

// GenerateAPIKeySecret creates a new API key and returns the full value and lookup prefix.
func GenerateAPIKeySecret(env APIKeyEnvironment) (fullKey, prefix string, err error) {
	secret := make([]byte, apiKeySecretLen)
	if _, err := rand.Read(secret); err != nil {
		return "", "", fmt.Errorf("generate api key secret: %w", err)
	}

	base := apiKeyPrefixLive
	if env == APIKeyEnvTest {
		base = apiKeyPrefixTest
	}

	encoded := base64.RawURLEncoding.EncodeToString(secret)
	fullKey = base + encoded
	if len(fullKey) < apiKeyPrefixLen {
		return "", "", fmt.Errorf("%w: generated key too short", ErrInvalidAPIKey)
	}
	return fullKey, fullKey[:apiKeyPrefixLen], nil
}

// ExtractAPIKeyPrefix returns the lookup prefix from a presented API key.
func ExtractAPIKeyPrefix(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, apiKeyPrefixLive) && !strings.HasPrefix(raw, apiKeyPrefixTest) {
		return "", fmt.Errorf("%w: invalid key format", ErrInvalidAPIKey)
	}
	if len(raw) < apiKeyPrefixLen {
		return "", fmt.Errorf("%w: key too short", ErrInvalidAPIKey)
	}
	return raw[:apiKeyPrefixLen], nil
}

// RedactAPIKey masks an API key for safe display.
func RedactAPIKey(prefix string) string {
	if len(prefix) <= 8 {
		return prefix + "****"
	}
	return prefix[:8] + "****..."
}
