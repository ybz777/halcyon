# Testing Guide

## Why This Document Exists

Halcyon is an operations platform — bugs cause real downtime for customers.
This document defines what "tested" means and what every PR must include before merge.

## Who Reads This

| Audience | Purpose |
|---|---|
| Feature engineers | Know what tests to write |
| Reviewers | Verify test adequacy |
| CI maintainers | Configure coverage gates |

## Standards for Every PR

- No feature merges without appropriate tests.
- Domain and application layers require unit tests.
- API changes require handler tests.
- Infrastructure adapters require integration tests (when touching adapters).
- Contract changes require schema validation in CI.
- Coverage regression blocks merge (once CI gates are enabled).

---

## Testing Pyramid

```
                    ┌─────────┐
                    │   E2E   │  Few, critical paths
                   ┌┴─────────┴┐
                   │ Integration │  Service + real deps (testcontainers)
                  ┌┴───────────┴┐
                  │  Unit Tests   │  Domain, application, API handlers
                  └───────────────┘
```

| Layer | Location | Dependencies | Speed |
|---|---|---|---|
| Unit | `services/*/internal/**/*_test.go` | Mocks only | < 1s |
| Integration | `services/*/internal/infrastructure/*_test.go` | testcontainers | < 30s |
| Contract | `contracts/` validated in CI | Schema validators | < 10s |
| E2E | `tests/e2e/` | Staging environment | Minutes |
| Load | `tests/load/` | Staging / dedicated | Nightly |

---

## Unit Tests

### Domain Layer

- **Pure logic only** — no I/O, no mocks
- Test value object validation, aggregate behavior, state transitions
- Every aggregate method with business rules must have tests

```go
func TestEmail_InvalidFormat(t *testing.T) { ... }
func TestIncident_TransitionOpenToInvestigating(t *testing.T) { ... }
```

### Application Layer

- Test use cases with **mocked ports** (repository interfaces)
- Verify orchestration logic, error paths, event emission calls
- One test file per use case handler

### API Layer

- `net/http/httptest` — no real server
- Test status codes, response shapes, error mapping
- Test auth middleware behavior (when added)

---

## Integration Tests

### When Required

- New or modified PostgreSQL repository
- New or modified Redis adapter
- New or modified Kafka producer/consumer

### Tools

- [testcontainers-go](https://github.com/testcontainers/testcontainers-go) for Postgres, Redis, Kafka
- Run in CI with Docker-in-Docker
- Tag: `//go:build integration` if separation needed

### Rules

- Each test creates its own data — no shared state
- Clean up after test (transaction rollback or truncate)
- Use realistic SQL migrations (same as production)

---

## Contract Tests

| Contract | Validation |
|---|---|
| OpenAPI specs | Spectral lint + breaking change detection |
| Event schemas | JSON Schema validation in CI |
| gRPC protos (future) | `buf` lint and breaking detection |

Location: `contracts/`

Any PR modifying `contracts/` must pass contract validation CI.

---

## End-to-End Tests

Location: `tests/e2e/`

| Flow | Priority |
|---|---|
| Register → login → create incident | P0 |
| Webhook ingest → correlation → incident | P0 |
| AI brief generation | P1 |
| Search incident | P1 |

Tool: Playwright for `apps/web/` (when added).

Run: staging deploy pipeline, not every PR (too slow).

---

## Load Tests

Location: `tests/load/`

Tool: k6

| Scenario | Target |
|---|---|
| API read throughput | 1,000 RPS sustained |
| Webhook ingestion burst | 10,000 events/min |
| Kafka consumer lag | < 10,000 under load |

Run: nightly + pre-release.

---

## Coverage Targets

| Package | Minimum Coverage |
|---|---|
| `domain` | 90% |
| `application` | 85% |
| `api` | 80% |
| `infrastructure` | 70% (integration tests count) |
| `pkg/platform` | 85% |

Measured by CI (`go test -coverprofile`).

---

## Test Naming Convention

```
Test<Unit>_<Scenario>_<ExpectedOutcome>

Examples:
  TestRegisterUser_DuplicateEmail_ReturnsConflict
  TestHealthHandler_Ready_NoDatabase_ReturnsSkipped
  TestIncidentTransition_ResolvedToOpen_ReturnsError
```

---

## What Not to Test

- Generated code (OpenAPI clients)
- Third-party library internals
- Trivial getters with no logic
- Framework behavior (chi routing itself)

---

## CI Integration

| Trigger | Tests Run |
|---|---|
| Every PR | Unit + contract + lint |
| Merge to main | Unit + integration + deploy staging |
| Nightly | E2E + load |
| Pre-release | Full suite |

---

## Related Documents

- [CODE_STYLE.md](CODE_STYLE.md)
- [API_GUIDELINES.md](API_GUIDELINES.md)
- [EVENT_GUIDELINES.md](EVENT_GUIDELINES.md)
