# Idempotency Keys

Halcyon mutation endpoints support **idempotency keys** so clients can safely retry requests without creating duplicate side effects.

## Header

```http
Idempotency-Key: <client-generated-unique-key>
```

- Case-insensitive header name
- Maximum length: **255** characters
- Allowed characters: `A-Z`, `a-z`, `0-9`, `.`, `_`, `:`, `-`
- Invalid or missing keys (on required routes) return **HTTP 400** with RFC 7807 Problem Details

## When to use

Send an idempotency key when:

- Creating organizations, invitations, API keys, webhook endpoints, incidents, or workflows
- A network timeout or client retry could otherwise duplicate the operation

Do **not** send keys for `GET`, `HEAD`, or `OPTIONS` requests.

## Scope

A key is scoped to:

- Authenticated principal (user or API key)
- Organization (when applicable)
- HTTP method and route
- Request body fingerprint

User A cannot replay user B's result. The same key with a different payload returns **HTTP 409 Conflict**.

## Retry behavior

### First request

```http
POST /api/v1/organizations
Idempotency-Key: org-create-2026-07-30-001
Content-Type: application/json

{"name":"Acme"}
```

Response (example):

```http
HTTP/1.1 201 Created
Idempotency-Replayed: false
```

### Safe retry (same key + same body)

```http
POST /api/v1/organizations
Idempotency-Key: org-create-2026-07-30-001
```

Response:

```http
HTTP/1.1 201 Created
Idempotency-Replayed: true
```

The original status and body are replayed. No second organization is created.

### Conflict (same key + different body)

```http
POST /api/v1/organizations
Idempotency-Key: org-create-2026-07-30-001

{"name":"Other"}
```

```http
HTTP/1.1 409 Conflict
Content-Type: application/problem+json

{
  "type": "https://halcyon.io/problems/idempotency-conflict",
  "title": "Idempotency Conflict",
  "detail": "idempotency key was already used with a different request"
}
```

### In progress

If the original request is still processing:

```http
HTTP/1.1 409 Conflict
Content-Type: application/problem+json

{
  "type": "https://halcyon.io/problems/idempotency-in-progress",
  "title": "Idempotency In Progress",
  "detail": "a request with this idempotency key is already being processed"
}
```

## TTL and guarantees

| Setting | Default | Description |
|---------|---------|-------------|
| `IDEMPOTENCY_IN_PROGRESS_TTL` | `5m` | Lease for in-flight requests (crash recovery) |
| `IDEMPOTENCY_COMPLETED_TTL` | `24h` | How long completed responses are replayable |

Idempotency guarantees apply only within the completed TTL window.

## Cached responses

- **Cached:** 2xx success, deterministic 4xx business errors
- **Not cached:** 5xx transient failures (key is released for retry)
- **Never stored:** `Set-Cookie`, `Authorization`, bodies over `IDEMPOTENCY_MAX_RESPONSE_SIZE` (64 KiB default)

## Required routes (initial rollout)

| Service | Endpoint |
|---------|----------|
| Organization | `POST /api/v1/organizations` |
| Organization | `POST /api/v1/organizations/{id}/invitations` |
| Organization | `POST /api/v1/invitations/{token}/accept` |
| Auth | `POST /api/v1/api-keys` |
| Auth | `POST /api/v1/api-keys/{id}/rotate` |
| Integrations | `POST /api/v1/webhook-endpoints` |
| Workflow | `POST /api/v1/incidents` |
| Workflow | `POST /api/v1/workflows` |

`POST /api/v1/workflows/{id}/executions` uses domain-level Postgres idempotency (not HTTP response replay).

## Redis requirement

Multi-instance idempotency requires Redis (`REDIS_URL`). Required routes **fail closed** when Redis is unavailable and idempotency is enabled.

## Problem types

| Type | Status |
|------|--------|
| `idempotency-key-required` | 400 |
| `idempotency-key-invalid` | 400 |
| `idempotency-conflict` | 409 |
| `idempotency-in-progress` | 409 |
