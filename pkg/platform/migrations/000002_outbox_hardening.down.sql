DROP INDEX IF EXISTS idx_outbox_organization_id;
DROP INDEX IF EXISTS idx_outbox_aggregate_id;
DROP INDEX IF EXISTS idx_outbox_next_attempt_at;
DROP INDEX IF EXISTS idx_outbox_published_at;
DROP INDEX IF EXISTS idx_outbox_event_id;

ALTER TABLE outbox DROP COLUMN IF EXISTS next_attempt_at;
