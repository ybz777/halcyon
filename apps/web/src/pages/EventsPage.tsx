import { useCallback, useEffect, useState } from 'react'
import { listAuditEvents, type AuditEvent } from '../api/audit'
import { StateMessage } from '../components/StateMessage'
import { useOrg } from '../context/OrgContext'
import { formatApiError } from '../utils/errors'

export function EventsPage() {
  const { orgId } = useOrg()
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [cursor, setCursor] = useState<string | undefined>()
  const [hasMore, setHasMore] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [filterResult, setFilterResult] = useState('')

  const load = useCallback(
    async (nextCursor?: string, append = false) => {
      if (!orgId) return
      if (append) setLoadingMore(true)
      else setLoading(true)
      setError(null)
      try {
        const page = await listAuditEvents({
          cursor: nextCursor,
          result: filterResult || undefined,
        })
        setEvents((prev) => (append ? [...prev, ...page.items] : page.items))
        setCursor(page.next_cursor)
        setHasMore(page.has_more)
      } catch (err) {
        setError(formatApiError(err))
        if (!append) setEvents([])
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [orgId, filterResult],
  )

  useEffect(() => {
    void load()
  }, [load])

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
      <div className="page-header page-header-row">
        <div>
          <h1>Events</h1>
          <p className="page-subtitle">Audit activity from the backend</p>
        </div>
        <label className="form-field form-field-inline">
          <span>Result</span>
          <select value={filterResult} onChange={(e) => setFilterResult(e.target.value)}>
            <option value="">All</option>
            <option value="SUCCESS">SUCCESS</option>
            <option value="FAILURE">FAILURE</option>
            <option value="DENIED">DENIED</option>
          </select>
        </label>
      </div>

      {loading && <StateMessage variant="loading" title="Loading audit events…" />}

      {error && !loading && (
        <StateMessage variant="error" title="Failed to load events" message={error} />
      )}

      {!loading && !error && events.length === 0 && (
        <StateMessage variant="empty" title="No audit events" message="No events match the current filters." />
      )}

      {!loading && events.length > 0 && (
        <section className="card card-flush">
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Event type</th>
                  <th>Action</th>
                  <th>Actor</th>
                  <th>Target</th>
                  <th>Result</th>
                </tr>
              </thead>
              <tbody>
                {events.map((ev) => (
                  <tr key={ev.id}>
                    <td className="text-muted">{new Date(ev.occurred_at).toLocaleString()}</td>
                    <td>
                      <code>{ev.event_type}</code>
                    </td>
                    <td>{ev.action}</td>
                    <td>
                      {ev.actor.type}
                      {ev.actor.id ? ` · ${ev.actor.id}` : ''}
                    </td>
                    <td>
                      {ev.target.type}
                      {ev.target.id ? ` · ${ev.target.id}` : ''}
                    </td>
                    <td>
                      <span className={`status-pill status-${ev.result.toLowerCase()}`}>{ev.result}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {hasMore && (
            <div className="table-footer">
              <button
                type="button"
                className="btn btn-secondary"
                disabled={loadingMore}
                onClick={() => void load(cursor, true)}
              >
                {loadingMore ? 'Loading…' : 'Load more'}
              </button>
            </div>
          )}
        </section>
      )}
    </>
  )
}
