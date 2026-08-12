package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/halcyon-io/halcyon/services/integrations/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EndpointRepository struct {
	pool *pgxpool.Pool
}

func NewEndpointRepository(pool *pgxpool.Pool) *EndpointRepository {
	return &EndpointRepository{pool: pool}
}

func (r *EndpointRepository) Create(ctx context.Context, endpoint domain.WebhookEndpoint, secretCiphertext string) error {
	events, _ := json.Marshal(endpoint.SubscribedEvents)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO webhook_endpoints (
			id, organization_id, url, description, secret_hash, secret_ciphertext, secret_prefix,
			status, subscribed_events, failure_count, created_by, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, endpoint.ID.String(), endpoint.OrganizationID, endpoint.URL, endpoint.Description,
		endpoint.SecretHash, secretCiphertext, endpoint.SecretPrefix, string(endpoint.Status),
		events, endpoint.FailureCount, endpoint.CreatedBy, endpoint.CreatedAt, endpoint.UpdatedAt)
	return err
}

func (r *EndpointRepository) Update(ctx context.Context, endpoint domain.WebhookEndpoint) error {
	events, _ := json.Marshal(endpoint.SubscribedEvents)
	_, err := r.pool.Exec(ctx, `
		UPDATE webhook_endpoints
		SET url=$2, description=$3, status=$4, subscribed_events=$5, failure_count=$6,
			updated_at=$7, disabled_at=$8
		WHERE id=$1 AND organization_id=$9
	`, endpoint.ID.String(), endpoint.URL, endpoint.Description, string(endpoint.Status),
		events, endpoint.FailureCount, endpoint.UpdatedAt, endpoint.DisabledAt, endpoint.OrganizationID)
	return err
}

func (r *EndpointRepository) FindByID(ctx context.Context, orgID string, id domain.EndpointID) (domain.WebhookEndpoint, error) {
	row := r.pool.QueryRow(ctx, endpointSelectSQL()+` WHERE id=$1 AND organization_id=$2`, id.String(), orgID)
	return scanEndpoint(row)
}

func (r *EndpointRepository) FindByIDWithSecret(ctx context.Context, orgID string, id domain.EndpointID) (domain.WebhookEndpoint, string, error) {
	row := r.pool.QueryRow(ctx, endpointSelectSQL()+`, secret_ciphertext FROM webhook_endpoints WHERE id=$1 AND organization_id=$2`, id.String(), orgID)
	return scanEndpointWithCiphertext(row)
}

func (r *EndpointRepository) FindByIDWithSecretByID(ctx context.Context, id domain.EndpointID) (domain.WebhookEndpoint, string, error) {
	row := r.pool.QueryRow(ctx, endpointSelectSQL()+`, secret_ciphertext FROM webhook_endpoints WHERE id=$1`, id.String())
	return scanEndpointWithCiphertext(row)
}

func (r *EndpointRepository) ListByOrganization(ctx context.Context, orgID string, limit int, cursor string) ([]domain.WebhookEndpoint, string, error) {
	return listEndpointsPage(ctx, r.pool, orgID, limit, cursor)
}

func (r *EndpointRepository) ListActiveByOrganization(ctx context.Context, orgID string) ([]domain.WebhookEndpoint, error) {
	rows, err := r.pool.Query(ctx, endpointSelectSQL()+` WHERE organization_id=$1 AND status='ACTIVE'`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.WebhookEndpoint
	for rows.Next() {
		endpoint, err := scanEndpoint(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, endpoint)
	}
	return out, rows.Err()
}

func (r *EndpointRepository) CountByOrganization(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_endpoints WHERE organization_id=$1`, orgID).Scan(&count)
	return count, err
}

func endpointSelectSQL() string {
	return `
		SELECT id, organization_id, url, description, secret_hash, secret_prefix, status,
			subscribed_events, failure_count, created_by, created_at, updated_at, disabled_at
		FROM webhook_endpoints`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEndpoint(row rowScanner) (domain.WebhookEndpoint, error) {
	endpoint, _, err := scanEndpointFields(row, false)
	return endpoint, err
}

func scanEndpointWithCiphertext(row rowScanner) (domain.WebhookEndpoint, string, error) {
	return scanEndpointFields(row, true)
}

func scanEndpointFields(row rowScanner, withCipher bool) (domain.WebhookEndpoint, string, error) {
	var (
		id, orgID, url, desc, secretHash, secretPrefix, status, createdBy string
		eventsJSON                                                        []byte
		failureCount                                                      int
		createdAt, updatedAt                                              time.Time
		disabledAt                                                        *time.Time
		ciphertext                                                        string
	)
	dest := []any{&id, &orgID, &url, &desc, &secretHash, &secretPrefix, &status,
		&eventsJSON, &failureCount, &createdBy, &createdAt, &updatedAt, &disabledAt}
	if withCipher {
		dest = append(dest, &ciphertext)
	}
	if err := row.Scan(dest...); err != nil {
		if err == pgx.ErrNoRows {
			return domain.WebhookEndpoint{}, "", domain.ErrEndpointNotFound
		}
		return domain.WebhookEndpoint{}, "", err
	}
	parsedID, err := domain.ParseEndpointID(id)
	if err != nil {
		return domain.WebhookEndpoint{}, "", err
	}
	var events []string
	_ = json.Unmarshal(eventsJSON, &events)
	return domain.WebhookEndpoint{
		ID: parsedID, OrganizationID: orgID, URL: url, Description: desc,
		SecretHash: secretHash, SecretPrefix: secretPrefix, Status: domain.EndpointStatus(status),
		SubscribedEvents: events, FailureCount: failureCount, CreatedBy: createdBy,
		CreatedAt: createdAt.UTC(), UpdatedAt: updatedAt.UTC(), DisabledAt: disabledAt,
	}, ciphertext, nil
}
