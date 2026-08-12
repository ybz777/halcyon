package domain

import (
	"fmt"
	"time"
)

// RefreshTokenRecord is the refresh token aggregate root.
type RefreshTokenRecord struct {
	tokenID       JTI
	familyID      FamilyID
	parentTokenID *JTI
	userID        UserID
	tokenHash     string
	issuedAt      time.Time
	expiresAt     time.Time
	consumedAt    *time.Time
	revokedAt     *time.Time
	ipAddress     string
	userAgent     string
	status        TokenStatus
}

// NewRefreshTokenRecord creates an active refresh token aggregate.
func NewRefreshTokenRecord(
	tokenID JTI,
	familyID FamilyID,
	parentTokenID *JTI,
	userID UserID,
	tokenHash string,
	issuedAt, expiresAt time.Time,
	ipAddress, userAgent string,
) (*RefreshTokenRecord, error) {
	if tokenID.IsZero() || familyID.IsZero() || userID.IsZero() || tokenHash == "" {
		return nil, fmt.Errorf("%w: incomplete refresh token record", ErrInvalidRefreshToken)
	}
	if !expiresAt.After(issuedAt) {
		return nil, fmt.Errorf("%w: invalid expiry", ErrInvalidRefreshToken)
	}
	return &RefreshTokenRecord{
		tokenID:       tokenID,
		familyID:      familyID,
		parentTokenID: parentTokenID,
		userID:        userID,
		tokenHash:     tokenHash,
		issuedAt:      issuedAt.UTC(),
		expiresAt:     expiresAt.UTC(),
		ipAddress:     ipAddress,
		userAgent:     userAgent,
		status:        TokenStatusActive,
	}, nil
}

// RehydrateRefreshTokenRecord restores a persisted refresh token aggregate.
func RehydrateRefreshTokenRecord(
	tokenID JTI,
	familyID FamilyID,
	parentTokenID *JTI,
	userID UserID,
	tokenHash string,
	issuedAt, expiresAt time.Time,
	consumedAt, revokedAt *time.Time,
	ipAddress, userAgent string,
	status TokenStatus,
) *RefreshTokenRecord {
	return &RefreshTokenRecord{
		tokenID:       tokenID,
		familyID:      familyID,
		parentTokenID: parentTokenID,
		userID:        userID,
		tokenHash:     tokenHash,
		issuedAt:      issuedAt.UTC(),
		expiresAt:     expiresAt.UTC(),
		consumedAt:    consumedAt,
		revokedAt:     revokedAt,
		ipAddress:     ipAddress,
		userAgent:     userAgent,
		status:        status,
	}
}

func (r *RefreshTokenRecord) TokenID() JTI           { return r.tokenID }
func (r *RefreshTokenRecord) FamilyID() FamilyID     { return r.familyID }
func (r *RefreshTokenRecord) ParentTokenID() *JTI    { return r.parentTokenID }
func (r *RefreshTokenRecord) UserID() UserID         { return r.userID }
func (r *RefreshTokenRecord) TokenHash() string      { return r.tokenHash }
func (r *RefreshTokenRecord) IssuedAt() time.Time    { return r.issuedAt }
func (r *RefreshTokenRecord) ExpiresAt() time.Time   { return r.expiresAt }
func (r *RefreshTokenRecord) ConsumedAt() *time.Time { return r.consumedAt }
func (r *RefreshTokenRecord) RevokedAt() *time.Time  { return r.revokedAt }
func (r *RefreshTokenRecord) IPAddress() string      { return r.ipAddress }
func (r *RefreshTokenRecord) UserAgent() string      { return r.userAgent }
func (r *RefreshTokenRecord) Status() TokenStatus    { return r.status }

// ValidateForRefresh enforces domain rules before rotation.
func (r *RefreshTokenRecord) ValidateForRefresh(now time.Time) error {
	switch r.status {
	case TokenStatusConsumed:
		return ErrRefreshTokenReused
	case TokenStatusRevoked:
		return ErrRefreshTokenRevoked
	case TokenStatusExpired:
		return ErrTokenExpired
	case TokenStatusActive:
		if now.UTC().After(r.expiresAt) {
			return ErrTokenExpired
		}
		return nil
	default:
		return ErrInvalidRefreshToken
	}
}

// MarkConsumed transitions an active token to consumed.
func (r *RefreshTokenRecord) MarkConsumed(now time.Time) error {
	if r.status != TokenStatusActive {
		return r.ValidateForRefresh(now)
	}
	now = now.UTC()
	r.status = TokenStatusConsumed
	r.consumedAt = &now
	return nil
}

// MarkRevoked transitions a token to revoked.
func (r *RefreshTokenRecord) MarkRevoked(now time.Time) {
	now = now.UTC()
	r.status = TokenStatusRevoked
	r.revokedAt = &now
}
