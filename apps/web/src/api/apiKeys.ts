import { apiDelete, apiGet, apiPost, withQuery } from './client'
import type { PaginatedResponse } from './types'

export interface ApiKey {
  id: string
  keyId: string
  serviceAccountId: string
  name: string
  prefix: string
  scopes: string[]
  status: string
  createdAt: string
  expiresAt?: string
  lastUsedAt?: string
  revokedAt?: string
  createdBy: string
}

export interface CreateApiKeyInput {
  name: string
  serviceAccountId: string
  organizationId?: string
  scopes: string[]
  environment: string
  expiresAt?: string
}

export interface CreateApiKeyResponse {
  apiKey: ApiKey
  plaintext: string
}

export async function listApiKeys(
  serviceAccountId: string,
  cursor?: string,
  pageSize = 50,
): Promise<PaginatedResponse<ApiKey>> {
  const path = withQuery('/api/v1/auth/api-keys', {
    serviceAccountId,
    page_size: pageSize,
    cursor,
  })
  return apiGet<PaginatedResponse<ApiKey>>(path)
}

export async function getApiKey(id: string): Promise<ApiKey> {
  return apiGet<ApiKey>(`/api/v1/auth/api-keys/${encodeURIComponent(id)}`)
}

export async function createApiKey(input: CreateApiKeyInput): Promise<CreateApiKeyResponse> {
  return apiPost<CreateApiKeyResponse>('/api/v1/auth/api-keys', input, { idempotent: true })
}

export async function revokeApiKey(id: string): Promise<void> {
  return apiDelete(`/api/v1/auth/api-keys/${encodeURIComponent(id)}`)
}

export async function rotateApiKey(id: string, environment?: string): Promise<CreateApiKeyResponse> {
  const path = withQuery(`/api/v1/auth/api-keys/${encodeURIComponent(id)}/rotate`, {
    environment,
  })
  return apiPost<CreateApiKeyResponse>(path, undefined, { idempotent: true })
}
