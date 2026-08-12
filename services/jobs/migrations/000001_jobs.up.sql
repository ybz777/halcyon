CREATE TABLE jobs (
    id               TEXT PRIMARY KEY,
    organization_id  TEXT,
    job_type         TEXT NOT NULL,
    status           TEXT NOT NULL CHECK (status IN ('PENDING','RUNNING','RETRY_WAIT','SUCCEEDED','FAILED','CANCELLED')),
    payload          JSONB NOT NULL DEFAULT '{}',
    priority         TEXT NOT NULL DEFAULT 'NORMAL' CHECK (priority IN ('LOW','NORMAL','HIGH')),
    attempt_count    INT NOT NULL DEFAULT 0,
    max_attempts     INT NOT NULL DEFAULT 6,
    idempotency_key  TEXT,
    available_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at        TIMESTAMPTZ,
    locked_by        TEXT,
    last_error       TEXT NOT NULL DEFAULT '',
    last_error_category TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at       TIMESTAMPTZ,
    completed_at     TIMESTAMPTZ,
    failed_at        TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_jobs_idempotency ON jobs (organization_id, job_type, idempotency_key)
    WHERE idempotency_key IS NOT NULL AND idempotency_key <> '';

CREATE INDEX idx_jobs_claim ON jobs (status, priority DESC, available_at ASC)
    WHERE status IN ('PENDING', 'RETRY_WAIT');

CREATE INDEX idx_jobs_locked ON jobs (locked_at) WHERE status = 'RUNNING';

CREATE INDEX idx_jobs_org_created ON jobs (organization_id, created_at DESC, id DESC);

CREATE INDEX idx_jobs_type_status ON jobs (job_type, status);

CREATE TABLE job_attempts (
    id              TEXT PRIMARY KEY,
    job_id          TEXT NOT NULL REFERENCES jobs(id),
    attempt_number  INT NOT NULL,
    worker_id       TEXT NOT NULL,
    status          TEXT NOT NULL,
    error_category  TEXT NOT NULL DEFAULT '',
    error_message   TEXT NOT NULL DEFAULT '',
    started_at      TIMESTAMPTZ NOT NULL,
    completed_at    TIMESTAMPTZ,
    duration_ms     INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_job_attempts_job ON job_attempts (job_id, attempt_number DESC);

CREATE TABLE dead_letter_jobs (
    id               TEXT PRIMARY KEY,
    job_id           TEXT NOT NULL REFERENCES jobs(id),
    organization_id  TEXT,
    job_type         TEXT NOT NULL,
    failure_category TEXT NOT NULL,
    last_error       TEXT NOT NULL DEFAULT '',
    attempt_count    INT NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_dead_letter_org ON dead_letter_jobs (organization_id, created_at DESC);
