package domain

import (
	"fmt"
	"strings"
	"time"
)

// APIKey is the API key aggregate root.
type APIKey struct {
	id               APIKeyID
	keyID            KeyID
	serviceAccountID ServiceAccountID
	name             string
	prefix           string
	hashedSecret     PasswordHash
	scopes           []APIScope
	status           APIKeyStatus
	createdAt        time.Time
	expiresAt        *time.Time
	lastUsedAt       *time.Time
	revokedAt        *time.Time
	createdBy        UserID
}

// NewAPIKey creates an active API key aggregate.
func NewAPIKey(
	id APIKeyID,
	keyID KeyID,
	serviceAccountID ServiceAccountID,
	name, prefix string,
	hashedSecret PasswordHash,
	scopes []APIScope,
	createdBy UserID,
	createdAt time.Time,
	expiresAt *time.Time,
) (*APIKey, error) {
	if id.IsZero() || keyID.String() == "" || serviceAccountID.IsZero() ||
		strings.TrimSpace(name) == "" || strings.TrimSpace(prefix) == "" ||
		hashedSecret.IsZero() || createdBy.IsZero() {
		return nil, fmt.Errorf("%w: incomplete api key", ErrInvalidAPIKey)
	}
	if len(scopes) == 0 {
		return nil, fmt.Errorf("%w: scopes required", ErrInvalidAPIScope)
	}
	if expiresAt != nil && !expiresAt.After(createdAt) {
		return nil, fmt.Errorf("%w: invalid expiry", ErrInvalidAPIKey)
	}
	return &APIKey{
		id:               id,
		keyID:            keyID,
		serviceAccountID: serviceAccountID,
		name:             strings.TrimSpace(name),
		prefix:           prefix,
		hashedSecret:     hashedSecret,
		scopes:           append([]APIScope(nil), scopes...),
		status:           APIKeyStatusActive,
		createdAt:        createdAt.UTC(),
		expiresAt:        expiresAt,
		createdBy:        createdBy,
	}, nil
}

// RehydrateAPIKey restores a persisted API key aggregate.
func RehydrateAPIKey(
	id APIKeyID,
	keyID KeyID,
	serviceAccountID ServiceAccountID,
	name, prefix string,
	hashedSecret PasswordHash,
	scopes []APIScope,
	status APIKeyStatus,
	createdAt time.Time,
	expiresAt, lastUsedAt, revokedAt *time.Time,
	createdBy UserID,
) *APIKey {
	return &APIKey{
		id:               id,
		keyID:            keyID,
		serviceAccountID: serviceAccountID,
		name:             name,
		prefix:           prefix,
		hashedSecret:     hashedSecret,
		scopes:           append([]APIScope(nil), scopes...),
		status:           status,
		createdAt:        createdAt.UTC(),
		expiresAt:        expiresAt,
		lastUsedAt:       lastUsedAt,
		revokedAt:        revokedAt,
		createdBy:        createdBy,
	}
}

func (k *APIKey) ID() APIKeyID                     { return k.id }
func (k *APIKey) KeyID() KeyID                     { return k.keyID }
func (k *APIKey) ServiceAccountID() ServiceAccountID { return k.serviceAccountID }
func (k *APIKey) Name() string                     { return k.name }
func (k *APIKey) Prefix() string                   { return k.prefix }
func (k *APIKey) HashedSecret() PasswordHash       { return k.hashedSecret }
func (k *APIKey) Scopes() []APIScope               { return append([]APIScope(nil), k.scopes...) }
func (k *APIKey) Status() APIKeyStatus             { return k.status }
func (k *APIKey) CreatedAt() time.Time             { return k.createdAt }
func (k *APIKey) ExpiresAt() *time.Time            { return k.expiresAt }
func (k *APIKey) LastUsedAt() *time.Time           { return k.lastUsedAt }
func (k *APIKey) RevokedAt() *time.Time            { return k.revokedAt }
func (k *APIKey) CreatedBy() UserID                { return k.createdBy }

// ValidateForAuth enforces domain rules before authentication.
func (k *APIKey) ValidateForAuth(now time.Time) error {
	switch k.status {
	case APIKeyStatusRevoked:
		return ErrAPIKeyRevoked
	case APIKeyStatusDisabled:
		return ErrAPIKeyDisabled
	case APIKeyStatusExpired:
		return ErrAPIKeyExpired
	case APIKeyStatusActive:
		if k.expiresAt != nil && now.UTC().After(*k.expiresAt) {
			return ErrAPIKeyExpired
		}
		return nil
	default:
		return ErrInvalidAPIKey
	}
}

// MarkRevoked transitions the key to revoked.
func (k *APIKey) MarkRevoked(now time.Time) {
	now = now.UTC()
	k.status = APIKeyStatusRevoked
	k.revokedAt = &now
}

// MarkUsed records the last usage timestamp.
func (k *APIKey) MarkUsed(now time.Time) {
	now = now.UTC()
	k.lastUsedAt = &now
}

// RedactedPrefix returns a safe display prefix.
func (k *APIKey) RedactedPrefix() string {
	return RedactAPIKey(k.prefix)
}
