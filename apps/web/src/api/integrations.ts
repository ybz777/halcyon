import { apiGet, apiPost, withQuery } from './client'
import type { PaginatedResponse } from './types'

export interface WebhookEndpoint {
  id: string
  organization_id: string
  url: string
  description: string
  status: string
  subscribed_events: string[]
  secret_prefix: string
  failure_count: number
  created_at: string
}

export interface CreateWebhookInput {
  url: string
  description?: string
  subscribed_events: string[]
}

export interface CreateWebhookResult extends WebhookEndpoint {
  secret: string
}

export async function listWebhookEndpoints(
  cursor?: string,
  pageSize = 50,
): Promise<PaginatedResponse<WebhookEndpoint>> {
  const path = withQuery('/api/v1/webhook-endpoints', { page_size: pageSize, cursor })
  return apiGet<PaginatedResponse<WebhookEndpoint>>(path, { org: true, requireOrg: true })
}

export async function createWebhookEndpoint(
  input: CreateWebhookInput,
): Promise<CreateWebhookResult> {
  return apiPost<CreateWebhookResult>('/api/v1/webhook-endpoints', input, {
    org: true,
    requireOrg: true,
    idempotent: true,
  })
}
