# OpenAPI Specifications

REST API contracts for all Halcyon services.

## Layout

```
openapi/<service>/v1/openapi.yaml
```

## Rules

- OpenAPI 3.1
- All paths prefixed with `/api/v1/`
- Errors documented as RFC 7807 Problem Details
- See [API_GUIDELINES.md](../../API_GUIDELINES.md)

## Status

Specifications will be added as services are implemented. Auth service spec is
the first priority (Phase 0 / Phase 1).
