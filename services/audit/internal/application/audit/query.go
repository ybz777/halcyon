package audit

import (
	"context"
	"time"

	"github.com/halcyon-io/halcyon/services/audit/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
)

// QueryUseCase lists and retrieves audit events.
type QueryUseCase struct {
	repo   ports.AuditRepository
	authz  *Authorizer
}

// NewQueryUseCase creates a QueryUseCase.
func NewQueryUseCase(repo ports.AuditRepository, authz *Authorizer) *QueryUseCase {
	return &QueryUseCase{repo: repo, authz: authz}
}

// ListInput carries list query parameters.
type ListInput struct {
	Principal      ports.Principal
	TrustedOrgID   string
	RequestedOrgID string
	Filter         domain.ListFilter
}

// ListResult contains a page of audit events.
type ListResult struct {
	Events     []domain.AuditEvent
	NextCursor string
}

// List returns a cursor-paginated list of audit events.
func (uc *QueryUseCase) List(ctx context.Context, input ListInput) (ListResult, error) {
	orgID, err := uc.authz.ResolveOrganizationScope(input.Principal, input.TrustedOrgID, input.RequestedOrgID)
	if err != nil {
		return ListResult{}, err
	}

	filter := input.Filter
	filter.OrganizationID = orgID
	filter.PageSize = domain.NormalizePageSize(filter.PageSize)
	filter.IncludeSecurity = input.Principal.HasPermission(ports.PermissionAuditReadSecurity)

	if err := validateTimeRange(filter.StartTime, filter.EndTime); err != nil {
		return ListResult{}, err
	}

	events, nextCursor, err := uc.repo.List(ctx, filter)
	if err != nil {
		return ListResult{}, err
	}
	events = uc.authz.FilterSecurityEvents(input.Principal, events)
	return ListResult{Events: events, NextCursor: nextCursor}, nil
}

// GetInput carries get-by-id parameters.
type GetInput struct {
	Principal      ports.Principal
	TrustedOrgID   string
	RequestedOrgID string
	ID             domain.AuditEventID
}

// Get returns a single audit event.
func (uc *QueryUseCase) Get(ctx context.Context, input GetInput) (domain.AuditEvent, error) {
	orgID, err := uc.authz.ResolveOrganizationScope(input.Principal, input.TrustedOrgID, input.RequestedOrgID)
	if err != nil {
		return domain.AuditEvent{}, err
	}

	event, err := uc.repo.FindByID(ctx, orgID, input.ID)
	if err != nil {
		return domain.AuditEvent{}, err
	}
	if err := uc.authz.CanViewEvent(input.Principal, orgID, event); err != nil {
		return domain.AuditEvent{}, err
	}
	return event, nil
}

func validateTimeRange(start, end *time.Time) error {
	if start != nil && end != nil && end.Before(*start) {
		return domain.ErrInvalidFilter
	}
	if start != nil && end != nil {
		if end.Sub(*start) > 90*24*time.Hour {
			return domain.ErrInvalidFilter
		}
	}
	return nil
}
