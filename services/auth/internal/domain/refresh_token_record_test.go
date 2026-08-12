package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
)

func TestRefreshTokenRecord_ValidateForRefresh(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	jti, _ := domain.ParseJTI("11111111-1111-1111-1111-111111111111")
	family, _ := domain.NewFamilyID()
	userID, _ := domain.ParseUserID("usr_integrationtest0501")

	active, err := domain.NewRefreshTokenRecord(
		jti, family, nil, userID, "hash",
		now, now.Add(time.Hour), "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	if err := active.ValidateForRefresh(now); err != nil {
		t.Fatalf("active token error = %v", err)
	}

	expired := domain.RehydrateRefreshTokenRecord(
		jti, family, nil, userID, "hash",
		now.Add(-2*time.Hour), now.Add(-time.Hour), nil, nil,
		"", "", domain.TokenStatusActive,
	)
	if !errors.Is(expired.ValidateForRefresh(now), domain.ErrTokenExpired) {
		t.Fatalf("expired token error = %v", domain.ErrTokenExpired)
	}

	consumed := domain.RehydrateRefreshTokenRecord(
		jti, family, nil, userID, "hash",
		now, now.Add(time.Hour), &now, nil,
		"", "", domain.TokenStatusConsumed,
	)
	if !errors.Is(consumed.ValidateForRefresh(now), domain.ErrRefreshTokenReused) {
		t.Fatalf("consumed token error = %v", domain.ErrRefreshTokenReused)
	}
}

func TestRefreshTokenRecord_MarkConsumed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	jti, _ := domain.ParseJTI("22222222-2222-2222-2222-222222222222")
	family, _ := domain.NewFamilyID()
	userID, _ := domain.ParseUserID("usr_integrationtest0502")

	record, err := domain.NewRefreshTokenRecord(
		jti, family, nil, userID, "hash",
		now, now.Add(time.Hour), "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	if err := record.MarkConsumed(now); err != nil {
		t.Fatalf("MarkConsumed() error = %v", err)
	}
	if record.Status() != domain.TokenStatusConsumed {
		t.Fatalf("status = %s", record.Status())
	}
}

func TestSecureCompareHashes(t *testing.T) {
	t.Parallel()

	token, _ := domain.ParseRefreshToken("abcdefghijklmnopqrstuvwxyz1234567890abcdef")
	hash := domain.HashRefreshToken(token)
	if !domain.SecureCompareHashes(hash, hash) {
		t.Fatal("expected hashes to match")
	}
	if domain.SecureCompareHashes(hash, hash+"x") {
		t.Fatal("expected hashes to differ")
	}
}
