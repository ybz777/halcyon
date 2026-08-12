# Service Architecture Documents

Every Halcyon microservice has a dedicated architecture page in this directory.
A new engineer should understand any service in under five minutes.

## Required Sections

Each `docs/services/<name>.md` must include:

- Purpose
- Responsibilities
- Dependencies
- Public APIs (with performance budgets)
- Events Produced
- Events Consumed
- Database Ownership
- Scaling Characteristics
- Failure Modes
- Future Work

## Index

| Service | Document | Status |
|---|---|---|
| auth | [auth.md](auth.md) | PR #1 |
