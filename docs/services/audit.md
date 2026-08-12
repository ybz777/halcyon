# Audit Service

The audit service provides durable, tenant-scoped, append-only audit logging for Halcyon.

## Architecture

```
Application → Domain Event → Outbox → Kafka → audit-log-consumer → halcyon_audit → Audit API
```

Security events from the gateway are published directly to `halcyon.audit`.

## Data model

Audit records are immutable rows in `audit_events` with actor/target/result metadata. Records are never updated or deleted through application APIs.

## Permissions

| Permission | Description |
|---|---|
| `audit.read` | View organization audit events |
| `audit.read.security` | View security-sensitive events (login failures, invalid tokens, rate limits) |
| `audit.read.platform` | Cross-organization audit queries (platform administrators) |

## Tenant isolation

All queries are scoped to an organization from trusted headers (`X-Halcyon-Organization-ID`). Query parameter `organization_id` cannot override tenant context unless the caller has `audit.read.platform`.

## API

- `GET /api/v1/audit-events` — cursor pagination (`cursor`, `page_size`, filters)
- `GET /api/v1/audit-events/{audit_event_id}`

## Consumer

- Group: `audit-log-consumer`
- Topics: `halcyon.auth`, `halcyon.organizations`, `halcyon.workflows`, `halcyon.audit`
- Idempotency: `processed_events` with `(event_id, consumer_group)` in the same transaction as insert

## Retention

Configure `AUDIT_RETENTION_DAYS` (default 365). The audit consumer runs a daily job to delete records older than the retention window.

## DLQ

Malformed or permanently failing events are routed to topic DLQs via the platform messaging pipeline (`halcyon.*.dlq`).

## Observability

Prometheus metrics: `audit_events_consumed_total`, `audit_events_persisted_total`, `audit_events_failed_total`, `audit_events_duplicate_total`, `audit_persistence_duration_seconds`, `audit_query_duration_seconds`, `audit_query_errors_total`.

Recommended alerts:
- Consumer lag increasing for `audit-log-consumer`
- `audit_events_failed_total` rate > 0 sustained
- `audit_query_duration_seconds` p95 > 2s

## Scaling

The `audit_events` table is indexed for tenant-scoped queries. Future monthly partitioning and cold storage archival are supported by the `occurred_at` column design.

## Security

Never stored: passwords, tokens, API key secrets, authorization headers, cookies, or request bodies. Metadata is sanitized before persistence.
