-- Append-only audit event store.

CREATE TABLE audit_events (
    id              TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    action          TEXT NOT NULL,
    actor_type      TEXT NOT NULL CHECK (actor_type IN ('USER', 'SERVICE', 'API_KEY', 'SYSTEM', 'ANONYMOUS')),
    actor_id        TEXT,
    target_type     TEXT NOT NULL,
    target_id       TEXT,
    result          TEXT NOT NULL CHECK (result IN ('SUCCESS', 'FAILURE', 'DENIED')),
    occurred_at     TIMESTAMPTZ NOT NULL,
    request_id      TEXT NOT NULL DEFAULT '',
    correlation_id  TEXT NOT NULL DEFAULT '',
    ip_address      TEXT NOT NULL DEFAULT '',
    user_agent      TEXT NOT NULL DEFAULT '',
    source_service  TEXT NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_events_org_occurred ON audit_events (organization_id, occurred_at DESC, id DESC);
CREATE INDEX idx_audit_events_org_actor ON audit_events (organization_id, actor_id, occurred_at DESC) WHERE actor_id IS NOT NULL AND actor_id <> '';
CREATE INDEX idx_audit_events_org_type ON audit_events (organization_id, event_type, occurred_at DESC);
CREATE INDEX idx_audit_events_org_target ON audit_events (organization_id, target_type, target_id, occurred_at DESC) WHERE target_id IS NOT NULL AND target_id <> '';
CREATE INDEX idx_audit_events_org_result ON audit_events (organization_id, result, occurred_at DESC);
