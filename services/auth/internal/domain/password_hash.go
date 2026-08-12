package domain

import (
	"fmt"
	"strings"
)

const (
	bcryptHashLength = 60
)

// PasswordHash is an immutable stored credential hash value object.
// Hashing occurs outside the domain; this type validates persisted hash format.
type PasswordHash struct {
	value string
}

// ParsePasswordHash validates a stored password hash produced by the application layer.
func ParsePasswordHash(raw string) (PasswordHash, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return PasswordHash{}, fmt.Errorf("%w: empty", ErrInvalidPasswordHash)
	}

	if !isSupportedHashFormat(raw) {
		return PasswordHash{}, fmt.Errorf("%w: unsupported hash format", ErrInvalidPasswordHash)
	}

	if len(raw) < bcryptHashLength {
		return PasswordHash{}, fmt.Errorf("%w: hash too short", ErrInvalidPasswordHash)
	}

	return PasswordHash{value: raw}, nil
}

func isSupportedHashFormat(hash string) bool {
	return strings.HasPrefix(hash, "$2a$") ||
		strings.HasPrefix(hash, "$2b$") ||
		strings.HasPrefix(hash, "$2y$") ||
		strings.HasPrefix(hash, "$argon2id$")
}

// String returns the hash value. Callers must never log this value.
func (h PasswordHash) String() string {
	return h.value
}

// IsZero reports whether the PasswordHash is unset.
func (h PasswordHash) IsZero() bool {
	return h.value == ""
}

// Equal reports whether two PasswordHash values are identical.
func (h PasswordHash) Equal(other PasswordHash) bool {
	return h.value == other.value
}

// Bytes returns a copy of the hash bytes for constant-time comparison in adapters.
func (h PasswordHash) Bytes() []byte {
	return []byte(h.value)
}
