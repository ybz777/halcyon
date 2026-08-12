# Jobs Service

Durable background job infrastructure for asynchronous platform work.

## Architecture

```
Application / Dispatcher
        |
        v
   Job Producer (pkg/platform/jobs)
        |
        v
 PostgreSQL Job Store (halcyon_jobs)
        |
        v
   Worker Pool (job-worker)
        |
        v
 Handler Registry
        |
   +----+----+----+
   |    |    |    |
success retry FAILED -> DLQ
```

Kafka remains the platform event bus. Jobs handle durable execution with retries.

## Job Types

Registered types:

- `webhook.delivery`
- `email.delivery`
- `notification.dispatch`
- `workflow.execution`
- `audit.persistence`

Unknown types fail permanently at execution time.

## Lifecycle

```
PENDING -> RUNNING -> SUCCEEDED
                   -> RETRY_WAIT -> (available_at) -> RUNNING
                   -> FAILED (permanent)
                   -> DEAD_LETTERED (DLQ)
                   -> CANCELLED
```

Expired `RUNNING` leases are reclaimed to `RETRY_WAIT` or `DEAD_LETTERED` when max attempts exceeded.

## Leases

- `JOB_LEASE_DURATION` controls claim lease expiry stored in `locked_at`
- `JOB_HEARTBEAT_INTERVAL` renews leases for long-running handlers
- Stale workers cannot complete jobs after lease loss

## Retries

Exponential backoff with jitter via `pkg/platform/jobs/backoff.go`:

- Default initial delay: 1s
- Multiplier: 2
- Max backoff: 5m

Handlers return `HandlerError` with `Retryable` flag to control policy.

## Idempotency

Optional `idempotency_key` per `(organization_id, job_type)` prevents duplicate scheduling.

## DLQ

Jobs exceeding `max_attempts` move to `DEAD_LETTERED` and a row is written to `dead_letter_jobs`.
Permanent non-retryable failures move to `FAILED` without DLQ.

## Worker Pool

Configuration:

| Variable | Description |
|----------|-------------|
| `JOB_WORKER_COUNT` | Logical workers (poll loops) |
| `JOB_MAX_CONCURRENT` | Max in-flight handler goroutines |
| `JOB_POLL_INTERVAL` | Idle poll delay |
| `JOB_SHUTDOWN_TIMEOUT` | Graceful shutdown wait |
| `JOB_LEASE_DURATION` | Claim lease duration |
| `JOB_HEARTBEAT_INTERVAL` | Lease renewal interval |

## Concurrency

Global concurrency is bounded by `JOB_MAX_CONCURRENT`. Per-job-type registration allows dedicated handlers without starving the pool.

## Security

- Admin APIs require `jobs.read` / `jobs.cancel` or platform equivalents
- Tenant scope derived from gateway headers, never from `?organization_id=`
- Payloads are summarized in APIs; secrets are redacted

## Operations

Admin APIs (via gateway):

- `GET /api/v1/jobs`
- `GET /api/v1/jobs/{id}`
- `GET /api/v1/jobs/{id}/attempts`
- `POST /api/v1/jobs/{id}/cancel`

## Retention

| Variable | Default |
|----------|---------|
| `JOB_SUCCESS_RETENTION_DAYS` | 7 |
| `JOB_FAILURE_RETENTION_DAYS` | 30 |
| `JOB_ATTEMPT_RETENTION_DAYS` | 30 |

## Migration from legacy workers

Webhook and email dispatchers enqueue jobs when `JOBS_DATABASE_URL` is set. The unified `job-worker` replaces standalone `webhook-worker` and `email-worker` in Docker Compose. Legacy workers remain available under the `legacy-workers` compose profile.

## Debugging

1. Check queue depth: `jobs_queue_depth` metric
2. Inspect job status via admin API
3. Review `job_attempts` for execution history
4. Check `dead_letter_jobs` for permanent failures
