# Organization Service

## Purpose

The organization service owns **multi-tenant identity, membership, invitations, and RBAC** for Halcyon. It manages organizations, memberships, role-based permissions, and invitation workflows with transactional outbox events.

## Responsibilities

| In scope | Out of scope |
|---|---|
| Organization CRUD (soft delete) | User credentials (auth service) |
| Membership lifecycle | JWT issuance (auth service) |
| Email invitations | Incident/workflow execution |
| Centralized RBAC | Direct Kafka publishing from handlers |

## Multi-Tenancy

- Shared PostgreSQL schema; tenant isolation via `organization_id` on every query.
- URL path `{organization_id}` is validated against authenticated membership.
- Client-supplied organization IDs in request bodies are never authoritative.
- See [Multi-Tenancy Architecture](../architecture/multi-tenancy.md).

## API Endpoints

| Method | Path | Auth | Permission |
|---|---|---|---|
| `POST` | `/api/v1/organizations` | JWT | (creates org + owner membership) |
| `GET` | `/api/v1/organizations/{organization_id}` | JWT | `organization.read` |
| `PATCH` | `/api/v1/organizations/{organization_id}` | JWT | `organization.update` |
| `DELETE` | `/api/v1/organizations/{organization_id}` | JWT | `organization.delete` |
| `GET` | `/api/v1/organizations/{organization_id}/members` | JWT | `members.read` |
| `PATCH` | `/api/v1/organizations/{organization_id}/members/{user_id}` | JWT | `members.update` |
| `DELETE` | `/api/v1/organizations/{organization_id}/members/{user_id}` | JWT | `members.remove` |
| `POST` | `/api/v1/organizations/{organization_id}/invitations` | JWT | `members.invite` |
| `GET` | `/api/v1/organizations/{organization_id}/invitations` | JWT | `members.read` |
| `DELETE` | `/api/v1/organizations/{organization_id}/invitations/{id}` | JWT | `members.invite` |
| `POST` | `/api/v1/invitations/{token}/accept` | JWT | (email must match invitation) |

OpenAPI: `contracts/openapi/organization/v1/openapi.yaml`

## RBAC Roles

`owner`, `admin`, `member`, `viewer`

Permission matrix is centralized in `services/organization/internal/domain/role.go` and seeded in migration `000002_rbac_multitenancy.up.sql`.

## Authentication vs Authorization

1. **Authenticate** middleware validates JWT bearer tokens.
2. **OrganizationContext** middleware resolves membership for `{organization_id}` routes.
3. **Authorizer** enforces permissions via `RequirePermission`.
4. Domain layer enforces invariants (last owner, invitation lifecycle).

API keys authenticate via the auth service; service account principals intersect API key scopes with organization permissions.

## Domain Events

Events are written to the transactional outbox (`organization.created`, `membership.role_changed`, `invitation.accepted`, etc.). See [Event Catalog](../events/catalog.md).

## Local Development

```bash
make run-organization   # Port 8081
make migrate-org        # Requires ORGANIZATION_DATABASE_URL
```

## Database

PostgreSQL database: `halcyon_organization` (see `.env.example`).

Migrations: `services/organization/migrations/`

Tables: `organizations`, `memberships`, `invitations`, `roles`, `permissions`, `role_permissions`, `outbox`

## Metrics

- `organizations_created_total`
- `memberships_created_total`
- `memberships_removed_total`
- `invitations_created_total`
- `invitations_accepted_total`
- `authorization_denied_total`
- `cross_tenant_access_denied_total`
- `role_changes_total`

## Security Invariants

- Cross-tenant access returns 403 without leaking foreign organization existence.
- Last owner cannot be removed or demoted without ownership transfer.
- Invitation tokens are hashed at rest; raw tokens returned only at creation.
- Deleted/suspended organizations and memberships reject normal operations.
