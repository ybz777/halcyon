# Observability Guide

## Why This Document Exists

Halcyon sells operational intelligence. Our own services must be fully observable.
This document defines the logging, metrics, and tracing standards every service
must implement from the first PR.

## Who Reads This

| Audience | Purpose |
|---|---|
| Service engineers | Instrument new code correctly |
| SRE / on-call | Know what signals exist |
| Reviewers | Verify observability in PRs |

## Standards for Every PR

- Every new service exposes `/api/v1/health`, `/api/v1/ready`, `/metrics`.
- Every new HTTP handler emits request logs and metrics.
- Every new Kafka consumer reports lag metrics.
- No `fmt.Println` in production paths.
- No PII in logs, traces, or metric labels.
- PRs without observability for new code paths are rejected.

---

## Three Pillars

| Pillar | Tool | Collection |
|---|---|---|
| **Logs** | `log/slog` (JSON) | stdout → Loki / CloudWatch |
| **Metrics** | Prometheus | `/metrics` endpoint |
| **Traces** | OpenTelemetry | OTLP → Tempo |

All bootstrapped via `pkg/platform/observability`.

---

## Structured Logging

### Required Fields (Every Log Line)

| Field | Source | Example |
|---|---|---|
| `timestamp` | slog (automatic) | |
| `level` | slog | `info` |
| `service` | config | `auth` |
| `version` | build info | `v1.4.2` |
| `message` | caller | `request completed` |
| `trace_id` | OTel context | `abc123` |
| `span_id` | OTel context | `def456` |
| `request_id` | middleware | `req_xyz` |
| `org_id` | JWT (when available) | `org_abc` |

### Log Levels

| Level | Use |
|---|---|
| `debug` | Development only — disabled in production |
| `info` | Normal operations: request completed, event processed |
| `warn` | Recoverable issues: retry, degraded mode |
| `error` | Failures requiring attention: DB error, external API failure |

### Rules

- JSON format in production and staging
- Text format allowed in local development only
- Never log: passwords, tokens, API keys, full request bodies with PII
- Error logs include `error` field with wrapped message

---

## Metrics

### Endpoint

`GET /metrics` — Prometheus exposition format (unversioned).

Restrict access via Kubernetes NetworkPolicy in production.

### Naming Convention

```
halcyon_<service>_<metric>_<unit>
```

Examples:

```
halcyon_auth_http_requests_total{method, path, status}
halcyon_auth_http_request_duration_seconds{method, path}
halcyon_workflow_incidents_active
halcyon_ingestion_signals_received_total{source, type}
halcyon_kafka_consumer_lag{topic, group}
```

### Cardinality Rules

| Allowed labels | Forbidden labels |
|---|---|
| `method`, `path` (templated), `status` | `org_id` (unbounded) |
| `service`, `source`, `type` | `user_id` |
| `topic`, `consumer_group` | `incident_id` |

**At 10K orgs, per-tenant metric labels will crash Prometheus.** Use logs and
traces for per-tenant debugging. Use aggregated metrics for dashboards.

### Required Metrics (Every Service)

| Metric | Type |
|---|---|
| `http_requests_total` | Counter |
| `http_request_duration_seconds` | Histogram |
| `http_requests_in_flight` | Gauge |

### Required Metrics (Kafka Consumers)

| Metric | Type |
|---|---|
| `kafka_messages_consumed_total` | Counter |
| `kafka_consumer_lag` | Gauge |
| `kafka_processing_duration_seconds` | Histogram |

---

## Distributed Tracing

### Instrumentation

- OpenTelemetry SDK via `pkg/platform/observability`
- Auto-instrument HTTP (incoming and outgoing)
- Manual spans for business operations: `incident.create`, `ai.analyze`

### Context Propagation

| Transport | Header |
|---|---|
| HTTP | `traceparent` (W3C Trace Context) |
| Kafka | `traceparent` in message headers |

### Sampling

| Trace Type | Sample Rate |
|---|---|
| Errors (5xx) | 100% |
| Slow (> 2s) | 100% |
| AI inference | 100% |
| Normal | 10% |
| Health checks | 0% (dropped) |

---

## Health and Readiness

| Endpoint | Purpose | K8s Probe |
|---|---|---|
| `GET /api/v1/health` | Liveness | `livenessProbe` |
| `GET /api/v1/ready` | Readiness | `readinessProbe` |

### Readiness Check States

| State | Meaning |
|---|---|
| `ok` | Dependency healthy |
| `pending` | Configured but not yet verified |
| `skipped` | Not configured (pre-integration) |
| `failed` | Dependency unreachable — do not route traffic |

---

## Dashboards and Alerts

| Dashboard | Location |
|---|---|
| Platform overview | `infra/observability/grafana/` |
| Per-service | Same, tagged by service |

| Alert | Condition | Severity |
|---|---|---|
| High error rate | 5xx > 5% for 3 min | Critical |
| High latency | p99 > 2s for 5 min | Warning |
| Kafka consumer lag | > 10,000 | Critical |
| Pod crash loop | restarts > 0 in 5 min | Critical |

---

## Dogfooding

Halcyon monitors itself with the same patterns it helps customers understand.
This is a product feature, not an afterthought.

---

## Related Documents

- [ARCHITECTURE.md](ARCHITECTURE.md)
- [PERFORMANCE_GUIDELINES.md](PERFORMANCE_GUIDELINES.md)
- [docs/adr/0004-observability-baseline.md](docs/adr/0004-observability-baseline.md)
