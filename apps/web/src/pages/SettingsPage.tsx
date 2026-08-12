import { useCallback, useEffect, useState } from 'react'
import {
  listNotificationPreferences,
  updateNotificationPreference,
  type NotificationPreference,
} from '../api/notifications'
import { StateMessage } from '../components/StateMessage'
import { useOrg } from '../context/OrgContext'
import { formatApiError } from '../utils/errors'

export function SettingsPage() {
  const { orgId } = useOrg()
  const [prefs, setPrefs] = useState<NotificationPreference[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const page = await listNotificationPreferences()
      setPrefs(page.items)
    } catch (err) {
      setError(formatApiError(err))
      setPrefs([])
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void load()
  }, [load])

  const togglePref = async (pref: NotificationPreference) => {
    setSaving(pref.notification_type)
    setError(null)
    try {
      const updated = await updateNotificationPreference(
        pref.notification_type,
        pref.channel,
        !pref.enabled,
      )
      setPrefs((prev) =>
        prev.map((p) => (p.notification_type === updated.notification_type ? updated : p)),
      )
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setSaving(null)
    }
  }

  if (!orgId) {
    return (
      <StateMessage
        variant="info"
        title="Organization required"
        message="Select or create an organization on the Dashboard."
      />
    )
  }

  return (
    <>
      <div className="page-header">
        <h1>Settings</h1>
        <p className="page-subtitle">Notification preferences from the backend API</p>
      </div>

      {loading && <StateMessage variant="loading" title="Loading preferences…" />}

      {error && !loading && (
        <StateMessage variant="error" title="Failed to load settings" message={error} />
      )}

      {!loading && !error && prefs.length === 0 && (
        <StateMessage
          variant="empty"
          title="No notification preferences"
          message="The backend returned no preference records for your account."
        />
      )}

      {!loading && prefs.length > 0 && (
        <section className="card card-flush">
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Notification type</th>
                  <th>Channel</th>
                  <th>Enabled</th>
                  <th>Updated</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {prefs.map((pref) => (
                  <tr key={`${pref.notification_type}-${pref.channel}`}>
                    <td>
                      <code>{pref.notification_type}</code>
                    </td>
                    <td>{pref.channel}</td>
                    <td>
                      <span className={`status-pill status-${pref.enabled ? 'active' : 'disabled'}`}>
                        {pref.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td className="text-muted">
                      {pref.updated_at ? new Date(pref.updated_at).toLocaleString() : '—'}
                    </td>
                    <td>
                      <button
                        type="button"
                        className="btn btn-secondary btn-sm"
                        disabled={saving === pref.notification_type}
                        onClick={() => void togglePref(pref)}
                      >
                        {saving === pref.notification_type
                          ? 'Saving…'
                          : pref.enabled
                            ? 'Disable'
                            : 'Enable'}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      <section className="card">
        <h2>About settings</h2>
        <p className="card-hint">
          This page shows notification preferences from{' '}
          <code>GET /api/v1/notification-preferences</code>. There is no general org/user settings API
          in the backend.
        </p>
      </section>
    </>
  )
}
