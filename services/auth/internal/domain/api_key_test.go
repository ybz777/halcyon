package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

func TestAPIKey_ValidateForAuth(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	id, _ := domain.NewAPIKeyID()
	keyID, _ := domain.NewKeyID()
	saID, _ := domain.NewServiceAccountID()
	userID, _ := domain.ParseUserID("usr_integrationtest0601")
	hash, _ := domain.ParsePasswordHash("$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHQ$hashvaluehashvaluehashvaluehashvaluehashvaluehashval")

	active, err := domain.NewAPIKey(id, keyID, saID, "CI", "hal_live_12345678", hash,
		[]domain.APIScope{domain.ScopeUsersRead}, userID, now, nil)
	if err != nil {
		t.Fatalf("NewAPIKey() error = %v", err)
	}
	if err := active.ValidateForAuth(now); err != nil {
		t.Fatalf("active key error = %v", err)
	}

	revoked := domain.RehydrateAPIKey(id, keyID, saID, "CI", "hal_live_12345678", hash,
		[]domain.APIScope{domain.ScopeUsersRead}, domain.APIKeyStatusRevoked,
		now, nil, nil, nil, userID)
	if !errors.Is(revoked.ValidateForAuth(now), domain.ErrAPIKeyRevoked) {
		t.Fatalf("revoked key error = %v", domain.ErrAPIKeyRevoked)
	}
}

func TestGenerateAPIKeySecret(t *testing.T) {
	t.Parallel()

	full, prefix, err := domain.GenerateAPIKeySecret(domain.APIKeyEnvLive)
	if err != nil {
		t.Fatalf("GenerateAPIKeySecret() error = %v", err)
	}
	if full[:9] != "hal_live_" {
		t.Fatalf("prefix = %s", full[:9])
	}
	if prefix != full[:17] {
		t.Fatalf("lookup prefix mismatch")
	}
}

func TestRedactAPIKey(t *testing.T) {
	t.Parallel()

	redacted := domain.RedactAPIKey("hal_live_3Fk8a2b1")
	if redacted != "hal_live****..." {
		t.Fatalf("redacted = %s", redacted)
	}
}

func TestHasScope(t *testing.T) {
	t.Parallel()

	scopes := []domain.APIScope{domain.ScopeUsersRead}
	if !domain.HasScope(scopes, domain.ScopeUsersRead) {
		t.Fatal("expected users.read scope")
	}
	if domain.HasScope(scopes, domain.ScopeAdmin) {
		t.Fatal("did not expect admin scope")
	}
}
