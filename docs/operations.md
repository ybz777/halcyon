# Operations Runbook

Operational guide for Halcyon services. For deployment steps see [deployment.md](deployment.md).

## Service inventory

| Service | Port | Health | Ready | Metrics |
|---------|------|--------|-------|---------|
| auth | 8080 | ✓ | ✓ (DB) | ✓ |
| organization | 8081 | ✓ | ✓ | ✓ |
| workflow | 8082 | ✓ | ✓ | ✓ |
| audit | 8083 | ✓ | ✓ | ✓ |
| integrations | 8084 | ✓ | ✓ (DB) | ✓ |
| notifications | 8085 | ✓ | ✓ (DB) | ✓ |
| jobs | 8086 | ✓ | ✓ | ✓ |
| gateway | 8088 | ✓ | ✓ (upstream) | ✓ |

**Workers (no HTTP):** workflow-worker, job-worker, audit-consumer, webhook-dispatcher, notification-dispatcher, outbox-relay

## Startup (local)

```bash
make up
# Wait for healthchecks (compose uses httpprobe + dependency conditions)
bash scripts/deploy/smoke-deploy.sh
```

## Logs

All services log structured JSON to **stdout/stderr**. Log rotation is the platform responsibility (Kubernetes, Docker logging driver, etc.).

Useful log events:

- `security.rate_limited` — rate limit rejections
- `job succeeded` / `job scheduled for retry` — worker activity
- `event_dlq` — Kafka consumer dead letters
- `event_publish_failure` — outbox relay issues

**Never log:** passwords, tokens, API keys, raw JWTs, idempotency keys as metric labels.

## Metrics (post-deployment checklist)

Inspect after each deploy:

| Signal | Where |
|--------|-------|
| HTTP error rate | `http_requests_total` per service |
| Latency p95 | HTTP histogram metrics |
| Rate limit rejections | `rate_limit_*` |
| Job queue depth | jobs service metrics |
| DB pool wait | pgx pool stats (if exposed) |
| Kafka lag | consumer group lag (external) |
| Redis errors | rate limit / idempotency metrics |

## Common failures

### Service `/ready` returns 503

- **Cause:** database unreachable or pool exhausted
- **Check:** `DATABASE_URL`, network, Postgres health, connection limits
- **Action:** restore DB connectivity; scale pool only after measurement

### Gateway unhealthy

- **Cause:** upstream auth/organization not ready, Redis down, missing `SERVICE_PRIVATE_KEY`
- **Check:** upstream `/ready`, Redis ping, gateway logs

### job-worker not processing

- **Cause:** missing `INTEGRATIONS_DATABASE_URL` / `NOTIFICATIONS_DATABASE_URL`, handler registration failed at startup
- **Check:** worker logs at startup; jobs table for `PENDING` rows

### outbox-relay not publishing

- **Cause:** Kafka unavailable, migration not applied, stuck `FAILED` rows
- **Check:** Redpanda health, `pkg/platform/outbox` table, relay logs

### Migration failed on deploy

- **Action:** stop rollout; do not route traffic; fix migration forward; see [deployment.md](deployment.md)

## Graceful shutdown

Services handle `SIGTERM`:

1. Stop accepting new HTTP connections (`server.Shutdown`)
2. Workers finish or abort in-flight jobs per lease/timeout policy
3. Flush telemetry (`OTEL_SHUTDOWN_TIMEOUT`)
4. Close DB/Redis pools

Compose/Kubernetes should allow ≥30s termination grace.

## Incident debugging

1. Confirm scope (single service vs platform)
2. Check `/api/v1/ready` on affected services
3. Review recent deploy / migration
4. Inspect metrics and structured logs
5. For auth issues: verify `JWT_SECRET` consistency across services
6. For tenant issues: verify organization ID in JWT and query filters

## Smoke test

```bash
bash scripts/deploy/smoke-deploy.sh
# Optional load smoke (non-destructive):
./scripts/perf/load-smoke.sh http://localhost:8088/api/v1/health 10 100
```

## Security operations

- Rotate `JWT_SECRET` with planned session invalidation
- Rotate `INTEGRATIONS_ENCRYPTION_KEY` requires re-encrypting stored webhook secrets
- `SERVICE_PRIVATE_KEY` rotation: update gateway + all `SERVICE_TOKEN_PUBLIC_KEY` validators
- Run `make security-vuln` in CI; review Dependabot PRs

## Autoscaling

Not configured in this repository. If using Kubernetes HPA:

- Scale HTTP services on CPU/latency
- Scale workers on queue depth / consumer lag
- Ensure DB pool limits × replicas do not exceed Postgres `max_connections`
