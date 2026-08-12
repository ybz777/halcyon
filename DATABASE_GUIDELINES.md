# Database Guidelines

## Why This Document Exists

Halcyon runs 13 microservices, each owning its data. Poor database practices
cause cross-tenant leaks, migration failures, and production outages. This document
defines how we design schemas, write queries, run migrations, and operate databases
at scale.

## Who Reads This

| Audience | Purpose |
|---|---|
| Service engineers | Schema design and query patterns |
| Reviewers | Catch N+1 queries, missing indexes, tenant isolation gaps |
| DBAs / SRE | Migration windows, backup, and recovery procedures |
| Security | Tenant isolation and RLS requirements |

## Standards for Every PR

- Every table with tenant data includes `org_id` and is indexed.
- Migrations are forward-only, idempotent where possible, and reviewed.
- No cross-service database access — ever.
- Repository interfaces live in `application/ports/`; SQL lives in `infrastructure/`.
- PRs that add queries without `EXPLAIN` evidence for hot paths may be rejected.
- Breaking schema changes require ADR and coordinated rollout plan.

---

## Database-per-Service

Each service owns its PostgreSQL database (or schema in early development).

| Service | Primary Store | Notes |
|---|---|---|
| auth | PostgreSQL | Users, sessions, API keys |
| user | PostgreSQL | Profiles, preferences |
| organization | PostgreSQL | Orgs, RBAC, memberships |
| workflow | PostgreSQL | Incidents, state machines |
| ingestion | PostgreSQL + S3 | Metadata in PG, payloads in S3 |
| correlation | PostgreSQL + Redis | Windows in Redis, matches in PG |
| search | Elasticsearch | Read model — not source of truth |
| analytics | PostgreSQL / Cassandra | Aggregates at scale |
| audit | PostgreSQL + S3 | Hot in PG, cold archive in S3 |

**Rule:** Service A never queries Service B's database. Use API or events.

---

## Schema Design

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Tables | `snake_case`, plural | `incidents`, `api_keys` |
| Columns | `snake_case` | `created_at`, `org_id` |
| Primary keys | `id` (prefixed string) | `inc_abc123` |
| Foreign keys | `<entity>_id` | `org_id`, `user_id` |
| Indexes | `idx_<table>_<columns>` | `idx_incidents_org_status` |
| Constraints | `chk_<table>_<rule>` | `chk_incidents_status` |

### Required Columns (Tenant Tables)

```sql
CREATE TABLE incidents (
    id          TEXT PRIMARY KEY,
    org_id      TEXT NOT NULL,
    -- business columns ...
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ          -- soft delete when applicable
);

CREATE INDEX idx_incidents_org_status ON incidents (org_id, status)
    WHERE deleted_at IS NULL;
```

### ID Generation

- Use `pkg/platform/ids` — never auto-increment integers for public IDs
- Prefixes: `org_`, `usr_`, `inc_`, `evt_`, `sig_`, `key_`
- UUIDv7 or ULID for sortable, globally unique IDs

### Soft Deletes

- Use `deleted_at` for user-facing resources
- Hard delete for ephemeral data (sessions, correlation windows)
- Queries must filter `WHERE deleted_at IS NULL` by default

---

## Multi-Tenancy

### Tenant Isolation (Required)

Every query on tenant data **must** include `org_id`:

```sql
-- Correct
SELECT * FROM incidents WHERE org_id = $1 AND id = $2;

-- Forbidden — cross-tenant risk
SELECT * FROM incidents WHERE id = $1;
```

### Row-Level Security (Production)

```sql
ALTER TABLE incidents ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON incidents
    USING (org_id = current_setting('app.org_id'));
```

Set `app.org_id` at connection time from JWT claims. Defense in depth — not a
substitute for application-level filtering.

---

## Migrations

### Location

```
services/<service>/migrations/
  001_create_users.up.sql
  001_create_users.down.sql   -- optional; prefer forward-only
```

Managed by `tools/migrate` or golang-migrate.

### Rules

| Rule | Rationale |
|---|---|
| Forward-only in production | Rollback = forward migration, not down |
| One concern per migration | Easier review and rollback |
| No `DROP TABLE` without ADR | Data loss is irreversible |
| Add columns as nullable first | Zero-downtime deploys |
| Backfill in separate migration | Avoid long locks |
| Test on copy of production data | Catch lock and duration issues |

### Zero-Downtime Pattern

```
Migration 1: ADD COLUMN new_col TEXT NULL
Deploy:      Write to both old and new columns
Migration 2: Backfill new_col from old_col
Migration 3: ADD NOT NULL constraint
Migration 4: Drop old_col
```

### Migration Review Checklist

- [ ] Includes `org_id` on tenant tables
- [ ] Indexes for expected query patterns
- [ ] No table locks on large tables during peak hours
- [ ] Estimated duration documented for migrations > 1s
- [ ] Rollback plan documented in PR description

---

## Query Patterns

### Repository Pattern

```go
// application/ports/incident_repository.go
type IncidentRepository interface {
    FindByID(ctx context.Context, orgID, id string) (*domain.Incident, error)
    Save(ctx context.Context, incident *domain.Incident) error
}

// infrastructure/postgres/incident_repository.go
// SQL implementation only here
```

### Connection Pooling

- Use PgBouncer in production (transaction pooling mode)
- Max connections per service pod: 10–25 (tune per service)
- Set `statement_timeout` and `idle_in_transaction_session_timeout`

### N+1 Prevention

- Use JOINs or batch `WHERE id IN (...)` — never loop queries
- For lists: keyset pagination, not OFFSET on large tables

### Pagination

```sql
-- Keyset (preferred)
SELECT * FROM incidents
WHERE org_id = $1 AND created_at < $2
ORDER BY created_at DESC
LIMIT 25;

-- OFFSET only for small tables or admin tools
```

---

## Redis Guidelines

| Use Case | Pattern | TTL |
|---|---|---|
| Session cache | `session:{session_id}` | 24h |
| Rate limiting | Sliding window counter | 1 min |
| Correlation windows | Sorted sets per org | 15 min |
| Distributed locks | Redlock (sparingly) | 30s max |

**Never** store source-of-truth data only in Redis.

---

## Elasticsearch Guidelines

- Shared indices with `org_id` filter — **not** per-org indices at scale
- Index naming: `halcyon-incidents-v1`
- Reindex strategy documented before mapping changes
- Vector fields for RAG: `dense_vector` with `org_id` filter

See [PERFORMANCE_GUIDELINES.md](PERFORMANCE_GUIDELINES.md) for query budgets.

---

## Backup and Recovery

| Store | Backup | RPO | RTO |
|---|---|---|---|
| PostgreSQL (RDS) | Automated daily + PITR | 5 min | 1 hour |
| Redis | AOF + snapshot | 1 min | 15 min |
| Elasticsearch | Snapshot to S3 | 1 hour | 2 hours |
| S3 | Versioning + cross-region | 0 | 4 hours |

Restore procedures documented in `docs/runbooks/`.

---

## Observability

- Log slow queries (> 100ms) at `warn` level
- Metric: `halcyon_<service>_db_query_duration_seconds`
- Alert on connection pool exhaustion
- Never log query parameters containing PII

---

## Related Documents

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [EVENT_GUIDELINES.md](EVENT_GUIDELINES.md)
- [SECURITY.md](SECURITY.md)
- [PERFORMANCE_GUIDELINES.md](PERFORMANCE_GUIDELINES.md)
