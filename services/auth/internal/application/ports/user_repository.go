package ports

import (
	"context"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// UserRepository persists User aggregates.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id domain.UserID) error
	FindByID(ctx context.Context, id domain.UserID) (*domain.User, error)
	FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email domain.Email) (bool, error)
}

// TransactionManager runs callbacks inside a database transaction.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// NoopTransactionManager runs callbacks without starting a transaction.
type NoopTransactionManager struct{}

// WithinTransaction implements TransactionManager.
func (NoopTransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}
