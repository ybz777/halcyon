# Multi-Tenancy and RBAC

Halcyon uses a shared-database, shared-schema multi-tenant model. Organizations are isolated logically through membership, RBAC, and tenant-scoped repository queries.

## Tenant Model

- Every organization-owned resource is scoped by `organization_id`.
- Users may belong to multiple organizations with independent roles.
- API keys and service accounts belong to exactly one organization.
- Client-supplied `organization_id` values are never trusted without membership validation.

## Authentication vs Authorization

| Layer | Responsibility |
|-------|----------------|
| Authentication | Validates JWT or API key identity |
| Organization context | Resolves membership and role for `{organization_id}` routes |
| Authorization | Central `Authorizer` checks permissions from role matrix |
| Domain | Enforces invariants (last owner, invitation lifecycle) |

## Role Matrix

| Role | Scope |
|------|-------|
| owner | Full organization access |
| admin | Administrative access without org delete |
| member | Read + incident create/update |
| viewer | Read-only |

Permissions are defined in `services/organization/internal/domain/role.go` and seeded in migration `000002_rbac_multitenancy`.

## Invitation Lifecycle

1. Admin invites by email → raw token returned once, hash stored
2. Invitee accepts with JWT matching email → membership created atomically
3. Revoked/expired/accepted invitations cannot be replayed

## Security Invariants

- Cross-tenant access returns 403 without leaking foreign org existence
- Last owner cannot be removed or demoted without transfer
- Deleted/suspended organizations reject normal operations
- Outbox events never contain tokens, passwords, or API secrets

## Related

- [Event-Driven Architecture](event-driven.md)
- [Organization Service](../services/organization.md)
