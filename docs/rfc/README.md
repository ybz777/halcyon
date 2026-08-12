# Request for Comments (RFC)

## Why This Directory Exists

Large or controversial changes need design review before implementation. RFCs
provide a structured forum for proposals — similar to Kubernetes KEPs, Rust RFCs,
or Envoy design docs.

## Who Reads This

| Audience | Purpose |
|---|---|
| Engineers proposing major work | Submit RFC before coding |
| Reviewers and maintainers | Evaluate tradeoffs |
| Future contributors | Understand why features exist |

## Standards for Every PR

- Work that meets RFC triggers (below) **must** have an accepted RFC before merge.
- RFC number assigned sequentially: `RFC-0001`, `RFC-0002`, ...
- Status must be `Accepted` before implementation PRs reference the RFC.
- Superseded RFCs are marked `Superseded by RFC-XXXX` — never deleted.

---

## When an RFC Is Required

| Trigger | Example |
|---|---|
| New microservice | `services/billing/` |
| Breaking API or event change | `incidents.incident.created` v2 |
| New data store | Adding Cassandra |
| Cross-cutting platform change | New auth model |
| Performance architecture change | Per-tenant Kafka topics |
| Security model change | New token format |
| > 2 weeks of engineering effort | Incident Command Center UI |
| Roadmap phase change | Skipping Phase 2 |

**When in doubt, write an RFC.** Small RFCs are cheap; large reversals are expensive.

---

## RFC Lifecycle

```
Draft → Under Review → Accepted → Implemented → (Closed)
                    ↘ Rejected
                    ↘ Withdrawn
```

| Status | Meaning |
|---|---|
| `Draft` | Author writing; not ready for review |
| `Under Review` | Open for comment (minimum 5 business days) |
| `Accepted` | Approved; implementation may begin |
| `Rejected` | Not proceeding; rationale documented |
| `Withdrawn` | Author withdrew |
| `Implemented` | Landed in `main`; link to PRs |
| `Superseded` | Replaced by newer RFC |

---

## How to Submit

1. Copy [template.md](template.md) to `RFC-NNNN-short-title.md`
2. Fill all sections — incomplete RFCs are sent back
3. Open PR with label `rfc` (or note in PR description)
4. Request review from relevant code owners
5. Address feedback; update status to `Under Review`
6. After acceptance, create ADR if architectural decision is involved

---

## Index

| RFC | Title | Status |
|---|---|---|
| — | *No RFCs yet* | — |

---

## Related Documents

- [docs/adr/README.md](../adr/README.md)
- [CONTRIBUTING.md](../../CONTRIBUTING.md)
- [ARCHITECTURE.md](../../ARCHITECTURE.md)
