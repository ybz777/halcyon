package ports

import (
	"context"

	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
)

// AuditRepository persists and queries audit events.
type AuditRepository interface {
	Insert(ctx context.Context, event domain.AuditEvent) error
	FindByID(ctx context.Context, organizationID string, id domain.AuditEventID) (domain.AuditEvent, error)
	List(ctx context.Context, filter domain.ListFilter) ([]domain.AuditEvent, string, error)
}

// TransactionalIndexer persists audit events with consumer idempotency in one transaction.
type TransactionalIndexer interface {
	PersistIdempotent(ctx context.Context, eventID, consumerGroup, organizationID string, event domain.AuditEvent) (duplicate bool, err error)
}

// Principal carries authenticated identity and permissions.
type Principal struct {
	UserID      string
	Permissions map[string]struct{}
}

// HasPermission reports whether the principal has a permission.
func (p Principal) HasPermission(permission string) bool {
	_, ok := p.Permissions[permission]
	return ok
}

// TokenValidator validates JWT access tokens.
type TokenValidator interface {
	Validate(ctx context.Context, bearerToken string) (Principal, error)
}

// ReadinessChecker reports dependency readiness.
type ReadinessChecker interface {
	Check(ctx context.Context) map[string]string
}

const (
	PermissionAuditRead         = "audit.read"
	PermissionAuditReadSecurity = "audit.read.security"
	PermissionAuditReadPlatform = "audit.read.platform"
)
