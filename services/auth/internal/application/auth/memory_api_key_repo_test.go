package auth_test

import (
	"context"
	"sync"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

type memoryAPIKeyRepo struct {
	mu   sync.Mutex
	keys map[string]*domain.APIKey
}

func newMemoryAPIKeyRepo() *memoryAPIKeyRepo {
	return &memoryAPIKeyRepo{keys: make(map[string]*domain.APIKey)}
}

func (r *memoryAPIKeyRepo) Save(_ context.Context, key *domain.APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys[key.ID().String()] = key
	r.keys[key.Prefix()] = key
	return nil
}

func (r *memoryAPIKeyRepo) FindByPrefix(_ context.Context, prefix string) (*domain.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key, ok := r.keys[prefix]
	if !ok {
		return nil, domain.ErrAPIKeyNotFound
	}
	return key, nil
}

func (r *memoryAPIKeyRepo) FindByID(_ context.Context, id domain.APIKeyID) (*domain.APIKey, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, key := range r.keys {
		if key.ID().Equal(id) {
			return key, nil
		}
	}
	return nil, domain.ErrAPIKeyNotFound
}

func (r *memoryAPIKeyRepo) ListByServiceAccount(_ context.Context, serviceAccountID domain.ServiceAccountID, _ int, _ string) ([]*domain.APIKey, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var keys []*domain.APIKey
	for _, key := range r.keys {
		if key.ServiceAccountID().Equal(serviceAccountID) {
			keys = append(keys, key)
		}
	}
	return keys, "", nil
}

func (r *memoryAPIKeyRepo) Update(_ context.Context, key *domain.APIKey) error {
	return r.Save(context.Background(), key)
}

func (r *memoryAPIKeyRepo) DeleteExpired(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

func (r *memoryAPIKeyRepo) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type memoryServiceAccountRepo struct {
	mu       sync.Mutex
	accounts map[string]*domain.ServiceAccount
}

func newMemoryServiceAccountRepo() *memoryServiceAccountRepo {
	return &memoryServiceAccountRepo{accounts: make(map[string]*domain.ServiceAccount)}
}

func (r *memoryServiceAccountRepo) Save(_ context.Context, account *domain.ServiceAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts[account.ID().String()] = account
	return nil
}

func (r *memoryServiceAccountRepo) FindByID(_ context.Context, id domain.ServiceAccountID) (*domain.ServiceAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	account, ok := r.accounts[id.String()]
	if !ok {
		return nil, domain.ErrServiceAccountNotFound
	}
	return account, nil
}

type stubAPIKeyHasher struct{}

func (stubAPIKeyHasher) Hash(_ context.Context, plaintext string) (domain.PasswordHash, error) {
	return domain.ParsePasswordHash("$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$" + plaintext)
}

func (stubAPIKeyHasher) Verify(_ context.Context, plaintext string, stored domain.PasswordHash) (bool, error) {
	expected := "$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$" + plaintext
	return stored.String() == expected, nil
}
