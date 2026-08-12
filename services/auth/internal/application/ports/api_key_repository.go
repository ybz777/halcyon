package ports

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// APIKeyRepository persists API key aggregates.
type APIKeyRepository interface {
	Save(ctx context.Context, key *domain.APIKey) error
	FindByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error)
	FindByID(ctx context.Context, id domain.APIKeyID) (*domain.APIKey, error)
	ListByServiceAccount(ctx context.Context, serviceAccountID domain.ServiceAccountID, limit int, cursor string) ([]*domain.APIKey, string, error)
	Update(ctx context.Context, key *domain.APIKey) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
