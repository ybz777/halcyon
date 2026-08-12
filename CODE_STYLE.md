# Code Style Guide

## Why This Document Exists

Consistent code style reduces cognitive load across 100+ contributors and makes
review faster. This document defines enforceable standards — not suggestions.

## Who Reads This

| Audience | Purpose |
|---|---|
| All engineers | Write consistent code |
| Reviewers | Objective style feedback |
| CI systems | Automated lint rules |

## Standards for Every PR

- All new code must follow this guide.
- `golangci-lint` and `eslint` enforce what can be automated.
- Reviewers reject PRs with style violations not caught by CI.
- Exceptions require a comment explaining why.

---

## Universal Principles

1. **Readability over cleverness** — boring code is good code
2. **Explicit over implicit** — no magic
3. **Small files** — target ~200 lines; split at ~150 if growing
4. **Small functions** — single responsibility; early returns
5. **No commented-out code** — delete it; git remembers
6. **No TODOs** — open an issue instead (unless explicitly approved in RFC)
7. **Meaningful names** — `incidentRepository` not `repo`; `ctx` for context
8. **Errors are values** — never ignore; wrap with context in application layer

---

## Go (Backend Services)

### Formatting

- `gofmt` and `goimports` — non-negotiable
- `golangci-lint` in CI — must pass

### Naming

| Item | Convention | Example |
|---|---|---|
| Packages | lowercase, single word | `domain`, `postgres` |
| Interfaces | noun or -er suffix | `UserRepository`, `EventPublisher` |
| Implementations | descriptive struct | `PostgresUserRepository` |
| Errors | `Err` prefix | `ErrInvalidEmail` |
| Tests | `Test<Function>_<Case>` | `TestRegister_DuplicateEmail` |

### Error Handling

```go
// Domain: sentinel errors
var ErrInvalidEmail = errors.New("invalid email")

// Application: wrap with context
return fmt.Errorf("register user: %w", err)

// API: map to HTTP status — never expose internals
```

### Dependency Injection

- Constructor functions: `New<UserRepository>(db *pgxpool.Pool) UserRepository`
- No global state, no service locator, no `init()` with side effects
- Wire dependencies in `cmd/<name>/main.go` only

### Logging

- Use `pkg/platform/observability` — never raw `fmt.Println`
- Structured key-value pairs
- Never log secrets, tokens, passwords, or PII

### Tests

- Table-driven tests for multiple cases
- `t.Parallel()` where safe
- Colocated `*_test.go` files
- See [TESTING.md](TESTING.md)

---

## TypeScript (apps/)

### Formatting

- Prettier + ESLint — enforced in CI
- 2-space indent, single quotes, trailing commas

### Naming

| Item | Convention |
|---|---|
| Components | PascalCase (`IncidentTimeline`) |
| Hooks | camelCase with `use` prefix (`useIncident`) |
| Files | kebab-case or PascalCase for components |
| Types/Interfaces | PascalCase, no `I` prefix |

### Components

- Functional components with typed props
- No business logic in components — use hooks/services layer
- API calls via generated clients from `contracts/openapi/`

---

## Python (AI Service — Future)

- `ruff` for linting and formatting
- Type hints on all public functions
- `pydantic` for data validation
- Follow PEP 8

---

## File Organization

| Rule | Detail |
|---|---|
| One type per file (domain) | `user.go`, `email.go` — not `entities.go` |
| Handlers thin | Delegate to application layer immediately |
| No circular imports | Enforced by Go compiler; design to avoid |
| Config in `internal/config/` | Not scattered across packages |

---

## Comments

- Explain **why**, not **what**
- Package-level `doc.go` for public packages
- No redundant comments on obvious code
- ADRs for architectural decisions — not code comments

---

## Imports

Go import order (enforced by `goimports`):

1. Standard library
2. Third-party
3. `pkg/platform/...`
4. Internal packages

---

## Related Documents

- [ARCHITECTURE.md](ARCHITECTURE.md) — layer rules
- [TESTING.md](TESTING.md) — test standards
- [API_GUIDELINES.md](API_GUIDELINES.md) — API handler patterns
