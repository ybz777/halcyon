# Performance Guidelines

## Why This Document Exists

Halcyon targets 100M events/day across 10,000 organizations. Performance is not
an optimization pass — it is a design constraint. This document defines latency
budgets, profiling requirements, and anti-patterns that block merge.

## Who Reads This

| Audience | Purpose |
|---|---|
| Service engineers | Design within latency budgets |
| Reviewers | Reject performance regressions |
| SRE | Capacity planning and alerting |
| Load test authors | Define acceptance criteria |

## Standards for Every PR

- New API endpoints must document expected latency budget in PR description.
- Hot paths require benchmarks (`go test -bench`) or load test updates.
- No unbounded queries, unbounded fan-out, or unbounded goroutines.
- No `org_id` or `user_id` as Prometheus metric labels.
- PRs that add synchronous cross-service calls in request path require ADR.

---

## Latency Budgets

End-to-end user-facing request: **< 500ms p99** (excluding AI inference).

### Per-Layer Budget (Typical Read)

| Layer | Budget |
|---|---|
| Gateway (auth, routing) | 10ms |
| Service handler | 5ms |
| Application logic | 10ms |
| Database query | 50ms |
| Cache (Redis) | 5ms |
| Elasticsearch | 100ms |
| Serialization | 5ms |
| **Total (no AI)** | **~185ms** |

### AI Inference

| Operation | Budget |
|---|---|
| Brief generation (streaming start) | < 2s to first token |
| Full brief | < 15s |
| Embedding (single doc) | < 500ms |
| Semantic search | < 300ms |

AI calls are **async by default**. Synchronous AI only for explicit user-initiated
queries with loading UX.

---

## Throughput Targets

| Path | Target |
|---|---|
| API reads | 1,000 RPS per service (sustained) |
| API writes | 200 RPS per service |
| Webhook ingestion | 10,000 events/min per org burst |
| Kafka consumption | < 10,000 lag under normal load |
| Search queries | 500 RPS |

Load tests in `tests/load/` validate these before major releases.

---

## Caching Strategy

| Data | Cache | TTL | Invalidation |
|---|---|---|---|
| User session | Redis | 24h | On logout / password change |
| Org settings | Redis | 5 min | Event: `organizations.settings.updated` |
| Incident list (dashboard) | Redis | 30s | Event: `incidents.*` |
| Search results | None (ES is the cache) | — | — |
| AI briefs | Redis | 1h | On new evidence linked |

### Cache Rules

- Cache keys include `org_id`: `incidents:list:{org_id}:{cursor_hash}`
- Never cache across tenants
- Cache-aside pattern — DB is source of truth
- Document cache invalidation in PR when adding new cached data

---

## Database Performance

- All tenant queries use composite indexes starting with `org_id`
- `EXPLAIN ANALYZE` required in PR for queries on tables > 100K rows
- No `SELECT *` in production paths
- Batch inserts for bulk operations (outbox relay, analytics)
- Connection pool sizing documented per service

See [DATABASE_GUIDELINES.md](DATABASE_GUIDELINES.md).

---

## Kafka Performance

| Setting | Value | Rationale |
|---|---|---|
| Partition key | `org_id` | Per-tenant ordering |
| Compression | `lz4` | CPU vs bandwidth tradeoff |
| Batch size | 16KB default | Producer throughput |
| Consumer concurrency | 1 per partition max | Ordering guarantee |

### Hot Partition Mitigation

- Monitor partition lag per key
- Large tenants may get dedicated partition strategy (ADR required)
- Alert when single partition lag > 2x average

---

## HTTP and API

- Pagination required for all list endpoints (default limit: 25, max: 100)
- Response compression (gzip) at gateway
- ETags for cacheable reads where applicable
- No synchronous fan-out to > 3 services in one request — use aggregation service or GraphQL

---

## Frontend Performance (When `apps/` Exists)

| Metric | Target |
|---|---|
| LCP | < 2.5s |
| FID | < 100ms |
| CLS | < 0.1 |
| Bundle size (initial) | < 200KB gzipped |

---

## Profiling and Benchmarking

### When Required

- New hot path code (request handlers, Kafka consumers)
- Algorithm changes (correlation, search ranking)
- Before every major release

### Tools

| Tool | Use |
|---|---|
| `go test -bench` | Micro-benchmarks |
| `pprof` | CPU and memory profiling |
| k6 | Load testing (`tests/load/`) |
| Grafana dashboards | Production latency tracking |

### PR Requirement

For performance-sensitive changes, include in PR description:

```
## Performance
- Benchmark: BenchmarkIncidentCreate (before: 1.2ms, after: 0.8ms)
- Load test: not required (cold path)
```

---

## Anti-Patterns (Merge Blockers)

| Anti-Pattern | Why |
|---|---|
| N+1 queries | Linear DB growth per request |
| Unbounded `LIMIT` or no pagination | Memory exhaustion |
| Synchronous Kafka publish in request path | Latency coupling |
| Global mutex on hot path | Contention at scale |
| Per-tenant Prometheus labels | Cardinality explosion |
| `time.Sleep` in production retry | Blocks goroutines |
| Loading full incident history for brief | Unbounded payload |

---

## Capacity Planning

Review quarterly and before major launches:

- RDS instance sizing and connection counts
- Kafka partition count and broker sizing
- Elasticsearch cluster nodes and shard count
- EKS node pool autoscaling thresholds

Document capacity decisions in ADRs when changing infrastructure tiers.

---

## Related Documents

- [OBSERVABILITY.md](OBSERVABILITY.md)
- [DATABASE_GUIDELINES.md](DATABASE_GUIDELINES.md)
- [TESTING.md](TESTING.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
