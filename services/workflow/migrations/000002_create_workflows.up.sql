-- Durable workflow execution engine tables.

CREATE TABLE workflows (
    id              TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT workflows_org_name_unique UNIQUE (organization_id, name)
);

CREATE INDEX idx_workflows_organization_id ON workflows (organization_id);

CREATE TABLE workflow_versions (
    id              TEXT PRIMARY KEY,
    workflow_id     TEXT NOT NULL REFERENCES workflows (id) ON DELETE CASCADE,
    organization_id TEXT NOT NULL,
    version         INT NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('draft', 'published')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT workflow_versions_unique UNIQUE (workflow_id, version)
);

CREATE INDEX idx_workflow_versions_workflow_id ON workflow_versions (workflow_id);
CREATE INDEX idx_workflow_versions_org_status ON workflow_versions (organization_id, status);

CREATE TABLE workflow_steps (
    id                  TEXT PRIMARY KEY,
    workflow_version_id TEXT NOT NULL REFERENCES workflow_versions (id) ON DELETE CASCADE,
    organization_id     TEXT NOT NULL,
    step_key            TEXT NOT NULL,
    name                TEXT NOT NULL,
    step_type           TEXT NOT NULL CHECK (step_type IN ('HTTP_ACTION', 'EVENT_EMIT', 'DELAY')),
    configuration       JSONB NOT NULL DEFAULT '{}',
    timeout_seconds     INT NOT NULL DEFAULT 30,
    retry_policy        JSONB NOT NULL DEFAULT '{}',
    dependencies        JSONB NOT NULL DEFAULT '[]',
    step_order          INT NOT NULL,
    CONSTRAINT workflow_steps_version_key_unique UNIQUE (workflow_version_id, step_key)
);

CREATE INDEX idx_workflow_steps_version_order ON workflow_steps (workflow_version_id, step_order);

CREATE TABLE workflow_executions (
    id                  TEXT PRIMARY KEY,
    workflow_id         TEXT NOT NULL REFERENCES workflows (id),
    workflow_version_id TEXT NOT NULL REFERENCES workflow_versions (id),
    organization_id     TEXT NOT NULL,
    triggered_by        TEXT NOT NULL,
    status              TEXT NOT NULL CHECK (status IN ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'CANCELLED')),
    correlation_id      TEXT NOT NULL DEFAULT '',
    trigger_depth       INT NOT NULL DEFAULT 0,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_workflow_executions_org_status ON workflow_executions (organization_id, status);
CREATE INDEX idx_workflow_executions_workflow_id ON workflow_executions (workflow_id, created_at DESC);

CREATE TABLE workflow_step_executions (
    id               TEXT PRIMARY KEY,
    execution_id     TEXT NOT NULL REFERENCES workflow_executions (id) ON DELETE CASCADE,
    organization_id  TEXT NOT NULL,
    step_key         TEXT NOT NULL,
    status           TEXT NOT NULL CHECK (status IN ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'RETRY_WAIT', 'CANCELLED')),
    attempt_count    INT NOT NULL DEFAULT 0,
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    last_error       TEXT,
    output_reference JSONB,
    CONSTRAINT workflow_step_executions_unique UNIQUE (execution_id, step_key)
);

CREATE INDEX idx_workflow_step_executions_execution ON workflow_step_executions (execution_id);

CREATE TABLE workflow_jobs (
    id                TEXT PRIMARY KEY,
    organization_id   TEXT NOT NULL,
    execution_id      TEXT NOT NULL REFERENCES workflow_executions (id) ON DELETE CASCADE,
    step_execution_id TEXT NOT NULL REFERENCES workflow_step_executions (id) ON DELETE CASCADE,
    step_key          TEXT NOT NULL,
    status            TEXT NOT NULL CHECK (status IN ('READY', 'RUNNING', 'SUCCEEDED', 'FAILED', 'RETRY_WAIT', 'CANCELLED')),
    attempt           INT NOT NULL DEFAULT 0,
    available_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at         TIMESTAMPTZ,
    locked_by         TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at      TIMESTAMPTZ
);

CREATE INDEX idx_workflow_jobs_claim ON workflow_jobs (status, available_at) WHERE status IN ('READY', 'RETRY_WAIT');
CREATE INDEX idx_workflow_jobs_execution ON workflow_jobs (execution_id);
CREATE INDEX idx_workflow_jobs_locked_at ON workflow_jobs (locked_at) WHERE status = 'RUNNING';

CREATE TABLE execution_attempts (
    id               TEXT PRIMARY KEY,
    job_id           TEXT NOT NULL REFERENCES workflow_jobs (id) ON DELETE CASCADE,
    organization_id  TEXT NOT NULL,
    attempt          INT NOT NULL,
    status           TEXT NOT NULL,
    started_at       TIMESTAMPTZ NOT NULL,
    completed_at     TIMESTAMPTZ,
    error_message    TEXT,
    output_reference JSONB
);

CREATE INDEX idx_execution_attempts_job ON execution_attempts (job_id, attempt);

CREATE TABLE workflow_idempotency_keys (
    organization_id  TEXT NOT NULL,
    idempotency_key  TEXT NOT NULL,
    execution_id     TEXT NOT NULL REFERENCES workflow_executions (id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, idempotency_key)
);
