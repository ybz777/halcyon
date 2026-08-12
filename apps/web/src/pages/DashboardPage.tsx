import { useCallback, useEffect, useState } from 'react'
import type { MeResponse } from '../api/auth'
import { listApiKeys } from '../api/apiKeys'
import { listAuditEvents } from '../api/audit'
import { listWebhookEndpoints } from '../api/integrations'
import { getOrganization, listMembers } from '../api/organizations'
import { DemoServiceAccountsList, OrgSelector } from '../components/ConsoleLayout'
import { StateMessage } from '../components/StateMessage'
import { demoConfig, isDemoMode } from '../config/demo'
import { useOrg } from '../context/OrgContext'
import { formatApiError } from '../utils/errors'

interface DashboardPageProps {
  user: MeResponse
}

interface DashboardStats {
  members: number
  apiKeys: number
  webhooks: number
  events: number
}

const PLATFORM_SERVICES = [
  { name: 'API Gateway', status: 'healthy', port: 8088 },
  { name: 'Authentication', status: 'healthy', port: 8080 },
  { name: 'Organization', status: 'healthy', port: 8081 },
  { name: 'Workflow', status: 'healthy', port: 8082 },
  { name: 'Audit', status: 'healthy', port: 8083 },
  { name: 'Integrations', status: 'healthy', port: 8084 },
  { name: 'Notifications', status: 'healthy', port: 8085 },
  { name: 'Jobs', status: 'healthy', port: 8086 },
]

export function DashboardPage({ user }: DashboardPageProps) {
  const { orgId, setOrgId } = useOrg()
  const [orgName, setOrgName] = useState<string | null>(null)
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [recentEvents, setRecentEvents] = useState<Awaited<ReturnType<typeof listAuditEvents>>['items']>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadDashboard = useCallback(async (id: string) => {
    setLoading(true)
    setError(null)
    try {
      const org = await getOrganization(id)
      setOrgName(org.name)

      const [membersPage, keysPage, webhooksPage, eventsPage] = await Promise.all([
        listMembers(id).catch(() => ({ items: [], has_more: false })),
        listApiKeys(demoConfig.primaryServiceAccountId, undefined, 50).catch(() => ({ items: [], has_more: false })),
        listWebhookEndpoints(undefined, 50).catch(() => ({ items: [], has_more: false })),
        listAuditEvents({ pageSize: 5 }).catch(() => ({ items: [], has_more: false })),
      ])

      setStats({
        members: membersPage.items?.length ?? 0,
        apiKeys: keysPage.items?.length ?? 0,
        webhooks: webhooksPage.items?.length ?? 0,
        events: eventsPage.items?.length ?? 0,
      })
      setRecentEvents(eventsPage.items ?? [])
    } catch (err) {
      setOrgName(null)
      setStats(null)
      setRecentEvents([])
      setError(formatApiError(err))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (orgId) {
      void loadDashboard(orgId)
    } else {
      setOrgName(null)
      setStats(null)
      setRecentEvents([])
      setError(null)
    }
  }, [orgId, loadDashboard])

  return (
    <>
      <div className="page-header">
        <h1>{isDemoMode ? 'Halcyon Operations Console' : 'Welcome back'}</h1>
        <p className="page-subtitle">
          {isDemoMode
            ? 'Enterprise engineering operations — organizations, identities, credentials, integrations, and audit activity.'
            : 'Authenticated Halcyon console'}
        </p>
      </div>

      {!isDemoMode && (
        <section className="card">
          <h2>Organization context</h2>
          <p className="card-hint">Select an organization to load dashboard metrics.</p>
          <OrgSelector orgId={orgId} onOrgChange={setOrgId} />
        </section>
      )}

      {loading && <StateMessage variant="loading" title="Loading dashboard…" />}

      {error && !loading && <StateMessage variant="error" title="Dashboard error" message={error} />}

      {orgId && !loading && !error && (
        <>
          <section className="stat-grid">
            <div className="stat-card">
              <span className="stat-label">Organization</span>
              <span className="stat-value">{orgName ?? '—'}</span>
            </div>
            <div className="stat-card">
              <span className="stat-label">Members</span>
              <span className="stat-value">{stats?.members ?? 0}</span>
            </div>
            <div className="stat-card">
              <span className="stat-label">API keys</span>
              <span className="stat-value">{stats?.apiKeys ?? 0}</span>
            </div>
            <div className="stat-card">
              <span className="stat-label">Webhooks</span>
              <span className="stat-value">{stats?.webhooks ?? 0}</span>
            </div>
            <div className="stat-card">
              <span className="stat-label">Recent events</span>
              <span className="stat-value">{stats?.events ?? 0}</span>
            </div>
          </section>

          <div className="dashboard-columns">
            <section className="card">
              <h2>Platform services</h2>
              <p className="card-hint">Local microservices in the Halcyon stack.</p>
              <div className="table-wrap">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Service</th>
                      <th>Status</th>
                      <th>Port</th>
                    </tr>
                  </thead>
                  <tbody>
                    {PLATFORM_SERVICES.map((svc) => (
                      <tr key={svc.name}>
                        <td>{svc.name}</td>
                        <td>
                          <span className="status-pill status-active">{svc.status}</span>
                        </td>
                        <td className="text-muted">{svc.port}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </section>

            <section className="card">
              <h2>Service accounts</h2>
              {isDemoMode ? (
                <DemoServiceAccountsList />
              ) : (
                <StateMessage
                  variant="info"
                  title="No list API"
                  message="Service account listing is not exposed via HTTP. Use API Keys with a known service account ID."
                />
              )}
            </section>
          </div>

          <section className="card card-flush">
            <h2 className="card-section-title">Recent audit events</h2>
            {recentEvents.length === 0 ? (
              <StateMessage
                variant="empty"
                title="No events yet"
                message="Audit events appear as platform activity is recorded."
              />
            ) : (
              <div className="table-wrap">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>Action</th>
                      <th>Actor</th>
                      <th>Result</th>
                      <th>When</th>
                    </tr>
                  </thead>
                  <tbody>
                    {recentEvents.map((ev) => (
                      <tr key={ev.id}>
                        <td>{ev.action}</td>
                        <td className="text-muted">{ev.actor?.id ?? ev.actor?.type ?? '—'}</td>
                        <td>
                          <span className={`status-pill status-${ev.result.toLowerCase()}`}>{ev.result}</span>
                        </td>
                        <td className="text-muted">{new Date(ev.occurred_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="card">
            <h2>Signed-in identity</h2>
            <dl className="info-grid">
              <div>
                <dt>Email</dt>
                <dd>{user.email}</dd>
              </div>
              <div>
                <dt>Roles</dt>
                <dd>{user.roles.length ? user.roles.join(', ') : '—'}</dd>
              </div>
              <div>
                <dt>Permissions</dt>
                <dd>{user.permissions.length ? user.permissions.join(', ') : '—'}</dd>
              </div>
            </dl>
          </section>
        </>
      )}

      {!orgId && !isDemoMode && !loading && (
        <StateMessage
          variant="info"
          title="No organization selected"
          message="Enter an organization ID above to load dashboard data."
        />
      )}
    </>
  )
}
