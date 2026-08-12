package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpmiddleware "github.com/halcyon-io/halcyon/services/auth/internal/api/http/middleware"
	appauth "github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

func TestAPIKeyAuthMiddleware_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMiddlewareAPIKeyRepo()
	saRepo := newMiddlewareServiceAccountRepo()
	hasher := middlewareStubHasher{}
	account := mustMiddlewareServiceAccount(t)
	_ = saRepo.Save(ctx, account)

	plaintext, prefix, _ := domain.GenerateAPIKeySecret(domain.APIKeyEnvLive)
	hash, _ := hasher.Hash(ctx, plaintext)
	id, _ := domain.NewAPIKeyID()
	keyID, _ := domain.NewKeyID()
	creator, _ := domain.ParseUserID("usr_integrationtest0605")
	now := time.Now().UTC()
	key, _ := domain.NewAPIKey(id, keyID, account.ID(), "Bot", prefix, hash,
		[]domain.APIScope{domain.ScopeUsersRead}, creator, now, nil)
	_ = repo.Save(ctx, key)

	authenticator := appauth.NewAuthenticateAPIKeyHandler(repo, saRepo, hasher, nil, nil, nil)
	called := false
	handler := httpmiddleware.APIKeyAuthMiddleware(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		principal, ok := httpmiddleware.PrincipalFromContext(r.Context())
		if !ok || principal.AuthMethod != appauth.AuthMethodAPIKey {
			t.Fatalf("principal missing or wrong auth method")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+plaintext)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !called {
		t.Fatalf("status = %d called = %v", rec.Code, called)
	}
}

func TestAPIKeyAuthMiddleware_InvalidKey(t *testing.T) {
	t.Parallel()

	authenticator := appauth.NewAuthenticateAPIKeyHandler(newMiddlewareAPIKeyRepo(), newMiddlewareServiceAccountRepo(), middlewareStubHasher{}, nil, nil, nil)
	handler := httpmiddleware.APIKeyAuthMiddleware(authenticator)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-API-Key", "hal_live_invalid-key-value")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func mustMiddlewareServiceAccount(t *testing.T) *domain.ServiceAccount {
	t.Helper()
	id, _ := domain.NewServiceAccountID()
	org, _ := domain.ParseOrganizationID("org_integration2")
	account, err := domain.NewServiceAccount(id, org, "bot", "", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewServiceAccount() error = %v", err)
	}
	return account
}

type middlewareStubHasher struct{}

func (middlewareStubHasher) Hash(_ context.Context, plaintext string) (domain.PasswordHash, error) {
	return domain.ParsePasswordHash("$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$" + plaintext)
}

func (middlewareStubHasher) Verify(_ context.Context, plaintext string, stored domain.PasswordHash) (bool, error) {
	return stored.String() == "$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$"+plaintext, nil
}

type middlewareAPIKeyRepo struct {
	keys map[string]*domain.APIKey
}

func newMiddlewareAPIKeyRepo() *middlewareAPIKeyRepo {
	return &middlewareAPIKeyRepo{keys: make(map[string]*domain.APIKey)}
}

func (r *middlewareAPIKeyRepo) Save(_ context.Context, key *domain.APIKey) error {
	r.keys[key.Prefix()] = key
	return nil
}

func (r *middlewareAPIKeyRepo) FindByPrefix(_ context.Context, prefix string) (*domain.APIKey, error) {
	key, ok := r.keys[prefix]
	if !ok {
		return nil, domain.ErrAPIKeyNotFound
	}
	return key, nil
}

func (r *middlewareAPIKeyRepo) FindByID(context.Context, domain.APIKeyID) (*domain.APIKey, error) {
	return nil, domain.ErrAPIKeyNotFound
}

func (r *middlewareAPIKeyRepo) ListByServiceAccount(context.Context, domain.ServiceAccountID, int, string) ([]*domain.APIKey, string, error) {
	return nil, "", nil
}

func (r *middlewareAPIKeyRepo) Update(_ context.Context, key *domain.APIKey) error {
	r.keys[key.Prefix()] = key
	return nil
}

func (r *middlewareAPIKeyRepo) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func (r *middlewareAPIKeyRepo) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type middlewareServiceAccountRepo struct {
	accounts map[string]*domain.ServiceAccount
}

func newMiddlewareServiceAccountRepo() *middlewareServiceAccountRepo {
	return &middlewareServiceAccountRepo{accounts: make(map[string]*domain.ServiceAccount)}
}

func (r *middlewareServiceAccountRepo) Save(_ context.Context, account *domain.ServiceAccount) error {
	r.accounts[account.ID().String()] = account
	return nil
}

func (r *middlewareServiceAccountRepo) FindByID(_ context.Context, id domain.ServiceAccountID) (*domain.ServiceAccount, error) {
	account, ok := r.accounts[id.String()]
	if !ok {
		return nil, domain.ErrServiceAccountNotFound
	}
	return account, nil
}
