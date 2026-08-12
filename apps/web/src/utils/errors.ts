import { ApiError } from '../api/client'

export function formatApiError(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.status === 404) {
      return `${err.message} (HTTP 404 — resource not found or route unavailable via gateway)`
    }
    if (err.status === 403) {
      return `${err.message} (HTTP 403 — insufficient permissions)`
    }
    if (err.status === 401) {
      return `${err.message} (HTTP 401 — authentication required)`
    }
    if (err.fieldErrors?.length) {
      return err.fieldErrors.map((e) => e.message || e.field).join('; ')
    }
    return err.message
  }
  if (err instanceof Error) {
    return err.message
  }
  return 'An unexpected error occurred.'
}

export async function copyToClipboard(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    return false
  }
}
