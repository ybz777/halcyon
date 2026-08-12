//go:build integration

package postgres_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appengine "github.com/halcyon-io/halcyon/services/workflow/internal/application/engine"
	appworkflow "github.com/halcyon-io/halcyon/services/workflow/internal/application/workflow"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/events"
	workflowmetrics "github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/metrics"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/postgres"
	"github.com/halcyon-io/halcyon/services/workflow/internal/infrastructure/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func testPrincipal(userID domain.UserID) ports.Principal {
	return ports.Principal{
		UserID: userID,
		Permissions: map[string]struct{}{
			ports.PermissionWorkflowRead:    {},
			ports.PermissionWorkflowWrite:   {},
			ports.PermissionWorkflowExecute: {},
			ports.PermissionWorkflowCancel:  {},
		},
	}
}

func TestWorkflowExecutionEndToEnd(t *testing.T) {
	ctx := context.Background()
	pgContainer, err := tcpostgres.Run(ctx, "postgres:16-alpine",
		tcpostgres.WithDatabase("halcyon_workflow"),
		tcpostgres.WithUsername("halcyon"),
		tcpostgres.WithPassword("halcyon"),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	defer pgContainer.Terminate(ctx)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("conn: %v", err)
	}
	if err := postgres.RunMigrations(connStr); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()

	workflowRepo := postgres.NewWorkflowRepository(pool)
	executionRepo := postgres.NewExecutionRepository(pool)
	jobRepo := postgres.NewJobRepository(pool)
	idempotencyRepo := postgres.NewIdempotencyRepository(pool)
	txManager := postgres.NewTransactionManager(pool)
	publisher := events.NewOutboxPublisher(pool)
	metrics := workflowmetrics.NewEngineMetrics()
	limits := domain.DefaultWorkflowLimits()

	orgID, _ := domain.ParseOrganizationID("org_test12345678901")
	userID, _ := domain.ParseUserID("usr_test12345678901")

	createUC := appworkflow.NewCreateWorkflowUseCase(workflowRepo, txManager, limits)
	startUC := appworkflow.NewStartExecutionUseCase(workflowRepo, executionRepo, jobRepo, idempotencyRepo, publisher, txManager, metrics, limits)

	wf, _, err := createUC.Execute(ctx, appworkflow.CreateWorkflowInput{
		Principal: testPrincipal(userID), OrganizationID: orgID, Name: "e2e-delay",
		Steps: []domain.StepInput{{
			StepKey: "wait", Name: "Wait", Type: "DELAY",
			Configuration: json.RawMessage(`{"duration":"100ms"}`),
		}},
	})
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	execution, err := startUC.Execute(ctx, appworkflow.StartExecutionInput{
		Principal: testPrincipal(userID), OrganizationID: orgID, WorkflowID: wf.ID, IdempotencyKey: "e2e-key-1",
	})
	if err != nil {
		t.Fatalf("start execution: %v", err)
	}

	registry := appengine.NewRegistry(appengine.NewDelayExecutor())
	orchestrator := appengine.NewOrchestrator(workflowRepo, executionRepo, jobRepo, publisher, txManager, registry, metrics, appengine.OrchestratorConfig{LeaseDuration: 5 * time.Second})

	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	w := worker.NewWorker(jobRepo, orchestrator, metrics, worker.Config{
		WorkerID: "test-worker", PollInterval: 50 * time.Millisecond, LeaseDuration: 5 * time.Second,
		MaxConcurrent: 1, ClaimBatchSize: 1,
	}, nil)
	go func() { _ = w.Run(runCtx) }()

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		got, err := executionRepo.FindExecution(ctx, orgID, execution.ID)
		if err != nil {
			t.Fatalf("find execution: %v", err)
		}
		if got.Status == domain.ExecutionSucceeded {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("execution did not succeed in time")
}
