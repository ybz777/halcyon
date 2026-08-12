DROP INDEX IF EXISTS idx_dead_letter_job_id;
DROP INDEX IF EXISTS idx_jobs_scheduled;

ALTER TABLE jobs DROP CONSTRAINT IF EXISTS jobs_status_check;
ALTER TABLE jobs ADD CONSTRAINT jobs_status_check
    CHECK (status IN ('PENDING','RUNNING','RETRY_WAIT','SUCCEEDED','FAILED','CANCELLED'));

UPDATE jobs SET status = 'FAILED' WHERE status = 'DEAD_LETTERED';
