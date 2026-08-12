package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const redacted = "[REDACTED]"

// Secret wraps a sensitive configuration value.
type Secret struct {
	value string
	name  string
}

// NewSecret constructs a secret value for derived or test configuration.
func NewSecret(name, value string) Secret {
	return Secret{value: value, name: name}
}

// Value returns the secret. Never log this value.
func (s Secret) Value() string {
	return s.value
}

// IsSet reports whether the secret is non-empty.
func (s Secret) IsSet() bool {
	return strings.TrimSpace(s.value) != ""
}

// String redacts the secret for safe display.
func (s Secret) String() string {
	if !s.IsSet() {
		return ""
	}
	if s.name != "" {
		return s.name + "=" + redacted
	}
	return redacted
}

// MarshalJSON always redacts secrets.
func (s Secret) MarshalJSON() ([]byte, error) {
	return json.Marshal(redacted)
}

// RedactURL masks credentials in a connection URL.
func RedactURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "[invalid-url]"
	}
	if u.User != nil {
		u.User = url.UserPassword("***", "***")
	}
	return u.String()
}

// RedactSecretsInMap replaces known secret keys with redacted values.
func RedactSecretsInMap(values map[string]string) map[string]string {
	secretKeys := regexp.MustCompile(`(?i)(secret|password|token|key|credential|private)`)
	out := make(map[string]string, len(values))
	for k, v := range values {
		switch {
		case secretKeys.MatchString(k):
			out[k] = redacted
		case strings.Contains(k, "URL") || strings.Contains(k, "_URL"):
			out[k] = RedactURL(v)
		default:
			out[k] = v
		}
	}
	return out
}

// ValidateSecretMinLength validates a secret meets minimum length without exposing it.
func ValidateSecretMinLength(name string, s Secret, min int) error {
	if !s.IsSet() {
		return fmt.Errorf("%s is required", name)
	}
	if len(s.Value()) < min {
		return fmt.Errorf("%s must be at least %d bytes", name, min)
	}
	return nil
}
