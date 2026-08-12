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

// RefreshTokenRepository persists refresh token aggregates in PostgreSQL.
type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

// NewRefreshTokenRepository creates a RefreshTokenRepository.
func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) db(ctx context.Context) DBTX {
	return QuerierFromContext(ctx, r.pool)
}

// Save inserts a new refresh token aggregate.
func (r *RefreshTokenRepository) Save(ctx context.Context, record *domain.RefreshTokenRecord) error {
	var parentID *string
	if parent := record.ParentTokenID(); parent != nil {
		value := parent.String()
		parentID = &value
	}

	_, err := r.db(ctx).Exec(ctx, `
		INSERT INTO refresh_tokens (
			id, family_id, parent_id, user_id, token_hash,
			issued_at, expires_at, consumed_at, revoked_at,
			ip_address, user_agent, status
		) VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		record.TokenID().String(),
		record.FamilyID().String(),
		parentID,
		record.UserID().String(),
		record.TokenHash(),
		record.IssuedAt(),
		record.ExpiresAt(),
		record.ConsumedAt(),
		record.RevokedAt(),
		record.IPAddress(),
		record.UserAgent(),
		string(record.Status()),
	)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

// FindByJTI loads a refresh token by its JTI.
func (r *RefreshTokenRepository) FindByJTI(ctx context.Context, jti domain.JTI) (*domain.RefreshTokenRecord, error) {
	return r.scanRecord(r.db(ctx).QueryRow(ctx, `
		SELECT id, family_id::text, parent_id, user_id, token_hash,
			issued_at, expires_at, consumed_at, revoked_at,
			ip_address, user_agent, status
		FROM refresh_tokens
		WHERE id = $1
	`, jti.String()))
}

// Consume atomically marks an active token as consumed.
func (r *RefreshTokenRepository) Consume(ctx context.Context, jti domain.JTI) (*domain.RefreshTokenRecord, error) {
	record, err := r.scanRecord(r.db(ctx).QueryRow(ctx, `
		UPDATE refresh_tokens
		SET status = 'consumed', consumed_at = NOW()
		WHERE id = $1 AND status = 'active'
		RETURNING id, family_id::text, parent_id, user_id, token_hash,
			issued_at, expires_at, consumed_at, revoked_at,
			ip_address, user_agent, status
	`, jti.String()))
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("consume refresh token: %w", err)
	}

	existing, findErr := r.FindByJTI(ctx, jti)
	if findErr != nil {
		return nil, findErr
	}
	switch existing.Status() {
	case domain.TokenStatusConsumed:
		return nil, domain.ErrRefreshTokenReused
	case domain.TokenStatusRevoked:
		return nil, domain.ErrRefreshTokenRevoked
	case domain.TokenStatusExpired:
		return nil, domain.ErrTokenExpired
	default:
		return nil, domain.ErrInvalidRefreshToken
	}
}

// RevokeFamily revokes every active token in a family.
func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID domain.FamilyID) error {
	_, err := r.db(ctx).Exec(ctx, `
		UPDATE refresh_tokens
		SET status = 'revoked', revoked_at = NOW()
		WHERE family_id = $1::uuid AND status IN ('active', 'consumed')
	`, familyID.String())
	if err != nil {
		return fmt.Errorf("revoke token family: %w", err)
	}
	return nil
}

// RevokeByJTI revokes a single refresh token.
func (r *RefreshTokenRepository) RevokeByJTI(ctx context.Context, jti domain.JTI) error {
	tag, err := r.db(ctx).Exec(ctx, `
		UPDATE refresh_tokens
		SET status = 'revoked', revoked_at = NOW()
		WHERE id = $1 AND status = 'active'
	`, jti.String())
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInvalidRefreshToken
	}
	return nil
}

// DeleteExpired removes expired refresh tokens.
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db(ctx).Exec(ctx, `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1
	`, before.UTC())
	if err != nil {
		return 0, fmt.Errorf("delete expired refresh tokens: %w", err)
	}
	return tag.RowsAffected(), nil
}

// WithinTransaction runs a function inside a database transaction.
func (r *RefreshTokenRepository) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	_, ok := TxFromContext(ctx)
	if ok {
		return fn(ctx)
	}

	pgxTx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer pgxTx.Rollback(ctx)

	if err := fn(ContextWithTx(ctx, pgxTx)); err != nil {
		return err
	}
	return pgxTx.Commit(ctx)
}

func (r *RefreshTokenRepository) scanRecord(row pgx.Row) (*domain.RefreshTokenRecord, error) {
	var (
		id, familyID, userID, tokenHash, ipAddress, userAgent, statusRaw string
		parentID                                                         *string
		issuedAt, expiresAt                                              time.Time
		consumedAt, revokedAt                                            *time.Time
	)

	if err := row.Scan(
		&id, &familyID, &parentID, &userID, &tokenHash,
		&issuedAt, &expiresAt, &consumedAt, &revokedAt,
		&ipAddress, &userAgent, &statusRaw,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidRefreshToken
		}
		return nil, fmt.Errorf("scan refresh token: %w", err)
	}

	jti, err := domain.ParseJTI(id)
	if err != nil {
		return nil, err
	}
	family, err := domain.ParseFamilyID(familyID)
	if err != nil {
		return nil, err
	}
	uid, err := domain.ParseUserID(userID)
	if err != nil {
		return nil, err
	}
	status, err := domain.ParseTokenStatus(statusRaw)
	if err != nil {
		return nil, err
	}

	var parent *domain.JTI
	if parentID != nil {
		parsed, err := domain.ParseJTI(*parentID)
		if err != nil {
			return nil, err
		}
		parent = &parsed
	}

	return domain.RehydrateRefreshTokenRecord(
		jti, family, parent, uid, tokenHash,
		issuedAt, expiresAt, consumedAt, revokedAt,
		ipAddress, userAgent, status,
	), nil
}

var _ ports.RefreshTokenRepository = (*RefreshTokenRepository)(nil)
