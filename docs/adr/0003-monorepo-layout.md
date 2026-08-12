# ADR-0003: Monorepo Layout

## Status

Accepted

## Context

Halcyon will ship many services, shared documentation, infrastructure definitions,
and ADRs. We need a repository structure that supports incremental PRs and
open-source contribution.

## Decision

Use a **monorepo** with this top-level layout:

```
halcyon/
├── docs/           # ADRs, engineering guides, PR records
├── infra/          # docker-compose, terraform, helm (later)
├── services/       # one directory per microservice
└── pkg/            # shared libraries (only when needed)
```

Each service is self-contained with its own `go.mod`, `cmd/`, and `internal/`.

## Alternatives

| Alternative | Why rejected |
|---|---|
| Polyrepo (one repo per service) | High coordination overhead for solo/small team |
| Monolith first | Conflicts with SDD microservices story and scale path |
| `pkg/` from day one | Premature abstraction before two services share code |

## Consequences

### Positive

- Single clone for full system context
- Unified ADRs and architecture docs
- Atomic cross-service documentation updates

### Negative

- CI must use path filters to avoid rebuilding all services
- Repo size grows; requires CODEOWNERS and conventions

## Scale Note

At very large team scale (100+ engineers), partial monorepo pain may justify
splitting hot services. For 10K orgs / 250K users, monorepo remains appropriate.
