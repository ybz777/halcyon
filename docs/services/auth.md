# Auth Service

## Purpose

The auth service is Halcyon's **identity and access** bounded context. It owns
authentication credentials, sessions, API keys, and token issuance for all
platform and domain services.

## Responsibilities

| In scope | Out of scope (other services) |
|---|---|
| User authentication (email/password, OAuth) | User profiles and preferences (`user`) |
| JWT access and refresh token lifecycle | Organization RBAC (`organization`) |
| API key issuance and validation | Edge routing (`gateway`) |
| Session management | Audit log persistence (`audit`) |
| MFA (TOTP) | |

## Dependencies

| Dependency | Type | Required | Notes |
|---|---|---|---|
| PostgreSQL | Data store | Yes | Users (PR #4); sessions, API keys later |
| Redis | Cache | PR #4+ | Session cache, rate limiting |
| Kafka | Events | Phase 2+ | `auth.user.registered`, etc. |

**PR #4:** With `DATABASE_URL` set, the service connects on startup, applies
migrations when `DATABASE_MIGRATE=true`, and readiness reports `database: ok`
or `database: failed`.

## Public APIs

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/health` | Liveness probe |
| `GET` | `/api/v1/ready` | Readiness probe with dependency checks |
| `GET` | `/metrics` | Prometheus metrics |

OpenAPI: [`contracts/openapi/auth/v1/openapi.yaml`](../../contracts/openapi/auth/v1/openapi.yaml)

### Performance Budgets

Initial targets for PR #1 endpoints (local, no DB):

| Endpoint | P50 | P95 | P99 |
|---|---|---|---|
| `GET /api/v1/health` | < 5 ms | < 20 ms | < 50 ms |
| `GET /api/v1/ready` | < 10 ms | < 30 ms | < 75 ms |
| `GET /metrics` | < 25 ms | < 75 ms | < 150 ms |

### Service-Level Targets

| Metric | Target |
|---|---|
| Idle RSS | < 100 MB |
| Container cold start | < 3 seconds |
| Graceful shutdown | < 15 seconds |

Budgets are validated in load tests starting PR #3 (CI). Adjust via RFC when
production data is available.

## Events Produced

| Event | Topic | Status |
|---|---|---|
| `auth.user.registered` | `halcyon.auth` | Phase 1 |
| `auth.user.logged_in` | `halcyon.auth` | Phase 1 |
| `auth.session.revoked` | `halcyon.auth` | Phase 1 |

See [`docs/events/catalog.md`](../events/catalog.md).

## Events Consumed

None in PR #1. Future: `organizations.membership.revoked` for session invalidation.

## Database Ownership

| Store | Schema owner | Status |
|---|---|---|
| PostgreSQL `auth` database | `services/auth` | PR #4 |

Tables: `users` (PR #4). Planned: `sessions`, `api_keys`, `refresh_tokens`, `outbox`.

### Persistence stack (PR #4)

| Component | Library |
|---|---|
| Driver / pool | pgx/v5 |
| Queries | sqlc (`internal/infrastructure/postgres/sqlcgen`) |
| Migrations | golang-migrate (`services/auth/migrations`) |

Repository ports live in `internal/application/ports`. PostgreSQL adapters
live in `internal/infrastructure/postgres`.

## Scaling Characteristics

| Dimension | Strategy |
|---|---|
| Read-heavy (token validation) | Horizontal pod autoscaling; Redis session cache |
| Write-heavy (login bursts) | Connection pooling via PgBouncer; rate limiting |
| Multi-tenancy | `org_id` on all tenant tables; partition keys in events |

Stateless HTTP handlers in PR #1 scale horizontally without coordination.

## Failure Modes

| Failure | Impact | Mitigation |
|---|---|---|
| PostgreSQL unavailable | Login/register fail; existing JWTs valid until expiry | Readiness `database: failed`; circuit breaker; cached JWKS |
| Redis unavailable | Session lookup slower (fallback to DB) | Degraded mode; alert on cache miss rate |
| Token signing key compromise | Forged tokens possible | Key rotation; short access token TTL (15 min) |
| High login brute force | Account lockout risk | Rate limiting at gateway + auth service |

**PR #1:** Process crash → K8s restarts via liveness probe. No data dependencies.

## Internal Architecture

```
cmd/auth/                    Composition root (DI wiring)
internal/
  domain/                    Aggregates, value objects, domain events
  application/               Use cases and ports
  infrastructure/            PostgreSQL, Redis, Kafka adapters
  api/http/                  HTTP handlers (no business logic)
  config/                    Environment configuration
```

## Future Work

| Item | PR / Phase | Status |
|---|---|---|
| User aggregate, Email VO, domain events | PR #2 | Done |
| GitHub Actions CI | PR #3 | Done |
| PostgreSQL migrations + repository | PR #4 | Done |
| Register/login use cases | PR #5–#6 | In progress |
| JWT authentication + RBAC foundation | PR #5 | Done |
| OAuth2 + PKCE | Phase 1 | Planned |
| MFA (TOTP) | Phase 1 | Planned |
| API keys | Phase 1 | Planned |

## Related Documents

- [ARCHITECTURE.md](../../ARCHITECTURE.md)
- [SECURITY.md](../../SECURITY.md)
- [API_GUIDELINES.md](../../API_GUIDELINES.md)
- [services/auth/README.md](../../services/auth/README.md)
