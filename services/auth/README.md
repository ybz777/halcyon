# Auth Service

Identity and access management for Halcyon.

## Responsibilities (SDD)

- User authentication (email/password, OAuth)
- JWT access and refresh tokens
- API key management
- Session lifecycle
- MFA (future)

## Run Locally

```bash
# From repository root
make up
make run-auth
```

## Configuration

| Variable | Default | Description |
|---|---|---|
| `AUTH_PORT` | `8080` | HTTP listen port |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |
| `SERVICE_NAME` | `auth` | Service identifier in logs and health |
| `DATABASE_URL` | _(empty)_ | PostgreSQL URL (wired in PR #4) |

## Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/health` | Liveness probe |
| `GET` | `/api/v1/ready` | Readiness probe |
| `GET` | `/metrics` | Prometheus metrics |

See [docs/services/auth.md](../../docs/services/auth.md) for architecture details.

## Architecture

```
cmd/auth/            Composition root
internal/
  domain/            Aggregates, value objects (PR #2+)
  application/       Use cases and ports
  infrastructure/    PostgreSQL, Redis adapters (PR #4+)
  api/http/          HTTP handlers
  config/            Environment configuration
```

## Tests

```bash
go test ./... -count=1
```

## Docker

```bash
make docker-build-auth
docker run --rm -p 8080:8080 -e AUTH_PORT=8080 halcyon/auth:local
```
