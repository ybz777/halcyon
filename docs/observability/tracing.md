# Distributed Tracing

Halcyon uses OpenTelemetry for distributed tracing across HTTP services, Kafka
consumers, background jobs, and external integrations.

## Architecture

Tracing lives in `pkg/platform/observability`. Business code does not import
OpenTelemetry directly.

```
HTTP request
  → Chi trace middleware (server span)
  → application layer
  → PostgreSQL / Redis (instrumented drivers where enabled)
  → Kafka producer (W3C traceparent in headers)
  → Kafka consumer (consumer span)
  → background job worker (job span)
  → external HTTP (client span)
```

## Identifiers

| Field | Purpose |
|-------|---------|
| `trace_id` | Distributed trace across services |
| `span_id` | Single operation within a trace |
| `request_id` | Per HTTP request correlation |
| `correlation_id` | Business workflow correlation |
| `causation_id` | Parent event that caused this action |

## Configuration

| Variable | Description |
|----------|-------------|
| `OTEL_ENABLED` | Enable tracing export (`false` default locally) |
| `OTEL_SERVICE_NAME` | Service name resource attribute |
| `SERVICE_VERSION` | Service version (`dev` default) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Collector endpoint |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` or `grpc` |
| `OTEL_TRACES_SAMPLER` | `parentbased_traceidratio` recommended |
| `OTEL_TRACES_SAMPLER_ARG` | Sample ratio `0.0-1.0` |
| `OTEL_RESOURCE_ATTRIBUTES` | Extra resource attributes |

When `OTEL_ENABLED=false`, services use a no-op tracer provider with zero network overhead.

## Kafka Propagation

Producers inject W3C `traceparent` into Kafka message headers. Consumers extract
the context and create child spans. Business event payloads are never modified.

## Job Tracing

Each job execution creates a span named `job <type>` with attributes:

- `job.type`
- `job.attempt`
- `job.priority`

Job IDs and organization IDs are not used as metric labels.

## Security

Tracing must never record:

- Authorization headers
- API keys or webhook secrets
- Passwords or JWTs
- Request/response bodies
- Kafka or job payloads containing secrets

Webhook deliveries do not forward internal trace headers to customer endpoints.

## Local Development

Start the optional collector:

```bash
docker compose -f infra/docker/compose.yml up -d otel-collector
```

Enable tracing in `.env`:

```env
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_TRACES_SAMPLER=always_on
```

Collector debug output appears in the `otel-collector` container logs.

## Sampling

Production recommendation:

```env
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1
```

## Logging Correlation

`NewLoggerWithTracing` enriches structured logs with `trace_id`, `span_id`,
`request_id`, and `correlation_id` when present in context.

## Shutdown

Services call `TracerProvider.Shutdown` during graceful shutdown to flush pending spans.
