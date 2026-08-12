import { FormEvent, useCallback, useEffect, useState } from 'react'
import {
  createWebhookEndpoint,
  listWebhookEndpoints,
  type WebhookEndpoint,
} from '../api/integrations'
import { StateMessage } from '../components/StateMessage'
import { useOrg } from '../context/OrgContext'
import { copyToClipboard, formatApiError } from '../utils/errors'

export function ServicesPage() {
  const { orgId } = useOrg()
  const [endpoints, setEndpoints] = useState<WebhookEndpoint[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [cursor, setCursor] = useState<string | undefined>()
  const [hasMore, setHasMore] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)

  const [formUrl, setFormUrl] = useState('')
  const [formDescription, setFormDescription] = useState('')
  const [formEvents, setFormEvents] = useState('organization.created')
  const [creating, setCreating] = useState(false)
  const [webhookSecret, setWebhookSecret] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const load = useCallback(
    async (nextCursor?: string, append = false) => {
      if (!orgId) return
      if (append) setLoadingMore(true)
      else setLoading(true)
      setError(null)
      try {
        const page = await listWebhookEndpoints(nextCursor)
        setEndpoints((prev) => (append ? [...prev, ...page.items] : page.items))
        setCursor(page.next_cursor)
        setHasMore(page.has_more)
      } catch (err) {
        setError(formatApiError(err))
        if (!append) setEndpoints([])
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [orgId],
  )

  useEffect(() => {
    void load()
  }, [load])

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    if (!formUrl.trim()) return
    setCreating(true)
    setError(null)
    setSuccess(null)
    setWebhookSecret(null)
    try {
      const subscribed_events = formEvents
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      const result = await createWebhookEndpoint({
        url: formUrl.trim(),
        description: formDescription.trim() || undefined,
        subscribed_events,
      })
      setWebhookSecret(result.secret)
      setCopied(false)
      setFormUrl('')
      setFormDescription('')
      setSuccess('Webhook endpoint created.')
      await load()
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setCreating(false)
    }
  }

  const handleCopySecret = async () => {
    if (!webhookSecret) return
    const ok = await copyToClipboard(webhookSecret)
    setCopied(ok)
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
        <h1>Services</h1>
        <p className="page-subtitle">Webhook integration endpoints</p>
      </div>

      {success && <StateMessage variant="success" title={success} />}

      <section className="card">
        <h2>Create webhook endpoint</h2>
        <form className="form-grid" onSubmit={handleCreate}>
          <label className="form-field form-field-full">
            <span>URL</span>
            <input
              type="url"
              value={formUrl}
              onChange={(e) => setFormUrl(e.target.value)}
              placeholder="https://example.com/webhooks/halcyon"
              disabled={creating}
              required
            />
          </label>
          <label className="form-field form-field-full">
            <span>Description</span>
            <input
              type="text"
              value={formDescription}
              onChange={(e) => setFormDescription(e.target.value)}
              placeholder="Optional description"
              disabled={creating}
            />
          </label>
          <label className="form-field form-field-full">
            <span>Subscribed events (comma-separated)</span>
            <input
              type="text"
              value={formEvents}
              onChange={(e) => setFormEvents(e.target.value)}
              disabled={creating}
            />
          </label>
          <div className="form-actions">
            <button type="submit" className="btn btn-primary" disabled={creating || !formUrl.trim()}>
              {creating ? 'Creating…' : 'Create endpoint'}
            </button>
          </div>
        </form>
      </section>

      {webhookSecret && (
        <section className="card card-warning" role="alert">
          <h2>Webhook signing secret — copy now</h2>
          <p className="card-hint">This secret is shown once and is not stored in the browser.</p>
          <code className="secret-display">{webhookSecret}</code>
          <div className="form-actions">
            <button type="button" className="btn btn-secondary" onClick={() => void handleCopySecret()}>
              {copied ? 'Copied' : 'Copy to clipboard'}
            </button>
            <button type="button" className="btn btn-ghost" onClick={() => setWebhookSecret(null)}>
              Dismiss
            </button>
          </div>
        </section>
      )}

      {loading && <StateMessage variant="loading" title="Loading webhook endpoints…" />}

      {error && !loading && (
        <StateMessage variant="error" title="Failed to load integrations" message={error} />
      )}

      {!loading && !error && endpoints.length === 0 && (
        <StateMessage
          variant="empty"
          title="No webhook endpoints"
          message="No integration endpoints are configured for this organization."
        />
      )}

      {!loading && endpoints.length > 0 && (
        <section className="card card-flush">
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>URL</th>
                  <th>Description</th>
                  <th>Status</th>
                  <th>Events</th>
                  <th>Failures</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {endpoints.map((ep) => (
                  <tr key={ep.id}>
                    <td className="cell-truncate" title={ep.url}>
                      {ep.url}
                    </td>
                    <td>{ep.description || '—'}</td>
                    <td>
                      <span className={`status-pill status-${ep.status.toLowerCase()}`}>{ep.status}</span>
                    </td>
                    <td className="text-muted">{ep.subscribed_events?.join(', ') || '—'}</td>
                    <td>{ep.failure_count}</td>
                    <td className="text-muted">
                      {ep.created_at ? new Date(ep.created_at).toLocaleString() : '—'}
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
