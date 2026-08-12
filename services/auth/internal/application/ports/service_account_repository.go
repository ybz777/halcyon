package ports

import (
	"context"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// ServiceAccountRepository persists service account aggregates.
type ServiceAccountRepository interface {
	Save(ctx context.Context, account *domain.ServiceAccount) error
	FindByID(ctx context.Context, id domain.ServiceAccountID) (*domain.ServiceAccount, error)
}
