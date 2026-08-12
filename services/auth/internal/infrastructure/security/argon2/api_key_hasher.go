package argon2

import (
	"context"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// APIKeyHasher adapts the Argon2id hasher for API key secrets.
type APIKeyHasher struct {
	hasher *Hasher
}

// NewAPIKeyHasher creates an API key hasher.
func NewAPIKeyHasher(hasher *Hasher) *APIKeyHasher {
	return &APIKeyHasher{hasher: hasher}
}

// Hash hashes an API key secret.
func (h *APIKeyHasher) Hash(ctx context.Context, plaintext string) (domain.PasswordHash, error) {
	return h.hasher.Hash(ctx, plaintext)
}

// Verify checks an API key secret against a stored hash.
func (h *APIKeyHasher) Verify(ctx context.Context, plaintext string, stored domain.PasswordHash) (bool, error) {
	return h.hasher.Verify(ctx, plaintext, stored)
}

var _ ports.APIKeyHasher = (*APIKeyHasher)(nil)
