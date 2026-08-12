//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	jobpostgres "github.com/halcyon-io/halcyon/pkg/platform/jobs/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("JOBS_DATABASE_URL")
	if url == "" {
		t.Skip("JOBS_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestPostgresClaimAndComplete(t *testing.T) {
	repo := jobpostgres.NewRepository(testPool(t))
	ctx := context.Background()

	job, _, err := repo.Create(ctx, platformjobs.CreateInput{
		Type: platformjobs.TypeWebhookDelivery, Payload: []byte(`{"delivery_id":"d1"}`),
		MaxAttempts: 3, IdempotencyKey: "integration-" + time.Now().Format("150405"),
	})
	if err != nil {
		t.Fatal(err)
	}

	claimed, err := repo.Claim(ctx, "worker-1", []platformjobs.Type{platformjobs.TypeWebhookDelivery}, 1, 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].ID != job.ID {
		t.Fatalf("unexpected claim: %+v", claimed)
	}

	if err := repo.Complete(ctx, "worker-1", job.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, job.ID)
	if err != nil || got.Status != platformjobs.StatusSucceeded {
		t.Fatalf("status=%s err=%v", got.Status, err)
	}
}

func TestPostgresLeaseRecovery(t *testing.T) {
	repo := jobpostgres.NewRepository(testPool(t))
	ctx := context.Background()

	job, _, err := repo.Create(ctx, platformjobs.CreateInput{
		Type: platformjobs.TypeEmailDelivery, Payload: []byte(`{}`),
		MaxAttempts: 5, IdempotencyKey: "recovery-" + time.Now().Format("150405.000"),
	})
	if err != nil {
		t.Fatal(err)
	}

	claimed, err := repo.Claim(ctx, "worker-crash", nil, 1, time.Millisecond)
	if err != nil || len(claimed) != 1 {
		t.Fatal("claim failed", err)
	}
	_ = claimed

	time.Sleep(5 * time.Millisecond)
	counts, err := repo.RecoverLeases(ctx, time.Millisecond, platformjobs.DefaultRetryConfig())
	if err != nil {
		t.Fatal(err)
	}
	if counts[job.Type] < 1 {
		t.Fatal("expected reclaimed job")
	}
	got, _ := repo.Get(ctx, job.ID)
	if got.Status != platformjobs.StatusRetryWait {
		t.Fatalf("expected RETRY_WAIT, got %s", got.Status)
	}
}
