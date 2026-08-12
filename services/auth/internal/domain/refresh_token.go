package domain

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const refreshTokenByteLength = 32

// RefreshToken is an opaque refresh token value object.
type RefreshToken struct {
	value string
}

// NewRefreshToken generates a cryptographically secure refresh token.
func NewRefreshToken() (RefreshToken, error) {
	b := make([]byte, refreshTokenByteLength)
	if _, err := rand.Read(b); err != nil {
		return RefreshToken{}, fmt.Errorf("generate refresh token: %w", err)
	}
	return ParseRefreshToken(base64.RawURLEncoding.EncodeToString(b))
}

// ParseRefreshToken validates a refresh token string.
func ParseRefreshToken(raw string) (RefreshToken, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return RefreshToken{}, fmt.Errorf("%w: empty", ErrInvalidRefreshToken)
	}
	if len(raw) < 32 {
		return RefreshToken{}, fmt.Errorf("%w: too short", ErrInvalidRefreshToken)
	}
	return RefreshToken{value: raw}, nil
}

// String returns the token value.
func (t RefreshToken) String() string {
	return t.value
}

// IsZero reports whether the token is unset.
func (t RefreshToken) IsZero() bool {
	return t.value == ""
}

// Equal reports whether two refresh tokens are identical.
func (t RefreshToken) Equal(other RefreshToken) bool {
	return t.value == other.value
}

// Bytes returns the token bytes for hashing in adapters.
func (t RefreshToken) Bytes() []byte {
	return []byte(t.value)
}
