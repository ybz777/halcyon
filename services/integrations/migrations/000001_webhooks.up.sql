CREATE TABLE webhook_endpoints (
    id               TEXT PRIMARY KEY,
    organization_id  TEXT NOT NULL,
    url              TEXT NOT NULL,
    description      TEXT NOT NULL DEFAULT '',
    secret_hash      TEXT NOT NULL,
    secret_ciphertext TEXT NOT NULL,
    secret_prefix    TEXT NOT NULL,
    status           TEXT NOT NULL CHECK (status IN ('ACTIVE', 'DISABLED')),
    subscribed_events JSONB NOT NULL DEFAULT '[]',
    failure_count    INT NOT NULL DEFAULT 0,
    created_by       TEXT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    disabled_at      TIMESTAMPTZ
);

CREATE INDEX idx_webhook_endpoints_org ON webhook_endpoints (organization_id, created_at DESC);
CREATE INDEX idx_webhook_endpoints_status ON webhook_endpoints (organization_id, status);

CREATE TABLE webhook_deliveries (
    id                  TEXT PRIMARY KEY,
    webhook_endpoint_id TEXT NOT NULL REFERENCES webhook_endpoints(id),
    organization_id     TEXT NOT NULL,
    event_id            TEXT NOT NULL,
    event_type          TEXT NOT NULL,
    payload             JSONB NOT NULL,
    attempt_count       INT NOT NULL DEFAULT 0,
    status              TEXT NOT NULL CHECK (status IN ('PENDING', 'DELIVERING', 'SUCCEEDED', 'FAILED', 'RETRY_WAIT', 'DISABLED')),
    available_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_attempt_at     TIMESTAMPTZ,
    next_attempt_at     TIMESTAMPTZ,
    response_status     INT,
    response_time_ms    INT,
    last_error          TEXT NOT NULL DEFAULT '',
    locked_at           TIMESTAMPTZ,
    locked_by           TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    UNIQUE (webhook_endpoint_id, event_id)
);

CREATE INDEX idx_webhook_deliveries_ready ON webhook_deliveries (status, available_at, next_attempt_at);
CREATE INDEX idx_webhook_deliveries_endpoint ON webhook_deliveries (webhook_endpoint_id, created_at DESC);
CREATE INDEX idx_webhook_deliveries_org ON webhook_deliveries (organization_id, created_at DESC);
