# Halcyon Event Catalog

## Why This Document Exists

A single registry of all Halcyon domain events — who produces them, who consumes
them, and what they mean. Prevents duplicate event types and orphaned consumers.

## Who Reads This

| Audience | Purpose |
|---|---|
| Event producers | Know what to publish |
| Event consumers | Know what to subscribe to |
| Reviewers | Validate new events against existing catalog |

## Standards for Every PR

- New events must be added to this catalog in the **same PR** as the schema in `contracts/events/`.
- Include producer, consumers, topic, and description for every event.
- Deprecated events are marked but not removed until all consumers migrate.

---

## Catalog

### Auth Domain

Topic: `halcyon.auth`

| Event Type | Version | Producer | Consumers | Description |
|---|---|---|---|---|
| `auth.user.registered` | 1.0 | auth | organization, audit | New user account created |
| `auth.user.logged_in` | 1.0 | auth | audit | User authenticated successfully |
| `auth.api_key.created` | 1.0 | auth | audit | API key created |
| `auth.api_key.revoked` | 1.0 | auth | audit | API key revoked |
| `auth.api_key.rotated` | 1.0 | auth | audit | API key rotated |
| `auth.user.email_changed` | 1.0 | auth | audit | User email updated |
| `auth.user.disabled` | 1.0 | auth | audit, notification | User account deactivated |
| `auth.user.enabled` | 1.0 | auth | audit | User account reactivated |

### Organization Domain

Topic: `halcyon.organizations`

| Event Type | Version | Producer | Consumers | Description |
|---|---|---|---|---|
| `organizations.organization.created` | 1.0 | organization | audit, workflow | Organization created |
| `organizations.organization.updated` | 1.0 | organization | audit | Organization updated |
| `organizations.organization.deleted` | 1.0 | organization | audit | Organization soft-deleted |
| `organizations.membership.created` | 1.0 | organization | audit | Membership created |
| `organizations.membership.role_changed` | 1.0 | organization | audit | Member role changed |
| `organizations.membership.suspended` | 1.0 | organization | audit | Membership suspended |
| `organizations.membership.removed` | 1.0 | organization | audit | Membership removed |
| `organizations.invitation.created` | 1.0 | organization | audit | Invitation created |
| `organizations.invitation.accepted` | 1.0 | organization | audit | Invitation accepted |
| `organizations.invitation.revoked` | 1.0 | organization | audit | Invitation revoked |
| `organizations.member.invited` | 1.0 | organization | audit | Member invited (legacy alias) |
| `organizations.member.joined` | 1.0 | organization | audit | Member joined |

### Signals Domain

Topic: `halcyon.signals`

| Event Type | Version | Producer | Consumers | Description |
|---|---|---|---|---|
| *Pending Phase 2* | | | | |

### Incidents Domain

Topic: `halcyon.incidents`

| Event Type | Version | Producer | Consumers | Description |
|---|---|---|---|---|
| *Pending Phase 3* | | | | |

### AI Domain

Topic: `halcyon.ai`

| Event Type | Version | Producer | Consumers | Description |
|---|---|---|---|---|
| *Pending Phase 4* | | | | |

---

## Envelope

All events use the standard envelope defined in `contracts/events/envelope/v1/schema.json`.

See [EVENT_GUIDELINES.md](../../EVENT_GUIDELINES.md).

---

## Related Documents

- [Event-Driven Architecture](../architecture/event-driven.md)
- [EVENT_GUIDELINES.md](../../EVENT_GUIDELINES.md)
- [contracts/events/README.md](../../contracts/events/README.md)
