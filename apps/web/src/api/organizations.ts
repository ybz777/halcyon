import { apiDelete, apiGet, apiPatch, apiPost, withQuery } from './client'
import type { PaginatedResponse } from './types'

export interface Organization {
  id: string
  name: string
  slug: string
  status: string
  created_at: string
  updated_at: string
}

export interface Member {
  id: string
  user_id: string
  role: string
  status: string
  joined_at: string
}

export interface Invitation {
  id: string
  email: string
  role: string
  status: string
  expires_at: string
  created_at: string
}

export interface CreateInvitationResult {
  id: string
  invitation_token: string
  expires_at: string
}

export async function createOrganization(name: string): Promise<Organization> {
  return apiPost<Organization>('/api/v1/organizations', { name }, { idempotent: true })
}

export async function getOrganization(organizationId: string): Promise<Organization> {
  return apiGet<Organization>(`/api/v1/organizations/${encodeURIComponent(organizationId)}`, {
    org: true,
  })
}

export async function updateOrganization(organizationId: string, name: string): Promise<Organization> {
  return apiPatch<Organization>(
    `/api/v1/organizations/${encodeURIComponent(organizationId)}`,
    { name },
    { org: true },
  )
}

export async function deleteOrganization(organizationId: string): Promise<void> {
  return apiDelete(`/api/v1/organizations/${encodeURIComponent(organizationId)}`, { org: true })
}

export async function listMembers(
  organizationId: string,
  cursor?: string,
  pageSize = 50,
): Promise<PaginatedResponse<Member>> {
  const path = withQuery(
    `/api/v1/organizations/${encodeURIComponent(organizationId)}/members`,
    { page_size: pageSize, cursor },
  )
  return apiGet<PaginatedResponse<Member>>(path, { org: true })
}

export async function updateMember(
  organizationId: string,
  userId: string,
  data: { role?: string; status?: string },
): Promise<Member> {
  return apiPatch<Member>(
    `/api/v1/organizations/${encodeURIComponent(organizationId)}/members/${encodeURIComponent(userId)}`,
    data,
    { org: true },
  )
}

export async function removeMember(organizationId: string, userId: string): Promise<void> {
  return apiDelete(
    `/api/v1/organizations/${encodeURIComponent(organizationId)}/members/${encodeURIComponent(userId)}`,
    { org: true },
  )
}

export async function createInvitation(
  organizationId: string,
  email: string,
  role: string,
): Promise<CreateInvitationResult> {
  return apiPost<CreateInvitationResult>(
    `/api/v1/organizations/${encodeURIComponent(organizationId)}/invitations`,
    { email, role },
    { org: true, idempotent: true },
  )
}

export async function listInvitations(
  organizationId: string,
  cursor?: string,
  pageSize = 50,
): Promise<PaginatedResponse<Invitation>> {
  const path = withQuery(
    `/api/v1/organizations/${encodeURIComponent(organizationId)}/invitations`,
    { page_size: pageSize, cursor },
  )
  return apiGet<PaginatedResponse<Invitation>>(path, { org: true })
}

export async function revokeInvitation(organizationId: string, invitationId: string): Promise<void> {
  return apiDelete(
    `/api/v1/organizations/${encodeURIComponent(organizationId)}/invitations/${encodeURIComponent(invitationId)}`,
    { org: true },
  )
}

export async function acceptInvitation(token: string): Promise<Member> {
  return apiPost<Member>(`/api/v1/invitations/${encodeURIComponent(token)}/accept`, {})
}

export const ORG_ID_HINT = 'Organization IDs use the org_ prefix (e.g. org_…).'
