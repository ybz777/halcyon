-- Extend processed_events for tenant-aware idempotent consumption.

ALTER TABLE processed_events
    ADD COLUMN IF NOT EXISTS organization_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'completed'
        CHECK (status IN ('processing', 'completed', 'failed'));

CREATE INDEX IF NOT EXISTS idx_processed_events_org
    ON processed_events (organization_id, consumer_group);

ALTER TABLE outbox
    ADD COLUMN IF NOT EXISTS request_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS actor_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS subject_id TEXT NOT NULL DEFAULT '';
