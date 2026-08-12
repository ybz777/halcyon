# Rate Limiting and Abuse Protection

Halcyon enforces distributed rate limits, request body size caps, and concurrency guards through `pkg/platform/ratelimit`. Business services consume a small `Limiter` interface; Redis details stay in infrastructure.

## Architecture

```
HTTP middleware (gateway/auth)
        │
        ▼
  Limiter.Allow(ctx, identity, policy)
        │
        ▼
 Redis (Lua atomic fixed-window)  |  DisabledLimiter (dev/tests)
```

- **Gateway**: multi-dimension checks (IP + user/org/api-key) per route class.
- **Auth service**: login and API-key authentication failure paths use fail-closed policies.
- **Disabled mode**: `RATE_LIMIT_ENABLED=false` — no Redis required; all requests pass.

## Algorithm

**Fixed-window counter** with atomic Redis Lua script:

1. `INCR` key
2. On first increment, `PEXPIRE` with window duration
3. Reject when count > limit
4. Return remaining quota and key TTL for `Retry-After`

Multi-key checks use a second Lua script that rolls back increments when any dimension exceeds its limit.

Keys always expire; no permanent counters.

Clock strategy: TTL is set server-side in Redis on first increment; `Retry-After` derives from `PTTL`.

## Policies

| Policy | Use case | Default (env) | Fail mode |
|--------|----------|---------------|-----------|
| `auth` | Login, refresh, password reset | `RATE_LIMIT_AUTH_REQUESTS` / `RATE_LIMIT_AUTH_WINDOW` | closed |
| `password_reset` | Password reset flows | half of auth limit | closed |
| `public` | Unauthenticated traffic | `RATE_LIMIT_DEFAULT_REQUESTS` | configurable |
| `api` | Authenticated API traffic | `RATE_LIMIT_DEFAULT_REQUESTS` | configurable |
| `workflow` | Executions, jobs | `RATE_LIMIT_WORKFLOW_REQUESTS` | configurable |
| `webhook` | Webhook registration/delivery | default window | configurable |

Health (`/api/v1/health`, `/api/v1/ready`) is excluded from gateway rate limits.

## Identity Model

Stable, bounded identity strings (never secrets):

- `ip:<normalized-ip>`
- `user:<user-id>`
- `org:<org-id>`
- `apikey:<key-id>` (prefix for auth failures; key ID at gateway)

Redis keys: `<prefix>:<policy>:<sha256(identity)[:16]>`

## Proxy Trust

`X-Forwarded-For` / `X-Real-IP` are honored only when `RATE_LIMIT_TRUST_PROXY=true`. Otherwise `RemoteAddr` is used. IPv4-mapped IPv6 addresses normalize to IPv4.

## Failure Modes

| Scenario | Auth endpoints | General API |
|----------|----------------|-------------|
| Redis unavailable | Fail **closed** (policy.FailClosed) | `RATE_LIMIT_FAILURE_MODE` (`open` or `closed`) |
| Fail-open | Structured warning log + `rate_limit_redis_errors_total` | Same |

Silent bypass is never allowed when fail-open is active.

## HTTP Behavior

- **429** `application/problem+json` with type `https://halcyon.io/problems/rate-limited`
- **Retry-After** header (seconds, minimum 1)
- Optional `X-RateLimit-Remaining`
- **413** for oversized bodies (`https://halcyon.io/problems/payload-too-large`)

## Request Body Limits

`REQUEST_MAX_BODY_BYTES` (alias `MAX_REQUEST_BODY_BYTES`) enforced via `http.MaxBytesReader` without buffering entire payloads.

## Concurrency Protection

`ConcurrencyLimiter` provides Redis-backed semaphores for expensive operations (`WORKFLOW_CONCURRENCY_LIMIT`).

## Metrics (low cardinality)

- `rate_limit_requests_total{service,route,method,policy,result}`
- `rate_limit_rejected_total{service,route,method,policy}`
- `rate_limit_redis_errors_total{service}`
- `request_body_rejected_total{service}`
- `concurrency_limit_rejected_total{service}`

No user, org, IP, or API-key labels.

## Configuration

```bash
RATE_LIMIT_ENABLED=false
RATE_LIMIT_DEFAULT_REQUESTS=300
RATE_LIMIT_DEFAULT_WINDOW=1m
RATE_LIMIT_AUTH_REQUESTS=20
RATE_LIMIT_AUTH_WINDOW=1m
RATE_LIMIT_WORKFLOW_REQUESTS=30
RATE_LIMIT_WORKFLOW_WINDOW=1m
RATE_LIMIT_REDIS_PREFIX=gateway:ratelimit
RATE_LIMIT_FAILURE_MODE=closed   # or open
RATE_LIMIT_TRUST_PROXY=false
REQUEST_MAX_BODY_BYTES=1048576
WORKFLOW_CONCURRENCY_LIMIT=8
REDIS_URL=redis://localhost:6379/0
```

## Operational Tuning

- Lower auth limits first under attack; keep health exclusions.
- Use fail-closed for edge gateways; fail-open only when Redis outage must not block read-heavy internal APIs.
- Keep windows short (1m) for brute-force protection; longer windows for tenant quotas.
- Monitor `rate_limit_redis_errors_total` and rejection rates by `policy` label.

## Security Considerations

- Never put API key secrets or passwords in Redis keys or logs.
- Normalize routes in metrics to avoid cardinality explosion.
- Audit new routes for appropriate policy class (`auth`, `workflow`, `api`).
- Rate limiting is abuse protection, not billing quota.

## Local Verification

```bash
make test-ratelimit
make integration-ratelimit   # requires Docker
```
