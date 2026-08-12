package ports

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

// RefreshTokenRepository persists refresh token aggregates.
type RefreshTokenRepository interface {
	Save(ctx context.Context, record *domain.RefreshTokenRecord) error
	FindByJTI(ctx context.Context, jti domain.JTI) (*domain.RefreshTokenRecord, error)
	Consume(ctx context.Context, jti domain.JTI) (*domain.RefreshTokenRecord, error)
	RevokeFamily(ctx context.Context, familyID domain.FamilyID) error
	RevokeByJTI(ctx context.Context, jti domain.JTI) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
