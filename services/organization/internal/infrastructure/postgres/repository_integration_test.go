//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	appauth "github.com/halcyon-io/halcyon/services/organization/internal/application/auth"
	apporg "github.com/halcyon-io/halcyon/services/organization/internal/application/organization"
	"github.com/halcyon-io/halcyon/services/organization/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/organization/internal/domain"
	pginfra "github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/postgres"
	"github.com/halcyon-io/halcyon/services/organization/internal/infrastructure/rbac"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("halcyon_org"),
		tcpostgres.WithUsername("halcyon"),
		tcpostgres.WithPassword("halcyon"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp")),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	if err := pginfra.RunMigrations(connStr); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pginfra.NewPool(ctx, pginfra.PoolConfig{URL: connStr, MaxConns: 5, MinConns: 1})
	if err != nil {
		t.Fatalf("pool: %v", err)
	}

	return pool, func() {
		pool.Close()
		_ = testcontainers.TerminateContainer(container)
	}
}

func testUserID(suffix string) domain.UserID {
	id, _ := domain.ParseUserID("usr_integrationtest" + suffix)
	return id
}

func TestOrganizationRepository_CreateAndFind(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := pginfra.NewOrganizationRepository(pool)
	memberRepo := pginfra.NewMembershipRepository(pool)
	tx := pginfra.NewTransactionManager(pool)
	_ = rbac.NewEvaluator()

	uc := apporg.NewCreateOrganizationUseCase(repo, memberRepo, tx, ports.NoopEventPublisher{}, nil)
	org, err := uc.Execute(context.Background(), apporg.CreateOrganizationInput{
		ActorUserID: testUserID("01"),
		Name:        "Integration Org",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	authz := apporg.NewAuthorizer(nil)
	member, err := memberRepo.FindActiveByOrganizationAndUser(context.Background(), org.ID(), testUserID("01"))
	if err != nil {
		t.Fatalf("find membership: %v", err)
	}
	email, _ := domain.ParseEmail("owner@example.com")
	principal := appauth.NewUserPrincipal(testUserID("01"), email, org.ID(), member.ID(), member.Role())

	getUC := apporg.NewGetOrganizationUseCase(repo, authz, nil)
	found, err := getUC.Execute(context.Background(), apporg.GetOrganizationInput{
		OrganizationID: org.ID(),
		Principal:      principal,
	})
	if err != nil {
		t.Fatalf("GetOrganization() error = %v", err)
	}
	if found.Name().String() != "Integration Org" {
		t.Fatalf("name = %q", found.Name().String())
	}
}

func TestMembershipRepository_ConcurrentCreate(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	orgRepo := pginfra.NewOrganizationRepository(pool)
	memberRepo := pginfra.NewMembershipRepository(pool)
	ctx := context.Background()

	id, _ := domain.NewOrganizationID()
	name, _ := domain.ParseOrganizationName("Concurrent Org")
	slug, _ := domain.ParseSlug("concurrent-org")
	now := time.Now().UTC()
	org, _ := domain.CreateOrganization(id, name, slug, now)
	if err := orgRepo.Create(ctx, org); err != nil {
		t.Fatalf("create org: %v", err)
	}

	memberID1, _ := domain.NewMembershipID()
	memberID2, _ := domain.NewMembershipID()
	userID := testUserID("02")

	m1, _ := domain.NewMembership(memberID1, org.ID(), userID, domain.RoleViewer, now)
	m2, _ := domain.NewMembership(memberID2, org.ID(), userID, domain.RoleViewer, now)

	errCh := make(chan error, 2)
	go func() { errCh <- memberRepo.Create(ctx, m1) }()
	go func() { errCh <- memberRepo.Create(ctx, m2) }()

	var successes, conflicts int
	for i := 0; i < 2; i++ {
		if err := <-errCh; err == nil {
			successes++
		} else if err == domain.ErrMembershipExists {
			conflicts++
		} else {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestRunMigrations_Idempotent(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()
	if err := pginfra.RunMigrations(pool.Config().ConnString()); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}
}
