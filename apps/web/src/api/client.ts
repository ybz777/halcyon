import { getStoredToken } from './auth'
import { getOrganizationId } from './orgContext'
import type { FieldError, ProblemDetails } from './types'

export class ApiError extends Error {
  readonly status: number
  readonly fieldErrors?: FieldError[]

  constructor(message: string, status: number, fieldErrors?: FieldError[]) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.fieldErrors = fieldErrors
  }
}

export interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown
  /** Attach X-Halcyon-Organization-ID when an org is selected */
  org?: boolean
  /** Require org header — fail fast if none selected */
  requireOrg?: boolean
  /** Send Idempotency-Key header (for supported mutations) */
  idempotent?: boolean
}

function newIdempotencyKey(): string {
  return crypto.randomUUID()
}

async function parseProblem(res: Response): Promise<ApiError> {
  try {
    const body = (await res.json()) as ProblemDetails
    const message = body.detail || body.title || `Request failed (${res.status})`
    return new ApiError(message, res.status, body.errors)
  } catch {
    return new ApiError(`Request failed (${res.status})`, res.status)
  }
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, org = false, requireOrg = false, headers: extraHeaders, ...rest } = options

  const headers = new Headers(extraHeaders)
  if (body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const token = getStoredToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }

  if (org || requireOrg) {
    const orgId = getOrganizationId()
    if (!orgId && requireOrg) {
      throw new ApiError('Select an organization to use this feature.', 0)
    }
    if (orgId) {
      // Use X-Organization-ID for direct service access. X-Halcyon-Organization-ID is
      // treated as gateway-forwarded identity and requires a gateway service token on
      // audit/notifications when those services are reached without gateway attestation.
      headers.set('X-Organization-ID', orgId)
    }
  }

  const base = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, '') ?? ''
  const url = `${base}${path}`

  const res = await fetch(url, {
    ...rest,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  if (!res.ok) {
    throw await parseProblem(res)
  }

  if (res.status === 204) {
    return undefined as T
  }

  const text = await res.text()
  if (!text) {
    return undefined as T
  }
  return JSON.parse(text) as T
}

export function apiGet<T>(path: string, options?: RequestOptions): Promise<T> {
  return apiRequest<T>(path, { ...options, method: 'GET' })
}

export function apiPost<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
  const headers = new Headers(options?.headers)
  if (options?.idempotent) {
    headers.set('Idempotency-Key', newIdempotencyKey())
  }
  return apiRequest<T>(path, {
    ...options,
    method: 'POST',
    body,
    headers,
  })
}

export function apiPut<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
  return apiRequest<T>(path, { ...options, method: 'PUT', body })
}

export function apiPatch<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
  return apiRequest<T>(path, { ...options, method: 'PATCH', body })
}

export function apiDelete(path: string, options?: RequestOptions): Promise<void> {
  return apiRequest<void>(path, { ...options, method: 'DELETE' })
}

export function withQuery(path: string, params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') {
      search.set(key, String(value))
    }
  }
  const qs = search.toString()
  return qs ? `${path}?${qs}` : path
}
