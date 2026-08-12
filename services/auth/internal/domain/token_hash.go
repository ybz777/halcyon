package domain

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// HashRefreshToken returns a SHA-256 hex digest of the refresh token.
func HashRefreshToken(token RefreshToken) string {
	sum := sha256.Sum256(token.Bytes())
	return hex.EncodeToString(sum[:])
}

// SecureCompareHashes compares two token hashes in constant time.
func SecureCompareHashes(stored, computed string) bool {
	if len(stored) != len(computed) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(computed)) == 1
}
