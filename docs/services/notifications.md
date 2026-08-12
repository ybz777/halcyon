# Notifications Service

Event-driven in-app and email notifications for Halcyon users.

## Architecture

```
Domain Event → Kafka → notification-dispatcher → notifications + email_deliveries
                                                    ↓
                                              email-worker → EmailSender (log/sink in dev)
```

Business services never send email directly.

## Event Mapping

| Domain Event | Notification Type |
|---|---|
| `workflow.execution.succeeded` | `workflow.execution.completed` |
| `workflow.execution.failed` | `workflow.execution.failed` |
| `organizations.membership.role_changed` | `organization.role_changed` |
| `auth.api_key.created` | `security.api_key_created` |
| `auth.api_key.revoked` | `security.api_key_revoked` |
| `auth.user.logged_in` | `security.login_alert` |

## Recipient Resolution

Dedicated resolver maps events to affected users (actor, subject, or payload user ID). Administrators are not spammed by default.

## Preferences

- `GET /api/v1/notification-preferences`
- `PUT /api/v1/notification-preferences/{notification_type}`

Explicit preference overrides defaults. Security email notifications cannot be disabled.

## Email Delivery

- Async worker with leases, retries, exponential backoff
- Default provider: `log` (safe local sink)
- Configure: `EMAIL_PROVIDER`, `EMAIL_FROM_ADDRESS`, `EMAIL_FROM_NAME`

## Idempotency

- Notifications: unique `(event_id, user_id, notification_type)`
- Kafka: `processed_events` per dispatcher consumer group

## Retention

`NOTIFICATION_RETENTION_DAYS` (default 90). Cleanup via maintenance job (future).

## Security

Never include API keys, JWTs, or passwords in email bodies. Tenant boundary: `organization_id + user_id`.

## Observability

Metrics: `notifications_created_total`, `email_deliveries_total`, `email_delivery_retries_total`, `email_queue_depth`.

## Future Channels

Domain supports `IN_APP` and `EMAIL`. `PUSH`, `SMS`, and `WEBHOOK` can be added via channel abstraction without rewriting core logic.
