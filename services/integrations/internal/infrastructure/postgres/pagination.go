package postgres

import (
	"context"
	"fmt"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func listEndpointsPage(ctx context.Context, pool *pgxpool.Pool, orgID string, limit int, cursor string) ([]domain.WebhookEndpoint, string, error) {
	limit = platformpagination.NormalizePageSize(limit, 50, platformpagination.MaxPageSize)
	args := []any{orgID}
	where := "organization_id = $1"
	argPos := 2
	if cursor != "" {
		cursorTime, cursorID, err := platformpagination.DecodeTimeIDCursor(cursor)
		if err != nil {
			return nil, "", domain.ErrInvalidCursor
		}
		where += fmt.Sprintf(" AND (created_at < $%d OR (created_at = $%d AND id < $%d))", argPos, argPos, argPos+1)
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}
	args = append(args, limit+1)
	query := fmt.Sprintf(`%s WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d`, endpointSelectSQL(), where, argPos)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	return scanEndpointPage(rows, limit)
}

func scanEndpointPage(rows pgx.Rows, limit int) ([]domain.WebhookEndpoint, string, error) {
	items := make([]domain.WebhookEndpoint, 0, limit)
	for rows.Next() {
		item, err := scanEndpoint(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	var next string
	if len(items) > limit {
		last := items[limit-1]
		next = platformpagination.EncodeTimeIDCursor(last.CreatedAt, last.ID.String())
		items = items[:limit]
	}
	return items, next, rows.Err()
}

func listDeliveriesPage(ctx context.Context, pool *pgxpool.Pool, endpointID, orgID string, limit int, cursor string) ([]domain.WebhookDelivery, string, error) {
	limit = platformpagination.NormalizePageSize(limit, 50, platformpagination.MaxPageSize)
	args := []any{endpointID, orgID}
	where := "webhook_endpoint_id=$1 AND organization_id=$2"
	argPos := 3
	if cursor != "" {
		cursorTime, cursorID, err := platformpagination.DecodeTimeIDCursor(cursor)
		if err != nil {
			return nil, "", domain.ErrInvalidCursor
		}
		where += fmt.Sprintf(" AND (created_at < $%d OR (created_at = $%d AND id < $%d))", argPos, argPos, argPos+1)
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}
	args = append(args, limit+1)
	query := fmt.Sprintf(`%s WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d`, deliverySelectSQL(), where, argPos)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]domain.WebhookDelivery, 0, limit)
	for rows.Next() {
		item, err := scanDelivery(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	var next string
	if len(items) > limit {
		last := items[limit-1]
		next = platformpagination.EncodeTimeIDCursor(last.CreatedAt, last.ID.String())
		items = items[:limit]
	}
	return items, next, rows.Err()
}
