package invitation

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// TokenService generates and hashes invitation tokens.
type TokenService struct{}

// NewTokenService creates a TokenService.
func NewTokenService() *TokenService {
	return &TokenService{}
}

// Generate creates a raw token and its hash.
func (TokenService) Generate() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, Hash(raw), nil
}

// Hash returns the SHA-256 hex digest of a token.
func (TokenService) Hash(raw string) string {
	return Hash(raw)
}

// Hash computes the SHA-256 hex digest of a token.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
