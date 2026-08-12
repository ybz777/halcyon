# Architecture Decision Records

## Why This Directory Exists

ADRs are the permanent record of significant engineering decisions. They prevent
the same debates from recurring and give future contributors context for why
the system looks the way it does.

## Who Reads This

| Audience | Purpose |
|---|---|
| Engineers | Understand past decisions before proposing changes |
| Reviewers | Verify new work aligns with accepted decisions |
| Architects | Evaluate when a new ADR is needed |

## Standards for Every PR

- Non-trivial architectural decisions require a new ADR in the same PR.
- ADRs are immutable once accepted — supersede with a new ADR, never edit in place.
- Use [template.md](template.md) for new ADRs.
- Large proposals should go through [RFC](../rfc/README.md) first.

---

## Index

| ADR | Title | Status |
|---|---|---|
| [0001](0001-clean-architecture.md) | Clean Architecture for all backend services | Accepted |
| [0002](0002-go-for-core-services.md) | Go for core backend services | Accepted |
| [0003](0003-monorepo-layout.md) | Monorepo layout | Accepted |
| [0004](0004-observability-baseline.md) | Observability baseline per service | Accepted |

---

## Format

Each ADR includes:

- **Context** — what problem we are solving
- **Decision** — what we chose
- **Alternatives** — what we rejected and why
- **Consequences** — tradeoffs and follow-up actions

When superseding a decision, add a new ADR and mark the old one `Superseded by ADR-XXXX`.