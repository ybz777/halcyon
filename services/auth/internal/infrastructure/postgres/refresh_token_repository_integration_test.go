//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	pginfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres"
)

func TestRefreshTokenRepository_RotateAndReplay(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := pginfra.NewRefreshTokenRepository(pool)
	ctx := context.Background()
	user := newTestUser(t, "rt01")
	userRepo := pginfra.NewUserRepository(pool, nil)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	jti, _ := domain.ParseJTI("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	family, _ := domain.NewFamilyID()
	now := time.Now().UTC()
	record, err := domain.NewRefreshTokenRecord(
		jti, family, nil, user.ID(), "hash-one",
		now, now.Add(time.Hour), "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	if err := repo.Save(ctx, record); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	consumed, err := repo.Consume(ctx, jti)
	if err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if consumed.Status() != domain.TokenStatusConsumed {
		t.Fatalf("status = %s", consumed.Status())
	}

	_, err = repo.Consume(ctx, jti)
	if !errors.Is(err, domain.ErrRefreshTokenReused) {
		t.Fatalf("replay error = %v", err)
	}

	if err := repo.RevokeFamily(ctx, family); err != nil {
		t.Fatalf("RevokeFamily() error = %v", err)
	}
}

func TestRefreshTokenRepository_ConcurrentConsume(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := pginfra.NewRefreshTokenRepository(pool)
	ctx := context.Background()
	user := newTestUser(t, "rt02")
	userRepo := pginfra.NewUserRepository(pool, nil)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	jti, _ := domain.ParseJTI("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	family, _ := domain.NewFamilyID()
	now := time.Now().UTC()
	record, err := domain.NewRefreshTokenRecord(
		jti, family, nil, user.ID(), "hash-two",
		now, now.Add(time.Hour), "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	if err := repo.Save(ctx, record); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := repo.Consume(ctx, jti)
			results <- err
		}()
	}
	wg.Wait()
	close(results)

	var successes int
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successes = %d, want 1", successes)
	}
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := pginfra.NewRefreshTokenRepository(pool)
	ctx := context.Background()
	user := newTestUser(t, "rt03")
	userRepo := pginfra.NewUserRepository(pool, nil)
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	jti, _ := domain.ParseJTI("cccccccc-cccc-cccc-cccc-cccccccccccc")
	family, _ := domain.NewFamilyID()
	now := time.Now().UTC()
	record, err := domain.NewRefreshTokenRecord(
		jti, family, nil, user.ID(), "hash-three",
		now.Add(-48*time.Hour), now.Add(-24*time.Hour), "", "",
	)
	if err != nil {
		t.Fatalf("NewRefreshTokenRecord() error = %v", err)
	}
	if err := repo.Save(ctx, record); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	deleted, err := repo.DeleteExpired(ctx, now)
	if err != nil {
		t.Fatalf("DeleteExpired() error = %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
}
