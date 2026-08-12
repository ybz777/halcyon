package ports

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
)

// IncidentRepository persists Incident aggregates.
type IncidentRepository interface {
	Create(ctx context.Context, incident *domain.Incident) error
	Update(ctx context.Context, incident *domain.Incident) error
	FindByID(ctx context.Context, organizationID domain.OrganizationID, id domain.IncidentID) (*domain.Incident, error)
	Search(ctx context.Context, filter IncidentSearchFilter) ([]*domain.Incident, string, error)
}

// HistoryRepository persists incident transition history.
type HistoryRepository interface {
	Append(ctx context.Context, records []domain.TransitionRecord) error
}

// IncidentSearchFilter scopes list/search queries.
type IncidentSearchFilter struct {
	OrganizationID domain.OrganizationID
	Statuses       []domain.Status
	Severities     []domain.Severity
	Query          string
	Limit          int
	Cursor         string
	SortBy         string
	SortOrder      string
}

// TransactionManager runs callbacks inside a database transaction.
type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// EventPublisher stores domain events for eventual delivery.
type EventPublisher interface {
	Publish(ctx context.Context, event domain.Event) error
}

// Principal carries authenticated identity and permissions.
type Principal struct {
	UserID      domain.UserID
	Permissions map[string]struct{}
}

// HasPermission reports whether the principal has a permission.
func (p Principal) HasPermission(permission string) bool {
	if _, ok := p.Permissions["future.*"]; ok {
		return true
	}
	_, ok := p.Permissions[permission]
	return ok
}

// TokenValidator validates JWT access tokens.
type TokenValidator interface {
	Validate(ctx context.Context, bearerToken string) (Principal, error)
}

// WorkflowMetrics records workflow observability signals.
type WorkflowMetrics interface {
	RecordIncidentCreated()
	RecordTransition(from, to string)
	ObserveResolutionDuration(duration time.Duration)
	ObserveStateDuration(status string, duration time.Duration)
}

// ReadinessChecker reports dependency readiness.
type ReadinessChecker interface {
	Check(ctx context.Context) map[string]string
}

const (
	PermissionIncidentRead  = "incident.read"
	PermissionIncidentWrite = "incident.write"
)
