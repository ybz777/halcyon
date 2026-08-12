# Integrations Service

Halcyon external integrations: API keys (via auth service) and outbound webhooks.

## API Keys

- Format: `hal_live_<entropy>` / `hal_test_<entropy>`
- Hashing: Argon2id via auth service (`services/auth`)
- Authentication: `Authorization: Bearer <api-key>` or `X-API-Key`
- Gateway validates keys through `POST /api/v1/auth/api-keys/introspect`
- Plaintext shown once at create/rotate only

### Scopes

| Scope | Capability |
|---|---|
| `workflows.read` | Read workflows |
| `workflows.execute` | Execute workflows |
| `audit.read` | Read audit logs |
| `integrations.read` | Manage webhook metadata |
| `integrations.write` | Create/update webhooks |

Effective permissions = organization RBAC ∩ API key scopes.

## Webhooks

### Endpoint lifecycle

1. Admin creates endpoint with HTTPS URL and explicit event subscriptions
2. Signing secret (`whsec_...`) returned once
3. Domain events flow: Kafka → webhook dispatcher → delivery rows
4. Webhook worker delivers signed POST requests asynchronously

### Signature protocol

Headers:
- `X-Webhook-Timestamp` — Unix seconds
- `X-Webhook-Signature` — `v1=<hex>` HMAC-SHA256 of `timestamp + "." + raw_payload`
- `X-Webhook-Event-ID`
- `X-Webhook-Event-Type`

Replay protection: reject timestamps outside ±5 minutes; deduplicate on `event_id`.

### Retry policy

- Retry: 429, 5xx, network errors
- No retry: most 4xx
- Exponential backoff with jitter, max 6 attempts (configurable)
- Respect `Retry-After` when present (bounded)

### SSRF defense

- HTTPS only in production (`WEBHOOK_ALLOW_HTTP=true` for local dev)
- Blocks localhost, RFC1918, link-local, metadata IPs
- DNS resolution checked before connect
- Redirects disabled

### Failure handling

- Delivery status tracked in `webhook_deliveries`
- Endpoints auto-disabled after `WEBHOOK_FAILURE_THRESHOLD` consecutive failures
- Re-enable via `POST /webhook-endpoints/{id}/enable`

## Observability

Bounded Prometheus metrics: `webhook_deliveries_total`, `webhook_delivery_failures_total`, `webhook_delivery_retries_total`, `webhook_delivery_latency_seconds`, `webhook_ssrf_rejections_total`, `webhook_queue_depth`.

## Security

Never log or return: API key plaintext, webhook secrets, Authorization headers, full request/response bodies.
