package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/auth"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

func TestCreateAPIKeyHandler_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryAPIKeyRepo()
	saRepo := newMemoryServiceAccountRepo()
	hasher := stubAPIKeyHasher{}
	handler := auth.NewCreateAPIKeyHandler(repo, saRepo, hasher, nil, nil, nil)

	account := mustServiceAccount(t)
	if err := saRepo.Save(ctx, account); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	creator, _ := domain.ParseUserID("usr_integrationtest0602")

	result, err := handler.Handle(ctx, auth.CreateAPIKeyCommand{
		Name:             "CI Key",
		ServiceAccountID: account.ID().String(),
		Scopes:           []string{"users.read"},
		Environment:      "live",
		CreatedBy:        creator,
	})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if result.Plaintext == "" || result.APIKey.Prefix == "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestAuthenticateAPIKeyHandler_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryAPIKeyRepo()
	saRepo := newMemoryServiceAccountRepo()
	hasher := stubAPIKeyHasher{}
	account := mustServiceAccount(t)
	_ = saRepo.Save(ctx, account)

	plaintext, prefix, _ := domain.GenerateAPIKeySecret(domain.APIKeyEnvLive)
	hash, _ := hasher.Hash(ctx, plaintext)
	id, _ := domain.NewAPIKeyID()
	keyID, _ := domain.NewKeyID()
	creator, _ := domain.ParseUserID("usr_integrationtest0603")
	now := time.Now().UTC()

	key, err := domain.NewAPIKey(id, keyID, account.ID(), "Bot", prefix, hash,
		[]domain.APIScope{domain.ScopeUsersRead}, creator, now, nil)
	if err != nil {
		t.Fatalf("NewAPIKey() error = %v", err)
	}
	_ = repo.Save(ctx, key)

	handler := auth.NewAuthenticateAPIKeyHandler(repo, saRepo, hasher, nil, nil, nil)
	principal, err := handler.Handle(ctx, auth.AuthenticateAPIKeyQuery{APIKey: plaintext})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if principal.AuthMethod != auth.AuthMethodAPIKey {
		t.Fatalf("auth method = %s", principal.AuthMethod)
	}
}

func TestAuthenticateAPIKeyHandler_Revoked(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := newMemoryAPIKeyRepo()
	saRepo := newMemoryServiceAccountRepo()
	hasher := stubAPIKeyHasher{}
	account := mustServiceAccount(t)
	_ = saRepo.Save(ctx, account)

	plaintext, prefix, _ := domain.GenerateAPIKeySecret(domain.APIKeyEnvLive)
	hash, _ := hasher.Hash(ctx, plaintext)
	id, _ := domain.NewAPIKeyID()
	keyID, _ := domain.NewKeyID()
	creator, _ := domain.ParseUserID("usr_integrationtest0604")
	now := time.Now().UTC()

	key, _ := domain.NewAPIKey(id, keyID, account.ID(), "Bot", prefix, hash,
		[]domain.APIScope{domain.ScopeUsersRead}, creator, now, nil)
	key.MarkRevoked(now)
	_ = repo.Save(ctx, key)

	handler := auth.NewAuthenticateAPIKeyHandler(repo, saRepo, hasher, nil, nil, nil)
	_, err := handler.Handle(ctx, auth.AuthenticateAPIKeyQuery{APIKey: plaintext})
	if !errors.Is(err, domain.ErrAPIKeyRevoked) {
		t.Fatalf("error = %v", err)
	}
}

func mustServiceAccount(t *testing.T) *domain.ServiceAccount {
	t.Helper()
	id, _ := domain.NewServiceAccountID()
	org, _ := domain.ParseOrganizationID("org_integration1")
	account, err := domain.NewServiceAccount(id, org, "ci-bot", "automation", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewServiceAccount() error = %v", err)
	}
	return account
}
