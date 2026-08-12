package ports

import (
	"context"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
)

// JobRepository exposes read and admin operations.
type JobRepository interface {
	platformjobs.Repository
}

// AttemptReader loads attempt history.
type AttemptReader interface {
	ListByJob(ctx context.Context, jobID platformjobs.ID, limit int, cursor string) ([]platformjobs.Attempt, string, error)
}

// Principal carries authenticated identity and permissions.
type Principal struct {
	UserID      string
	Permissions map[string]struct{}
}

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

// AuditPublisher records administrative actions.
type AuditPublisher interface {
	PublishJobCancelled(ctx context.Context, actorID, jobID, orgID string)
}

const (
	PermissionJobsRead         = "jobs.read"
	PermissionJobsCancel       = "jobs.cancel"
	PermissionJobsReadPlatform = "platform.jobs.read"
	PermissionJobsCancelPlatform = "platform.jobs.cancel"
)
