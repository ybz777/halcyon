# Gateway Service

The gateway is Halcyon's controlled external entry point. It handles cross-cutting delivery concerns and proxies requests to internal services.

## Architecture

```
External client
      |
      v
API Gateway (8088)
      |
      +-- JWT validation (JWKS)
      +-- Request / correlation IDs
      +-- Redis rate limiting
      +-- Security headers
      +-- Circuit breaker
      |
      v
Internal services (auth, organization)
```

The gateway performs **coarse authentication**. Destination services enforce fine-grained authorization (RBAC).

## Routing

| External prefix | Upstream |
|---|---|
| `/api/v1/auth/*` | `AUTH_SERVICE_URL` |
| `/api/v1/organizations/*` | `ORG_SERVICE_URL` |
| `/api/v1/invitations/*` | `ORG_SERVICE_URL` |

Routing destinations come from trusted configuration only. Arbitrary upstream URLs are rejected (SSRF protection).

## Authentication

### User JWT

- Validated via JWKS fetched from auth service (`/api/v1/auth/jwks`)
- HS256 with cached symmetric keys (oct JWK)
- Validates issuer, expiration, algorithm (rejects algorithm confusion)
- Audience validated when `JWT_AUDIENCE` is configured

### API Keys

- Detected by `hal_live_` / `hal_test_` prefix
- Passed through to downstream services for validation
- Rate limited separately

### Public routes

- `/api/v1/health`, `/api/v1/ready`, `/metrics`
- `/api/v1/auth/login`, `/api/v1/auth/refresh`, `/api/v1/auth/jwks`

## Service-to-Service Authentication

The gateway signs short-lived RS256 service tokens (`X-Halcyon-Service-Token`) when proxying to internal services.

Claims: `sid`, `iss`, `sub`, `aud`, `exp`, `kid`

Downstream services must validate service tokens cryptographically; never trust `X-Internal-Service: true` alone.

## Tenant Propagation

The gateway strips client-supplied `X-Organization-ID` headers to prevent tenant spoofing.

Validated user identity is propagated via:

- `X-Halcyon-User-ID`
- `X-Halcyon-User-Email`
- `Authorization` (original bearer token)

Organization authorization remains enforced by the destination service.

## Rate Limiting

Redis-backed fixed-window rate limiting with bounded keys (SHA-256 hashed).

| Route class | Default limit |
|---|---|
| Auth (`login`, `refresh`) | 20/min |
| Invitation accept | 30/min |
| General authenticated | 300/min |

When `RATE_LIMIT_FAIL_CLOSED=true` (default), Redis outages deny requests rather than allowing unlimited traffic.

## Observability

Prometheus metrics at `/metrics`:

- `gateway_requests_total`
- `gateway_request_duration_seconds`
- `gateway_rate_limited_total`
- `gateway_authentication_failures_total`
- `gateway_circuit_open_total`
- `gateway_jwks_refresh_total`

Labels are bounded: `method`, `route`, `status`, `service`, `reason`.

## Local Development

```bash
make run-gateway   # Port 8088
make test-gateway
```

Requires auth and organization services running, Redis for rate limiting, and `SERVICE_PRIVATE_KEY` set.

## Security Invariants

- No unsigned JWTs
- No algorithm confusion
- No client-controlled upstream URLs
- No tenant header spoofing
- No secrets in logs
- Request body size limited (`MAX_REQUEST_BODY_BYTES`, default 1 MiB)
- TLS verification enabled by default (`UPSTREAM_TLS_INSECURE` for local HTTP only)
