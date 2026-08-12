package idempotency

import (
	"regexp"
	"strings"
)

const (
	MaxKeyLength = 255
)

var keyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,255}$`)

// ParseKey extracts and validates the Idempotency-Key header.
func ParseKey(header string) (string, error) {
	key := strings.TrimSpace(header)
	if key == "" {
		return "", ErrKeyRequired
	}
	if len(key) > MaxKeyLength {
		return "", ErrKeyInvalid
	}
	if !keyPattern.MatchString(key) {
		return "", ErrKeyInvalid
	}
	return key, nil
}

// NormalizeKey returns the validated key unchanged (semantic identity preserved).
func NormalizeKey(key string) string {
	return strings.TrimSpace(key)
}
