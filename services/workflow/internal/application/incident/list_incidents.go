package incident



import (

	"context"

	"strings"



	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"

	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"

	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"

)



// ListIncidentsInput carries list query parameters.

type ListIncidentsInput struct {

	Principal      ports.Principal

	OrganizationID domain.OrganizationID

	Status         string

	Severity       string

	Limit          int

	Cursor         string

	SortBy         string

	SortOrder      string

}



// ListIncidentsResult contains a page of incidents.

type ListIncidentsResult struct {

	Items      []*domain.Incident

	NextCursor string

}



// ListIncidentsUseCase lists incidents for an organization.

type ListIncidentsUseCase struct {

	incidents ports.IncidentRepository

}



// NewListIncidentsUseCase creates the use case.

func NewListIncidentsUseCase(incidents ports.IncidentRepository) *ListIncidentsUseCase {

	return &ListIncidentsUseCase{incidents: incidents}

}



// Execute returns incidents matching filters.

func (uc *ListIncidentsUseCase) Execute(ctx context.Context, input ListIncidentsInput) (ListIncidentsResult, error) {

	if err := ensureReadAccess(input.Principal); err != nil {

		return ListIncidentsResult{}, err

	}

	items, cursor, err := uc.incidents.Search(ctx, buildSearchFilter(input))

	if err != nil {

		return ListIncidentsResult{}, err

	}

	return ListIncidentsResult{Items: items, NextCursor: cursor}, nil

}



// SearchIncidentsInput carries search query parameters.

type SearchIncidentsInput struct {

	Principal      ports.Principal

	OrganizationID domain.OrganizationID

	Query          string

	Status         string

	Severity       string

	Limit          int

	Cursor         string

	SortBy         string

	SortOrder      string

}



// SearchIncidentsUseCase searches incidents by text and filters.

type SearchIncidentsUseCase struct {

	incidents ports.IncidentRepository

}



// NewSearchIncidentsUseCase creates the use case.

func NewSearchIncidentsUseCase(incidents ports.IncidentRepository) *SearchIncidentsUseCase {

	return &SearchIncidentsUseCase{incidents: incidents}

}



// Execute searches incidents for an organization.

func (uc *SearchIncidentsUseCase) Execute(ctx context.Context, input SearchIncidentsInput) (ListIncidentsResult, error) {

	if err := ensureReadAccess(input.Principal); err != nil {

		return ListIncidentsResult{}, err

	}

	filter := buildSearchFilter(ListIncidentsInput{

		Principal:      input.Principal,

		OrganizationID: input.OrganizationID,

		Status:         input.Status,

		Severity:       input.Severity,

		Limit:          input.Limit,

		Cursor:         input.Cursor,

		SortBy:         input.SortBy,

		SortOrder:      input.SortOrder,

	})

	filter.Query = strings.TrimSpace(input.Query)

	items, cursor, err := uc.incidents.Search(ctx, filter)

	if err != nil {

		return ListIncidentsResult{}, err

	}

	return ListIncidentsResult{Items: items, NextCursor: cursor}, nil

}



func buildSearchFilter(input ListIncidentsInput) ports.IncidentSearchFilter {

	filter := ports.IncidentSearchFilter{

		OrganizationID: input.OrganizationID,

		Limit:          input.Limit,

		Cursor:         input.Cursor,

		SortBy:         input.SortBy,

		SortOrder:      input.SortOrder,

	}

	if input.Status != "" {

		if status, err := domain.ParseStatus(input.Status); err == nil {

			filter.Statuses = []domain.Status{status}

		}

	}

	if input.Severity != "" {

		if severity, err := domain.ParseSeverity(input.Severity); err == nil {

			filter.Severities = []domain.Severity{severity}

		}

	}

	if filter.Limit <= 0 {

		filter.Limit = platformpagination.NormalizePageSize(0, 20, 100)

	}

	if filter.SortBy == "" {

		filter.SortBy = "created_at"

	}

	if filter.SortOrder == "" {

		filter.SortOrder = "desc"

	}

	return filter

}


