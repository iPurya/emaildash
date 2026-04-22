import { useEffect, useState } from 'react'
import { createBrowserRouter, Navigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { AppShell } from '../components/layout/AppShell'
import { api, clearCSRFToken, setCSRFToken } from '../lib/api'
import { Inbox } from '../pages/Inbox'
import { Login } from '../pages/Login'
import { Settings } from '../pages/Settings'
import { SettingsCloudflare } from '../pages/SettingsCloudflare'
import { SettingsPassword } from '../pages/SettingsPassword'
import { SetupWizard } from '../pages/SetupWizard'

function AppLoadingState() {
  return (
    <div className="page-wrap flex min-h-screen items-center justify-center py-10">
      <div className="surface max-w-lg px-8 py-10 text-center">
        <div className="badge-info">Emaildash</div>
        <h1 className="mt-5 text-3xl font-semibold tracking-tight text-white">Preparing workspace</h1>
        <p className="mt-3 text-sm leading-6 text-slate-300">
          Checking setup state, restoring session, and loading dashboard shell.
        </p>
        <div className="mt-8 grid gap-3 text-left">
          <div className="skeleton h-5 w-32" />
          <div className="skeleton h-16 w-full" />
          <div className="skeleton h-16 w-full" />
        </div>
      </div>
    </div>
  )
}

function Root() {
  const [csrfToken, setCSRFTokenState] = useState('')
  const setupQuery = useQuery({ queryKey: ['setup-status'], queryFn: api.setupStatus })
  const meQuery = useQuery({ queryKey: ['me'], queryFn: api.me, retry: false })

  useEffect(() => {
    if (meQuery.data?.csrfToken) {
      setCSRFToken(meQuery.data.csrfToken)
      setCSRFTokenState(meQuery.data.csrfToken)
      return
    }
    if (meQuery.isError) {
      clearCSRFToken()
      setCSRFTokenState('')
    }
  }, [meQuery.data?.csrfToken, meQuery.isError])

  if (setupQuery.isLoading || meQuery.isLoading) {
    return <AppLoadingState />
  }

  if (!setupQuery.data?.initialized) {
    return <SetupWizard onReady={(token) => {
      setCSRFToken(token)
      setCSRFTokenState(token)
      void setupQuery.refetch()
      void meQuery.refetch()
    }} />
  }

  if (meQuery.error && !csrfToken) {
    return <Login onLogin={(token) => {
      setCSRFToken(token)
      setCSRFTokenState(token)
      void meQuery.refetch()
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
