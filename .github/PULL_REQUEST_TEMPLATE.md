## Summary

<!-- What does this PR do? One paragraph. -->

## Motivation

### Business
<!-- Why does this matter to users or the product? -->

### Engineering
<!-- What technical problem does this solve? -->

## Design Decisions

<!-- What was chosen and why? Link ADR/RFC if applicable. -->

- **ADR/RFC:** <!-- ADR-0000 or RFC-0000, or N/A -->
- **Approach:**
- **Tradeoffs:**
- **Alternatives rejected:**

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation
- [ ] Refactor
- [ ] Performance
- [ ] Security
- [ ] Chore / CI

## Quality Gate

**All items must pass before merge.** See [CONTRIBUTING.md](../CONTRIBUTING.md).

### Build
- [ ] Workspace builds successfully

### Formatting
- [ ] `gofmt`
- [ ] `go vet`
- [ ] `golangci-lint`

### Tests
- [ ] All unit tests pass (`make test`)
- [ ] Coverage reported (`make coverage`)

### Architecture
- [ ] Clean Architecture dependency rules respected
- [ ] No cyclic dependencies
- [ ] No business logic in HTTP layer
- [ ] `docs/services/<name>.md` updated (if service changed)

### Security
- [ ] No secrets committed
- [ ] No hardcoded credentials
- [ ] Configuration via environment variables

### Observability
- [ ] Structured logs
- [ ] Metrics exposed
- [ ] Health endpoint
- [ ] Readiness endpoint

### Documentation
- [ ] README updated if needed
- [ ] OpenAPI updated for API changes
- [ ] ADR/RFC updated if architecture changed
- [ ] Performance budgets defined for new public endpoints

### Repository
- [ ] No file exceeds ~200 lines (when practical)
- [ ] No duplicated logic
- [ ] No TODO comments

## Testing

<!-- How was this tested? Include commands. -->

```bash
make test
make lint
make coverage
```

## Performance Budgets

<!-- For new public endpoints, document P50/P95 targets. N/A if no new endpoints. -->

| Endpoint | P50 | P95 |
|---|---|---|
| | | |

## Deployment Notes

<!-- Migrations, feature flags, rollout plan. N/A if not applicable. -->

## Related Issues

<!-- Fixes #123, Relates to RFC-0001 -->
