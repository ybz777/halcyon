# Deployment Guide

This document describes how Halcyon is built, validated, deployed, and rolled back. It reflects the **current repository** (Docker Compose for local/staging; Helm references in `RELEASE_PROCESS.md` are aspirational until charts land).

## Pipeline overview

```
commit → CI (fmt, vet, lint, test, race, build, integration, docker)
      → artifact (container image tagged with git SHA)
      → migrate (explicit job — not automatic on app startup in production)
      → deploy
      → health/readiness verification
      → traffic
```

## Prerequisites

- Go **1.25.x** (see `go.work` / module `go` directives)
- Docker with Compose v2
- `make`
- For production: secret manager / env injection for `JWT_SECRET`, `SERVICE_PRIVATE_KEY`, database URLs, `INTEGRATIONS_ENCRYPTION_KEY`

## Local / staging (Docker Compose)

```bash
cp .env.example .env
# Set SERVICE_PRIVATE_KEY (required for gateway) — generate with: openssl genrsa 2048
make up
```

Stack: PostgreSQL, Redis, Redpanda, all HTTP services, workers, outbox relay.

Verify:

```bash
bash scripts/deploy/smoke-deploy.sh
```

Or manually:

```bash
curl -sf http://localhost:8080/api/v1/health
curl -sf http://localhost:8080/api/v1/ready
curl -sf http://localhost:8088/api/v1/health
```

## Container images

| Image | Dockerfile | Default port |
|-------|------------|--------------|
| auth | `services/auth/Dockerfile` | 8080 |
| organization | `services/organization/Dockerfile` | 8081 |
| workflow | `services/workflow/Dockerfile` | 8082 |
| audit | `services/audit/Dockerfile` | 8083 |
| integrations | `services/integrations/Dockerfile` | 8084 |
| notifications | `services/notifications/Dockerfile` | 8085 |
| jobs | `services/jobs/Dockerfile` | 8086 |
| gateway | `services/gateway/Dockerfile` | 8088 |
| outbox-relay | `cmd/outbox-relay/Dockerfile` | — |

**Build all images locally:**

```bash
make docker-build-all
```

**Properties:**

- Multi-stage build (`golang:1.25-alpine` → `distroless/static-debian12:nonroot`)
- `CGO_ENABLED=0`, stripped binaries
- Non-root runtime user
- `httpprobe` binary for container `HEALTHCHECK` (no shell in distroless)
- No secrets in images or build args
- `.dockerignore` excludes tests, `.git`, credentials

## Image tagging

CI builds `halcyon/<service>:ci`. Production should use **immutable tags**:

- `halcyon/auth:<git-sha>`
- `halcyon/auth:v1.2.3` (release tag)

Never deploy only `:latest` to production.

## Database migrations

Migrations run **on startup in development** when `*_DATABASE_MIGRATE=true`. **Production:** run migrations as a separate deployment step before rolling out new app versions.

| Service | Env flag | Manual command |
|---------|----------|----------------|
| auth + platform outbox | `DATABASE_MIGRATE` | `make migrate` + `make migrate-platform` |
| organization | `ORGANIZATION_DATABASE_MIGRATE` | `make migrate-org` |
| workflow | `WORKFLOW_DATABASE_MIGRATE` | `make migrate-workflow` |
| audit | `AUDIT_DATABASE_MIGRATE` | `make migrate-audit` |
| integrations | `INTEGRATIONS_DATABASE_MIGRATE` | `make migrate-integrations` |
| notifications | `NOTIFICATIONS_DATABASE_MIGRATE` | `make migrate-notifications` |
| jobs | `JOBS_DATABASE_MIGRATE` | `make migrate-jobs` |

Validate migration files: `make validate-migrations`

### Migration failure

If migration fails:

1. **Do not** start new application instances against an incompatible schema.
2. Fix forward with a new migration (preferred) or roll back application to previous image.
3. Database rollback requires DBA review — down migrations exist for dev only.

### Deployment order (recommended)

1. Build and push images (tagged with SHA)
2. Run migration job against target database
3. Deploy workers/consumers compatible with schema
4. Deploy HTTP services (rolling)
5. Deploy gateway last (or with upstream health gates)
6. Run smoke tests
7. Monitor error rate, latency, worker queue depth

## Health and readiness

| Endpoint | Meaning |
|----------|---------|
| `GET /api/v1/health` | Process alive |
| `GET /api/v1/ready` | Dependencies OK (e.g. database ping) |
| `GET /metrics` | Prometheus metrics |

Workers (job-worker, workflow-worker, consumers) log structured status; no HTTP health endpoint. Monitor via logs and job/event metrics.

## Rollback

1. Identify previous image tag (git SHA or release version)
2. Redeploy previous artifacts for all affected services
3. Verify `/api/v1/health` and `/api/v1/ready`
4. Run `scripts/deploy/smoke-deploy.sh`
5. Database: only roll back schema if a down migration was tested; otherwise fix forward

## Configuration

See `.env.example` and `docs/configuration.md`. Required production secrets (never commit):

- `JWT_SECRET` (≥32 chars)
- `SERVICE_PRIVATE_KEY` / `SERVICE_TOKEN_PUBLIC_KEY`
- Database URLs with TLS in production
- `INTEGRATIONS_ENCRYPTION_KEY` (32 bytes)

Production must set `APP_ENV=production` — disables debug defaults via PR #27 config validation.

## CI/CD

GitHub Actions workflow: `.github/workflows/ci.yml`

Jobs: format/tidy/vet → lint → unit tests → race → OpenAPI → migrations → security (govulncheck, gitleaks) → build → integration (Postgres/Redis/Redpanda) → Docker matrix build (+ Trivy on `main`).

Reproduce locally: `make release-check` (subset) or `make ci`.

## TLS

TLS termination is expected at the **gateway / load balancer** in production. Services speak HTTP inside the cluster. Do not expose PostgreSQL or Redis publicly.

## Resource recommendations

Based on local measurements (`docs/performance.md`) — not production capacity guarantees:

| Component | Starting point |
|-----------|----------------|
| HTTP service | 256–512 MiB RAM, 0.25–0.5 CPU |
| job-worker | 512 MiB RAM, `JOB_MAX_CONCURRENT=8` |
| PostgreSQL pool | 10–25 conns per service |
| Redis | 128–256 MiB |

## Backup / DR assumptions

- **Application artifacts:** container registry retains tagged images
- **Database backups:** provided by deployment platform (RDS, managed Postgres, etc.) — not implemented in this repo
- **Secrets:** secret manager / vault — not in git
- **Kafka:** Redpanda/Kafka retention and replication per ops config

## Further reading

- [docs/operations.md](operations.md) — runbook
- [RELEASE_PROCESS.md](../RELEASE_PROCESS.md) — versioning
- [docs/onboarding/ci.md](onboarding/ci.md) — CI reproduction
- [docs/performance.md](performance.md) — benchmarks and limits
