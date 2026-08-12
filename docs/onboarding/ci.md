# Continuous Integration

## Why This Document Exists

Halcyon CI runs on every pull request and push to `main`. This guide explains
what runs in CI, how to reproduce checks locally, and how to interpret failures.

## Workflows

| Workflow | Trigger | Purpose |
|---|---|---|
| [CI](../../.github/workflows/ci.yml) | PR, push to `main`, `workflow_dispatch` | Quality gates, security, integration |
| [PR Checks](../../.github/workflows/pr-checks.yml) | Pull request | Conventional Commits title validation |

## CI Architecture

```mermaid
flowchart TB
  subgraph parallel [Parallel after quick-checks]
    LINT[lint]
    UNIT[unit-test + coverage]
    RACE[race-test]
  end
  QC[quick-checks: fmt, tidy, vet] --> parallel
  QC --> CONTRACTS[OpenAPI validation]
  QC --> MIGRATIONS[migration validation]
  QC --> SECURITY[govulncheck + gitleaks]
  parallel --> BUILD[build]
  BUILD --> INTEGRATION[integration tests]
  BUILD --> DOCKER[Docker builds + Trivy on main]
```

Path filtering skips expensive Docker builds when only documentation changes.

## Required Checks

PRs must pass:

| Check | Local command |
|---|---|
| Format | `make fmt-check` |
| Dependencies | `make tidy-check` |
| Vet | `make vet` |
| Lint | `make lint` |
| Unit tests | `make test` |
| Race detector | `make test-race` |
| Coverage | `make coverage` |
| OpenAPI | `make validate-openapi` |
| Migrations | `make validate-migrations` |
| Vulnerabilities | `make security-vuln` |
| Build | `make build` |
| Integration | `make test-integration` |

Run everything:

```bash
make ci
```

## Local Prerequisites

- Go 1.25+ (see `.go-version`)
- `golangci-lint` v1.62.2+
- Docker (integration tests and image builds)
- Bash (CI scripts under `scripts/ci/`)

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2
```

## Integration Tests

Integration tests use the `integration` build tag and expect test infrastructure:

```bash
export APP_ENV=test
export DATABASE_URL=postgres://halcyon:halcyon@localhost:5432/halcyon_test?sslmode=disable
export REDIS_URL=redis://localhost:6379/0
export KAFKA_BROKERS=localhost:9092
make test-integration
```

CI starts PostgreSQL, Redis, and Redpanda as GitHub Actions services with isolated
test database names per run.

## Security Scanning

| Tool | Policy |
|---|---|
| `govulncheck` | Fails on known vulnerabilities in module dependencies |
| `gitleaks` | Fails on detected secrets (with documented allowlist) |
| `trivy` | Runs on `main` after Docker builds; fails on CRITICAL/HIGH unfixed CVEs |

## Failure Diagnostics

On integration failure, CI uploads `integration-failure-logs` artifacts.
Coverage reports are uploaded on every unit test run.

## Branch Protection

See [branch-protection.md](branch-protection.md).

## Related Documents

- [CONTRIBUTING.md](../../CONTRIBUTING.md)
- [SECURITY.md](../../SECURITY.md)
- [docs/observability/tracing.md](../observability/tracing.md)
- [local-development.md](local-development.md)
