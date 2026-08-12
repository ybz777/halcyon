# ADR-0004: Observability Baseline per Service

## Status

Accepted

## Context

Halcyon is an operations intelligence platform. Our own services must be
observable from the first deploy. Waiting until “later” produces blind spots
and rework.

## Decision

Every service exposes from the earliest practical PR:

| Endpoint | Purpose |
|---|---|
| `GET /api/v1/health` | Liveness — process is running |
| `GET /api/v1/ready` | Readiness — dependency checks with explicit status |
| `GET /metrics` | Prometheus metrics (unversioned per convention) |

All services emit **structured JSON logs** via `log/slog`.

Readiness before database integration returns `200` with explicit
`database: skipped` rather than `503`, preventing false rollout failures.

## Alternatives

| Alternative | Why rejected |
|---|---|
| Health only until prod | No metrics during development; harder to debug |
| Readiness 503 without DB | Incorrect failure signal in Kubernetes |
| Logs as plain text | Poor queryability in Loki/CloudWatch |
| OpenTelemetry in PR #1 | Valuable but adds scope; metrics + slog first |

## Consequences

### Positive

- Consistent operational interface across all services
- Recruiters see production patterns immediately
- Kubernetes probes can be wired without API changes later

### Negative

- Slight boilerplate per service
- `/metrics` unauthenticated in PR #1—must restrict by network policy in prod
