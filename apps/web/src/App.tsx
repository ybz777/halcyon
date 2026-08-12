import { useCallback, useEffect, useState } from 'react'
import {
  clearStoredToken,
  getMe,
  getStoredToken,
  login,
  setStoredToken,
  type MeResponse,
} from './api/auth'
import { setOrganizationId, setServiceAccountId } from './api/orgContext'
import { ConsoleLayout, type PageId } from './components/ConsoleLayout'
import { applyDemoSessionDefaults, demoConfig, isDemoMode } from './config/demo'
import { useOrg } from './context/OrgContext'
import { ApiKeysPage } from './pages/ApiKeysPage'
import { DashboardPage } from './pages/DashboardPage'
import { EventsPage } from './pages/EventsPage'
import { Login } from './pages/Login'
import { ServicesPage } from './pages/ServicesPage'
import { SettingsPage } from './pages/SettingsPage'
import { UsersPage } from './pages/UsersPage'

type View = 'loading' | 'login' | 'app'

export default function App() {
  const { setOrgId } = useOrg()
  const [view, setView] = useState<View>('loading')
  const [user, setUser] = useState<MeResponse | null>(null)
  const [page, setPage] = useState<PageId>('dashboard')

  const applyDemoContext = useCallback(() => {
    if (!isDemoMode) return
    applyDemoSessionDefaults(setOrgId, (id) => setServiceAccountId(id))
    if (demoConfig.orgId) {
      setOrganizationId(demoConfig.orgId)
    }
    if (demoConfig.primaryServiceAccountId) {
      setServiceAccountId(demoConfig.primaryServiceAccountId)
    }
  }, [setOrgId])

  const loadSession = useCallback(async () => {
    const token = getStoredToken()
    if (!token) {
      if (isDemoMode) {
        try {
          const { accessToken } = await login(demoConfig.email, demoConfig.password)
          setStoredToken(accessToken)
          const me = await getMe()
          setUser(me)
          applyDemoContext()
          setView('app')
          return
        } catch {
          setUser(null)
          setView('login')
          return
        }
      }
      setUser(null)
      setView('login')
      return
    }

    try {
      const me = await getMe()
      setUser(me)
      if (isDemoMode) {
        applyDemoContext()
      }
      setView('app')
    } catch {
      clearStoredToken()
      setOrgId(null)
      setUser(null)
      setView('login')
    }
  }, [applyDemoContext, setOrgId])

  useEffect(() => {
    void loadSession()
  }, [loadSession])

  const handleLogin = async (email: string, password: string) => {
    const { accessToken } = await login(email, password)
    setStoredToken(accessToken)
    const me = await getMe()
    setUser(me)
    if (isDemoMode) {
      applyDemoContext()
    }
    setView('app')
    setPage('dashboard')
  }

  const handleLogout = () => {
    clearStoredToken()
    setOrgId(null)
    setUser(null)
    if (isDemoMode) {
      void loadSession()
      return
    }
    setView('login')
    setPage('dashboard')
  }

  if (view === 'loading') {
    return (
      <div className="app-loading">
        <div className="spinner" aria-hidden="true" />
        <p>Loading Halcyon…</p>
      </div>
    )
  }

  if (view === 'login' || !user) {
    return <Login onLogin={handleLogin} demoMode={isDemoMode} />
  }

  const renderPage = () => {
    switch (page) {
      case 'dashboard':
        return <DashboardPage user={user} />
      case 'users':
        return <UsersPage />
      case 'api-keys':
        return <ApiKeysPage />
      case 'services':
        return <ServicesPage />
      case 'events':
        return <EventsPage />
      case 'settings':
        return <SettingsPage />
      default:
        return <DashboardPage user={user} />
    }
  }

  return (
    <ConsoleLayout
      user={user}
      page={page}
      onPageChange={setPage}
      onLogout={handleLogout}
      demoMode={isDemoMode}
    >
      {renderPage()}
    </ConsoleLayout>
  )
}
