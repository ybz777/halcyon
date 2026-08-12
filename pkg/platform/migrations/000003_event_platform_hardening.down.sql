ALTER TABLE outbox
    DROP COLUMN IF EXISTS request_id,
    DROP COLUMN IF EXISTS actor_id,
    DROP COLUMN IF EXISTS subject_id;

ALTER TABLE processed_events
    DROP COLUMN IF EXISTS organization_id,
    DROP COLUMN IF EXISTS status;

DROP INDEX IF EXISTS idx_processed_events_org;
