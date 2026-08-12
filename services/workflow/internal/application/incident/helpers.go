package incident

import (
	"context"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

func ensureWriteAccess(principal ports.Principal) error {
	if !principal.HasPermission(ports.PermissionIncidentWrite) {
		return domain.ErrForbidden
	}
	return nil
}

func ensureReadAccess(principal ports.Principal) error {
	if !principal.HasPermission(ports.PermissionIncidentRead) {
		return domain.ErrForbidden
	}
	return nil
}

func persistIncident(ctx context.Context, incidents ports.IncidentRepository, history ports.HistoryRepository, publisher ports.EventPublisher, incident *domain.Incident, create bool) error {
	if create {
		if err := incidents.Create(ctx, incident); err != nil {
			return err
		}
	} else {
		if err := incidents.Update(ctx, incident); err != nil {
			return err
		}
	}

	if len(incident.PendingHistory()) > 0 {
		if err := history.Append(ctx, incident.PendingHistory()); err != nil {
			return err
		}
	}

	for _, event := range incident.UncommittedEvents() {
		if err := publisher.Publish(ctx, event); err != nil {
			return fmt.Errorf("publish event: %w", err)
		}
	}

	incident.ClearPendingHistory()
	incident.ClearUncommittedEvents()
	return nil
}

func nowUTC() time.Time { return time.Now().UTC() }
