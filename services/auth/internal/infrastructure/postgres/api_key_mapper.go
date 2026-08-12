package postgres

import (
	"fmt"
	"time"

	"github.com/halcyon-io/halcyon/services/auth/internal/domain"
	"github.com/jackc/pgx/v5"
)

func scanServiceAccount(rawID, orgID, name, description, status string, createdAt time.Time) (*domain.ServiceAccount, error) {
	id, err := domain.ParseServiceAccountID(rawID)
	if err != nil {
		return nil, err
	}
	organizationID, err := domain.ParseOrganizationID(orgID)
	if err != nil {
		return nil, err
	}
	parsedStatus, err := domain.ParseServiceAccountStatus(status)
	if err != nil {
		return nil, err
	}
	return domain.RehydrateServiceAccount(id, organizationID, name, description, parsedStatus, createdAt), nil
}

func scanAPIKey(row pgx.Row) (*domain.APIKey, error) {
	var (
		rawID, keyID, serviceAccountID, name, prefix, hashedSecret, statusRaw, createdBy string
		scopes                                                                           []string
		createdAt                                                                        time.Time
		expiresAt, lastUsedAt, revokedAt                                                 *time.Time
	)

	if err := row.Scan(
		&rawID, &keyID, &serviceAccountID, &name, &prefix, &hashedSecret, &scopes, &statusRaw,
		&createdAt, &expiresAt, &lastUsedAt, &revokedAt, &createdBy,
	); err != nil {
		return nil, err
	}

	id, err := domain.ParseAPIKeyID(rawID)
	if err != nil {
		return nil, err
	}
	parsedKeyID, err := domain.ParseKeyID(keyID)
	if err != nil {
		return nil, err
	}
	saID, err := domain.ParseServiceAccountID(serviceAccountID)
	if err != nil {
		return nil, err
	}
	hash, err := domain.ParsePasswordHash(hashedSecret)
	if err != nil {
		return nil, fmt.Errorf("parse api key hash: %w", err)
	}
	parsedScopes, err := domain.ParseAPIScopes(scopes)
	if err != nil {
		return nil, err
	}
	status, err := domain.ParseAPIKeyStatus(statusRaw)
	if err != nil {
		return nil, err
	}
	creator, err := domain.ParseUserID(createdBy)
	if err != nil {
		return nil, err
	}

	return domain.RehydrateAPIKey(
		id, parsedKeyID, saID, name, prefix, hash, parsedScopes, status,
		createdAt, expiresAt, lastUsedAt, revokedAt, creator,
	), nil
}

const apiKeySelectColumns = `
	id, key_id, service_account_id, name, prefix, hashed_secret, scopes, status,
	created_at, expires_at, last_used_at, revoked_at, created_by
`
