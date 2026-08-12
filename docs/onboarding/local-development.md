# Local Development Guide

This guide gets you from zero to a running Halcyon auth service locally.

## 1. Clone and Configure

```bash
git clone <repository-url> halcyon
cd halcyon
cp .env.example .env
```

Edit `.env` if you need non-default ports.

## 2. Start Infrastructure

```bash
make up
```

This starts:

| Service | Port | Credentials |
|---|---|---|
| PostgreSQL 16 | 5432 | `halcyon` / `halcyon` / db `halcyon` |
| Redis 7 | 6379 | no password (local only) |
| Redpanda (Kafka API) | 9092 | no auth (local only) |

Verify:

```bash
docker compose -f infra/docker/compose.yml ps
```

## 3. Run Auth Service

```bash
make run-auth
```

Expected log output (JSON):

```json
{"level":"INFO","msg":"starting server","service":"auth","port":"8080"}
```

## 4. Smoke Test

```bash
curl -s http://localhost:8080/api/v1/health | jq .
curl -s http://localhost:8080/api/v1/ready | jq .
curl -s http://localhost:8080/metrics | head -5
```

### Expected Responses

**Health (liveness):**

```json
{"status":"ok","service":"auth"}
```

**Readiness:**

```json
{
  "status": "ready",
  "service": "auth",
  "checks": {
    "database": "skipped"
  }
}
```

With `DATABASE_URL` set in `.env`, the service applies migrations on startup
(when `DATABASE_MIGRATE=true`) and readiness reports `database: ok` when
PostgreSQL is reachable.

## 5. Run Migrations (optional)

Migrations run automatically on auth service startup. To apply them manually:

```bash
export DATABASE_URL=postgres://halcyon:halcyon@localhost:5432/halcyon?sslmode=disable
make migrate
```

## 6. Run Tests

```bash
make test
make test-race
make test-ratelimit
make integration-ratelimit   # Requires Docker
make test-integration   # Requires Docker services or CI infrastructure
make coverage
make ci        # Full CI suite (format, tidy, lint, test, race, build)
```

See [docs/onboarding/ci.md](../onboarding/ci.md) for CI details.

## 7. Optional: Distributed Tracing

```bash
docker compose -f infra/docker/compose.yml --profile observability up -d otel-collector
```

Set in `.env`:

```env
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
```

See [docs/observability/tracing.md](../observability/tracing.md).

## 8. Rate Limiting (optional)

Gateway rate limiting uses Redis when `RATE_LIMIT_ENABLED=true` (default in `.env.example`).
Auth service login/API-key throttling uses the same Redis instance when enabled for auth.

```bash
make test-ratelimit
make integration-ratelimit   # Docker required
```

See [docs/security/rate-limiting.md](../security/rate-limiting.md).

## 9. Idempotency (optional)

Mutation endpoints on organization, auth, integrations, and workflow services use Redis-backed idempotency when `IDEMPOTENCY_ENABLED=true`.

```bash
make test-idempotency
make integration-idempotency   # Docker / Redis required
```

Send `Idempotency-Key` on supported `POST` routes. See [docs/api/idempotency.md](../api/idempotency.md).

## 10. Configuration

All environment variables are loaded through `pkg/platform/config`. See [docs/configuration.md](../configuration.md).

```bash
make test-config
```

## 11. Background Jobs

The unified `job-worker` executes durable jobs from PostgreSQL (`halcyon_jobs`).

```bash
make run-worker
make test-jobs
make integration-jobs   # PostgreSQL required (JOBS_DATABASE_URL)
```

See [docs/infrastructure/jobs.md](../infrastructure/jobs.md).

## 12. Stop Infrastructure

```bash
make down
```

## Troubleshooting

| Issue | Fix |
|---|---|
| Port 5432 in use | Change `POSTGRES_PORT` in `.env` |
| Port 8080 in use | Set `AUTH_PORT=8081` in `.env` |
| Docker not running | Start Docker Desktop |
| `go: command not found` | Install Go 1.25+ (see `.go-version`) |

## Next Steps

- [Auth service architecture](../services/auth.md)
- [services/auth README](../../services/auth/README.md)
