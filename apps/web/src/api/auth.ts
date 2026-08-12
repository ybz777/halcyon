import { apiGet, apiPost } from './client'

import { clearOrganizationId, clearServiceAccountId } from './orgContext'

const TOKEN_KEY = 'halcyon_access_token'

export function getStoredToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function setStoredToken(token: string): void {
  sessionStorage.setItem(TOKEN_KEY, token)
}

export function clearStoredToken(): void {
  sessionStorage.removeItem(TOKEN_KEY)
  clearOrganizationId()
  clearServiceAccountId()
}

export interface LoginResponse {
  accessToken: string
  refreshToken?: string
  expiresIn?: number
  tokenType?: string
}

export interface MeResponse {
  id: string
  email: string
  roles: string[]
  permissions: string[]
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  return apiPost<LoginResponse>('/api/v1/auth/login', { email, password })
}

export async function getMe(): Promise<MeResponse> {
  return apiGet<MeResponse>('/api/v1/auth/me')
}
