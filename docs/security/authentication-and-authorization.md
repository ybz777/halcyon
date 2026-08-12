# Authentication and Authorization

Halcyon separates **authentication** (who is calling) from **authorization** (what they may do). Handlers must not parse JWTs or implement ad-hoc permission checks; middleware and centralized authorizers enforce boundaries.

## Identity Model

`pkg/platform/auth.Identity` is the normalized caller attached to request context after authentication:

| Field | Description |
|---|---|
| `SubjectID` | User ID, API key ID, or service ID |
| `PrincipalType` | `user`, `service`, or `system` |
| `OrganizationID` | Tenant scope when resolved |
| `Roles` | Role names when present in token |
| `Permissions` | Explicit permission strings |
| `Method` | `bearer_jwt`, `api_key`, `service_token`, or `session` |

Service-specific principals (for example `organization/internal/application/auth.Principal`) are built **after** authentication from trusted membership data — never from client-supplied role fields.

## Authentication

### Bearer JWT (users)

- Signed with **HS256** using `JWT_SECRET` from centralized configuration (`pkg/platform/config`).
- Validated by `pkg/platform/auth.HS256Validator` with pinned algorithm, issuer (`JWT_ISSUER`), optional audience (`JWT_AUDIENCE`), expiration, and subject checks.
- Rejects `alg=none`, unexpected algorithms, expired tokens, and malformed tokens.
- Returns **HTTP 401** with Problem Details; error messages do not reveal whether a subject exists.

### API keys

- Issued and stored only in `services/auth` (Argon2 hashed, scoped, revocable).
- Browser clients never send raw API keys to internal services directly.
- The gateway validates API keys and forwards scoped identity headers.
- Downstream services require a valid `X-Halcyon-Service-Token` before trusting `X-Halcyon-API-Key-*` headers.

### Service-to-service tokens

- Gateway signs short-lived RS256 tokens (`pkg/platform/servicetoken`) with `SERVICE_PRIVATE_KEY`.
- Downstream services validate with `SERVICE_TOKEN_PUBLIC_KEY` (or derived public key in local dev), `SERVICE_TOKEN_ISSUER`, and `SERVICE_TOKEN_AUDIENCE`.
- `pkg/platform/auth.GatewayTrust` middleware rejects spoofed forwarded identity headers.

### Public routes

Explicitly public endpoints include health/readiness, login/register, and documentation. Unlisted routes default to authenticated.

## Authorization

### Organization RBAC

`services/organization` maps roles to permissions centrally:

| Role | Capabilities |
|---|---|
| `owner` | Full organization control including delete |
| `admin` | Manage members, settings; cannot delete org |
| `member` | Standard read/write within org |
| `viewer` | Read-only |

`organization.Authorizer` enforces `RequirePermission` and `RequireRole`. Membership resolution verifies active organization membership; revoked members lose access.

### Default deny

New protected routes must declare required permissions via middleware (`RequirePermission`) or use-case authorizer checks. Authorization fails closed.

### Object-level access

- Cross-tenant access returns `ErrCrossTenantAccess` / HTTP 403.
- Unknown or unauthorized organizations return forbidden responses without confirming resource existence where appropriate.
- Handlers use explicit request DTOs; sensitive fields (`role`, `organization_id`, `owner_id`) are not mass-assignable from JSON.

## Tenant Isolation

Authorization complements — does not replace — database tenant filtering (PR #24). Every organization-scoped query includes tenant predicates. Idempotency scope (PR #25) includes principal and organization ID so users cannot replay another tenant's results.

## Background Jobs

Job payloads store stable principal and organization identifiers, not full JWTs. Workers verify the job was created by an authorized operation and enforce tenant scope at execution time (PR #26).

## Observability

Safe tracing/log attributes: principal type, auth method, route, permission, authorization result.

Never log: raw JWT, API key, `Authorization` header, passwords, or full identity tokens.

Metrics (`auth_attempts_total`, `auth_failures_total`, `authorization_denied_total`) use low-cardinality labels only.

## Configuration

All auth settings load through `pkg/platform/config`:

| Variable | Purpose |
|---|---|
| `JWT_SECRET` | HS256 signing/validation secret |
| `JWT_ISSUER` | Expected token issuer |
| `JWT_AUDIENCE` | Expected audience (optional) |
| `JWT_ACCESS_TTL` / `JWT_REFRESH_TTL` | Token lifetimes |
| `SERVICE_PRIVATE_KEY` | Gateway signing key |
| `SERVICE_TOKEN_PUBLIC_KEY` | Downstream validation key |
| `SERVICE_TOKEN_ISSUER` | Service token issuer (default `gateway`) |
| `SERVICE_TOKEN_AUDIENCE` | Internal audience (default `halcyon-internal`) |

## Token Revocation

- **Access tokens**: short-lived; no global blacklist.
- **Refresh tokens / sessions**: revocable in auth service storage.
- **API keys**: disabled/revoked state checked on every use.

## Threat Model Summary

| Threat | Mitigation | Limitation |
|---|---|---|
| Credential theft | TLS, HttpOnly cookies, secret redaction | Client-side malware out of scope |
| Token replay | Short TTL, scoped idempotency keys | Stolen token valid until expiry |
| IDOR | Object-level authz + tenant queries | Requires consistent handler coverage |
| Tenant escape | RBAC + DB scoping + idempotency scope | New endpoints must opt in |
| Privilege escalation | Central RBAC; DTOs exclude role fields | Review required for new update endpoints |
| Mass assignment | Explicit request models | Legacy endpoints need audit |
| API key leakage | Hash-at-rest, gateway-only forwarding | Operator logging discipline |
| Service credential leakage | RS256, short TTL, not exposed to browsers | Key rotation manual today |

See also [threat-model.md](./threat-model.md).
