//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	pginfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/security/argon2"
)

func TestAPIKeyRepository_SaveFindRevoke(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	saRepo := pginfra.NewServiceAccountRepository(pool)
	keyRepo := pginfra.NewAPIKeyRepository(pool)
	hasher := argon2.NewAPIKeyHasher(argon2.NewHasher(argon2.DefaultParams()))

	saID, _ := domain.NewServiceAccountID()
	org, _ := domain.ParseOrganizationID("org_integrationtest1")
	account, err := domain.NewServiceAccount(saID, org, "ci-bot", "automation", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewServiceAccount() error = %v", err)
	}
	if err := saRepo.Save(ctx, account); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	plaintext, prefix, err := domain.GenerateAPIKeySecret(domain.APIKeyEnvLive)
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret() error = %v", err)
	}
	hash, err := hasher.Hash(ctx, plaintext)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	id, _ := domain.NewAPIKeyID()
	keyID, _ := domain.NewKeyID()
	creator, _ := domain.ParseUserID("usr_integrationtest0701")
	now := time.Now().UTC()
	key, err := domain.NewAPIKey(id, keyID, saID, "CI", prefix, hash,
		[]domain.APIScope{domain.ScopeUsersRead}, creator, now, nil)
	if err != nil {
		t.Fatalf("NewAPIKey() error = %v", err)
	}
	if err := keyRepo.Save(ctx, key); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	found, err := keyRepo.FindByPrefix(ctx, prefix)
	if err != nil {
		t.Fatalf("FindByPrefix() error = %v", err)
	}
	if found.ID().String() != id.String() {
		t.Fatalf("id mismatch")
	}

	ok, err := hasher.Verify(ctx, plaintext, found.HashedSecret())
	if err != nil || !ok {
		t.Fatalf("Verify() ok=%v err=%v", ok, err)
	}

	found.MarkRevoked(time.Now().UTC())
	if err := keyRepo.Update(ctx, found); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
}
