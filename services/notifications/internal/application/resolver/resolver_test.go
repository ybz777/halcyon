package resolver_test

import (
	"testing"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/services/notifications/internal/application/resolver"
)

func TestResolveWorkflowActor(t *testing.T) {
	recipients := resolver.Resolve(platformevents.Envelope{
		EventType: platformevents.TypeWorkflowExecutionFailed,
		OrganizationID: "org_a",
		ActorID: "usr_actor",
		OccurredAt: time.Now().UTC(),
	})
	if len(recipients) != 1 || recipients[0].UserID != "usr_actor" {
		t.Fatalf("unexpected recipients: %+v", recipients)
	}
}

func TestResolveSkipsMissingOrg(t *testing.T) {
	if len(resolver.Resolve(platformevents.Envelope{EventType: platformevents.TypeWorkflowExecutionFailed})) != 0 {
		t.Fatal("expected no recipients without org")
	}
}
