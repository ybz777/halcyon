DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS service_accounts;

CREATE TABLE service_accounts (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_service_accounts_organization_id ON service_accounts (organization_id);
CREATE INDEX idx_service_accounts_status ON service_accounts (status);

CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    key_id TEXT NOT NULL UNIQUE,
    service_account_id TEXT NOT NULL REFERENCES service_accounts (id),
    name TEXT NOT NULL,
    prefix TEXT NOT NULL UNIQUE,
    hashed_secret TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled', 'revoked', 'expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_by TEXT NOT NULL
);

CREATE INDEX idx_api_keys_prefix ON api_keys (prefix);
CREATE INDEX idx_api_keys_service_account_id ON api_keys (service_account_id);
CREATE INDEX idx_api_keys_status ON api_keys (status);
CREATE INDEX idx_api_keys_expires_at ON api_keys (expires_at);
