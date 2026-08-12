# Testing Guide

Halcyon uses a layered test pyramid: many fast unit tests, moderate integration tests, and a small set of end-to-end scenarios.

## Test Layers

| Layer | Location | Build tag | Dependencies |
|---|---|---|---|
| Unit | `pkg/platform/**`, `services/**/internal/**` | none | none |
| Integration | `*_integration_test.go`, `tests/integration` | `integration` | Postgres, Redis, Kafka (optional) |
| Security | `tests/security`, `**/security_test.go`, `test-auth` | none | none |
| E2E | `tests/e2e` | `e2e` | Running services |

## Commands

```bash
make test              # all unit tests (no integration tag)
make test-unit         # alias for unit-only platform + services
make test-integration  # integration-tagged tests
make test-security     # security regression suite
make test-e2e          # end-to-end (requires running stack)
make test-race         # race detector (Linux/macOS with CGO)
make test-auth         # auth JWT matrix + middleware
make test-idempotency  # idempotency unit tests
make test-jobs         # background job unit tests
make coverage          # coverage profiles for core modules
```

## Configuration in Tests

Use `pkg/platform/testutil` for deterministic secrets and environment:

```go
cleanup := platformconfig.SetTestEnv(map[string]string{
    "APP_ENV": "test",
    "JWT_SECRET": testutil.TestJWTSecret,
})
defer cleanup()
```

Never commit real credentials. Fake secrets are validated by `config/leak_regression_test.go`.

## Integration Dependencies

Start local infrastructure:

```bash
docker compose -f infra/docker/compose.yml up -d postgres redis redpanda
```

Set environment for `tests/integration`:

```bash
export APP_ENV=test
export DATABASE_URL=postgres://halcyon:halcyon@localhost:5432/halcyon_test?sslmode=disable
export REDIS_URL=redis://localhost:6379/0
export KAFKA_BROKERS=localhost:19092
```

## Security Testing

Security regression tests cover:

- JWT validation matrix (`pkg/platform/auth`)
- Gateway trust / API key header spoofing
- Idempotency principal and tenant scope isolation
- Organization RBAC, tenant escape, IDOR, mass assignment
- Secret redaction in config serialization

Run: `make test-security`

## Concurrency Testing

- Idempotency concurrent single-execution: `pkg/platform/idempotency/idempotency_test.go`
- Job pool single-execution and retry: `pkg/platform/jobs/pool_test.go`
- Pagination completeness: `pkg/platform/pagination/completeness_test.go`

Run race detector on Linux CI: `make test-race`

## Fuzz Testing

Bounded fuzz targets:

- `pkg/platform/auth/fuzz_test.go` — JWT parser
- `pkg/platform/pagination/fuzz_test.go` — cursor decoder

```bash
cd pkg/platform && go test -fuzz=FuzzHS256Validator_ParseToken ./auth -fuzztime=30s
```

## Failure Injection

Integration tests use testcontainers or env-based services. Subsystems document fail-closed vs fail-open behavior:

| Subsystem | Redis down policy |
|---|---|
| Rate limit | configurable (`RATE_LIMIT_FAILURE_MODE`) |
| Idempotency | fail-closed when required |

## Debugging Failed Tests

1. Run the specific package: `go test -v ./path/to/pkg -run TestName`
2. For integration: ensure Docker services are healthy
3. Check `APP_ENV=test` is set for integration smoke tests
4. Use `-count=1` to disable caching

## CI

GitHub Actions runs `make ci` (fmt, vet, lint, test, test-race on Linux, coverage, build). Integration job starts Postgres, Redis, and Redpanda service containers.

## OpenAPI Validation

```bash
make validate-openapi
```

## Coverage

```bash
make coverage
go tool cover -func=coverage/platform.out
```

Prioritize coverage on authentication, authorization, tenant isolation, idempotency, jobs, and pagination — not arbitrary 100%.
