import { useEffect, useState } from 'react'
import { createBrowserRouter, Navigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { AppShell } from '../components/layout/AppShell'
import { api, setCSRFToken } from '../lib/api'
import { Inbox } from '../pages/Inbox'
import { Login } from '../pages/Login'
import { Settings } from '../pages/Settings'
import { SettingsCloudflare } from '../pages/SettingsCloudflare'
import { SettingsPassword } from '../pages/SettingsPassword'
import { SetupWizard } from '../pages/SetupWizard'

function Root() {
  const [csrfToken, setCSRFTokenState] = useState('')
  const setupQuery = useQuery({ queryKey: ['setup-status'], queryFn: api.setupStatus })
  const meQuery = useQuery({ queryKey: ['me'], queryFn: api.me, retry: false })

  useEffect(() => {
    if (meQuery.data?.csrfToken) {
      setCSRFToken(meQuery.data.csrfToken)
      setCSRFTokenState(meQuery.data.csrfToken)
    }
  }, [meQuery.data?.csrfToken])

  if (setupQuery.isLoading || meQuery.isLoading) {
    return <div className="p-8 text-slate-300">Loading…</div>
  }
  if (!setupQuery.data?.initialized) {
    return <SetupWizard onReady={() => window.location.reload()} />
  }
  if (meQuery.error && !csrfToken) {
    return <Login onLogin={(token) => {
      setCSRFToken(token)
      setCSRFTokenState(token)
      window.location.reload()
    }} />
  }
  return <AppShell />
}

export const router = createBrowserRouter([
  {
    path: '/',
    element: <Root />,
    children: [
      { index: true, element: <Navigate to="/inbox" replace /> },
      { path: 'inbox', element: <Inbox /> },
      {
        path: 'settings',
        element: <Settings />,
        children: [
          { index: true, element: <Navigate to="password" replace /> },
          { path: 'password', element: <SettingsPassword /> },
          { path: 'cloudflare', element: <SettingsCloudflare /> },
        ],
      },
    ],
  },
])
