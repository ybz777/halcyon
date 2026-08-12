# API Guidelines

## Why This Document Exists

Halcyon is API-first. Consistent API design across 13 services reduces client
bugs, simplifies gateway routing, and demonstrates Stripe-level API craft in
the portfolio.

## Who Reads This

| Audience | Purpose |
|---|---|
| API implementers | Build endpoints correctly |
| Frontend engineers | Know what to expect |
| External integrators | Integration documentation |
| Reviewers | Enforce consistency |

## Standards for Every PR

- New endpoints must be defined in `contracts/openapi/` **before** implementation.
- Handlers must not contain business logic.
- All public endpoints use `/api/v1/` prefix.
- Errors return RFC 7807 Problem Details.
- PRs without OpenAPI spec updates for new/changed endpoints are rejected.

---

## API Surfaces

| Surface | Use | Location |
|---|---|---|
| REST | Commands, CRUD, webhooks | `contracts/openapi/` |
| GraphQL | Complex read aggregation | `contracts/graphql/` (future) |
| gRPC | Internal service-to-service | `contracts/proto/` (future) |
| WebSocket | Real-time push | Documented in OpenAPI + WS protocol doc |

---

## REST Conventions

### URL Structure

```
https://api.halcyon.io/api/v1/<resource>
https://api.halcyon.io/api/v1/<resource>/{id}
https://api.halcyon.io/api/v1/<resource>/{id}/<sub-resource>
```

### Versioning

- Path-based: `/api/v1/`, `/api/v2/`
- Breaking changes require new version
- Deprecation: `Deprecation: true` + `Sunset: <RFC 7231 date>` headers
- Minimum 6-month deprecation window

### HTTP Methods

| Method | Use | Idempotent |
|---|---|---|
| `GET` | Read | Yes |
| `POST` | Create, commands, transitions | No* |
| `PUT` | Full replace | Yes |
| `PATCH` | Partial update | No |
| `DELETE` | Remove | Yes |

*Use `X-Idempotency-Key` header on POST that create resources.

### Status Codes

| Code | Use |
|---|---|
| `200` | Success with body |
| `201` | Resource created |
| `202` | Accepted (async processing) |
| `204` | Success, no body |
| `400` | Malformed request |
| `401` | Unauthenticated |
| `403` | Unauthorized (authenticated but not permitted) |
| `404` | Resource not found |
| `409` | Conflict (duplicate, invalid state transition) |
| `422` | Validation failure |
| `429` | Rate limited |
| `500` | Internal server error |

### Pagination

Cursor-based for real-time data:

```
GET /api/v1/incidents?limit=25&cursor=eyJ...

Response:
{
  "data": [...],
  "pagination": {
    "next_cursor": "eyJ...",
    "has_more": true
  }
}
```

### Filtering

Query parameters: `?status=open&severity=sev1&service=payment`

### Sorting

`?sort=created_at&order=desc`

---

## Error Format (RFC 7807)

```json
{
  "type": "https://api.halcyon.io/errors/incident-not-found",
  "title": "Incident Not Found",
  "status": 404,
  "detail": "Incident inc_abc does not exist in organization org_xyz.",
  "instance": "/api/v1/incidents/inc_abc",
  "request_id": "req_123",
  "errors": [
    {
      "field": "incident_id",
      "code": "NOT_FOUND",
      "message": "No incident with this ID exists."
    }
  ]
}
```

Implemented via `pkg/platform/httperrors`.

---

## Authentication

| Method | Header | Use |
|---|---|---|
| JWT | `Authorization: Bearer <token>` | User sessions |
| API key | `Authorization: Bearer cp_live_...` | Programmatic access |
| Webhook | `X-Halcyon-Signature: sha256=...` | Inbound webhooks |

---

## ID Format

Prefix-based, URL-safe:

```
org_  usr_  inc_  evt_  sig_  ana_  key_  aud_
```

Generated via `pkg/platform/ids`.

---

## Rate Limiting

| Client | Limit |
|---|---|
| Authenticated user | 1,000 req/min per org |
| API key | 500 req/min per key |
| Webhook ingress | 10,000 req/min per org |
| AI queries | 60 req/min per org |

Response headers:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 987
X-RateLimit-Reset: 1722384000
```

---

## OpenAPI Workflow

1. Define or update spec in `contracts/openapi/<service>/v1/openapi.yaml`
2. Validate with Spectral in CI
3. Implement handler in `services/<service>/internal/api/http/`
4. Handler tests verify spec compliance

**Spec is the source of truth.** Implementation follows spec, not the reverse.

---

## GraphQL Guidelines (Future)

- Read-only queries in GraphQL; mutations stay in REST
- GraphQL for complex page loads (Incident Command Center)
- Schema in `contracts/graphql/`
- No GraphQL for state machine transitions

---

## WebSocket Guidelines

- Endpoint: `wss://ws.halcyon.io/ws/v1/<channel>`
- Auth: Bearer token in upgrade request
- JSON message protocol with `type` field
- Documented in `docs/api/websocket-protocol.md`

---

## Related Documents

- [EVENT_GUIDELINES.md](EVENT_GUIDELINES.md)
- [SECURITY.md](SECURITY.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- `contracts/openapi/README.md`
