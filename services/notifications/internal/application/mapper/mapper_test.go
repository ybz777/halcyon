package mapper_test

import (
	"testing"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/services/notifications/internal/application/mapper"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

func TestMapWorkflowFailed(t *testing.T) {
	specs, ok := mapper.MapEnvelope(platformevents.Envelope{
		EventType: platformevents.TypeWorkflowExecutionFailed,
		AggregateID: "wex_123",
		OrganizationID: "org_1",
		OccurredAt: time.Now().UTC(),
	})
	if !ok || len(specs) != 1 {
		t.Fatalf("expected one spec")
	}
	if specs[0].NotificationType != domain.TypeWorkflowExecutionFailed {
		t.Fatalf("unexpected type %s", specs[0].NotificationType)
	}
}

func TestMapUnknownSkipped(t *testing.T) {
	_, ok := mapper.MapEnvelope(platformevents.Envelope{EventType: "unknown.event"})
	if ok {
		t.Fatal("expected skip")
	}
}
