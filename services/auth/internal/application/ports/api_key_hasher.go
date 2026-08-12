package ports

import (
	"context"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// APIKeyHasher hashes and verifies API key secrets using Argon2id.
type APIKeyHasher interface {
	Hash(ctx context.Context, plaintext string) (domain.PasswordHash, error)
	Verify(ctx context.Context, plaintext string, stored domain.PasswordHash) (bool, error)
}

// APIKeyRateLimiter gates failed API key authentication attempts.
type APIKeyRateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// NoopAPIKeyRateLimiter allows all API key auth attempts.
type NoopAPIKeyRateLimiter struct{}

// Allow implements APIKeyRateLimiter.
func (NoopAPIKeyRateLimiter) Allow(context.Context, string) (bool, error) {
	return true, nil
}
