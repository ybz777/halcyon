# Performance and Scalability

This document records **local/test-environment** measurements and evidence-based optimizations from PR #30. Numbers here are **not** production guarantees.

## Test environment

| Item | Value |
|------|-------|
| OS | Windows 11 (10.0.26100) |
| CPU | Intel Core i7-13700HX (representative local dev machine) |
| Go | 1.25.x (via `go.work` toolchain) |
| Database | PostgreSQL when integration tests run; unit benchmarks use in-memory stores |
| Redis | localhost:6379 when integration tests run |
| Date captured | 2026-07-30 |

Re-run benchmarks:

```bash
make bench-all
```

## Baseline measurements (local)

Run `make bench-all` and compare against the table below. Values vary by machine load.

| Benchmark | ns/op (local) | B/op | allocs/op | Package |
|-----------|---------------|------|-----------|---------|
| `BenchmarkEncodeTimeIDCursor` | 392 | 320 | 5 | `pkg/platform/pagination` |
| `BenchmarkDecodeTimeIDCursor` | 1106 | 344 | 7 | `pkg/platform/pagination` |
| `BenchmarkParsePageSize` | 10.6 | 0 | 0 | `pkg/platform/pagination` |
| `BenchmarkFingerprint` | 1756 | 1545 | 30 | `pkg/platform/idempotency` |
| `BenchmarkStorageKey` | 633 | 288 | 7 | `pkg/platform/idempotency` |
| `BenchmarkHS256ValidateBearer` | 4381 | 3312 | 56 | `pkg/platform/auth` |
| `BenchmarkStaticAuthorizer_Can` | 3.6 | 0 | 0 | `pkg/platform/auth` |
| `BenchmarkBackoffDelay` | 1.8 | 0 | 0 | `pkg/platform/jobs` |
| `BenchmarkPolicyLookup` | 13.3 | 0 | 0 | `pkg/platform/jobs` |
| `BenchmarkAllow_Memory` | 64.5 | 0 | 0 | `pkg/platform/ratelimit` |

Race detection (`go test -race`) is skipped on Windows without CGO; run on Linux CI.

## Profiling findings

### CPU

- JWT HS256 validation dominates authenticated request CPU in micro-benchmarks; acceptable for current scale.
- Cursor encode/decode and idempotency fingerprinting are sub-microsecond to low-microsecond — not hot paths relative to network I/O.
- Job backoff calculation is negligible.

### Memory

- Pagination cursor encoding allocates small fixed buffers (5 allocs/op) — acceptable.
- Idempotency middleware buffers request bodies up to configured `REQUEST_MAX_BODY_BYTES` (default 1 MiB).
- No unbounded in-memory job batches; worker `BatchSize` capped at 500 by config validation.

### Goroutines

- Workers use bounded concurrency (`JOB_MAX_CONCURRENT`, `MAX_CONCURRENT_JOBS`).
- HTTP servers use standard `net/http` per-connection goroutines with configured timeouts.

## Optimizations applied (PR #30)

### Database

**Finding:** Job claim query orders by `CASE priority WHEN 'HIGH' THEN 3 ...` but `idx_jobs_claim` used text `priority DESC`, which does not match semantic ordering.

**Change:** Migration `000003_claim_priority_index` adds `idx_jobs_claim_priority` aligned with the claim `ORDER BY`.

| Query | Before | After | Improvement | Environment |
|-------|--------|-------|-------------|-------------|
| Job claim (`SELECT ... ORDER BY priority CASE`) | Sequential scan or suboptimal index | Index scan on expression index | Planner can use matching index (verify with `EXPLAIN ANALYZE` in staging) | Local migration review |

### Redis / idempotency

**Finding:** `RedisStore.Claim` performed GET before SETNX on every new key (2 round trips).

**Change:** SETNX-first fast path; GET only on lost race or existing key.

| Operation | Before | After | Improvement | Environment |
|-----------|--------|-------|-------------|-------------|
| New idempotency claim | 2 Redis commands | 1 Redis command | ~50% fewer round trips on first claim | Code path analysis |

Completed records use shared `encodeRecord` for consistent serialization.

### HTTP server

**Finding:** Several services set only `ReadHeaderTimeout`, leaving read/write/idle timeouts at Go defaults.

**Change:** `pkg/platform/httputil` provides `DefaultServerTimeouts()` (read 15s, write 30s, idle 60s, read-header 5s). Applied to integrations, jobs, audit, and notifications.

### Connection pools

**Finding:** `workflow-worker` used `pgxpool.New` without pool limits.

**Change:** Uses `postgres.NewPool` with env-configurable max/min connections and lifetimes.

### Rate limiting

**Finding:** `MemoryLimiter.AllowAll` did not roll back partial increments on failure (diverged from Redis Lua semantics). `RecoverBodyLimitMiddleware` did not compose body limiting with panic recovery.

**Change:** Atomic batch with rollback in memory limiter; middleware composes `BodyLimitMiddleware` + recovery.

## Cache behavior

- Rate limiting: fixed-window counters in Redis with TTL on first increment; keys expire with window.
- Idempotency: completed responses stored with `IDEMPOTENCY_COMPLETED_TTL` (default 24h); in-progress with `IDEMPOTENCY_IN_PROGRESS_TTL` (default 5m).
- No application-level read-through cache on authorization paths (intentional — invalidation safety).

## Worker throughput

- Job workers claim in batches (`JOB_BATCH_SIZE`, default 10).
- Workflow worker: `MAX_CONCURRENT_JOBS` (default 4), `WORKER_CLAIM_BATCH_SIZE` (default 5).
- Retry backoff uses exponential delay with jitter (`JOB_RETRY_JITTER`) to avoid synchronized retry storms.

## Load-test methodology

Local smoke load test (requires running auth or organization health endpoint):

```powershell
.\scripts\perf\load-smoke.ps1 -Url http://localhost:8080/health -Concurrency 20 -Requests 500
```

```bash
./scripts/perf/load-smoke.sh http://localhost:8080/health 20 500
```

Measures: total duration, requests/sec, status code distribution. **Not** a production capacity test.

### Scenarios covered locally

| Scenario | Method |
|----------|--------|
| Normal traffic | Sequential health checks in unit/integration tests |
| Concurrent requests | `load-smoke` script |
| Idempotency races | `pkg/platform/idempotency` unit + integration tests |
| Rate limit bursts | `pkg/platform/ratelimit` middleware tests |
| Pagination stability | Keyset cursor benchmarks + org list integration tests |

## Resource limits

| Resource | Limit | Configuration | Reason |
|----------|-------|---------------|--------|
| HTTP request body | 1 MiB (default) | `REQUEST_MAX_BODY_BYTES` | Prevent unbounded memory per request |
| Idempotency stored response | 64 KiB (default) | `IDEMPOTENCY_MAX_RESPONSE_SIZE` | Bound Redis value size |
| Pagination page size | 200 max (service-specific) | Query param + `ParsePageSize` | Prevent huge result sets |
| Job batch size | 1–500 | `JOB_BATCH_SIZE` | Cap claim fan-out |
| Job max concurrent | ≥1 | `JOB_MAX_CONCURRENT` | Bound worker goroutines |
| DB pool (jobs example) | service config | `DATABASE_MAX_CONNS` | Avoid connection exhaustion |
| Workflow DB pool | 10 default max | `WORKFLOW_DATABASE_MAX_CONNS` | Worker + API share DB |
| HTTP read timeout | 15s | `HTTP_READ_TIMEOUT` / `httputil.DefaultServerTimeouts` | Slowloris / hung clients |
| HTTP write timeout | 30s | `HTTP_WRITE_TIMEOUT` | Long responses bounded |
| HTTP idle timeout | 60s | `HTTP_IDLE_TIMEOUT` | Connection reuse without stale sockets |
| Rate limit window | per-policy | `RATE_LIMIT_*_WINDOW` | Bounded Redis key TTL |
| Outbox relay batch | 100 default | `OUTBOX_BATCH_SIZE` | Kafka publish batching |

## Known bottlenecks (remaining)

1. **JWT validation** — per-request HS256 verify; consider caching validated token claims only if safe (not implemented).
2. **Full table scans on uncommon filters** — only indexed paths for high-frequency list/claim queries.
3. **Kafka consumer throughput** — not benchmarked locally without Redpanda stack running.
4. **Cross-service `internal/` import in jobs worker** — blocks worker build; see PR #31.
5. **Offset-free pagination** — requires stable sort keys; documented in pagination package.

## Security impact

All optimizations preserve:

- Authentication and JWT validation semantics
- Authorization checks (no unsafe caching added)
- Tenant isolation in queries
- Rate limiting fail-closed defaults
- Idempotency fingerprint and conflict detection
- Request body size limits

## Regression guards

Broad benchmark thresholds in `pkg/platform/perf/regression_test.go` catch order-of-magnitude regressions without brittle nanosecond assertions.

## Staff engineer review summary

The platform is structured for measurable scaling: keyset pagination, Redis-backed rate limits and idempotency, bounded worker pools, and indexed job claims. The highest real-world latency will remain network + PostgreSQL + Redis round trips. PR #30 focused on eliminating an extra Redis round trip on idempotency claims, aligning job claim indexes with query semantics, standardizing HTTP timeouts, and fixing rate-limit correctness bugs that could cause false rejections or partial state.

## Recommended follow-up (PR #32 — final production audit)

Full production audit: security review, end-to-end validation in staging, load testing at representative scale, and sign-off checklist before production traffic.
