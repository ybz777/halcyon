-- Align claim index ordering with worker claim query priority semantics.
-- The prior idx_jobs_claim used TEXT priority DESC which does not match
-- CASE priority WHEN 'HIGH' THEN 3 WHEN 'NORMAL' THEN 2 ELSE 1 END ordering.

CREATE INDEX IF NOT EXISTS idx_jobs_claim_priority
    ON jobs (
        status,
        (CASE priority WHEN 'HIGH' THEN 3 WHEN 'NORMAL' THEN 2 ELSE 1 END) DESC,
        available_at ASC,
        created_at ASC
    )
    WHERE status IN ('PENDING', 'RETRY_WAIT');
