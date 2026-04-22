import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../lib/api'

type Props = { onLogin: (csrfToken: string) => void }

export function Login({ onLogin }: Props) {
  const [password, setPassword] = useState('')
  const mutation = useMutation({
    mutationFn: () => api.login(password),
    onSuccess: (data) => onLogin(data.csrfToken),
  })

  return (
    <div className="page-wrap flex min-h-screen items-center justify-center py-10">
      <div className="grid w-full max-w-5xl gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <section className="surface px-8 py-10 sm:px-10">
          <div className="badge-info">Emaildash</div>
          <h1 className="mt-5 text-4xl font-semibold tracking-tight text-white">Welcome back</h1>
          <p className="mt-4 max-w-xl text-base leading-7 text-slate-300">
            Open dashboard, review inbound mail, and manage Cloudflare routing without leaving this workspace.
          </p>
          <div className="mt-8 grid gap-4 sm:grid-cols-3">
            <div className="stat-card">
              <div className="section-label">One password</div>
              <div className="mt-2 text-sm leading-6 text-slate-300">Single-user auth. No signup maze. Fast re-entry.</div>
            </div>
            <div className="stat-card">
              <div className="section-label">Inbox first</div>
              <div className="mt-2 text-sm leading-6 text-slate-300">Recipient groups, message list, and viewer stay in one workflow.</div>
            </div>
            <div className="stat-card">
              <div className="section-label">Cloudflare ready</div>
              <div className="mt-2 text-sm leading-6 text-slate-300">Provision routing and worker deployment from same control panel.</div>
            </div>
          </div>
        </section>

        <section className="surface px-8 py-10 sm:px-10">
          <div className="section-label">Sign in</div>
          <h2 className="mt-3 text-2xl font-semibold tracking-tight text-white">Unlock dashboard</h2>
          <p className="mt-3 text-sm leading-6 text-slate-400">
            Use dashboard password created during setup. Session and CSRF token restore automatically after login.
          </p>

          <div className="mt-8 grid gap-4">
            <div className="grid gap-2">
              <label className="field-label" htmlFor="password">Password</label>
              <input
                id="password"
                className="field"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter dashboard password"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && password && !mutation.isPending) {
                    mutation.mutate()
                  }
                }}
              />
            </div>

            <button className="btn-primary" onClick={() => mutation.mutate()} disabled={mutation.isPending || !password}>
              {mutation.isPending ? 'Signing in…' : 'Log in'}
            </button>
          </div>

          {mutation.isError && <div className="notice-error mt-5">{(mutation.error as Error).message}</div>}
          {!mutation.isError && <div className="notice-info mt-5">If session expired, sign in again and app will reopen current workspace state.</div>}
        </section>
      </div>
    </div>
  )
}
