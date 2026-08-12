//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	appincident "github.com/halcyon-io/halcyon/services/workflow/internal/application/incident"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/events"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupWorkflowDB(t *testing.T) (*pgxpool.Pool, func()) {
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
		t.Fatalf("start postgres: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	if err := postgres.RunMigrations(connStr); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	pool, err := postgres.NewPool(ctx, postgres.PoolConfig{URL: connStr, MaxConns: 5, MinConns: 1})
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	cleanup := func() {
		pool.Close()
		_ = container.Terminate(ctx)
	}
	return pool, cleanup
}

func testPrincipal() ports.Principal {
	userID, _ := domain.ParseUserID("usr_actor12345678901")
	return ports.Principal{
		UserID: userID,
		Permissions: map[string]struct{}{
			ports.PermissionIncidentRead:  {},
			ports.PermissionIncidentWrite: {},
		},
	}
}

func TestIncidentRepository_CreateAndGet(t *testing.T) {
	pool, cleanup := setupWorkflowDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := postgres.NewIncidentRepository(pool)
	history := postgres.NewHistoryRepository(pool)
	tx := postgres.NewTransactionManager(pool)
	publisher := events.NoopPublisher{}

	createUC := appincident.NewCreateIncidentUseCase(repo, history, publisher, tx, nil)
	orgID, _ := domain.ParseOrganizationID("org_test123456789012")

	incident, err := createUC.Execute(ctx, appincident.CreateIncidentInput{
		Principal:      testPrincipal(),
		OrganizationID: orgID,
		Title:          "Database latency",
		Description:    "Slow queries detected",
		Severity:       "HIGH",
		Priority:       "HIGH",
		Source:         "monitoring",
	})
	if err != nil {
		t.Fatalf("create incident: %v", err)
	}

	got, err := repo.FindByID(ctx, orgID, incident.ID())
	if err != nil {
		t.Fatalf("find incident: %v", err)
	}
	if got.Status() != domain.StatusOpen {
		t.Fatalf("status = %s", got.Status())
	}
}

func TestIncidentRepository_OptimisticLocking(t *testing.T) {
	pool, cleanup := setupWorkflowDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := postgres.NewIncidentRepository(pool)
	history := postgres.NewHistoryRepository(pool)
	tx := postgres.NewTransactionManager(pool)
	publisher := events.NoopPublisher{}
	createUC := appincident.NewCreateIncidentUseCase(repo, history, publisher, tx, nil)

	orgID, _ := domain.ParseOrganizationID("org_test123456789012")
	incident, err := createUC.Execute(ctx, appincident.CreateIncidentInput{
		Principal:      testPrincipal(),
		OrganizationID: orgID,
		Title:          "Lock test",
		Description:    "",
		Severity:       "LOW",
		Priority:       "LOW",
		Source:         "manual",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	staleA, err := repo.FindByID(ctx, orgID, incident.ID())
	if err != nil {
		t.Fatalf("find A: %v", err)
	}
	staleB, err := repo.FindByID(ctx, orgID, incident.ID())
	if err != nil {
		t.Fatalf("find B: %v", err)
	}

	actor := testPrincipal().UserID
	now := time.Now().UTC()
	if err := staleA.TransitionTo(domain.StatusAcknowledged, actor, now); err != nil {
		t.Fatalf("transition A: %v", err)
	}
	if err := repo.Update(ctx, staleA); err != nil {
		t.Fatalf("update A: %v", err)
	}

	if err := staleB.TransitionTo(domain.StatusAcknowledged, actor, now); err != nil {
		t.Fatalf("transition B: %v", err)
	}
	if err := repo.Update(ctx, staleB); err != domain.ErrVersionConflict {
		t.Fatalf("expected version conflict, got %v", err)
	}
}
