package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	platformpagination "github.com/halcyon-io/halcyon/pkg/platform/pagination"
	"github.com/halcyon-io/halcyon/services/auth/internal/application/ports"
	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// APIKeyRepository persists API keys in PostgreSQL.
type APIKeyRepository struct {
	pool *pgxpool.Pool
}

// NewAPIKeyRepository creates an APIKeyRepository.
func NewAPIKeyRepository(pool *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{pool: pool}
}

func (r *APIKeyRepository) db(ctx context.Context) DBTX {
	return QuerierFromContext(ctx, r.pool)
}

// Save inserts a new API key.
func (r *APIKeyRepository) Save(ctx context.Context, key *domain.APIKey) error {
	scopes := scopesToStrings(key.Scopes())
	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO api_keys (
			id, key_id, service_account_id, name, prefix, hashed_secret, scopes, status,
			created_at, expires_at, last_used_at, revoked_at, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`, key.ID().String(), key.KeyID().String(), key.ServiceAccountID().String(), key.Name(),
		key.Prefix(), key.HashedSecret().String(), scopes, string(key.Status()),
		key.CreatedAt(), key.ExpiresAt(), key.LastUsedAt(), key.RevokedAt(), key.CreatedBy().String())
	if err != nil {
		return fmt.Errorf("save api key: %w", err)
	}
	return nil
}

// FindByPrefix loads an API key by its lookup prefix.
func (r *APIKeyRepository) FindByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error) {
	row := r.db(ctx).QueryRow(ctx, `
		SELECT `+apiKeySelectColumns+` FROM api_keys WHERE prefix = $1
	`, prefix)
	key, err := scanAPIKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("find api key by prefix: %w", err)
	}
	return key, nil
}

// FindByID loads an API key by aggregate ID.
func (r *APIKeyRepository) FindByID(ctx context.Context, id domain.APIKeyID) (*domain.APIKey, error) {
	row := r.db(ctx).QueryRow(ctx, `
		SELECT `+apiKeySelectColumns+` FROM api_keys WHERE id = $1
	`, id.String())
	key, err := scanAPIKey(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("find api key by id: %w", err)
	}
	return key, nil
}

// ListByServiceAccount returns API keys for a service account.
func (r *APIKeyRepository) ListByServiceAccount(ctx context.Context, serviceAccountID domain.ServiceAccountID, limit int, cursor string) ([]*domain.APIKey, string, error) {
	limit = platformpagination.NormalizePageSize(limit, platformpagination.DefaultPageSize, platformpagination.MaxPageSize)
	args := []any{serviceAccountID.String()}
	where := "service_account_id = $1"
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
	rows, err := r.db(ctx).Query(ctx, fmt.Sprintf(`
		SELECT `+apiKeySelectColumns+`
		FROM api_keys WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d
	`, where, argPos), args...)
	if err != nil {
		return nil, "", fmt.Errorf("list api keys: %w", err)
	}
	defer rows.Close()

	keys := make([]*domain.APIKey, 0, limit)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, "", fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	var next string
	if len(keys) > limit {
		last := keys[limit-1]
		next = platformpagination.EncodeTimeIDCursor(last.CreatedAt(), last.ID().String())
		keys = keys[:limit]
	}
	return keys, next, nil
}

// Update persists API key changes.
func (r *APIKeyRepository) Update(ctx context.Context, key *domain.APIKey) error {
	scopes := scopesToStrings(key.Scopes())
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE api_keys
		SET name = $2, hashed_secret = $3, scopes = $4, status = $5,
			expires_at = $6, last_used_at = $7, revoked_at = $8
		WHERE id = $1
	`, key.ID().String(), key.Name(), key.HashedSecret().String(), scopes, string(key.Status()),
		key.ExpiresAt(), key.LastUsedAt(), key.RevokedAt())
	if err != nil {
		return fmt.Errorf("update api key: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrAPIKeyNotFound
	}
	return nil
}

// DeleteExpired removes expired API keys.
func (r *APIKeyRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		DELETE FROM api_keys WHERE expires_at IS NOT NULL AND expires_at < $1
	`, before.UTC())
	if err != nil {
		return 0, fmt.Errorf("delete expired api keys: %w", err)
	}
	return tag.RowsAffected(), nil
}

// WithinTransaction runs a function inside a database transaction.
func (r *APIKeyRepository) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := TxFromContext(ctx); ok {
		return fn(ctx)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := fn(ContextWithTx(ctx, tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func scopesToStrings(scopes []domain.APIScope) []string {
	out := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		out = append(out, scope.String())
	}
	return out
}

var _ ports.APIKeyRepository = (*APIKeyRepository)(nil)
