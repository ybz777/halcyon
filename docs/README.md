# Halcyon Documentation

## Why This Directory Exists

Central index for all engineering documentation beyond the root constitution files.
New contributors start at the repository root; this directory holds deeper references,
decision records, proposals, and operational guides.

## Who Reads This

| Audience | Start Here |
|---|---|
| New engineers | [onboarding/local-development.md](onboarding/local-development.md) |
| Contributors fixing CI | [onboarding/ci.md](onboarding/ci.md) |
| Architects | [adr/](adr/) |
| Proposers of large changes | [rfc/](rfc/) |
| Security reviewers | [security/threat-model.md](security/threat-model.md) |

## Standards for Every PR

- Documentation changes ship in the **same PR** as code changes they describe.
- New ADRs for architectural decisions; RFCs for proposals before large work.
- Do not duplicate content — link to root constitution files instead.

---

## Directory Map

| Path | Purpose |
|---|---|
| [adr/](adr/) | Architecture Decision Records (accepted decisions) |
| [rfc/](rfc/) | Request for Comments (proposals under discussion) |
| [onboarding/](onboarding/) | Getting started guides, CI, branch protection |
| [security/](security/) | Threat model, security runbooks |
| [services/](services/) | Per-service architecture pages |
| [events/](events/) | Event catalog |
| [api/](api/) | WebSocket and extended API docs (when populated) |
| [runbooks/](runbooks/) | Operational procedures (when populated) |

---

## Root Constitution (Authoritative)

These files live at the repository root and govern all contributions:

| Document | Scope |
|---|---|
| [README.md](../README.md) | Project overview and quickstart |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Contribution workflow |
| [ARCHITECTURE.md](../ARCHITECTURE.md) | System structure |
| [SYSTEM_DESIGN.md](../SYSTEM_DESIGN.md) | Product and platform design |
| [CODE_STYLE.md](../CODE_STYLE.md) | Coding standards |
| [SECURITY.md](../SECURITY.md) | Security policy |
| [TESTING.md](../TESTING.md) | Testing requirements |
| [OBSERVABILITY.md](../OBSERVABILITY.md) | Logging, metrics, tracing |
| [API_GUIDELINES.md](../API_GUIDELINES.md) | REST and API design |
| [EVENT_GUIDELINES.md](../EVENT_GUIDELINES.md) | Event-driven patterns |
| [DATABASE_GUIDELINES.md](../DATABASE_GUIDELINES.md) | Schema and migration rules |
| [PERFORMANCE_GUIDELINES.md](../PERFORMANCE_GUIDELINES.md) | Latency budgets |
| [RELEASE_PROCESS.md](../RELEASE_PROCESS.md) | Versioning and deployment |
| [ROADMAP.md](../ROADMAP.md) | Delivery phases |

---

## ADR vs RFC

| Type | When | Outcome |
|---|---|---|
| **RFC** | Before large or controversial work | Discussion → accept/reject |
| **ADR** | After decision is made | Permanent record |

Workflow: RFC proposed → discussed → accepted → ADR created → implementation PRs.
