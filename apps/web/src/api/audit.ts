import { apiGet, withQuery } from './client'
import type { PaginatedResponse } from './types'

export interface AuditActor {
  type: string
  id?: string
}

export interface AuditTarget {
  type: string
  id?: string
}

export interface AuditEvent {
  id: string
  event_type: string
  action: string
  actor: AuditActor
  target: AuditTarget
  result: string
  occurred_at: string
  request_id?: string
  correlation_id?: string
  source_service?: string
  metadata?: Record<string, string>
}

export interface AuditListParams {
  cursor?: string
  pageSize?: number
  eventType?: string
  actorId?: string
  result?: string
}

export async function listAuditEvents(params: AuditListParams = {}): Promise<PaginatedResponse<AuditEvent>> {
  const path = withQuery('/api/v1/audit-events', {
    page_size: params.pageSize ?? 50,
    cursor: params.cursor,
    event_type: params.eventType,
    actor_id: params.actorId,
    result: params.result,
  })
  return apiGet<PaginatedResponse<AuditEvent>>(path, { org: true, requireOrg: true })
}

export async function getAuditEvent(id: string): Promise<AuditEvent> {
  return apiGet<AuditEvent>(`/api/v1/audit-events/${encodeURIComponent(id)}`, {
    org: true,
    requireOrg: true,
  })
}
