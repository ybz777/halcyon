import { apiGet, apiPut } from './client'
import { withQuery } from './client'
import type { PaginatedResponse } from './types'

export interface NotificationPreference {
  notification_type: string
  channel: string
  enabled: boolean
  updated_at: string
}

export async function listNotificationPreferences(
  cursor?: string,
  pageSize = 50,
): Promise<PaginatedResponse<NotificationPreference>> {
  const path = withQuery('/api/v1/notification-preferences', { page_size: pageSize, cursor })
  return apiGet<PaginatedResponse<NotificationPreference>>(path, { org: true, requireOrg: true })
}

export async function updateNotificationPreference(
  notificationType: string,
  channel: string,
  enabled: boolean,
): Promise<NotificationPreference> {
  return apiPut<NotificationPreference>(
    `/api/v1/notification-preferences/${encodeURIComponent(notificationType)}`,
    { channel, enabled },
    { org: true, requireOrg: true },
  )
}
