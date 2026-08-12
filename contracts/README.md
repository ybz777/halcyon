# Halcyon Contracts

## Why This Directory Exists

`contracts/` is the **source of truth** for all external interfaces — REST APIs,
event schemas, and (future) GraphQL and gRPC definitions. Implementation follows
contracts, not the reverse.

## Who Reads This

| Audience | Purpose |
|---|---|
| API implementers | Know what to build |
| Consumer engineers (frontend, integrations) | Know what to expect |
| CI | Validate breaking changes |
| External integrators | Generate clients from specs |

## Standards for Every PR

- New or changed endpoints: update `contracts/openapi/` **before** handler code.
- New or changed events: update `contracts/events/` **before** producer/consumer code.
- Contract changes require CI validation to pass.
- Breaking changes require version bump per [API_GUIDELINES.md](../API_GUIDELINES.md)
  and [EVENT_GUIDELINES.md](../EVENT_GUIDELINES.md).

---

## Directory Structure

```
contracts/
├── openapi/           # REST API specifications
│   ├── auth/
│   │   └── v1/
│   │       └── openapi.yaml
│   ├── workflow/
│   └── ...
├── events/            # Kafka event schemas
│   ├── envelope/
│   │   └── v1/
│   │       └── schema.json
│   ├── auth/
│   ├── incidents/
│   └── ...
├── graphql/           # GraphQL schemas (future)
└── proto/             # gRPC definitions (future)
```

---

## Validation

| Contract Type | Tool | CI Job |
|---|---|---|
| OpenAPI | Spectral | `contract-lint` |
| OpenAPI breaking changes | oasdiff | `contract-breaking` |
| Event schemas | JSON Schema validator | `event-schema-validate` |

---

## Code Generation

Generated clients and types are produced by `tools/codegen/` — never hand-edit
generated output.

---

## Related Documents

- [API_GUIDELINES.md](../API_GUIDELINES.md)
- [EVENT_GUIDELINES.md](../EVENT_GUIDELINES.md)
- [TESTING.md](../TESTING.md)
