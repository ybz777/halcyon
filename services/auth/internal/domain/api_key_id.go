package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const apiKeyIDPrefix = "key_"

// APIKeyID is an immutable identifier for an API key aggregate.
type APIKeyID struct {
	value string
}

// NewAPIKeyID generates a new API key identifier.
func NewAPIKeyID() (APIKeyID, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return APIKeyID{}, fmt.Errorf("generate api key id: %w", err)
	}
	return ParseAPIKeyID(apiKeyIDPrefix + base64.RawURLEncoding.EncodeToString(b))
}

// ParseAPIKeyID validates an API key identifier.
func ParseAPIKeyID(raw string) (APIKeyID, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, apiKeyIDPrefix) {
		return APIKeyID{}, fmt.Errorf("%w: missing %q prefix", ErrInvalidAPIKey, apiKeyIDPrefix)
	}
	if len(strings.TrimPrefix(raw, apiKeyIDPrefix)) < 16 {
		return APIKeyID{}, fmt.Errorf("%w: id suffix too short", ErrInvalidAPIKey)
	}
	return APIKeyID{value: raw}, nil
}

func (id APIKeyID) String() string  { return id.value }
func (id APIKeyID) IsZero() bool     { return id.value == "" }
func (id APIKeyID) Equal(other APIKeyID) bool { return id.value == other.value }

// KeyID is a public lookup identifier separate from the aggregate ID.
type KeyID struct {
	value string
}

// NewKeyID generates a new public key identifier.
func NewKeyID() (KeyID, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return KeyID{}, fmt.Errorf("generate key id: %w", err)
	}
	return KeyID{value: base64.RawURLEncoding.EncodeToString(b)}, nil
}

// ParseKeyID validates a public key identifier.
func ParseKeyID(raw string) (KeyID, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 8 {
		return KeyID{}, fmt.Errorf("%w: invalid key id", ErrInvalidAPIKey)
	}
	return KeyID{value: raw}, nil
}

func (k KeyID) String() string { return k.value }
