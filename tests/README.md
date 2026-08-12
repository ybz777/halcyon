# Integration Tests

Platform integration tests live in `tests/integration/` and use the `integration` build tag.

## Running locally

```bash
export APP_ENV=test
export DATABASE_URL=postgres://halcyon:halcyon@localhost:5432/halcyon_test?sslmode=disable
export REDIS_URL=redis://localhost:6379/0
export KAFKA_BROKERS=localhost:9092
make test-integration
```

Service-specific integration tests also exist under each service's
`internal/infrastructure/postgres` package.

## CI

GitHub Actions starts PostgreSQL, Redis, and Redpanda and runs `make test-integration`
with isolated `_test` database names.
