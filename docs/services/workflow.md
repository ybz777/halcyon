# Workflow Service

Halcyon workflow service provides durable workflow execution with PostgreSQL-backed jobs, a worker runtime, and transactional outbox events.

## APIs

| Method | Path | Permission |
|--------|------|------------|
| POST | `/api/v1/workflows` | `workflows.write` |
| GET | `/api/v1/workflows` | `workflows.read` |
| GET | `/api/v1/workflows/{workflow_id}` | `workflows.read` |
| POST | `/api/v1/workflows/{workflow_id}/executions` | `workflows.execute` |
| GET | `/api/v1/workflow-executions/{execution_id}` | `workflows.read` |
| POST | `/api/v1/workflow-executions/{execution_id}/cancel` | `workflows.cancel` |

Start execution supports `Idempotency-Key` header (scoped per organization).

## Step Types

- `HTTP_ACTION` — outbound HTTP with SSRF protection
- `EVENT_EMIT` — publishes via transactional outbox
- `DELAY` — schedules continuation via `available_at` (no worker sleep)

## Local Development

```bash
export WORKFLOW_DATABASE_URL=postgres://halcyon:halcyon@localhost:5432/halcyon_workflow?sslmode=disable
export JWT_SECRET=change-me-to-a-secret-with-at-least-32-characters
make run-workflow
make run-workflow-worker
```

## Related

- [Workflow Engine Architecture](../architecture/workflow-engine.md)
