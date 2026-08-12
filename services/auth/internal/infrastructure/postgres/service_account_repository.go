package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ServiceAccountRepository persists service accounts in PostgreSQL.
type ServiceAccountRepository struct {
	pool *pgxpool.Pool
}

// NewServiceAccountRepository creates a ServiceAccountRepository.
func NewServiceAccountRepository(pool *pgxpool.Pool) *ServiceAccountRepository {
	return &ServiceAccountRepository{pool: pool}
}

func (r *ServiceAccountRepository) db(ctx context.Context) DBTX {
	return QuerierFromContext(ctx, r.pool)
}

// Save inserts a service account.
func (r *ServiceAccountRepository) Save(ctx context.Context, account *domain.ServiceAccount) error {
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO service_accounts (id, organization_id, name, description, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, account.ID().String(), account.OrganizationID().String(), account.Name(),
		account.Description(), string(account.Status()), account.CreatedAt())
	if err != nil {
		return fmt.Errorf("save service account: %w", err)
	}
	return nil
}

// FindByID loads a service account by ID.
func (r *ServiceAccountRepository) FindByID(ctx context.Context, id domain.ServiceAccountID) (*domain.ServiceAccount, error) {
	row := r.db(ctx).QueryRow(ctx, `
		SELECT id, organization_id, name, description, status, created_at
		FROM service_accounts WHERE id = $1
	`, id.String())

	var rawID, orgID, name, description, status string
	var createdAt time.Time
	if err := row.Scan(&rawID, &orgID, &name, &description, &status, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrServiceAccountNotFound
		}
		return nil, fmt.Errorf("find service account: %w", err)
	}
	return scanServiceAccount(rawID, orgID, name, description, status, createdAt)
}

var _ ports.ServiceAccountRepository = (*ServiceAccountRepository)(nil)
