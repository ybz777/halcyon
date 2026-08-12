import { FormEvent, useCallback, useEffect, useState } from 'react'
import {
  createApiKey,
  listApiKeys,
  revokeApiKey,
  rotateApiKey,
  type ApiKey,
} from '../api/apiKeys'
import {
  clearServiceAccountId,
  getServiceAccountId,
  setServiceAccountId,
} from '../api/orgContext'
import { DemoServiceAccountsList } from '../components/ConsoleLayout'
import { StateMessage } from '../components/StateMessage'
import { demoConfig, isDemoMode } from '../config/demo'
import { useOrg } from '../context/OrgContext'
import { copyToClipboard, formatApiError } from '../utils/errors'

export function ApiKeysPage() {
  const { orgId } = useOrg()
  const [serviceAccountId, setServiceAccountIdState] = useState(() => {
    if (isDemoMode && demoConfig.primaryServiceAccountId) {
      return demoConfig.primaryServiceAccountId
    }
    return getServiceAccountId() ?? ''
  })
  const [keys, setKeys] = useState<ApiKey[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [plaintext, setPlaintext] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)
  const [creating, setCreating] = useState(false)
  const [revokingId, setRevokingId] = useState<string | null>(null)
  const [rotatingId, setRotatingId] = useState<string | null>(null)

  const [formName, setFormName] = useState('')
  const [formScopes, setFormScopes] = useState('integrations.read')
  const [formEnv, setFormEnv] = useState('test')

  const activeServiceAccountId = serviceAccountId.trim()

  const loadKeys = useCallback(async (saId: string) => {
    setLoading(true)
    setError(null)
    try {
      const page = await listApiKeys(saId)
      setKeys(page.items)
    } catch (err) {
      setError(formatApiError(err))
      setKeys([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (isDemoMode && demoConfig.primaryServiceAccountId) {
      setServiceAccountId(demoConfig.primaryServiceAccountId)
      setServiceAccountIdState(demoConfig.primaryServiceAccountId)
    }
  }, [])

  useEffect(() => {
    if (!activeServiceAccountId) {
      setKeys([])
      return
    }
    const timer = window.setTimeout(() => {
      void loadKeys(activeServiceAccountId)
    }, 300)
    return () => window.clearTimeout(timer)
  }, [activeServiceAccountId, loadKeys])

  const applyServiceAccount = () => {
    const id = activeServiceAccountId
    if (id) {
      setServiceAccountId(id)
    } else {
      clearServiceAccountId()
    }
  }

  const showPlaintext = (secret: string) => {
    setPlaintext(secret)
    setCopied(false)
  }

  const handleCopy = async () => {
    if (!plaintext) return
    const ok = await copyToClipboard(plaintext)
    setCopied(ok)
  }

  const handleCreate = async (e: FormEvent) => {
    e.preventDefault()
    if (!activeServiceAccountId || !formName.trim()) return

    setCreating(true)
    setError(null)
    setPlaintext(null)
    try {
      const scopes = formScopes
        .split(',')
        .map((s) => s.trim())
        .filter(Boolean)
      const result = await createApiKey({
        name: formName.trim(),
        serviceAccountId: activeServiceAccountId,
        organizationId: orgId ?? undefined,
        scopes,
        environment: formEnv,
      })
      showPlaintext(result.plaintext)
      setFormName('')
      await loadKeys(activeServiceAccountId)
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setCreating(false)
    }
  }

  const handleRevoke = async (id: string) => {
    if (!window.confirm('Revoke this API key? This cannot be undone.')) return
    setRevokingId(id)
    setError(null)
    try {
      await revokeApiKey(id)
      if (activeServiceAccountId) await loadKeys(activeServiceAccountId)
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setRevokingId(null)
    }
  }

  const handleRotate = async (id: string) => {
    if (!window.confirm('Rotate this API key? The old key will be revoked and a new secret issued.')) return
    setRotatingId(id)
    setError(null)
    setPlaintext(null)
    try {
      const result = await rotateApiKey(id, formEnv)
      showPlaintext(result.plaintext)
      if (activeServiceAccountId) await loadKeys(activeServiceAccountId)
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setRotatingId(null)
    }
  }

  return (
    <>
      <div className="page-header">
        <h1>API Keys</h1>
        <p className="page-subtitle">Manage API keys for service accounts</p>
      </div>

      {isDemoMode ? (
        <section className="card">
          <h2>Demo service accounts</h2>
          <p className="card-hint">Keys are managed for the primary demo service account automatically.</p>
          <DemoServiceAccountsList />
        </section>
      ) : (
        <StateMessage
          variant="info"
          title="Service account required"
          message="There is no backend API to list service accounts. Enter a service account ID to load and manage keys."
        />
      )}

      {!isDemoMode && (
        <section className="card">
          <h2>Service account</h2>
          <div className="org-selector">
            <label className="form-field form-field-inline">
              <span>Service account ID</span>
              <input
                type="text"
                value={serviceAccountId}
                onChange={(e) => setServiceAccountIdState(e.target.value)}
                placeholder="svc_…"
              />
            </label>
            <button type="button" className="btn btn-secondary" onClick={applyServiceAccount}>
              Load keys
            </button>
          </div>
        </section>
      )}

      {activeServiceAccountId && (
        <section className="card">
          <h2>Create API key</h2>
          <form className="form-grid" onSubmit={handleCreate}>
            <label className="form-field">
              <span>Name</span>
              <input
                type="text"
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                disabled={creating}
                required
              />
            </label>
            <label className="form-field">
              <span>Environment</span>
              <select value={formEnv} onChange={(e) => setFormEnv(e.target.value)} disabled={creating}>
                <option value="test">test</option>
                <option value="live">live</option>
              </select>
            </label>
            <label className="form-field form-field-full">
              <span>Scopes (comma-separated)</span>
              <input
                type="text"
                value={formScopes}
                onChange={(e) => setFormScopes(e.target.value)}
                disabled={creating}
              />
            </label>
            <div className="form-actions">
              <button type="submit" className="btn btn-primary" disabled={creating || !formName.trim()}>
                {creating ? 'Creating…' : 'Create API key'}
              </button>
            </div>
          </form>
        </section>
      )}

      {plaintext && (
        <section className="card card-warning" role="alert">
          <h2>API key secret — copy now</h2>
          <p className="card-hint">This secret is shown once and is not stored in the browser.</p>
          <code className="secret-display">{plaintext}</code>
          <div className="form-actions">
            <button type="button" className="btn btn-secondary" onClick={() => void handleCopy()}>
              {copied ? 'Copied' : 'Copy to clipboard'}
            </button>
            <button type="button" className="btn btn-ghost" onClick={() => setPlaintext(null)}>
              I have copied the key
            </button>
          </div>
        </section>
      )}

      {error && <StateMessage variant="error" title="API key error" message={error} />}

      {loading && <StateMessage variant="loading" title="Loading API keys…" />}

      {!loading && activeServiceAccountId && keys.length === 0 && !error && (
        <StateMessage variant="empty" title="No API keys" message="No keys found for this service account." />
      )}

      {!loading && keys.length > 0 && (
        <section className="card card-flush">
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Prefix</th>
                  <th>Status</th>
                  <th>Scopes</th>
                  <th>Created</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {keys.map((key) => (
                  <tr key={key.id}>
                    <td>{key.name}</td>
                    <td>
                      <code>{key.prefix}</code>
                    </td>
                    <td>
                      <span className={`status-pill status-${key.status.toLowerCase()}`}>{key.status}</span>
                    </td>
                    <td className="text-muted">{key.scopes.join(', ')}</td>
                    <td className="text-muted">{new Date(key.createdAt).toLocaleString()}</td>
                    <td className="cell-actions">
                      {key.status !== 'revoked' && (
                        <>
                          <button
                            type="button"
                            className="btn btn-ghost btn-sm"
                            disabled={rotatingId === key.id}
                            onClick={() => void handleRotate(key.id)}
                          >
                            {rotatingId === key.id ? 'Rotating…' : 'Rotate'}
                          </button>
                          <button
                            type="button"
                            className="btn btn-ghost btn-sm btn-danger"
                            disabled={revokingId === key.id}
                            onClick={() => void handleRevoke(key.id)}
                          >
                            {revokingId === key.id ? 'Revoking…' : 'Revoke'}
                          </button>
                        </>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </>
  )
}
