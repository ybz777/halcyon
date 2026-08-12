package httpapi

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

type httpRefreshRepo struct {
	records       map[string]*domain.RefreshTokenRecord
	familyRevoked bool
}

func (r *httpRefreshRepo) Save(_ context.Context, record *domain.RefreshTokenRecord) error {
	r.records[record.TokenID().String()] = record
	return nil
}

func (r *httpRefreshRepo) FindByJTI(_ context.Context, jti domain.JTI) (*domain.RefreshTokenRecord, error) {
	record, ok := r.records[jti.String()]
	if !ok {
		return nil, domain.ErrInvalidRefreshToken
	}
	return record, nil
}

func (r *httpRefreshRepo) Consume(_ context.Context, jti domain.JTI) (*domain.RefreshTokenRecord, error) {
	record, ok := r.records[jti.String()]
	if !ok {
		return nil, domain.ErrInvalidRefreshToken
	}
	if record.Status() == domain.TokenStatusConsumed {
		return nil, domain.ErrRefreshTokenReused
	}
	if err := record.MarkConsumed(time.Now().UTC()); err != nil {
		return nil, err
	}
	return record, nil
}

func (r *httpRefreshRepo) RevokeFamily(_ context.Context, _ domain.FamilyID) error {
	r.familyRevoked = true
	return nil
}

func (r *httpRefreshRepo) RevokeByJTI(_ context.Context, jti domain.JTI) error {
	record, ok := r.records[jti.String()]
	if !ok {
		return domain.ErrInvalidRefreshToken
	}
	record.MarkRevoked(time.Now().UTC())
	return nil
}

func (r *httpRefreshRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func (r *httpRefreshRepo) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
