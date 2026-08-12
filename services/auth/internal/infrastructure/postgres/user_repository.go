package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"github.com/halcyon-io/halcyon/services/auth/internal/infrastructure/postgres/sqlcgen"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepository persists users in PostgreSQL.
type UserRepository struct {
	pool    *pgxpool.Pool
	metrics *Metrics
}

// NewUserRepository creates a PostgreSQL-backed UserRepository.
func NewUserRepository(pool *pgxpool.Pool, metrics *Metrics) *UserRepository {
	return &UserRepository{pool: pool, metrics: metrics}
}

func (r *UserRepository) queries(ctx context.Context) *sqlcgen.Queries {
	return sqlcgen.New(QuerierFromContext(ctx, r.pool))
}

// Create inserts a new user row.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	start := time.Now()
	params := fromDomainUser(user)

	_, err := r.queries(ctx).CreateUser(ctx, params)
	r.metrics.observe("create_user", start, err)

	if err != nil {
		return fmt.Errorf("create user: %w", mapSQLError(err))
	}

	return nil
}

// Update updates an existing user with optimistic locking.
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	start := time.Now()
	params := updateParamsFromUser(user)

	_, err := r.queries(ctx).UpdateUser(ctx, params)
	r.metrics.observe("update_user", start, err)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrVersionConflict
		}
		return fmt.Errorf("update user: %w", mapSQLError(err))
	}

	return nil
}

// Delete soft-deletes a user by external ID.
func (r *UserRepository) Delete(ctx context.Context, id domain.UserID) error {
	start := time.Now()
	now := time.Now().UTC()

	rows, err := r.queries(ctx).SoftDeleteUser(ctx, sqlcgen.SoftDeleteUserParams{
		ExternalID: id.String(),
		DeletedAt:  timeToTimestamptz(now),
	})
	r.metrics.observe("delete_user", start, err)

	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if rows == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// FindByID loads a user by external ID.
func (r *UserRepository) FindByID(ctx context.Context, id domain.UserID) (*domain.User, error) {
	start := time.Now()

	row, err := r.queries(ctx).GetUserByExternalID(ctx, id.String())
	r.metrics.observe("find_user_by_id", start, err)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return toDomainUser(row)
}

// FindByEmail loads a user by email address.
func (r *UserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	start := time.Now()

	row, err := r.queries(ctx).GetUserByEmail(ctx, email.String())
	r.metrics.observe("find_user_by_email", start, err)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return toDomainUser(row)
}

// ExistsByEmail reports whether an active user exists for the email.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email domain.Email) (bool, error) {
	start := time.Now()

	exists, err := r.queries(ctx).UserExistsByEmail(ctx, email.String())
	r.metrics.observe("exists_user_by_email", start, err)

	if err != nil {
		return false, fmt.Errorf("exists user by email: %w", err)
	}

	return exists, nil
}

var _ ports.UserRepository = (*UserRepository)(nil)
