package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	processedStatusProcessing = "processing"
	processedStatusCompleted  = "completed"
)

// AuditRepository implements append-only audit persistence.
type AuditRepository struct {
	pool *pgxpool.Pool
}

// NewAuditRepository creates an AuditRepository.
func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

// Insert appends an audit event.
func (r *AuditRepository) Insert(ctx context.Context, event domain.AuditEvent) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO audit_events (
			id, organization_id, event_type, action, actor_type, actor_id,
			target_type, target_id, result, occurred_at, request_id, correlation_id,
			ip_address, user_agent, source_service, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, now()
		)
	`, event.ID.String(), event.OrganizationID, event.EventType, event.Action, string(event.ActorType), nullIfEmpty(event.ActorID),
		string(event.TargetType), nullIfEmpty(event.TargetID), string(event.Result), event.OccurredAt.UTC(),
		event.RequestID, event.CorrelationID, event.IPAddress, event.UserAgent, event.SourceService, metadata)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

// PersistIdempotent inserts audit event and idempotency record in one transaction.
func (r *AuditRepository) PersistIdempotent(ctx context.Context, eventID, consumerGroup, organizationID string, event domain.AuditEvent) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO processed_events (event_id, consumer_group, organization_id, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_id, consumer_group) DO NOTHING
		RETURNING true
	`, eventID, consumerGroup, organizationID, processedStatusProcessing).Scan(&inserted)
	if err != nil {
		if err == pgx.ErrNoRows {
			return true, nil
		}
		return false, fmt.Errorf("claim processed event: %w", err)
	}

	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return false, fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO audit_events (
			id, organization_id, event_type, action, actor_type, actor_id,
			target_type, target_id, result, occurred_at, request_id, correlation_id,
			ip_address, user_agent, source_service, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15, $16, now()
		)
	`, event.ID.String(), event.OrganizationID, event.EventType, event.Action, string(event.ActorType), nullIfEmpty(event.ActorID),
		string(event.TargetType), nullIfEmpty(event.TargetID), string(event.Result), event.OccurredAt.UTC(),
		event.RequestID, event.CorrelationID, event.IPAddress, event.UserAgent, event.SourceService, metadata)
	if err != nil {
		return false, fmt.Errorf("insert audit event: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE processed_events
		SET status = $3, processed_at = now()
		WHERE event_id = $1 AND consumer_group = $2
	`, eventID, consumerGroup, processedStatusCompleted)
	if err != nil {
		return false, fmt.Errorf("mark processed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit tx: %w", err)
	}
	return false, nil
}

// FindByID returns an audit event scoped to organization.
func (r *AuditRepository) FindByID(ctx context.Context, organizationID string, id domain.AuditEventID) (domain.AuditEvent, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, organization_id, event_type, action, actor_type, actor_id,
			target_type, target_id, result, occurred_at, request_id, correlation_id,
			ip_address, user_agent, source_service, metadata, created_at
		FROM audit_events
		WHERE organization_id = $1 AND id = $2
	`, organizationID, id.String())

	event, err := scanAuditEvent(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return domain.AuditEvent{}, domain.ErrAuditEventNotFound
		}
		return domain.AuditEvent{}, err
	}
	return event, nil
}

// List returns cursor-paginated audit events.
func (r *AuditRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.AuditEvent, string, error) {
	if filter.OrganizationID == "" {
		return nil, "", domain.ErrInvalidFilter
	}

	limit := domain.NormalizePageSize(filter.PageSize)
	args := []any{filter.OrganizationID}
	where := []string{"organization_id = $1"}

	argPos := 2
	addFilter := func(clause string, value any) {
		where = append(where, fmt.Sprintf(clause, argPos))
		args = append(args, value)
		argPos++
	}

	if filter.EventType != "" {
		addFilter("event_type = $%d", filter.EventType)
	}
	if filter.ActorID != "" {
		addFilter("actor_id = $%d", filter.ActorID)
	}
	if filter.ActorType != "" {
		addFilter("actor_type = $%d", string(filter.ActorType))
	}
	if filter.TargetType != "" {
		addFilter("target_type = $%d", string(filter.TargetType))
	}
	if filter.TargetID != "" {
		addFilter("target_id = $%d", filter.TargetID)
	}
	if filter.Result != "" {
		addFilter("result = $%d", string(filter.Result))
	}
	if filter.StartTime != nil {
		addFilter("occurred_at >= $%d", filter.StartTime.UTC())
	}
	if filter.EndTime != nil {
		addFilter("occurred_at <= $%d", filter.EndTime.UTC())
	}
	if !filter.IncludeSecurity {
		where = append(where, "event_type NOT LIKE 'security.%'")
		where = append(where, "event_type <> 'auth.login.failed'")
		where = append(where, "event_type NOT LIKE 'authz.%'")
	}

	if filter.Cursor != "" {
		cursorTime, cursorID, err := domain.DecodeCursor(filter.Cursor)
		if err != nil {
			return nil, "", err
		}
		where = append(where, fmt.Sprintf("(occurred_at < $%d OR (occurred_at = $%d AND id < $%d))", argPos, argPos, argPos+1))
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}

	query := fmt.Sprintf(`
		SELECT id, organization_id, event_type, action, actor_type, actor_id,
			target_type, target_id, result, occurred_at, request_id, correlation_id,
			ip_address, user_agent, source_service, metadata, created_at
		FROM audit_events
		WHERE %s
		ORDER BY occurred_at DESC, id DESC
		LIMIT %d
	`, strings.Join(where, " AND "), limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.AuditEvent, 0, limit)
	for rows.Next() {
		event, err := scanAuditEvent(rows)
		if err != nil {
			return nil, "", err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(events) > limit {
		last := events[limit-1]
		nextCursor = domain.EncodeCursor(last.OccurredAt, last.ID.String())
		events = events[:limit]
	}
	return events, nextCursor, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanAuditEvent(row scannable) (domain.AuditEvent, error) {
	var (
		id, orgID, eventType, action, actorType, actorID, targetType, targetID, result string
		requestID, correlationID, ipAddress, userAgent, sourceService                  string
		metadataJSON                                                                   []byte
		occurredAt, createdAt                                                          time.Time
	)
	if err := row.Scan(&id, &orgID, &eventType, &action, &actorType, &actorID,
		&targetType, &targetID, &result, &occurredAt, &requestID, &correlationID,
		&ipAddress, &userAgent, &sourceService, &metadataJSON, &createdAt); err != nil {
		return domain.AuditEvent{}, err
	}

	parsedID, err := domain.ParseAuditEventID(id)
	if err != nil {
		return domain.AuditEvent{}, err
	}

	metadata := map[string]string{}
	if len(metadataJSON) > 0 {
		_ = json.Unmarshal(metadataJSON, &metadata)
	}

	return domain.AuditEvent{
		ID:             parsedID,
		OrganizationID: orgID,
		EventType:      eventType,
		Action:         action,
		ActorType:      domain.ActorType(actorType),
		ActorID:        actorID,
		TargetType:     domain.TargetType(targetType),
		TargetID:       targetID,
		Result:         domain.Result(result),
		OccurredAt:     occurredAt.UTC(),
		RequestID:      requestID,
		CorrelationID:  correlationID,
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
		SourceService:  sourceService,
		Metadata:       metadata,
		CreatedAt:      createdAt.UTC(),
	}, nil
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
