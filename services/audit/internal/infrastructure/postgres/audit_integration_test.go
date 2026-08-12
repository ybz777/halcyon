//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	appaudit "github.com/halcyon-io/halcyon/services/audit/internal/application/audit"
	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
	"github.com/halcyon-io/halcyon/services/audit/internal/infrastructure/postgres"
)

func TestPersistIdempotentCreatesSingleRow(t *testing.T) {
	t.Skip("requires AUDIT_DATABASE_URL and Kafka; run with -tags=integration in CI")

	ctx := context.Background()
	repo := postgres.NewAuditRepository(nil)
	indexer := appaudit.NewIndexUseCase(repo, nil, nil)

	payload, _ := platformevents.MarshalPayload(platformevents.TypeUserLoggedIn, platformevents.UserLoggedInPayload{
		UserID: "usr_test1", Email: "a@example.com",
	})
	envelope := platformevents.Envelope{
		EventID: "evt_integration1", EventType: platformevents.TypeUserLoggedIn,
		OrganizationID: "org_test1", ActorID: "usr_test1", Payload: payload,
		OccurredAt: time.Now().UTC(),
	}

	if err := indexer.HandleEnvelope(ctx, envelope, "audit-log-consumer", "auth"); err != nil {
		t.Fatal(err)
	}
	if err := indexer.HandleEnvelope(ctx, envelope, "audit-log-consumer", "auth"); err != nil {
		t.Fatal(err)
	}

	events, _, err := repo.List(ctx, domain.ListFilter{OrganizationID: "org_test1", PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 audit row, got %d", len(events))
	}
}
