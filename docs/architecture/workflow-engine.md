# Workflow Engine Architecture

## Overview

```
API → Application Command → PostgreSQL (workflow + execution + job)
                                    ↓
                            Workflow Worker (SKIP LOCKED claim)
                                    ↓
                            Step Executors → state update + outbox event
```

PostgreSQL is the durable source of truth. Kafka delivers events via the transactional outbox (PR #15).

## Domain Model

| Entity | Purpose |
|--------|---------|
| `Workflow` | Identity + organization scope |
| `WorkflowDefinition` | Immutable versioned step graph |
| `WorkflowExecution` | One concrete run |
| `WorkflowStepExecution` | Per-step state |
| `Job` | Durable executable work unit |
| `ExecutionAttempt` | Attempt history |

## State Machines

**Execution:** `PENDING → RUNNING → SUCCEEDED | FAILED | CANCELLED`

**Step:** `PENDING → RUNNING → SUCCEEDED | FAILED | RETRY_WAIT → PENDING | CANCELLED`

**Job:** `READY → RUNNING → SUCCEEDED | FAILED | RETRY_WAIT → READY | CANCELLED`

Invalid transitions are rejected in domain and tested.

## Job Claiming

Workers claim jobs with:

```sql
UPDATE workflow_jobs ... WHERE id IN (
  SELECT id FROM workflow_jobs
  WHERE status IN ('READY','RETRY_WAIT') AND available_at <= now()
  FOR UPDATE SKIP LOCKED LIMIT N
)
```

## Lease Recovery

`JOB_LEASE_DURATION` (default 30s). Running jobs with expired `locked_at` return to `READY` on worker startup and during polling recovery.

## Retry Strategy

Exponential backoff with jitter per step `retry_policy`. Permanent failures fail the step and workflow. Transient failures retry until `max_attempts`.

## Idempotency

`Idempotency-Key` + `organization_id` → unique execution. Same key returns existing execution.

## Event Integration

Workflow events (`workflow.execution.*`, `workflow.step.*`) publish via outbox to `halcyon.workflows`. Partition key: `execution_id` for ordering within a run.

## Tenant Isolation

All queries include `organization_id`. Cross-tenant access is forbidden at repository layer.

## Security

- SSRF blocking on HTTP_ACTION URLs (private networks, metadata endpoints)
- Forbidden headers: `Authorization`, `Cookie`
- Workflow limits: max steps, timeouts, retries, recursion depth
- No credential logging

## Scaling Considerations

Bottlenecks: PostgreSQL job claiming, worker concurrency, external HTTP latency, outbox relay throughput. Metrics exposed for load testing preparation.

## Failure Modes

| Failure | Behavior |
|---------|----------|
| Postgres outage | API/worker fail; jobs remain durable |
| Kafka outage | Outbox accumulates; executions complete |
| Worker crash | Lease expires; job recovered |
| Duplicate execution request | Idempotency key deduplicates |
