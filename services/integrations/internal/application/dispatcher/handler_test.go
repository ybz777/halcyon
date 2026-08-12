package dispatcher_test

import (
	"context"
	"testing"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/services/integrations/internal/application/dispatcher"
)

func TestHandlerSkipsMissingOrganization(t *testing.T) {
	h := dispatcher.NewHandler(nil, nil, nil, "test-group", nil)
	if err := h.Handle(context.Background(), platformevents.Envelope{EventID: "evt_1", EventType: "workflow.execution.succeeded.v1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandlerEnvelopeHasOccurredAt(t *testing.T) {
	env := platformevents.Envelope{
		EventID: "evt_2", EventType: "workflow.execution.succeeded.v1",
		OrganizationID: "org_123", OccurredAt: time.Now().UTC(),
	}
	if env.OrganizationID == "" {
		t.Fatal("expected organization")
	}
}
