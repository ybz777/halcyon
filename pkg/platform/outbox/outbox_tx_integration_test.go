//go:build integration

package outbox_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/pkg/platform/outbox"
)

func TestOutboxTransactionRollback(t *testing.T) {
	pool, _, cleanup := setupMessagingTest(t)
	defer cleanup()

	ctx := context.Background()
	store := outbox.NewPostgresStore(pool)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	record, err := events.BuildRecord(
		events.TypeUserLoggedIn,
		"usr_rollbacktest001",
		"user",
		events.SystemOrganizationID,
		time.Now().UTC(),
		events.UserLoggedInPayload{UserID: "usr_rollbacktest001", Email: "rollback@example.com"},
	)
	if err != nil {
		t.Fatalf("build record: %v", err)
	}

	txStore := outbox.NewTxStore(tx)
	pub := outbox.NewPublisher(txStore, "integration-test")
	if err := pub.Publish(ctx, record); err != nil {
		t.Fatalf("publish in tx: %v", err)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox WHERE aggregate_id = $1`, "usr_rollbacktest001").Scan(&count); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count != 0 {
		t.Fatalf("outbox rows after rollback = %d, want 0", count)
	}
}

func TestOutboxTransactionCommit(t *testing.T) {
	pool, _, cleanup := setupMessagingTest(t)
	defer cleanup()

	ctx := context.Background()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}

	record, err := events.BuildRecord(
		events.TypeAPIKeyCreated,
		"apk_commit_test0001",
		"api_key",
		events.SystemOrganizationID,
		time.Now().UTC(),
		events.APIKeyCreatedPayload{
			APIKeyID:         "apk_commit_test0001",
			KeyID:            "key_commit_test01",
			ServiceAccountID: "sac_commit_test01",
			OrganizationID:   events.SystemOrganizationID,
			Name:             "CI",
		},
	)
	if err != nil {
		t.Fatalf("build record: %v", err)
	}

	txStore := outbox.NewTxStore(tx)
	pub := outbox.NewPublisher(txStore, "integration-test")
	if err := pub.Publish(ctx, record); err != nil {
		t.Fatalf("publish in tx: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM outbox WHERE event_id IS NOT NULL AND aggregate_id = $1`, "apk_commit_test0001").Scan(&count); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if count != 1 {
		t.Fatalf("outbox rows after commit = %d, want 1", count)
	}
}

func TestClaimPendingSkipLocked(t *testing.T) {
	pool, _, cleanup := setupMessagingTest(t)
	defer cleanup()

	ctx := context.Background()
	store := outbox.NewPostgresStore(pool)

	for i := 0; i < 2; i++ {
		userID := fmt.Sprintf("usr_skip_locked%04d", i)
		record, err := events.BuildRecord(
			events.TypeUserLoggedIn,
			userID,
			"user",
			events.SystemOrganizationID,
			time.Now().UTC(),
			events.UserLoggedInPayload{UserID: userID, Email: "skip@example.com"},
		)
		if err != nil {
			t.Fatalf("build record: %v", err)
		}
		if err := outbox.NewPublisher(store, "test").Publish(ctx, record); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		UPDATE outbox SET status = 'publishing'
		WHERE id IN (
			SELECT id FROM outbox WHERE status = 'pending'
			ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1
		)
		RETURNING id
	`)
	if err != nil {
		t.Fatalf("claim in tx: %v", err)
	}
	defer rows.Close()

	var claimed int
	for rows.Next() {
		claimed++
	}
	if claimed != 1 {
		t.Fatalf("claimed = %d, want 1", claimed)
	}

	claimed2, err := store.ClaimPending(ctx, 10)
	if err != nil {
		t.Fatalf("claim pending: %v", err)
	}
	if len(claimed2) != 1 {
		t.Fatalf("second claim = %d rows, want 1 while first tx holds lock", len(claimed2))
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}
