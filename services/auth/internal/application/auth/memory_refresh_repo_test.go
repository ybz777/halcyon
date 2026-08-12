package auth_test

import (
	"context"
	"sync"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

type memoryRefreshRepo struct {
	mu            sync.Mutex
	records       map[string]*domain.RefreshTokenRecord
	consumeCount  int
	saveCount     int
	familyRevoked bool
}

func newMemoryRefreshRepo() *memoryRefreshRepo {
	return &memoryRefreshRepo{records: make(map[string]*domain.RefreshTokenRecord)}
}

func (r *memoryRefreshRepo) Save(_ context.Context, record *domain.RefreshTokenRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[record.TokenID().String()] = record
	r.saveCount++
	return nil
}

func (r *memoryRefreshRepo) FindByJTI(_ context.Context, jti domain.JTI) (*domain.RefreshTokenRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[jti.String()]
	if !ok {
		return nil, domain.ErrInvalidRefreshToken
	}
	return record, nil
}

func (r *memoryRefreshRepo) Consume(_ context.Context, jti domain.JTI) (*domain.RefreshTokenRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[jti.String()]
	if !ok {
		return nil, domain.ErrInvalidRefreshToken
	}
	if record.Status() == domain.TokenStatusConsumed {
		return nil, domain.ErrRefreshTokenReused
	}
	if record.Status() != domain.TokenStatusActive {
		return nil, domain.ErrRefreshTokenRevoked
	}
	now := time.Now().UTC()
	if err := record.MarkConsumed(now); err != nil {
		return nil, err
	}
	r.consumeCount++
	return record, nil
}

func (r *memoryRefreshRepo) RevokeFamily(_ context.Context, _ domain.FamilyID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.familyRevoked = true
	return nil
}

func (r *memoryRefreshRepo) RevokeByJTI(_ context.Context, jti domain.JTI) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[jti.String()]
	if !ok {
		return domain.ErrInvalidRefreshToken
	}
	record.MarkRevoked(time.Now().UTC())
	return nil
}

func (r *memoryRefreshRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func (r *memoryRefreshRepo) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
