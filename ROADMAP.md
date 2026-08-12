# Halcyon Roadmap

## Why This Document Exists

This document aligns engineering execution with product strategy. It tells
contributors what we are building, in what order, and why — so PRs land in the
right sequence and architectural foundations are not skipped.

## Who Reads This

| Audience | Purpose |
|---|---|
| Engineers | Prioritize work and understand dependencies |
| Product / leadership | Track delivery against milestones |
| Contributors | Find high-impact areas to help |
| Recruiters / evaluators | Assess project scope and maturity |

## Standards for Every PR

- PRs should map to a roadmap phase or open RFC.
- Work outside the current phase requires RFC approval.
- Completing a milestone requires updating this document in the same PR.
- Do not mark items complete without meeting acceptance criteria listed here.

---

## Vision

Halcyon becomes the default **intelligence layer** for engineering operations —
the place teams go to understand what changed, what broke, and what to do next.

**North star metric:** Time from alert to actionable brief < 60 seconds.

---

## Phase 0: Foundation (Current)

**Goal:** Production-grade monorepo scaffold that demonstrates senior engineering craft.

| Item | Status | Acceptance Criteria |
|---|---|---|
| Repository structure frozen | Done | ADR-0003, constitution docs |
| Auth service skeleton | Done | Health, ready, metrics, clean architecture |
| Observability baseline | Done | slog, Prometheus via `pkg/platform` |
| Local dev environment | Done | `make up` starts Postgres, Redis, Redpanda |
| `pkg/platform` libraries | Done | observability, httputil, httperrors, ids |
| Contract stubs | Done | OpenAPI auth v1 + event envelope schema |
| CI pipeline | Done | GitHub Actions: test, lint, build, Docker |

**Exit criteria:** New engineer clones, runs `make test`, gets green in < 15 minutes.

---

## Phase 1: Identity and Tenancy (MVP Core)

**Goal:** Secure multi-tenant platform with user and organization management.

| Item | Depends On | Acceptance Criteria |
|---|---|---|
| User registration and login | Phase 0 | JWT + refresh rotation |
| Organization CRUD | Auth | `org_id` on all tenant paths |
| RBAC (roles, permissions) | Organization | Enforced at service layer |
| API keys | Auth | Scoped, hashed, audited |
| MFA (TOTP) | Auth | Step-up for sensitive ops |
| Audit log (hash chain) | Auth, Org | Immutable, queryable |
| Gateway service | Auth | JWT validation, rate limiting |

**Exit criteria:** Two orgs cannot see each other's data. Pen test checklist passes.

---

## Phase 2: Signal Ingestion

**Goal:** Halcyon receives signals from the tools teams already use.

| Item | Depends On | Acceptance Criteria |
|---|---|---|
| Webhook ingestion service | Phase 1 | HMAC validation, idempotent |
| GitHub integration | Ingestion | Deploy events normalized |
| Datadog integration | Ingestion | Alert webhooks ingested |
| PagerDuty integration | Ingestion | Incident webhooks ingested |
| Signal normalization | Ingestion | Unified `Signal` schema |
| Kafka event backbone | Phase 0 | Outbox pattern, DLQ |
| S3 raw payload archive | Ingestion | 90-day retention |

**Exit criteria:** Deploy event from GitHub appears in Halcyon within 30 seconds.

---

## Phase 3: Correlation and Incidents

**Goal:** Automatic linking of changes to signals; incident lifecycle management.

| Item | Depends On | Acceptance Criteria |
|---|---|---|
| Correlation engine | Phase 2 | Time-window matching |
| Incident state machine | Phase 1 | Open → Investigating → Resolved |
| Incident Command Center UI | Phase 3 | Real-time timeline |
| Topology service | Phase 2 | Service dependency graph |
| Operational Brief (basic) | Phase 3 + AI | Evidence-linked summary |
| Search (Elasticsearch) | Phase 3 | Full-text incident search |

**Exit criteria:** Alert auto-correlates to recent deploy. Brief generated in < 60s.

---

## Phase 4: AI Platform

**Goal:** Evidence-linked AI that compounds organizational knowledge.

| Item | Depends On | Acceptance Criteria |
|---|---|---|
| LLM gateway | Phase 0 | Model routing, cost tracking |
| RAG pipeline | Phase 3, Search | Citations validated |
| Embedding service | Phase 2 | Document indexing |
| Prompt management | AI | Versioned prompts |
| Postmortem draft generation | Phase 3 + AI | Human-reviewable output |
| PII guardrails | AI, Security | Scan before inference |
| Conversation memory | AI | Context within incident |

**Exit criteria:** AI brief cites specific deploy event and alert. No hallucinated evidence.

---

## Phase 5: Analytics and Reliability

**Goal:** Leadership visibility into engineering reliability.

| Item | Depends On | Acceptance Criteria |
|---|---|---|
| MTTR / MTTA dashboards | Phase 3 | Per-service, per-team |
| Repeat incident detection | Phase 3 | Same root cause flagged |
| Reliability scorecards | Analytics | Weekly auto-generated |
| AI effectiveness metrics | Phase 4 | Brief accuracy tracking |
| Notification service | Phase 1 | Email, Slack, PagerDuty out |

**Exit criteria:** VP Engineering can view org-wide reliability trends.

---

## Phase 6: Enterprise and Scale

**Goal:** Production readiness for 10K orgs, 100M events/day.

| Item | Depends On | Acceptance Criteria |
|---|---|---|
| SSO (SAML/OIDC) | Phase 1 | Okta, Azure AD |
| SOC 2 alignment | All phases | Control mapping documented |
| Multi-region deployment | Infra | Active-passive failover |
| Hot-tenant isolation | Phase 2 | Dedicated partitions for large orgs |
| Cassandra analytics | Phase 5 | Time-series at scale |
| GraphQL read API | Phase 3 | Incident Command Center queries |
| Admin portal | Phase 1 | Org management, billing hooks |

**Exit criteria:** Load test passes 100M events/day. SOC 2 readiness assessment complete.

---

## Non-Goals (Explicit)

| Item | Reason |
|---|---|
| Replace Datadog / PagerDuty | Integration partner, not competitor |
| Custom observability agent | Ingest via webhooks and APIs |
| Mobile native apps | Web-first; responsive design |
| On-premise deployment (v1) | SaaS only until enterprise demand |
| Per-org Elasticsearch indices | Does not scale to 10K orgs |

---

## How to Propose Changes

Roadmap changes require an RFC in `docs/rfc/`. See [CONTRIBUTING.md](CONTRIBUTING.md).

Priority adjustments are discussed in weekly engineering sync (when team exists)
or via GitHub Discussion / RFC comment thread.

---

## Related Documents

- [SYSTEM_DESIGN.md](SYSTEM_DESIGN.md)
- [ARCHITECTURE.md](ARCHITECTURE.md)
- [RELEASE_PROCESS.md](RELEASE_PROCESS.md)
- [docs/rfc/README.md](docs/rfc/README.md)
