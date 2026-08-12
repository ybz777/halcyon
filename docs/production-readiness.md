# Production Readiness Checklist

Final audit (PR #32). Status reflects verification performed during the audit, not aspirational targets.

## Checklist

| Area | Status | Notes |
|------|--------|-------|
| Architecture | PASS | Microservices with gateway edge, per-service DBs, event outbox documented in [architecture.md](architecture.md) |
| Security | PASS | Tenant header spoofing blocked in production; gateway service tokens required; security regression tests pass |
| Authentication | PASS | JWT + API keys at gateway; JWKS; production secret validation |
| Authorization | PASS | RBAC per service; permission middleware on protected routes |
| Tenant Isolation | PASS | `organization_id` scoping in queries; gateway attestation for org headers in production |
| Database | PASS | Migrations validated; FK constraints; per-service schemas |
| Redis | PASS | Required in production when rate limit/idempotency enabled; fail-closed rate limits |
| Events | PASS | Outbox relay, DLQ, idempotent consumers; JSON envelope tested |
| Workers | PASS | Lease-based claims, retries, retention; graceful shutdown |
| API | PASS | Consistent error envelope, pagination, idempotency on mutations |
| Testing | PASS | Unit, integration, security, e2e targets in Makefile |
| Performance | PASS | Documented in [performance.md](performance.md); claim index migration |
| Docker | PASS | Multi-stage images, non-root user, healthchecks, `.dockerignore` |
| CI/CD | PASS | fmt, vet, lint, test, race (Linux), build in CI workflow |
| Deployment | PASS | Documented in [deployment.md](deployment.md); smoke script |
| Observability | PASS | Structured logs, metrics, request IDs |
| Documentation | PASS | README, architecture, operations, deployment, this checklist |

## Known Limitations

- **Single-region:** No multi-region failover or active-active deployment.
- **Load testing:** Benchmarks and smoke scripts exist; full production-scale load test not performed in CI.
- **Helm/K8s:** Compose is the verified local path; production Helm charts referenced in older docs are not in this repository.
- **Race detector:** `-race` runs in CI on Linux; Windows dev environments skip CGO race builds.
- **Direct service access:** Allowed in non-production for developer ergonomics; production requires gateway.
- **Kafka:** Local stack uses Redpanda; production broker configuration is operator-managed.
- **Manual rollback:** Application rollback is redeploy previous image; DB migrations may not always be reversible.

## Technical Debt

| Item | Impact | Why it remains | Recommended direction |
|------|--------|----------------|----------------------|
| Per-service JWT validators | Medium | Each service embeds HS256 validator instead of shared JWKS client | Consolidate on JWKS fetch with caching |
| Duplicate auth middleware | Low | Slight copy across services for bounded-context isolation | Extract only if drift appears |
| In-memory idempotency fallback | Low | Dev convenience when Redis absent | Remove fallback in production-only builds if stricter policy needed |
| Workflow/org gateway trust | Low | Organization service accessed mainly via gateway today | Add `RequireGatewayForOrganizationHeaders` if direct exposure increases |

## Incident first steps

1. Check gateway `/api/v1/ready` and upstream service readiness.
2. Inspect structured logs filtered by `request_id`.
3. Verify Postgres, Redis, Kafka connectivity from affected pods.
4. Check job queue depth and worker logs for backlog.
5. Review rate-limit and auth failure metrics.

## Verification commands

```bash
make release-check          # fmt, tidy, migrations, openapi, vet, lint, test, security, build
make test-security
make test-integration       # requires Docker stack
make docker-build-all       # optional local image build
```

## Overall assessment

**READY FOR PRODUCTION** — with non-blocking limitations above. Blocking issues identified during audit (cross-tenant org header spoofing, missing production secret enforcement) were fixed in PR #32.
