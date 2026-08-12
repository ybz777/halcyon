const ORG_KEY = 'halcyon_organization_id'
const SERVICE_ACCOUNT_KEY = 'halcyon_service_account_id'

export function getOrganizationId(): string | null {
  return sessionStorage.getItem(ORG_KEY)
}

export function setOrganizationId(id: string): void {
  sessionStorage.setItem(ORG_KEY, id.trim())
}

export function clearOrganizationId(): void {
  sessionStorage.removeItem(ORG_KEY)
}

export function getServiceAccountId(): string | null {
  return sessionStorage.getItem(SERVICE_ACCOUNT_KEY)
}

export function setServiceAccountId(id: string): void {
  sessionStorage.setItem(SERVICE_ACCOUNT_KEY, id.trim())
}

export function clearServiceAccountId(): void {
  sessionStorage.removeItem(SERVICE_ACCOUNT_KEY)
}
