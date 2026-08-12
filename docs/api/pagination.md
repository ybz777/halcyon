# Collection API Pagination

Halcyon collection endpoints use **cursor-based (keyset) pagination** for stable, efficient traversal of large datasets.

## Response shape

All paginated list endpoints return:

```json
{
  "items": [],
  "next_cursor": "opaque-token",
  "has_more": true
}
```

| Field | Description |
|-------|-------------|
| `items` | Current page of resources (never `null`; empty arrays when no results) |
| `next_cursor` | Opaque token for the next page; omitted when `has_more` is `false` |
| `has_more` | `true` when additional pages exist |

## Query parameters

| Parameter | Required | Default | Max | Description |
|-----------|----------|---------|-----|-------------|
| `page_size` | No | 50 | 200 | Number of items per page |
| `limit` | No | — | 200 | Alias for `page_size` (accepted where documented) |
| `cursor` | No | — | — | Opaque cursor from a previous response |

### Limit validation

| Input | Behavior |
|-------|----------|
| Omitted | Default page size (50) |
| Valid `1..max` | Accepted |
| `0`, negative, non-numeric | HTTP 400 |
| Above max (200) | HTTP 400 |

Invalid limits are **rejected**, not silently clamped.

## Cursor model

Cursors are:

- **Opaque** — base64-encoded JSON; clients must not parse or construct them
- **Stateless** — no server-side cursor store
- **Versioned** — include a version field for forward compatibility
- **Validated** — malformed or tampered cursors return HTTP 400

Invalid cursors are **never** silently reset to page one.

### Ordering

Every paginated query uses deterministic ordering with a unique tie-breaker:

```
ORDER BY <sort_field> <direction>, <id_field> <direction>
```

Example: `created_at DESC, id DESC`

This prevents duplicates and gaps when traversing pages.

### Consistency model

Pagination provides **best-effort sequential traversal**, not snapshot isolation:

- **Concurrent inserts** — new rows may appear on later pages; already-seen pages are unaffected
- **Concurrent deletes** — rows may disappear between pages; cursors remain valid but may skip removed rows

## Client usage

```http
GET /api/v1/notifications?page_size=50
```

```json
{
  "items": [ ... ],
  "next_cursor": "eyJ2IjoxLCJ0Ijoi...",
  "has_more": true
}
```

```http
GET /api/v1/notifications?page_size=50&cursor=eyJ2IjoxLCJ0Ijoi...
```

Continue until `has_more` is `false` and `next_cursor` is absent.

## Filtering and sorting

Where supported, filters and sorts use **explicit allowlists**. Arbitrary field names are not accepted.

### Date filters

Use RFC 3339 timestamps (e.g. `2026-07-30T12:00:00Z`). Ambiguous local timestamps are rejected.

### Sort direction

`asc` or `desc` (case-insensitive). Invalid values return HTTP 400.

## Errors

Invalid query parameters return **HTTP 400** with RFC 7807 Problem Details:

| Error | Typical cause |
|-------|---------------|
| Invalid cursor | Malformed, tampered, or unsupported cursor |
| Invalid page size | Zero, negative, non-numeric, or above max |
| Invalid sort | Unknown sort field or direction |
| Invalid filter | Unknown filter field, bad enum, or malformed date |

## Authorization

Pagination does not bypass tenant isolation. Organization-scoped lists always filter by organization ID in the database query, before pagination is applied.

## Endpoints

| Service | Endpoint | Default sort |
|---------|----------|--------------|
| Audit | `GET /api/v1/audit-events` | `occurred_at DESC, id DESC` |
| Auth | `GET /api/v1/auth/api-keys` | `created_at DESC, id DESC` |
| Jobs | `GET /api/v1/jobs` | `created_at DESC, id DESC` |
| Jobs | `GET /api/v1/jobs/{id}/attempts` | `attempt_number DESC, id DESC` |
| Notifications | `GET /api/v1/notifications` | `created_at DESC, id DESC` |
| Notifications | `GET /api/v1/notification-preferences` | `notification_type ASC, channel ASC` |
| Integrations | `GET /api/v1/webhook-endpoints` | `created_at DESC, id DESC` |
| Integrations | `GET /api/v1/webhook-endpoints/{id}/deliveries` | `created_at DESC, id DESC` |
| Organization | `GET /api/v1/organizations/{id}/members` | `joined_at DESC, id DESC` |
| Organization | `GET /api/v1/organizations/{id}/invitations` | `created_at DESC, id DESC` |
| Workflow | `GET /api/v1/incidents` | configurable (allowlisted) |
| Workflow | `GET /api/v1/workflows` | `created_at DESC, id DESC` |
