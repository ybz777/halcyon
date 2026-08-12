# Halcyon Event Contracts

Shared event schemas live in `pkg/platform/events/`.

## Versioning Convention

Halcyon uses **separate** `event_type` and `event_version` fields:

| Field | Example | Meaning |
|-------|---------|---------|
| `event_type` | `organizations.organization.created` | Stable domain identifier |
| `event_version` | `1.0` | Schema version (`VersionV1`) |

Breaking changes require a new `event_version`. Optional fields may be added to v1 when consumers tolerate unknown fields.

## Serialization

- **Canonical wire format:** JSON (`application/vnd.halcyon.event+json`)
- **Legacy support:** Protobuf wire envelopes are still accepted on consume
- Payloads inside envelopes use typed codecs registered in `DefaultRegistry()`

JSON is appropriate for early-stage interoperability, debuggability, and schema evolution without a registry dependency. The `MarshalPayload` abstraction allows future Avro/Protobuf payload codecs.

## Envelope Fields

| Field | Required | Description |
|-------|----------|-------------|
| `event_id` | yes | Globally unique, stable across retries |
| `event_type` | yes | Registered domain event type |
| `event_version` | yes | Schema version |
| `occurred_at` | yes | UTC timestamp |
| `producer` | yes | Service that emitted the event |
| `correlation_id` | recommended | Workflow correlation |
| `request_id` | recommended | Originating HTTP request |
| `organization_id` | tenant events | Tenant scope |
| `actor_id` | user actions | Acting principal |
| `subject_id` | optional | Event subject when distinct from aggregate |
| `payload` | yes | Typed event body |

## Topics

Domain-bounded topics (never per-organization):

- `halcyon.auth`
- `halcyon.organizations`
- `halcyon.incidents`

DLQ: `{topic}.dlq`

## Partition Key

Default: `organization_id` → `aggregate_id` → `event_id`
