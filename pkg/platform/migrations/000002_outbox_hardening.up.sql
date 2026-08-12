ALTER TABLE outbox
    ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_outbox_event_id ON outbox (event_id);
CREATE INDEX IF NOT EXISTS idx_outbox_published_at ON outbox (published_at) WHERE published_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_outbox_next_attempt_at ON outbox (next_attempt_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_outbox_aggregate_id ON outbox (aggregate_id);
CREATE INDEX IF NOT EXISTS idx_outbox_organization_id ON outbox (organization_id);
