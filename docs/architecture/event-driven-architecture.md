# Event-Driven Architecture

Halcyon uses a **transactional outbox** with **Kafka/Redpanda** for reliable, at-least-once domain event delivery.

## End-to-End Flow

```
HTTP Request
    → Application use case
    → PostgreSQL transaction (business state + outbox INSERT)
    → COMMIT
    → Outbox Relay (SKIP LOCKED claim)
    → Kafka publish (ack required)
    → Mark outbox published
    → Consumer group
    → Idempotency check (processed_events)
    → Business handler
    → Commit offset
```

**Never** publish directly to Kafka from HTTP handlers.

## Delivery Semantics

| Stage | Semantics |
|-------|-----------|
| Outbox → Kafka | At-least-once (relay retries with backoff) |
| Kafka → Consumer | At-least-once (offset commit after processing) |
| End-to-end | At-least-once; consumers **must** be idempotent |

Exactly-once is not claimed. The outbox bridges the database/Kafka boundary; duplicates are expected and safe.

## Event Envelope

Canonical JSON envelope in `pkg/platform/events/`:

- `event_id` — globally unique (`evt_*`), stable across retries
- `event_type` + `event_version` — separate fields (e.g. `organizations.organization.created` + `1.0`)
- `organization_id`, `actor_id`, `correlation_id`, `request_id`
- Typed `payload` registered in `DefaultRegistry()`

See [contracts/events/README.md](../../contracts/events/README.md).

## Topic Strategy

| Topic | Domain |
|-------|--------|
| `halcyon.auth` | Authentication events |
| `halcyon.organizations` | Organization/membership/invitation |
| `halcyon.incidents` | Workflow/incident events |

**Never** one topic per organization. DLQ topics: `{topic}.dlq`.

Initialize locally: `go run ./cmd/event-topics`

## Local Redpanda Development

Start Redpanda and PostgreSQL:

```bash
docker compose -f infra/docker/compose.yml up -d redpanda postgres
make init-event-topics
make run-relay
make run-organization-consumer
```

Inspect topics, consumer groups, offsets, and DLQ:

```bash
docker exec -it redpanda rpk topic list
docker exec -it redpanda rpk topic consume halcyon.organizations -n 1
docker exec -it redpanda rpk group list
docker exec -it redpanda rpk group describe organization-audit-consumer
docker exec -it redpanda rpk topic consume halcyon.organizations.dlq -n 5
```

## Partitioning

Default partition key priority:

1. `organization_id` — per-tenant ordering
2. `aggregate_id` — aggregate ordering
3. `event_id` — fallback

**Tradeoff:** `organization_id` ordering may create hot partitions for large tenants. Monitor partition lag; dedicated partitioning is a future scaling strategy.

## Outbox Relay

- Claims pending rows with `FOR UPDATE SKIP LOCKED`
- Publishes to Kafka only after broker acknowledgment
- Retries transient failures with exponential backoff (`next_attempt_at`)
- Routes exhausted messages to `{topic}.dlq`
- Configurable: `OUTBOX_BATCH_SIZE`, `OUTBOX_POLL_INTERVAL`

Run: `go run ./cmd/outbox-relay`

## Consumer Architecture

- Stable consumer group IDs (e.g. `organization-audit-consumer`)
- Bounded concurrency (`CONSUMER_CONCURRENCY`)
- Offset committed **after** successful processing
- Transactional idempotency via `IdempotentProcessor` (`processed_events` unique on `event_id + consumer_group`)

Run: `go run ./cmd/organization-consumer`

## Idempotency Strategy

```
BEGIN
  INSERT processed_events (event_id, consumer_group) ON CONFLICT → skip
  process business logic
  UPDATE status = completed
COMMIT
```

Same event may be processed by different consumer groups independently.

## Retry and DLQ

| Failure | Handling |
|---------|----------|
| Transient (DB timeout) | Bounded retry with jitter |
| Permanent (schema invalid) | Immediate DLQ |
| Business rejection | DLQ with `failure_category=business` |

DLQ messages preserve original event + metadata (topic, partition, offset, failure reason) for future replay.

## Schema Evolution

- v1 consumers must ignore unknown JSON fields
- Do not rename or change semantic meaning of existing fields
- Breaking changes → new `event_version`
- Optional fields may be added within a version

## Observability

| Metric | Labels |
|--------|--------|
| `events_published_total` | `event_type`, `topic` |
| `events_consumed_total` | `event_type`, `topic` |
| `events_duplicate_total` | `event_type`, `consumer_group` |
| `events_dlq_total` | `topic`, `event_type`, `failure_category` |
| `outbox_pending_events` | gauge |

Never use `organization_id`, `event_id`, or `correlation_id` as Prometheus labels.

Structured logs: `event_published`, `event_consumed`, `event_duplicate`, `event_retry`, `event_dlq` — never log secrets or full sensitive payloads.

## Scaling Path (100M events/day)

- Multiple gateway, relay, and consumer replicas
- Kafka partition parallelism (6 partitions per topic default)
- Idempotent consumers with inbox deduplication
- Hot-partition monitoring via consumer lag metrics
- Future: dedicated partitions for enterprise tenants

## Related

- [Event Catalog](../events/catalog.md)
- [event-driven.md](event-driven.md) (operational quick reference)
