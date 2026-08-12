# Security Policy

## Why This Document Exists

Halcyon handles authentication, tenant data, and operational intelligence.
A public security policy defines how vulnerabilities are reported, handled, and
disclosed. Enterprise adopters and mature OSS projects require this file.

## Who Reads This

| Audience | Purpose |
|---|---|
| Security researchers | Responsible disclosure process |
| Enterprise customers | Security posture evaluation |
| Engineers | Secure development requirements |
| Compliance auditors | Vulnerability management evidence |

## Standards for Every PR

- Security-sensitive changes require explicit review from a code owner listed in
  `.github/CODEOWNERS` under `@halcyon-io/security`.
- Never commit secrets, credentials, or real customer data.
- Security regressions block merge regardless of feature priority.
- Breaking security fixes may bypass normal release cadence — see [RELEASE_PROCESS.md](RELEASE_PROCESS.md).

---

## Supported Versions

| Version | Supported |
|---|---|
| `main` branch | Yes |
| Latest release tag | Yes |
| Older releases | Security fixes at maintainer discretion |

---

## Reporting a Vulnerability

**Do not open public GitHub issues for security vulnerabilities.**

Email: **security@halcyon.io**

Include:

- Description of the vulnerability
- Steps to reproduce
- Impact assessment
- Affected components (service name, endpoint, version)

### Response Timeline

| Stage | Target |
|---|---|
| Acknowledgment | 48 hours |
| Initial assessment | 5 business days |
| Fix or mitigation plan | 15 business days (severity-dependent) |
| Coordinated disclosure | After fix is deployed |

We follow coordinated disclosure. Researchers who report valid vulnerabilities
in good faith will be credited in the release notes (unless anonymity is requested).

---

## Security Requirements

### Authentication and Authorization

- JWT access tokens: short TTL (15 minutes default)
- Refresh token rotation with reuse detection
- RBAC enforced at the service layer — gateway auth is necessary but not sufficient
- API keys: scoped, hashed at rest, audited on use
- MFA for sensitive operations (step-up auth)

### Multi-Tenancy

- `org_id` required on every data access path
- PostgreSQL Row-Level Security on tenant tables
- Cross-tenant access is a **Severity-1** incident

### Secrets Management

- No secrets in source code, Docker images, or committed env files
- AWS Secrets Manager in production
- GitHub OIDC for CI — no long-lived cloud credentials

### Data Protection

- TLS 1.3 in transit
- AES-256 at rest (RDS, S3)
- PII minimization in logs and traces
- Customer data never used for model training

### Audit

- Immutable audit log for security-relevant actions
- Hash-chained entries for tamper detection
- Audit log access is itself audited

### Dependencies

- Automated vulnerability scanning in CI (`govulncheck`)
- Secret scanning in CI (`gitleaks`)
- Container image scanning on `main` (`trivy`)
- Dependabot weekly updates for Go modules and GitHub Actions
- **Policy:** CI fails on high-confidence critical/high vulnerabilities reported by `govulncheck` and unfixed CRITICAL/HIGH container CVEs on `main`
- Pin base image versions

### API Security

- Rate limiting at gateway
- HMAC validation on inbound webhooks
- Input validation on all endpoints
- RFC 7807 errors — no stack traces in production responses

---

## Threat Model Summary

See [docs/security/threat-model.md](docs/security/threat-model.md) for the full model.

Primary threats:

1. Cross-tenant data leakage
2. Stolen JWT or API key abuse
3. Webhook spoofing
4. Prompt injection via incident content
5. Privilege escalation via RBAC bugs
6. Audit log tampering

---

## Security Review Triggers

The following changes require explicit security review:

- Authentication or authorization logic
- Cryptography (JWT, hashing, encryption)
- Multi-tenant isolation boundaries
- Webhook ingestion and signature validation
- AI prompt handling and output guardrails
- Audit log implementation
- Infrastructure IAM or network policies
