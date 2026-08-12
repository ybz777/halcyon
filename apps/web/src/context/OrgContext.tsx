import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import {
  clearOrganizationId as clearStoredOrgId,
  getOrganizationId as readStoredOrgId,
  setOrganizationId as writeStoredOrgId,
} from '../api/orgContext'

interface OrgContextValue {
  orgId: string | null
  setOrgId: (id: string | null) => void
}

const OrgContext = createContext<OrgContextValue | null>(null)

export function OrgProvider({ children }: { children: ReactNode }) {
  const [orgId, setOrgIdState] = useState<string | null>(() => readStoredOrgId())

  const setOrgId = useCallback((id: string | null) => {
    const trimmed = id?.trim() || null
    if (trimmed) {
      writeStoredOrgId(trimmed)
    } else {
      clearStoredOrgId()
    }
    setOrgIdState(trimmed)
  }, [])

  const value = useMemo(() => ({ orgId, setOrgId }), [orgId, setOrgId])

  return <OrgContext.Provider value={value}>{children}</OrgContext.Provider>
}

export function useOrg(): OrgContextValue {
  const ctx = useContext(OrgContext)
  if (!ctx) {
    throw new Error('useOrg must be used within OrgProvider')
  }
  return ctx
}
