-- Add DEAD_LETTERED status and lease-expiration index for stale job recovery.

ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_status_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check
    CHECK (status IN ('PENDING','RUNNING','RETRY_WAIT','SUCCEEDED','FAILED','DEAD_LETTERED','CANCELLED'));

CREATE INDEX IF NOT EXISTS idx_jobs_scheduled ON jobs (available_at ASC)
    WHERE status IN ('PENDING', 'RETRY_WAIT');

CREATE UNIQUE INDEX IF NOT EXISTS idx_dead_letter_job_id ON dead_letter_jobs (job_id);

UPDATE jobs SET status = 'DEAD_LETTERED'
WHERE status = 'FAILED'
  AND id IN (SELECT job_id FROM dead_letter_jobs);
