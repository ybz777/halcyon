# Halcyon Architecture (Implemented)

Concise reference for the **currently implemented** Halcyon backend. For governance rules and long-term vision see [ARCHITECTURE.md](../ARCHITECTURE.md).

## Components

| Layer | Component | Role |
|-------|-----------|------|
| Edge | `services/gateway` | TLS termination, JWT/API-key auth, rate limits, reverse proxy, service-token signing |
| Identity | `services/auth` | Users, sessions, JWT issuance, JWKS |
| Tenancy | `services/organization` | Organizations, memberships, invitations, RBAC |
| Workflow | `services/workflow` | Workflow definitions and execution state |
| Audit | `services/audit` | Immutable audit event store and query API |
| Integrations | `services/integrations` | Webhook endpoints and delivery |
| Notifications | `services/notifications` | Email/notification dispatch |
| Jobs | `services/jobs` | Background job queue API and worker coordination |
| Platform | `pkg/platform` | Shared config, auth, idempotency, jobs, events, observability |
| Relay | `cmd/outbox-relay` | Transactional outbox → Kafka publisher |

**Workers:** workflow-worker, job-worker, audit-consumer, webhook-dispatcher, notification-dispatcher, organization-consumer.

## Request flow

```
Client → Gateway (auth, rate limit, strip client org headers)
      → signs X-Halcyon-Service-Token
      → re-sets X-Halcyon-Organization-ID from authenticated principal
      → upstream service (gateway-trust middleware, JWT validation, RBAC)
      → PostgreSQL / Redis
```

Direct service access in **development** (`APP_ENV != production`) may send JWT + org header without gateway attestation. In **production**, organization headers require a valid gateway service token.

## Authentication

- **Users:** HS256 JWT from auth service; validated via shared secret or JWKS at gateway.
- **API keys:** Validated at gateway; org and scopes forwarded as internal headers.
- **Service-to-service:** Short-lived signed service tokens (`X-Halcyon-Service-Token`) from gateway private key; verified by downstream `GatewayTrust` using `SERVICE_TOKEN_PUBLIC_KEY`.

## Authorization

- Organization-scoped RBAC enforced in each service's application layer.
- Permissions carried in JWT claims or API-key scopes.
- Destructive and admin operations require explicit permission checks.

## Database

- One PostgreSQL database per bounded context (auth, organization, workflow, audit, integrations, notifications, jobs).
- Migrations per service under `internal/infrastructure/postgres/migrations` plus shared platform migrations where applicable.
- Tenant isolation: all queries scoped by `organization_id`; no cross-service DB access.

## Redis

- Rate limiting (gateway, auth).
- Idempotency keys for mutating APIs.
- Failure policy: rate limits fail closed in production when configured; idempotency degrades to per-process memory only when Redis unavailable in dev (production requires Redis).

## Events

- Transactional outbox pattern in producing services.
- `outbox-relay` publishes to Kafka (Redpanda locally).
- Consumers: audit-consumer, organization-consumer, dispatchers.
- JSON envelope with base64-encoded protobuf payloads; DLQ on repeated failure.

## Background jobs

- Postgres-backed job queue in `pkg/platform/jobs`.
- Workers claim jobs with leases; retries with backoff; retention policies for success/failure rows.
- Job handlers registered per domain (integrations webhooks, notification email).

## Deployment

- **Local:** Docker Compose (`infra/docker/compose.yml`) — see [deployment.md](deployment.md).
- **CI:** GitHub Actions — fmt, vet, lint, test, race (Linux), build, integration tests.
- **Production:** Container images per service; migrations run as explicit pre-deploy job.

## Observability

- Structured JSON logs to stdout.
- Prometheus metrics on `/metrics`.
- Request IDs (`X-Request-ID`) propagated gateway → upstream.
- Health (`/api/v1/health`) and readiness (`/api/v1/ready`) on all HTTP services.

## Key documentation

| Doc | Purpose |
|-----|---------|
| [deployment.md](deployment.md) | Build, deploy, migrate, rollback |
| [operations.md](operations.md) | Incident runbook |
| [production-readiness.md](production-readiness.md) | Release checklist |
| [configuration.md](configuration.md) | Environment variables |
| [testing.md](testing.md) | Test strategy |
| [performance.md](performance.md) | Benchmarks and limits |
