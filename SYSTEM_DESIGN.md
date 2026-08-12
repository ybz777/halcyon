# Halcyon System Design

## Why This Document Exists

This document captures the product vision, business context, and high-level platform
design. It answers **what** Halcyon is and **why** it exists. Implementation
details live in [ARCHITECTURE.md](ARCHITECTURE.md) and service READMEs.

## Who Reads This

| Audience | Purpose |
|---|---|
| Product and engineering leadership | Strategic alignment |
| New engineers | Business context before diving into code |
| Investors / evaluators | Product-market fit narrative |
| Interviewers | Portfolio discussion depth |

## Standards for Every PR

- Feature PRs must align with the MVP scope and roadmap in this document.
- If a PR changes product scope, update this document and open an ADR in the same PR.
- Do not add implementation detail here — link to ARCHITECTURE.md instead.

---

## 1. Executive Summary

**Halcyon** is an Engineering Operations Intelligence Platform that unifies changes,
signals, incidents, decisions, and institutional knowledge into actionable
**Operational Briefs**.

**Category:** Engineering Operations Intelligence (EOI)

**One-liner:** Halcyon tells engineering teams what changed, what broke, why it
matters, and what to do next — in under 60 seconds.

---

## 2. Vision and Mission

| | |
|---|---|
| **Vision** | Become the default operational intelligence layer for companies running production software at scale |
| **Mission** | Give engineering teams a single source of operational truth |
| **Slogan** | *Operational calm, engineered.* |

### Product Principles

1. **Causality over correlation** — every insight links to evidence
2. **Brief, not dashboard** — answer "what should I do?" first
3. **Integrate, don't replace** — work with Datadog, GitHub, PagerDuty
4. **Memory compounds** — every incident makes the org smarter
5. **Enterprise-ready early** — RBAC, audit, tenancy from day one

---

## 3. Target Customers

### Primary ICP

- 50–2,000 engineers
- 20–300 microservices
- Kubernetes on AWS
- Stack: GitHub, Datadog/Grafana, PagerDuty

### Buyers

| Persona | Role |
|---|---|
| Economic buyer | VP Engineering / CTO |
| Champion | Head of Platform / SRE |
| Daily user | On-call engineer |

---

## 4. Problems Solved

| Problem | Impact |
|---|---|
| Change blindness | #1 postmortem root cause |
| Context switching during incidents | Hours lost per Sev-1 |
| Tribal knowledge | 30–40% repeat incidents |
| Cold on-call handoffs | Slow MTTA |
| Non-compounding postmortems | Same failures recur |
| Leadership without narrative | Poor investment decisions |

---

## 5. Core Capabilities

### Operational Graph

A living graph connecting changes, signals, incidents, decisions, and knowledge.
The data moat — org-specific intelligence that compounds over time.

### Operational Briefs (AI)

Context-aware summaries at critical moments:

- Before on-call shift
- When alert fires
- During incident investigation
- After resolution (postmortem draft)

Every brief is **evidence-linked** — no black-box answers.

### Incident Command Center

Real-time timeline, blast radius, role assignments, AI copilot, similar past
incidents.

### Change Correlation

Automatic linking of deploys, config changes, and feature flags to alerts and
incidents.

### Reliability Analytics

MTTR, repeat incident rate, service reliability scorecards, AI effectiveness.

---

## 6. Platform Architecture Summary

### Microservices (13)

| Layer | Services |
|---|---|
| Platform | auth, user, organization, gateway, audit |
| Domain | ingestion, correlation, workflow, topology, ai |
| Read models | search, analytics, notification |

### Data Stores

| Store | Use |
|---|---|
| PostgreSQL | OLTP per service |
| Redis | Cache, sessions, correlation windows |
| Elasticsearch | Search + vector RAG (shared indices) |
| S3 | Raw payloads, archives, audit cold storage |
| Kafka | Event backbone |

### AI Layer

- LLM Gateway (model routing, cost tracking)
- RAG over incidents, runbooks, postmortems
- Prompt management with versioning
- Guardrails: citation validation, PII scanning

---

## 7. Scale Targets

| Dimension | Target |
|---|---|
| Organizations | 10,000 |
| Users | 250,000 |
| Events/day | 100,000,000 |
| Engineers/contributors | 100+ |

### Scale Implications (Baked Into Design)

- Shared Elasticsearch indices with `org_id` filter (not per-org indices)
- Kafka partition key = `org_id` with hot-tenant monitoring
- PgBouncer for PostgreSQL connection pooling
- Idempotent Kafka consumers (at-least-once + dedup)
- No unbounded `org_id` on Prometheus metric labels

---

## 8. Security and Compliance

- JWT + OAuth2 + RBAC + MFA
- API keys with scoped permissions
- Immutable hash-chained audit log
- Tenant isolation at every layer
- SOC 2 alignment (future certification)

See [SECURITY.md](SECURITY.md).

---

## 9. Deployment

- AWS: EKS, RDS, S3, ALB, CloudFront, Lambda, Secrets Manager
- CI/CD: GitHub Actions → ECR → Helm → EKS
- Observability: OpenTelemetry → Prometheus, Grafana, Tempo, Loki

See [RELEASE_PROCESS.md](RELEASE_PROCESS.md) and `infra/`.

---

## 10. Competitive Positioning

Halcyon is the **intelligence layer** — not observability, not ITSM, not docs.

| Tool | Halcyon relationship |
|---|---|
| Datadog | Ingests signals from Datadog |
| PagerDuty | Ingests alerts; may notify via PagerDuty |
| GitHub | Ingests deploy events |
| incident.io | Overlaps on workflow; Halcyon differentiates on intelligence |
| Notion/Confluence | Ingests docs for RAG |

---

## Related Documents

- [ROADMAP.md](ROADMAP.md) — delivery timeline
- [ARCHITECTURE.md](ARCHITECTURE.md) — engineering structure
- [API_GUIDELINES.md](API_GUIDELINES.md) — API design
- [EVENT_GUIDELINES.md](EVENT_GUIDELINES.md) — event design
