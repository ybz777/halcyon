package ports

import (
	"context"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(ctx context.Context, password string) (domain.PasswordHash, error)
	Verify(ctx context.Context, password string, hash domain.PasswordHash) (bool, error)
}
