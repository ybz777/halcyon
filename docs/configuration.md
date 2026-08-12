# Configuration

Halcyon uses environment variables for configuration. All parsing is centralized in `pkg/platform/config`.

## Loading

```go
import platformconfig "github.com/halcyon-io/halcyon/pkg/platform/config"

env := platformconfig.FromEnv()
cfg, err := platformconfig.LoadAuthService()
```

Service-specific loaders remain in `services/<name>/internal/config` but delegate to the platform package.

**Do not call `os.Getenv` outside `pkg/platform/config`.**

## Environment

| Variable | Purpose | Default |
|----------|---------|---------|
| `APP_ENV` | `development`, `test`, `production` | `development` |
| `SERVICE_NAME` | Service identifier | per-service |
| `SERVICE_VERSION` | Release version | `dev` |
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `LOG_FORMAT` | `json`, `text` | `json` |

## Core subsystems

### HTTP

| Variable | Type | Required | Default |
|----------|------|----------|---------|
| `AUTH_PORT` / `GATEWAY_PORT` / etc. | int | optional | service-specific |
| `HTTP_READ_TIMEOUT` | duration | optional | `15s` |
| `HTTP_WRITE_TIMEOUT` | duration | optional | `15s` |
| `HTTP_SHUTDOWN_TIMEOUT` | duration | optional | `30s` |
| `REQUEST_MAX_BODY_BYTES` | int | optional | `1048576` |

**Deprecated:** `MAX_REQUEST_BODY_BYTES` → use `REQUEST_MAX_BODY_BYTES`

### Database

| Variable | Type | Secret | Notes |
|----------|------|--------|-------|
| `DATABASE_URL` | URL | **yes** | Auth service |
| `ORGANIZATION_DATABASE_URL` | URL | **yes** | Organization service |
| `WORKFLOW_DATABASE_URL` | URL | **yes** | Workflow service |
| `DATABASE_MAX_CONNS` | int | no | default `25` |
| `DATABASE_MIN_CONNS` | int | no | default `2` |
| `DATABASE_MIGRATE` | bool | no | default `true` |

### Redis

| Variable | Type | Secret |
|----------|------|--------|
| `REDIS_URL` | URL | **yes** (if password embedded) |

Required when `RATE_LIMIT_ENABLED=true` or idempotency is enabled.

### Kafka

| Variable | Type | Required |
|----------|------|----------|
| `KAFKA_BROKERS` | CSV | when using consumers |
| `KAFKA_CLIENT_ID` | string | optional |
| `EVENT_DLQ_ENABLED` | bool | optional |
| `EVENT_MAX_RETRIES` | int | optional |

### Authentication (secret)

| Variable | Type | Required in prod |
|----------|------|------------------|
| `JWT_SECRET` | secret | **yes** (min 32 bytes) |
| `JWT_ACCESS_TTL` | duration | optional |
| `JWT_REFRESH_TTL` | duration | optional |
| `SERVICE_PRIVATE_KEY` | secret | gateway |

### Rate limiting (PR #23)

| Variable | Default |
|----------|---------|
| `RATE_LIMIT_ENABLED` | `false` (auth), `true` (gateway) |
| `RATE_LIMIT_DEFAULT_REQUESTS` | `300` |
| `RATE_LIMIT_DEFAULT_WINDOW` | `1m` |
| `RATE_LIMIT_FAILURE_MODE` | `closed` |

**Deprecated:** `RATE_LIMIT_DEFAULT`, `RATE_LIMIT_AUTH`, `RATE_LIMIT_WINDOW`

### Idempotency (PR #25)

| Variable | Default |
|----------|---------|
| `IDEMPOTENCY_ENABLED` | `true` |
| `IDEMPOTENCY_IN_PROGRESS_TTL` | `5m` |
| `IDEMPOTENCY_COMPLETED_TTL` | `24h` |
| `IDEMPOTENCY_MAX_RESPONSE_SIZE` | `65536` |
| `IDEMPOTENCY_FAIL_CLOSED` | `true` |

### Jobs (PR #26)

| Variable | Default |
|----------|---------|
| `JOB_WORKER_COUNT` | `1` |
| `JOB_MAX_CONCURRENT` | `8` |
| `JOB_LEASE_DURATION` | `30s` |
| `JOB_LEASE_RENEW_INTERVAL` | `10s` |
| `JOB_MAX_ATTEMPTS` | `6` |
| `JOB_EXECUTION_TIMEOUT` | `5m` |

### Observability

| Variable | Default |
|----------|---------|
| `OTEL_ENABLED` | `false` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | required when enabled |
| `OTEL_TRACES_SAMPLER` | `parentbased_traceidratio` |

## Secret handling

Secrets are loaded via `env.Secret("KEY")`. Never log `.Value()` directly.

Safe logging:

```go
log.Info("starting", slog.String("config", cfg.Summary().String()))
```

URLs with credentials are redacted: `postgres://***:***@host/db`

## Production validation

When `APP_ENV=production`:

- `JWT_SECRET` must be at least 32 bytes when database is configured
- `OTEL_EXPORTER_OTLP_ENDPOINT` required when tracing enabled
- `REDIS_URL` required when rate limiting enabled

## Local development

Copy `.env.example` to `.env`. Docker Compose injects variables from the host environment.

```bash
cp .env.example .env
make up
```

## Testing

```go
env := platformconfig.ForEnv(t, map[string]string{
    "JWT_SECRET": strings.Repeat("x", 32),
})
```

Run config tests: `make test-config`

## Secret management

Production secrets must come from the deployment environment or secret manager. Never commit `.env` files with real credentials.
