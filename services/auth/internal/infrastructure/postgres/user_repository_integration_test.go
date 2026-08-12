//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	pginfra "github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const validBcryptHash = "$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/X4.G2oYbOuSO.8K6O"

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("halcyon"),
		tcpostgres.WithUsername("halcyon"),
		tcpostgres.WithPassword("halcyon"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	if err := pginfra.RunMigrations(connStr); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := pginfra.NewPool(ctx, pginfra.PoolConfig{URL: connStr, MaxConns: 5, MinConns: 1})
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	cleanup := func() {
		pool.Close()
		_ = testcontainers.TerminateContainer(container)
	}

	return pool, cleanup
}

func newTestUser(t *testing.T, suffix string) *domain.User {
	t.Helper()
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	id, err := domain.ParseUserID("usr_integrationtest" + suffix)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}

	email, err := domain.ParseEmail("user" + suffix + "@example.com")
	if err != nil {
		t.Fatalf("parse email: %v", err)
	}

	hash, err := domain.ParsePasswordHash(validBcryptHash)
	if err != nil {
		t.Fatalf("parse password hash: %v", err)
	}

	user, err := domain.RegisterUser(id, email, hash, now)
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	user.ClearUncommittedEvents()
	return user
}

func TestUserRepository_CreateFindDelete(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := pginfra.NewUserRepository(pool, pginfra.NewMetrics("auth", func() int32 {
		return pool.Stat().AcquiredConns()
	}))

	user := newTestUser(t, "01")
	ctx := context.Background()

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	found, err := repo.FindByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if !found.Email().Equal(user.Email()) {
		t.Fatalf("email mismatch")
	}

	exists, err := repo.ExistsByEmail(ctx, user.Email())
	if err != nil || !exists {
		t.Fatalf("ExistsByEmail() = %v, %v", exists, err)
	}

	if err := repo.Delete(ctx, user.ID()); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = repo.FindByID(ctx, user.ID())
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("FindByID() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepository_UpdateOptimisticLock(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := pginfra.NewUserRepository(pool, nil)
	user := newTestUser(t, "02")
	ctx := context.Background()

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stale, err := domain.ReconstituteUser(
		user.ID(), user.Email(), user.PasswordHash(), user.Status(),
		user.CreatedAt(), user.UpdatedAt(), user.Version(),
	)
	if err != nil {
		t.Fatalf("ReconstituteUser() error = %v", err)
	}

	if err := user.ChangeEmail(mustEmail(t, "updated02@example.com"), time.Now().UTC()); err != nil {
		t.Fatalf("ChangeEmail() error = %v", err)
	}

	if err := repo.Update(ctx, user); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if err := stale.ChangeEmail(mustEmail(t, "stale02@example.com"), time.Now().UTC()); err != nil {
		t.Fatalf("ChangeEmail() error = %v", err)
	}

	err = repo.Update(ctx, stale)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("Update() error = %v, want ErrVersionConflict", err)
	}
}

func TestTransactionManager_Rollback(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := pginfra.NewUserRepository(pool, nil)
	txManager := pginfra.NewTransactionManager(pool)
	user := newTestUser(t, "03")
	ctx := context.Background()

	err := txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := repo.Create(txCtx, user); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected transaction error")
	}

	_, err = repo.FindByID(ctx, user.ID())
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("FindByID() error = %v, want ErrUserNotFound", err)
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	connStr := pool.Config().ConnString()
	if err := pginfra.RunMigrations(connStr); err != nil {
		t.Fatalf("RunMigrations() second run error = %v", err)
	}
}

func TestUserRepository_ConcurrentUpdate(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := pginfra.NewUserRepository(pool, nil)
	user := newTestUser(t, "04")
	ctx := context.Background()

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	loaded, err := repo.FindByID(ctx, user.ID())
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}

	copyA, err := domain.ReconstituteUser(
		loaded.ID(), loaded.Email(), loaded.PasswordHash(), loaded.Status(),
		loaded.CreatedAt(), loaded.UpdatedAt(), loaded.Version(),
	)
	if err != nil {
		t.Fatalf("ReconstituteUser() error = %v", err)
	}
	copyB, err := domain.ReconstituteUser(
		loaded.ID(), loaded.Email(), loaded.PasswordHash(), loaded.Status(),
		loaded.CreatedAt(), loaded.UpdatedAt(), loaded.Version(),
	)
	if err != nil {
		t.Fatalf("ReconstituteUser() error = %v", err)
	}

	if err := copyA.ChangeEmail(mustEmail(t, "concurrenta04@example.com"), time.Now().UTC()); err != nil {
		t.Fatalf("ChangeEmail() error = %v", err)
	}
	if err := copyB.ChangeEmail(mustEmail(t, "concurrentb04@example.com"), time.Now().UTC()); err != nil {
		t.Fatalf("ChangeEmail() error = %v", err)
	}

	errCh := make(chan error, 2)
	go func() { errCh <- repo.Update(ctx, copyA) }()
	go func() { errCh <- repo.Update(ctx, copyB) }()

	var successes, conflicts int
	for i := 0; i < 2; i++ {
		switch err := <-errCh; {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrVersionConflict):
			conflicts++
		default:
			t.Fatalf("Update() error = %v", err)
		}
	}

	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want 1 each", successes, conflicts)
	}
}

func mustEmail(t *testing.T, raw string) domain.Email {
	t.Helper()
	email, err := domain.ParseEmail(raw)
	if err != nil {
		t.Fatalf("ParseEmail: %v", err)
	}
	return email
}
