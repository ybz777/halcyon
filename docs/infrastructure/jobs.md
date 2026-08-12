# Background Jobs

Production-grade durable job execution for Halcyon platform work.

## Architecture

```
Producer (application / dispatcher)
        |
        v
  pkg/platform/jobs.Producer
        |
        v
 PostgreSQL (halcyon_jobs.jobs)
        |
        v
   Worker Pool (job-worker)
        |
        v
  Handler Registry (per job type)
```

Kafka remains the event transport. Jobs provide durable execution with retries, leases, and dead-letter handling.

## State Machine

| State | Description |
|-------|-------------|
| `PENDING` | Queued, waiting for worker |
| `RUNNING` | Claimed by a worker with active lease |
| `RETRY_WAIT` | Failed retryably; waiting for `available_at` |
| `SUCCEEDED` | Completed successfully |
| `FAILED` | Permanent failure (non-retryable) |
| `DEAD_LETTERED` | Max attempts exceeded; row in `dead_letter_jobs` |
| `CANCELLED` | Cancelled before completion |

### Valid transitions

```
PENDING → RUNNING → SUCCEEDED
                 → RETRY_WAIT → RUNNING
                 → FAILED
                 → DEAD_LETTERED
                 → CANCELLED

Expired RUNNING lease → RETRY_WAIT (if attempts < max)
                     → DEAD_LETTERED (if attempts >= max)
```

## Worker lifecycle

1. Poll eligible jobs (`PENDING` / `RETRY_WAIT`, `available_at <= now`)
2. Atomically claim with `FOR UPDATE SKIP LOCKED`
3. Increment `attempt_count`, set `RUNNING` + lease
4. Execute handler with bounded timeout
5. Renew lease on heartbeat interval for long jobs
6. On success: `SUCCEEDED`
7. On retryable failure: `RETRY_WAIT` with backoff
8. On permanent failure: `FAILED`
9. On max attempts: `DEAD_LETTERED` + DLQ row

## Leases

- `JOB_LEASE_DURATION` — claim lease stored in `locked_at`
- `JOB_LEASE_RENEW_INTERVAL` — heartbeat renewal during execution
- Expired `RUNNING` jobs are recovered automatically (no manual repair)

## Retries

Exponential backoff with jitter (`pkg/platform/jobs/backoff.go`):

```
delay = min(maxDelay, baseDelay * multiplier^(attempt-1)) + jitter
```

Handlers return `HandlerError` with `Retryable` flag. Error categories:

- `TRANSIENT`, `DEPENDENCY`, `TIMEOUT`, `RATE_LIMITED` — typically retryable
- `VALIDATION`, `AUTHORIZATION`, `NOT_FOUND` — permanent

### Retry ownership

| Layer | Retries |
|-------|---------|
| HTTP client (webhook) | Bounded per delivery attempt |
| Job worker | Bounded by `max_attempts` + backoff |
| Kafka consumer | Separate inbox retry (dispatchers) |

Do not stack unbounded retries across all three layers.

## Dead letter

Jobs exceeding `max_attempts` transition to `DEAD_LETTERED`. Diagnostic metadata is stored in `dead_letter_jobs` (no payload secrets).

## Idempotency

`idempotency_key` per `(organization_id, job_type)` prevents duplicate enqueue. Handler side effects must be idempotent (delivery rows, provider idempotency keys).

## Graceful shutdown

On `SIGTERM`/`SIGINT`:

1. Stop claiming new jobs
2. Wait for in-flight handlers up to `JOB_SHUTDOWN_TIMEOUT`
3. Unfinished jobs remain recoverable via lease expiration

## Observability

### Metrics

- `jobs_created_total{job_type}`
- `jobs_started_total{job_type}`
- `jobs_succeeded_total{job_type}`
- `jobs_failed_total{job_type,error_category}`
- `jobs_retried_total{job_type}`
- `jobs_dead_lettered_total{job_type}`
- `job_execution_duration_seconds{job_type}`
- `job_queue_delay_seconds{job_type}`
- `job_active{job_type}`
- `job_lease_expirations_total{job_type}`

### Tracing

Each execution creates a new `job <type>` span with `job.type`, `job.attempt`, `job.state`.

### Logging

Structured fields: `job_id`, `job_type`, `attempt`, `error_category`, `duration`. Payloads are never logged.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `JOB_WORKER_ENABLED` | `true` | Enable worker |
| `JOB_WORKER_COUNT` | `1` | Logical worker processes (scale via replicas) |
| `JOB_MAX_CONCURRENT` | `8` | Max in-flight handlers per process |
| `JOB_BATCH_SIZE` | `10` | Claim batch size |
| `JOB_POLL_INTERVAL` | `1s` | Idle poll interval |
| `JOB_LEASE_DURATION` | `30s` | Claim lease |
| `JOB_LEASE_RENEW_INTERVAL` | `10s` | Heartbeat interval |
| `JOB_SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown wait |
| `JOB_MAX_ATTEMPTS` | `6` | Default max attempts |
| `JOB_EXECUTION_TIMEOUT` | `5m` | Handler timeout |
| `JOB_RETRY_BASE_DELAY` | `1s` | Backoff base |
| `JOB_RETRY_MAX_DELAY` | `5m` | Backoff cap |
| `JOB_RETRY_JITTER` | `0.5` | Jitter fraction |

## Operational signals

Monitor:

- High `jobs_retried_total` rate
- Growing `jobs_dead_lettered_total`
- Rising `job_queue_delay_seconds` p95
- `job_lease_expirations_total` spikes (worker crashes)
- `job_active` near `JOB_MAX_CONCURRENT` (saturation)

## Local development

```bash
# Start dependencies
docker compose -f infra/docker/compose.yml up -d postgres

# Run migrations + worker
make run-worker
```

Requires `JOBS_DATABASE_URL` and optional `INTEGRATIONS_DATABASE_URL` / `NOTIFICATIONS_DATABASE_URL` for handlers.

## Troubleshooting

| Symptom | Likely cause | Action |
|---------|--------------|--------|
| Jobs stuck in `RUNNING` | Worker crash | Wait for lease expiry or restart recovery |
| High retries | Downstream outage | Check webhook/email provider health |
| DLQ growth | Permanent failures or exhausted retries | Inspect `dead_letter_jobs.last_error` |
| No jobs processed | Handler DB not configured | Set integration/notification DB URLs |
