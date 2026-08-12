# Halcyon

[![CI](https://github.com/halcyon-io/halcyon/actions/workflows/ci.yml/badge.svg)](https://github.com/halcyon-io/halcyon/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?logo=go&logoColor=white)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

**Engineering Operations Intelligence Platform**

> *Operational calm, engineered.*

---

## Why This Document Exists

This is the front door of the repository. It orients new contributors, recruiters,
and customers to what Halcyon is, how the project is organized, and where to go next.

## Who Reads This

| Audience | What they need from this file |
|---|---|
| New engineers | Clone, build, run locally in under 15 minutes |
| Open-source contributors | Contribution path and standards |
| Recruiters / hiring managers | Scope, maturity signals, architecture pointers |
| Customers / evaluators | Product positioning and reliability posture |

## Standards for Every PR

- Update this README **only** when repository entry points change (new services,
  new prerequisites, broken quickstart steps).
- Do not add feature-level detail here — link to `SYSTEM_DESIGN.md` and service READMEs.
- Keep the quickstart verified and copy-pasteable.

---

## What Is Halcyon?

Halcyon is an **enterprise engineering operations platform** for managing organizations,
users, service identities, API credentials, workflows, integrations, events, and operational
activity. It provides a unified console and API surface for multi-tenant engineering teams.

The long-term vision is an **Engineering Operations Intelligence Platform** — unifying changes,
signals, incidents, and institutional knowledge into actionable operational briefs. The current
repository ships the operational foundation: authentication, tenancy, API keys, audit, integrations,
notifications, jobs, and a React console.

### Architecture

```
React / TypeScript (apps/web)
        ↓
    API Gateway (:8088)
        ↓
 ┌──────┼──────────────┬─────────────┐
 │      │              │             │
Auth   Organization   Audit    Integrations
 │      │              │             │
 └──────┼──────────────┴─────────────┘
        ↓
 PostgreSQL (per service)
        +
 Redis / Redpanda
        ↓
 Workers / Outbox relay
```

### Tech stack

- **Backend:** Go 1.25+, Chi, PostgreSQL, Redis, Redpanda (Kafka-compatible)
- **Frontend:** React 19, TypeScript, Vite (`apps/web`)
- **Infra:** Docker Compose (`infra/docker/compose.yml`)

### Features (implemented)

| Area | Capabilities |
|---|---|
| Authentication | Register, login, JWT, refresh, `/me`, JWKS |
| Authorization | Role-based permissions (`apikey:manage`, etc.) via gateway |
| Organizations | Create, members, invitations |
| API keys | Create, list, revoke, rotate (service-account scoped) |
| Integrations | Webhook endpoints |
| Audit | Organization-scoped audit event log |
| Notifications | Per-user notification preferences |
| Jobs / Workflow | Backend services (no console UI yet) |

---

## Local development

### 1. Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker Desktop](https://www.docker.com/) with Compose v2
- [Node.js 20+](https://nodejs.org/) for the web console

### 2. Configure environment

```powershell
cp .env.example .env
```

Generate an RSA key for gateway service tokens and set `SERVICE_PRIVATE_KEY` in `.env`:

```powershell
openssl genrsa 2048
```

Paste the PEM into `.env` (multiline values are supported). **Never commit `.env`** — it is gitignored.

### 3. Start the stack

```powershell
docker compose --env-file .env -f .\infra\docker\compose.yml up -d
```

Wait until gateway, auth, organization, postgres, and redis are healthy:

```powershell
curl http://localhost:8088/api/v1/health
```

### 4. Web console

```powershell
cd apps/web
cp .env.example .env.local
npm install
npm run dev
```

Open http://localhost:5173

### 5. Demo mode (portfolio)

Seed a demo identity, organization, service accounts, and sample API keys:

```powershell
.\scripts\demo-bootstrap.ps1
```

This writes `apps/web/.env.local` with `VITE_DEMO_MODE=true`. Restart `npm run dev` — the console
auto-signs in as `demo@halcyon.demo` using real JWT authentication and RBAC (admin role with
`apikey:manage`). No auth bypass.

### 6. Production frontend build

```powershell
cd apps/web
npm run build
```

Set `VITE_API_BASE_URL` at build time when the API is on another origin. Do not put secrets in
`VITE_*` variables — they are embedded in the client bundle.

### 7. Tests

```powershell
make test
# or per service:
cd services/gateway && go test ./...
cd services/auth && go test ./...
```

---

## Authentication

- Users obtain JWT access tokens via `POST /api/v1/auth/login`.
- The web console stores the token in `sessionStorage` under `halcyon_access_token`.
- All API requests send `Authorization: Bearer <token>`.
- Organization-scoped routes also send `X-Organization-ID`.
- The gateway validates JWTs and preserves `roles` / `permissions` claims for `RequirePermission()` checks.

---

## Project structure

```
halcyon/
├── apps/web/             # React/TypeScript operations console
├── contracts/            # OpenAPI + event schemas
├── infra/docker/         # Docker Compose for local development
├── pkg/platform/         # Shared Go libraries
├── scripts/              # Demo bootstrap and utilities
├── services/             # Microservices (auth, gateway, organization, …)
└── tests/                # Integration and E2E tests
```

---

## Quick Start (Makefile)

```bash
git clone https://github.com/halcyon-io/halcyon.git
cd halcyon
cp .env.example .env
# Set SERVICE_PRIVATE_KEY in .env before starting gateway

make up           # Start Postgres + Redis + Redpanda + services
make test         # Run Go unit tests
```

Verify:

```bash
curl http://localhost:8088/api/v1/health
curl http://localhost:8080/api/v1/health
```

Full guide: [docs/onboarding/local-development.md](docs/onboarding/local-development.md)

---

## Services

| Service | Directory | Port | Description |
|---|---|---|---|
| Auth | `services/auth/` | 8080 | Identity, JWT, API keys |
| Organization | `services/organization/` | 8081 | Tenancy, members, invitations |
| Workflow | `services/workflow/` | 8082 | Workflow execution |
| Audit | `services/audit/` | 8083 | Immutable audit log |
| Integrations | `services/integrations/` | 8084 | Webhook endpoints |
| Notifications | `services/notifications/` | 8085 | Notification preferences |
| Jobs | `services/jobs/` | 8086 | Background jobs API |
| Gateway | `services/gateway/` | 8088 | Edge routing and JWT enforcement |
| Web console | `apps/web/` | 5173 | React operations UI |

---

## Repository Map

```
halcyon/
├── apps/web/             # Web console (React + Vite)
├── contracts/            # OpenAPI + event schemas (source of truth)
├── deploy/               # Helm charts and environment values
├── docs/                 # ADRs, RFCs, onboarding, deployment, operations
├── infra/docker/         # Docker Compose for local development
├── pkg/platform/         # Shared Go platform libraries
├── services/             # Backend microservices (bounded contexts)
├── tests/                # E2E, integration, and load tests
└── tools/                # Codegen, migration helpers, ops CLIs
```

### Engineering Constitution

| Document | Audience | Purpose |
|---|---|---|
| [ARCHITECTURE.md](ARCHITECTURE.md) | All engineers | System structure and dependency rules |
| [SYSTEM_DESIGN.md](SYSTEM_DESIGN.md) | Product + engineering | Platform design overview |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributors | Contribution workflow and PR standards |
| [CODE_STYLE.md](CODE_STYLE.md) | Implementers | Language and code conventions |
| [SECURITY.md](SECURITY.md) | Everyone | Security policy and secure development |
| [TESTING.md](TESTING.md) | Implementers + reviewers | Testing pyramid and coverage gates |
| [OBSERVABILITY.md](OBSERVABILITY.md) | SRE + service owners | Logging, metrics, tracing standards |
| [API_GUIDELINES.md](API_GUIDELINES.md) | API authors | REST design and OpenAPI workflow |
| [EVENT_GUIDELINES.md](EVENT_GUIDELINES.md) | Event producers/consumers | Kafka patterns and schema rules |
| [DATABASE_GUIDELINES.md](DATABASE_GUIDELINES.md) | Service engineers | Schema, migrations, multi-tenancy |
| [PERFORMANCE_GUIDELINES.md](PERFORMANCE_GUIDELINES.md) | All engineers | Latency budgets and anti-patterns |
| [docs/deployment.md](docs/deployment.md) | Release managers | Build, deploy, migrate, rollback |
| [docs/operations.md](docs/operations.md) | SRE / on-call | Runbook and incident debugging |
| [docs/architecture.md](docs/architecture.md) | All engineers | Implemented system components and data flow |
| [docs/production-readiness.md](docs/production-readiness.md) | Release managers | Production checklist and known limitations |
| [docs/performance.md](docs/performance.md) | Performance | Benchmarks and resource limits |
| [ROADMAP.md](ROADMAP.md) | Everyone | Delivery phases and priorities |
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community | Expected behavior |
| [LICENSE](LICENSE) | Legal / adopters | Apache 2.0 terms |

### Deep References

| Path | Purpose |
|---|---|
| [docs/adr/](docs/adr/) | Architecture Decision Records |
| [docs/rfc/](docs/rfc/) | Request for Comments (proposals) |
| [docs/services/](docs/services/) | Per-service architecture pages |
| [docs/security/threat-model.md](docs/security/threat-model.md) | STRIDE threat analysis |
| [docs/security/rate-limiting.md](docs/security/rate-limiting.md) | Distributed rate limits and abuse protection |
| [contracts/](contracts/) | OpenAPI and event schemas (source of truth) |

---

## Prerequisites

- [Go 1.25+](https://go.dev/dl/)
- [Docker](https://www.docker.com/) with Compose v2
- [Node.js 20+](https://nodejs.org/) (for `apps/web`)
- `make` (recommended)

See **Local development** above for the full startup flow.

---

## Engineering Principles

1. **Production quality only** — no tutorial code paths in `main`
2. **Small, reviewable PRs** — one capability per change
3. **Clean Architecture** — dependencies point inward
4. **Schema-first APIs and events** — `contracts/` is the source of truth
5. **Observable by default** — logs, metrics, traces from day one
6. **Tenant isolation everywhere** — `org_id` at every layer
7. **Document decisions** — ADRs for non-trivial choices

---

## Community

- [Contributing Guide](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)
- [RFC Process](docs/rfc/README.md)
- [Architecture Decision Records](docs/adr/README.md)

---

## License

Apache License 2.0 — see [LICENSE](LICENSE).
