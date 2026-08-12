package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/services/workflow/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/workflow/internal/domain"
	"github.com/jackc/pgx/v5"
)

var incidentSortColumns = map[string]string{
	"created_at": "created_at",
	"updated_at": "updated_at",
	"severity":   "severity",
	"status":     "status",
	"title":      "title",
}

// Search lists incidents with filters, sorting, and cursor pagination.
func (r *IncidentRepository) Search(ctx context.Context, filter ports.IncidentSearchFilter) ([]*domain.Incident, string, error) {
	where := []string{"organization_id = $1"}
	args := []any{filter.OrganizationID.String()}
	argPos := 2

	if len(filter.Statuses) > 0 {
		where = append(where, fmt.Sprintf("status = $%d", argPos))
		args = append(args, filter.Statuses[0].String())
		argPos++
	}
	if len(filter.Severities) > 0 {
		where = append(where, fmt.Sprintf("severity = $%d", argPos))
		args = append(args, filter.Severities[0].String())
		argPos++
	}
	if filter.Query != "" {
		where = append(where, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+filter.Query+"%")
		argPos++
	}

	sort, err := platformpagination.ParseSort(filter.SortBy, filter.SortOrder, incidentSortColumns, "created_at")
	if err != nil {
		return nil, "", err
	}
	limit := platformpagination.NormalizePageSize(filter.Limit, 20, 100)

	if filter.Cursor != "" {
		keyset, err := platformpagination.DecodeKeysetCursor(filter.Cursor)
		if err != nil {
			return nil, "", domain.ErrInvalidCursor
		}
		if keyset.SortField != "" && keyset.SortField != sort.Field {
			return nil, "", domain.ErrInvalidCursor
		}
		if keyset.SortDir != "" && keyset.SortDir != sort.Direction {
			return nil, "", domain.ErrInvalidCursor
		}
		op := "<"
		if sort.Direction == platformpagination.Asc {
			op = ">"
		}
		switch sort.SQLColumn {
		case "created_at", "updated_at":
			cursorTime, err := time.Parse(time.RFC3339Nano, keyset.Primary)
			if err != nil {
				return nil, "", domain.ErrInvalidCursor
			}
			where = append(where, fmt.Sprintf("(%s %s $%d OR (%s = $%d AND external_id %s $%d))", sort.SQLColumn, op, argPos, sort.SQLColumn, argPos, op, argPos+1))
			args = append(args, cursorTime, keyset.ID)
			argPos += 2
		default:
			where = append(where, fmt.Sprintf("(%s %s $%d OR (%s = $%d AND external_id %s $%d))", sort.SQLColumn, op, argPos, sort.SQLColumn, argPos, op, argPos+1))
			args = append(args, keyset.Primary, keyset.ID)
			argPos += 2
		}
	}

	order := "DESC"
	if sort.Direction == platformpagination.Asc {
		order = "ASC"
	}
	whereSQL := strings.Join(where, " AND ")
	listSQL := fmt.Sprintf(`
		SELECT external_id, organization_id, title, description, severity, priority, status, source,
			created_at, updated_at, resolved_at, version
		FROM incidents
		WHERE %s
		ORDER BY %s %s, external_id %s
		LIMIT %d
	`, whereSQL, sort.SQLColumn, order, order, limit+1)

	rows, err := r.db(ctx).Query(ctx, listSQL, args...)
	if err != nil {
		return nil, "", fmt.Errorf("search incidents: %w", err)
	}
	defer rows.Close()

	incidents := make([]*domain.Incident, 0, limit)
	for rows.Next() {
		incident, err := scanIncidentRow(rows)
		if err != nil {
			return nil, "", err
		}
		incidents = append(incidents, incident)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var next string
	if len(incidents) > limit {
		last := incidents[limit-1]
		primary := primarySortValue(last, sort.SQLColumn)
		next = platformpagination.EncodeKeysetCursor(sort.Field, string(sort.Direction), primary, last.ID().String())
		incidents = incidents[:limit]
	}
	return incidents, next, nil
}

func scanIncidentRow(rows pgx.Rows) (*domain.Incident, error) {
	var externalID, orgID, title, description, severity, priority, status, source string
	var createdAt, updatedAt time.Time
	var resolvedAt *time.Time
	var version int
	if err := rows.Scan(
		&externalID, &orgID, &title, &description, &severity, &priority, &status, &source,
		&createdAt, &updatedAt, &resolvedAt, &version,
	); err != nil {
		return nil, fmt.Errorf("scan incident: %w", err)
	}
	return toDomainIncident(externalID, orgID, title, description, severity, priority, status, source, createdAt, updatedAt, resolvedAt, version)
}

func primarySortValue(incident *domain.Incident, column string) string {
	switch column {
	case "updated_at":
		return incident.UpdatedAt().Format(time.RFC3339Nano)
	case "severity":
		return incident.Severity().String()
	case "status":
		return incident.Status().String()
	case "title":
		return incident.Title().String()
	default:
		return incident.CreatedAt().Format(time.RFC3339Nano)
	}
}
