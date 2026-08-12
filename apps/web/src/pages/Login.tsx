import { FormEvent, useState } from 'react'
import { ApiError } from '../api/client'
import { demoConfig } from '../config/demo'

interface LoginProps {
  onLogin: (email: string, password: string) => Promise<void>
  demoMode?: boolean
}

export function Login({ onLogin, demoMode = false }: LoginProps) {
  const [email, setEmail] = useState(demoMode ? demoConfig.email : '')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError(null)

    const trimmedEmail = email.trim()
    if (!trimmedEmail) {
      setError('Email is required.')
      return
    }
    if (!password) {
      setError('Password is required.')
      return
    }

    setLoading(true)
    try {
      await onLogin(trimmedEmail, password)
    } catch (err) {
      if (err instanceof ApiError && err.fieldErrors?.length) {
        setError(err.fieldErrors.map((e) => e.message || e.field).join('; '))
      } else {
        setError(err instanceof Error ? err.message : 'Login failed. Please try again.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <div className="login-brand">
          <span className="brand-mark">H</span>
          <div>
            <h1>Halcyon</h1>
            <p>Engineering Operations Intelligence</p>
          </div>
        </div>

        {demoMode && (
          <div className="demo-login-hint" role="status">
            <strong>Demo mode</strong> — run <code>.\scripts\demo-bootstrap.ps1</code> after starting Docker,
            then refresh. Auto-login uses the seeded demo identity.
          </div>
        )}

        <form className="login-form" onSubmit={handleSubmit} noValidate>
          <h2>Sign in to your account</h2>

          {error && (
            <div className="form-error" role="alert">
              {error}
            </div>
          )}

          <label className="form-field">
            <span>Email</span>
            <input
              type="email"
              name="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              disabled={loading}
              placeholder="you@company.com"
            />
          </label>

          <label className="form-field">
            <span>Password</span>
            <input
              type="password"
              name="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              disabled={loading}
              placeholder="••••••••"
            />
          </label>

          <button type="submit" className="btn btn-primary" disabled={loading}>
            {loading ? 'Signing in…' : 'Sign in'}
          </button>
        </form>
      </div>
    </div>
  )
}
