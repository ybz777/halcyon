import type { MeResponse } from '../api/auth'
import { demoConfig, isDemoMode } from '../config/demo'
import { useOrg } from '../context/OrgContext'

export type PageId = 'dashboard' | 'users' | 'api-keys' | 'services' | 'events' | 'settings'

interface NavItem {
  id: PageId
  label: string
  hint?: string
}

const NAV_ITEMS: NavItem[] = [
  { id: 'dashboard', label: 'Dashboard' },
  { id: 'users', label: 'Users', hint: 'Organization members' },
  { id: 'api-keys', label: 'API Keys', hint: 'Requires apikey:manage' },
  { id: 'services', label: 'Services', hint: 'Webhook integrations' },
  { id: 'events', label: 'Events', hint: 'Audit log' },
  { id: 'settings', label: 'Settings', hint: 'Notification preferences' },
]

interface ConsoleLayoutProps {
  user: MeResponse
  page: PageId
  onPageChange: (page: PageId) => void
  onLogout: () => void
  demoMode?: boolean
  children: React.ReactNode
}

export function ConsoleLayout({
  user,
  page,
  onPageChange,
  onLogout,
  demoMode = false,
  children,
}: ConsoleLayoutProps) {
  const { orgId } = useOrg()

  return (
    <div className="console">
      <header className="console-header">
        <div className="console-header-left">
          <div className="console-brand">
            <span className="brand-mark brand-mark-sm">H</span>
            <span className="brand-name">HALCYON</span>
          </div>
          {demoMode && (
            <span className="demo-badge" title="Read-only portfolio demonstration">
              DEMO MODE
            </span>
          )}
          {orgId && !demoMode && (
            <span className="header-org" title="Active organization">
              <code>{orgId}</code>
            </span>
          )}
          {orgId && demoMode && (
            <span className="header-org header-org-demo" title="Active organization">
              Halcyon Demo
            </span>
          )}
        </div>
        <div className="console-user">
          <span className="user-email">{user.email}</span>
          <button type="button" className="btn btn-ghost" onClick={onLogout}>
            {demoMode ? 'Reset session' : 'Logout'}
          </button>
        </div>
      </header>

      <div className="console-body">
        <nav className="console-sidebar" aria-label="Main navigation">
          <ul>
            {NAV_ITEMS.map((item) => (
              <li key={item.id}>
                <button
                  type="button"
                  className={`nav-item${page === item.id ? ' nav-item-active' : ''}`}
                  onClick={() => onPageChange(item.id)}
                  aria-current={page === item.id ? 'page' : undefined}
                  title={item.hint}
                >
                  {item.label}
                </button>
              </li>
            ))}
          </ul>
        </nav>

        <main className="console-main">{children}</main>
      </div>
    </div>
  )
}

export function OrgSelector({
  orgId,
  onOrgChange,
  hiddenInDemo = false,
}: {
  orgId: string | null
  onOrgChange: (id: string | null) => void
  hiddenInDemo?: boolean
}) {
  if (hiddenInDemo && isDemoMode) {
    return (
      <p className="card-hint">
        Demo organization <strong>Halcyon Demo</strong> is selected automatically.
      </p>
    )
  }

  return (
    <div className="org-selector">
      <label className="form-field form-field-inline">
        <span>Organization ID</span>
        <input
          type="text"
          defaultValue={orgId ?? ''}
          key={orgId ?? 'empty'}
          placeholder="org_…"
          onBlur={(e) => {
            const value = e.target.value.trim()
            if (value !== (orgId ?? '')) {
              onOrgChange(value || null)
            }
          }}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              const value = (e.target as HTMLInputElement).value.trim()
              onOrgChange(value || null)
            }
          }}
        />
      </label>
      {orgId && (
        <button type="button" className="btn btn-ghost" onClick={() => onOrgChange(null)}>
          Clear
        </button>
      )}
    </div>
  )
}

export function DemoServiceAccountsList() {
  if (!isDemoMode) return null

  return (
    <ul className="service-account-list">
      {demoConfig.serviceAccounts.map((sa) => (
        <li key={sa.id} className="service-account-item">
          <div>
            <strong>{sa.name}</strong>
            <p className="text-muted">{sa.description}</p>
          </div>
          <code>{sa.id}</code>
        </li>
      ))}
    </ul>
  )
}
