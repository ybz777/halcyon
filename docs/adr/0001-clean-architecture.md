# ADR-0001: Clean Architecture for All Backend Services

## Status

Accepted

## Context

Halcyon is a multi-service platform that must remain maintainable as the team
and codebase grow. Business logic must be testable without HTTP, databases, or
third-party APIs.

## Decision

Every backend service follows Clean Architecture with four internal layers:

```
domain → application → infrastructure → api (delivery)
```

Dependency rule: source code dependencies point inward only.

- **domain** — aggregates, value objects, domain events, repository interfaces
- **application** — use cases (commands/queries), ports
- **infrastructure** — PostgreSQL, Redis, Kafka, external API adapters
- **api** — HTTP handlers, DTOs, routing

The composition root (`cmd/server/main.go`) is the only place where concrete
implementations are wired.

## Alternatives

| Alternative | Why rejected |
|---|---|
| MVC in handlers | Business logic leaks into controllers; hard to test |
| Hexagonal without strict folders | Boundaries erode without enforced package structure |
| Shared “god” internal package | Hidden coupling across features |

## Consequences

### Positive

- Unit tests can target domain and application without infrastructure
- Infrastructure swaps (e.g., ORM, broker) are isolated
- Interview and reviewer clarity: boundaries are visible in the repo

### Negative

- More packages and files per feature
- Requires discipline to avoid “just this once” layer violations
