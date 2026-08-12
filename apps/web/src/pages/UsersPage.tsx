import { FormEvent, useCallback, useEffect, useState } from 'react'
import {
  createInvitation,
  listInvitations,
  listMembers,
  removeMember,
  revokeInvitation,
  updateMember,
  type Invitation,
  type Member,
} from '../api/organizations'
import { StateMessage } from '../components/StateMessage'
import { useOrg } from '../context/OrgContext'
import { formatApiError } from '../utils/errors'

const MEMBER_ROLES = ['owner', 'admin', 'member', 'viewer'] as const
const MEMBER_STATUSES = ['active', 'suspended'] as const

export function UsersPage() {
  const { orgId } = useOrg()
  const [members, setMembers] = useState<Member[]>([])
  const [invitations, setInvitations] = useState<Invitation[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [memberCursor, setMemberCursor] = useState<string | undefined>()
  const [memberHasMore, setMemberHasMore] = useState(false)
  const [loadingMoreMembers, setLoadingMoreMembers] = useState(false)
  const [invCursor, setInvCursor] = useState<string | undefined>()
  const [invHasMore, setInvHasMore] = useState(false)
  const [loadingMoreInv, setLoadingMoreInv] = useState(false)

  const [inviteEmail, setInviteEmail] = useState('')
  const [inviteRole, setInviteRole] = useState<string>('member')
  const [inviting, setInviting] = useState(false)
  const [invitationToken, setInvitationToken] = useState<string | null>(null)

  const [editingUserId, setEditingUserId] = useState<string | null>(null)
  const [editRole, setEditRole] = useState('')
  const [editStatus, setEditStatus] = useState('')
  const [savingMember, setSavingMember] = useState(false)
  const [removingUserId, setRemovingUserId] = useState<string | null>(null)
  const [revokingId, setRevokingId] = useState<string | null>(null)

  const loadMembers = useCallback(
    async (nextCursor?: string, append = false) => {
      if (!orgId) return
      if (append) setLoadingMoreMembers(true)
      try {
        const page = await listMembers(orgId, nextCursor)
        setMembers((prev) => (append ? [...prev, ...page.items] : page.items))
        setMemberCursor(page.next_cursor)
        setMemberHasMore(page.has_more)
      } catch (err) {
        throw err
      } finally {
        setLoadingMoreMembers(false)
      }
    },
    [orgId],
  )

  const loadInvitations = useCallback(
    async (nextCursor?: string, append = false) => {
      if (!orgId) return
      if (append) setLoadingMoreInv(true)
      try {
        const page = await listInvitations(orgId, nextCursor)
        setInvitations((prev) => (append ? [...prev, ...page.items] : page.items))
        setInvCursor(page.next_cursor)
        setInvHasMore(page.has_more)
      } catch (err) {
        throw err
      } finally {
        setLoadingMoreInv(false)
      }
    },
    [orgId],
  )

  const loadAll = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      await Promise.all([loadMembers(), loadInvitations()])
    } catch (err) {
      setError(formatApiError(err))
      setMembers([])
      setInvitations([])
    } finally {
      setLoading(false)
    }
  }, [orgId, loadMembers, loadInvitations])

  useEffect(() => {
    void loadAll()
  }, [loadAll])

  const handleInvite = async (e: FormEvent) => {
    e.preventDefault()
    if (!orgId || !inviteEmail.trim()) return
    setInviting(true)
    setError(null)
    setSuccess(null)
    setInvitationToken(null)
    try {
      const result = await createInvitation(orgId, inviteEmail.trim(), inviteRole)
      setInvitationToken(result.invitation_token)
      setInviteEmail('')
      setSuccess('Invitation created.')
      await loadInvitations()
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setInviting(false)
    }
  }

  const startEditMember = (member: Member) => {
    setEditingUserId(member.user_id)
    setEditRole(member.role)
    setEditStatus(member.status)
  }

  const cancelEditMember = () => {
    setEditingUserId(null)
  }

  const saveMember = async (userId: string) => {
    if (!orgId) return
    setSavingMember(true)
    setError(null)
    setSuccess(null)
    try {
      await updateMember(orgId, userId, { role: editRole, status: editStatus })
      setEditingUserId(null)
      setSuccess('Member updated.')
      await loadMembers()
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setSavingMember(false)
    }
  }

  const handleRemoveMember = async (userId: string) => {
    if (!orgId || !window.confirm('Remove this member from the organization?')) return
    setRemovingUserId(userId)
    setError(null)
    setSuccess(null)
    try {
      await removeMember(orgId, userId)
      setSuccess('Member removed.')
      await loadMembers()
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setRemovingUserId(null)
    }
  }

  const handleRevokeInvitation = async (invitationId: string) => {
    if (!orgId || !window.confirm('Revoke this invitation?')) return
    setRevokingId(invitationId)
    setError(null)
    setSuccess(null)
    try {
      await revokeInvitation(orgId, invitationId)
      setSuccess('Invitation revoked.')
      await loadInvitations()
    } catch (err) {
      setError(formatApiError(err))
    } finally {
      setRevokingId(null)
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
        <h1>Users</h1>
        <p className="page-subtitle">Organization members and invitations</p>
      </div>

      {success && <StateMessage variant="success" title={success} />}

      {loading && <StateMessage variant="loading" title="Loading members…" />}

      {error && !loading && (
        <StateMessage variant="error" title="Request failed" message={error} />
      )}

      {!loading && (
        <>
          <section className="card">
            <h2>Invite member</h2>
            <form className="form-grid" onSubmit={handleInvite}>
              <label className="form-field">
                <span>Email</span>
                <input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  placeholder="colleague@example.com"
                  disabled={inviting}
                  required
                />
              </label>
              <label className="form-field">
                <span>Role</span>
                <select value={inviteRole} onChange={(e) => setInviteRole(e.target.value)} disabled={inviting}>
                  {MEMBER_ROLES.map((role) => (
                    <option key={role} value={role}>
                      {role}
                    </option>
                  ))}
                </select>
              </label>
              <div className="form-actions">
                <button type="submit" className="btn btn-primary" disabled={inviting || !inviteEmail.trim()}>
                  {inviting ? 'Sending…' : 'Send invitation'}
                </button>
              </div>
            </form>
          </section>

          {invitationToken && (
            <section className="card card-warning" role="alert">
              <h2>Invitation token — copy now</h2>
              <p className="card-hint">
                Share this token with the invitee. They can accept via{' '}
                <code>POST /api/v1/invitations/&#123;token&#125;/accept</code>.
              </p>
              <code className="secret-display">{invitationToken}</code>
              <div className="form-actions">
                <button type="button" className="btn btn-secondary" onClick={() => setInvitationToken(null)}>
                  Dismiss
                </button>
              </div>
            </section>
          )}

          <section className="card card-flush">
            <h2 className="card-section-title">Members</h2>
            {members.length === 0 ? (
              <StateMessage variant="empty" title="No members found" message="This organization has no members yet." />
            ) : (
              <>
                <div className="table-wrap">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>User ID</th>
                        <th>Role</th>
                        <th>Status</th>
                        <th>Joined</th>
                        <th />
                      </tr>
                    </thead>
                    <tbody>
                      {members.map((m) => (
                        <tr key={m.id}>
                          <td>
                            <code>{m.user_id}</code>
                          </td>
                          {editingUserId === m.user_id ? (
                            <>
                              <td>
                                <select value={editRole} onChange={(e) => setEditRole(e.target.value)}>
                                  {MEMBER_ROLES.map((role) => (
                                    <option key={role} value={role}>
                                      {role}
                                    </option>
                                  ))}
                                </select>
                              </td>
                              <td>
                                <select value={editStatus} onChange={(e) => setEditStatus(e.target.value)}>
                                  {MEMBER_STATUSES.map((status) => (
                                    <option key={status} value={status}>
                                      {status}
                                    </option>
                                  ))}
                                </select>
                              </td>
                              <td className="text-muted">{new Date(m.joined_at).toLocaleString()}</td>
                              <td className="cell-actions">
                                <button
                                  type="button"
                                  className="btn btn-secondary btn-sm"
                                  disabled={savingMember}
                                  onClick={() => void saveMember(m.user_id)}
                                >
                                  {savingMember ? 'Saving…' : 'Save'}
                                </button>
                                <button type="button" className="btn btn-ghost btn-sm" onClick={cancelEditMember}>
                                  Cancel
                                </button>
                              </td>
                            </>
                          ) : (
                            <>
                              <td>{m.role}</td>
                              <td>
                                <span className={`status-pill status-${m.status.toLowerCase()}`}>{m.status}</span>
                              </td>
                              <td className="text-muted">{new Date(m.joined_at).toLocaleString()}</td>
                              <td className="cell-actions">
                                <button
                                  type="button"
                                  className="btn btn-ghost btn-sm"
                                  onClick={() => startEditMember(m)}
                                >
                                  Edit
                                </button>
                                <button
                                  type="button"
                                  className="btn btn-ghost btn-sm btn-danger"
                                  disabled={removingUserId === m.user_id}
                                  onClick={() => void handleRemoveMember(m.user_id)}
                                >
                                  {removingUserId === m.user_id ? 'Removing…' : 'Remove'}
                                </button>
                              </td>
                            </>
                          )}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {memberHasMore && (
                  <div className="table-footer">
                    <button
                      type="button"
                      className="btn btn-secondary"
                      disabled={loadingMoreMembers}
                      onClick={() => void loadMembers(memberCursor, true)}
                    >
                      {loadingMoreMembers ? 'Loading…' : 'Load more members'}
                    </button>
                  </div>
                )}
              </>
            )}
          </section>

          <section className="card card-flush">
            <h2 className="card-section-title">Pending invitations</h2>
            {invitations.length === 0 ? (
              <StateMessage variant="empty" title="No invitations" message="No pending invitations for this organization." />
            ) : (
              <>
                <div className="table-wrap">
                  <table className="data-table">
                    <thead>
                      <tr>
                        <th>Email</th>
                        <th>Role</th>
                        <th>Status</th>
                        <th>Expires</th>
                        <th>Created</th>
                        <th />
                      </tr>
                    </thead>
                    <tbody>
                      {invitations.map((inv) => (
                        <tr key={inv.id}>
                          <td>{inv.email}</td>
                          <td>{inv.role}</td>
                          <td>
                            <span className={`status-pill status-${inv.status.toLowerCase()}`}>{inv.status}</span>
                          </td>
                          <td className="text-muted">{new Date(inv.expires_at).toLocaleString()}</td>
                          <td className="text-muted">{new Date(inv.created_at).toLocaleString()}</td>
                          <td>
                            {inv.status === 'pending' && (
                              <button
                                type="button"
                                className="btn btn-ghost btn-sm btn-danger"
                                disabled={revokingId === inv.id}
                                onClick={() => void handleRevokeInvitation(inv.id)}
                              >
                                {revokingId === inv.id ? 'Revoking…' : 'Revoke'}
                              </button>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {invHasMore && (
                  <div className="table-footer">
                    <button
                      type="button"
                      className="btn btn-secondary"
                      disabled={loadingMoreInv}
                      onClick={() => void loadInvitations(invCursor, true)}
                    >
                      {loadingMoreInv ? 'Loading…' : 'Load more invitations'}
                    </button>
                  </div>
                )}
              </>
            )}
          </section>
        </>
      )}
    </>
  )
}
