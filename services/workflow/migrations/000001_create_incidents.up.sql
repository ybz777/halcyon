CREATE TABLE IF NOT EXISTS incidents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_id TEXT NOT NULL UNIQUE,
    organization_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL,
    priority TEXT NOT NULL,
    status TEXT NOT NULL,
    source TEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_incidents_organization_id ON incidents (organization_id);
CREATE INDEX IF NOT EXISTS idx_incidents_status ON incidents (status);
CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents (severity);
CREATE INDEX IF NOT EXISTS idx_incidents_created_at ON incidents (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_org_status ON incidents (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_incidents_org_created_at ON incidents (organization_id, created_at DESC);

CREATE TABLE IF NOT EXISTS incident_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id TEXT NOT NULL REFERENCES incidents (external_id),
    organization_id TEXT NOT NULL,
    from_status TEXT,
    to_status TEXT NOT NULL,
    from_severity TEXT,
    to_severity TEXT,
    actor_id TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_incident_history_incident_id ON incident_history (incident_id);
CREATE INDEX IF NOT EXISTS idx_incident_history_organization_id ON incident_history (organization_id);
CREATE INDEX IF NOT EXISTS idx_incident_history_occurred_at ON incident_history (occurred_at DESC);
