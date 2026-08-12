package query

import (
	"context"
	"fmt"
	"time"

	platformjobs "github.com/halcyon-io/halcyon/pkg/platform/jobs"
	"github.com/halcyon-io/halcyon/services/jobs/internal/application/authorizer"
	"github.com/halcyon-io/halcyon/services/jobs/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/jobs/internal/domain"
)

// UseCase provides admin job queries and cancellation.
type UseCase struct {
	jobs     ports.JobRepository
	attempts ports.AttemptReader
	authz    *authorizer.Authorizer
	audit    ports.AuditPublisher
}

func New(jobs ports.JobRepository, attempts ports.AttemptReader, audit ports.AuditPublisher) *UseCase {
	return &UseCase{jobs: jobs, attempts: attempts, authz: authorizer.New(), audit: audit}
}

type ListInput struct {
	Principal ports.Principal
	OrgID     string
	Limit     int
	Cursor    string
}

func (uc *UseCase) List(ctx context.Context, input ListInput) ([]platformjobs.Job, string, error) {
	limit := domain.NormalizePageSize(input.Limit)
	if uc.authz.IsPlatformReader(input.Principal) {
		return uc.jobs.ListPlatform(ctx, limit, input.Cursor)
	}
	if input.OrgID == "" || !uc.authz.CanRead(input.Principal, input.OrgID) {
		return nil, "", fmt.Errorf("forbidden")
	}
	return uc.jobs.List(ctx, input.OrgID, limit, input.Cursor)
}

func (uc *UseCase) Get(ctx context.Context, principal ports.Principal, orgID string, id platformjobs.ID) (platformjobs.Job, error) {
	if uc.authz.IsPlatformReader(principal) {
		return uc.jobs.Get(ctx, id)
	}
	if orgID == "" || !uc.authz.CanRead(principal, orgID) {
		return platformjobs.Job{}, fmt.Errorf("forbidden")
	}
	return uc.jobs.GetForOrg(ctx, orgID, id)
}

func (uc *UseCase) ListAttempts(ctx context.Context, principal ports.Principal, orgID string, id platformjobs.ID, limit int, cursor string) ([]platformjobs.Attempt, string, error) {
	job, err := uc.Get(ctx, principal, orgID, id)
	if err != nil {
		return nil, "", err
	}
	_ = job
	return uc.attempts.ListByJob(ctx, id, limit, cursor)
}

func (uc *UseCase) Cancel(ctx context.Context, principal ports.Principal, orgID string, id platformjobs.ID) error {
	job, err := uc.Get(ctx, principal, orgID, id)
	if err != nil {
		return err
	}
	if !uc.authz.CanCancel(principal, job.OrganizationID) {
		return fmt.Errorf("forbidden")
	}
	if err := uc.jobs.Cancel(ctx, id, time.Now().UTC()); err != nil {
		return err
	}
	if uc.audit != nil {
		uc.audit.PublishJobCancelled(ctx, principal.UserID, id.String(), job.OrganizationID)
	}
	return nil
}
