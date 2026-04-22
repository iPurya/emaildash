import { useMemo, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api, setCSRFToken, type Zone } from '../lib/api'

type Props = { onReady: (csrfToken: string) => void }

type Stage = {
  id: string
  label: string
  description: string
  complete: boolean
  active: boolean
}

export function SetupWizard({ onReady }: Props) {
  const [password, setPassword] = useState('')
  const [email, setEmail] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [zones, setZones] = useState<Zone[]>([])
  const [selectedZone, setSelectedZone] = useState('')
  const [readyForCloudflare, setReadyForCloudflare] = useState(false)
  const [provisionedZone, setProvisionedZone] = useState<string>()

  const initMutation = useMutation({
    mutationFn: async () => {
      await api.initialize(password)
      const auth = await api.login(password)
      setCSRFToken(auth.csrfToken)
      setReadyForCloudflare(true)
      return auth
    },
  })

  const credentialsMutation = useMutation({
    mutationFn: () => api.saveCloudflareCredentials(email, apiKey),
    onSuccess: (data) => setZones(data.zones),
  })

  const provisionMutation = useMutation({
    mutationFn: () => api.provisionZone(selectedZone),
    onSuccess: () => {
      setProvisionedZone(selectedZone)
      onReady(initMutation.data?.csrfToken ?? '')
    },
  })

  const stages = useMemo<Stage[]>(() => {
    const hasZones = zones.length > 0
    return [
      {
        id: 'password',
        label: 'Create dashboard password',
        description: 'Initialize app and create first authenticated session.',
        complete: readyForCloudflare,
        active: !readyForCloudflare,
      },
      {
        id: 'credentials',
        label: 'Connect Cloudflare',
        description: 'Store account email and Global API key, then fetch available zones.',
        complete: hasZones,
        active: readyForCloudflare && !hasZones,
      },
      {
        id: 'domain',
        label: 'Select domain',
        description: 'Choose zone that should receive and route inbound mail.',
        complete: Boolean(selectedZone),
        active: hasZones && !selectedZone,
      },
      {
        id: 'deploy',
        label: 'Deploy receiver',
        description: 'Enable routing, publish worker, and point catch-all traffic at Emaildash.',
        complete: Boolean(provisionedZone),
        active: Boolean(selectedZone) && !provisionedZone,
      },
    ]
  }, [provisionedZone, readyForCloudflare, selectedZone, zones.length])

  const currentError = (initMutation.error || credentialsMutation.error || provisionMutation.error) as Error | null

  return (
    <div className="page-wrap py-8 sm:py-10">
      <div className="grid gap-6 xl:grid-cols-[320px_1fr]">
        <aside className="surface px-6 py-6 sm:px-7">
          <div className="badge-info">First-time setup</div>
          <h1 className="mt-4 text-3xl font-semibold tracking-tight text-white">Launch inbound email dashboard</h1>
          <p className="mt-3 text-sm leading-6 text-slate-300">
            Set dashboard password, connect Cloudflare, choose target zone, then deploy catch-all receiver.
          </p>

          <div className="mt-8 space-y-3">
            {stages.map((stage, index) => (
              <div
                key={stage.id}
                className={`rounded-2xl border px-4 py-4 ${stage.complete ? 'border-emerald-400/25 bg-emerald-500/10' : stage.active ? 'border-blue-400/25 bg-blue-500/10' : 'border-white/10 bg-white/5'}`}
              >
                <div className="flex items-center justify-between gap-3">
                  <div className="text-sm font-semibold text-white">
                    {index + 1}. {stage.label}
                  </div>
                  <span className={stage.complete ? 'badge-success' : stage.active ? 'badge-info' : 'badge-muted'}>
                    {stage.complete ? 'Done' : stage.active ? 'Now' : 'Next'}
                  </span>
                </div>
                <p className="mt-2 text-xs leading-5 text-slate-300">{stage.description}</p>
              </div>
            ))}
          </div>
        </aside>

        <section className="surface px-6 py-6 sm:px-8 sm:py-8">
          <div className="grid gap-8">
            <section className="rounded-3xl border border-white/10 bg-white/5 p-5 sm:p-6">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <div className="section-label">Step 1</div>
                  <h2 className="mt-2 text-2xl font-semibold text-white">Create dashboard password</h2>
                  <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">
                    Pick any password you want. After initialize, Emaildash logs in automatically so setup can continue without a page jump.
                  </p>
                </div>
                <span className={readyForCloudflare ? 'badge-success' : 'badge-muted'}>{readyForCloudflare ? 'Initialized' : 'Required'}</span>
              </div>

              <div className="mt-6 grid gap-4">
                <div className="grid gap-2">
                  <label className="field-label" htmlFor="setup-password">Dashboard password</label>
                  <input
                    id="setup-password"
                    className="field"
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Choose password"
                  />
                </div>
                <button className="btn-primary" onClick={() => initMutation.mutate()} disabled={initMutation.isPending || readyForCloudflare || !password}>
                  {initMutation.isPending ? 'Initializing…' : readyForCloudflare ? 'Password ready' : 'Initialize app'}
                </button>
                {readyForCloudflare && <div className="notice-success">Password saved. Auth session active. Cloudflare steps unlocked.</div>}
              </div>
            </section>

            <section className="rounded-3xl border border-white/10 bg-white/5 p-5 sm:p-6">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <div className="section-label">Step 2</div>
                  <h2 className="mt-2 text-2xl font-semibold text-white">Connect Cloudflare</h2>
                  <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">
                    Save Cloudflare account email and Global API key, then load zones available to that account.
                  </p>
                </div>
                <span className={zones.length > 0 ? 'badge-success' : readyForCloudflare ? 'badge-info' : 'badge-muted'}>
                  {zones.length > 0 ? `${zones.length} zones loaded` : readyForCloudflare ? 'Ready' : 'Locked'}
                </span>
              </div>

              <div className="mt-6 grid gap-4 sm:grid-cols-2">
                <div className="grid gap-2">
                  <label className="field-label" htmlFor="cf-email">Cloudflare account email</label>
                  <input id="cf-email" className="field" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="name@example.com" />
                </div>
                <div className="grid gap-2">
                  <label className="field-label" htmlFor="cf-api-key">Cloudflare Global API key</label>
                  <input id="cf-api-key" className="field" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="Paste Global API key" />
                </div>
              </div>
              <div className="mt-4 flex flex-wrap gap-3">
                <button
                  className="btn-secondary"
                  onClick={() => credentialsMutation.mutate()}
                  disabled={credentialsMutation.isPending || !readyForCloudflare || !email || !apiKey}
                >
                  {credentialsMutation.isPending ? 'Loading domains…' : 'Load domains'}
                </button>
                {!readyForCloudflare && <span className="badge-muted">Initialize app first</span>}
              </div>
            </section>

            <section className="rounded-3xl border border-white/10 bg-white/5 p-5 sm:p-6">
              <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <div className="section-label">Steps 3–4</div>
                  <h2 className="mt-2 text-2xl font-semibold text-white">Choose domain and deploy receiver</h2>
                  <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-400">
                    Pick zone, then enable Email Routing and deploy worker that forwards parsed mail into backend webhook.
                  </p>
                </div>
                <span className={provisionedZone ? 'badge-success' : selectedZone ? 'badge-info' : 'badge-muted'}>
                  {provisionedZone ? 'Provisioned' : selectedZone ? 'Zone selected' : 'Waiting'}
                </span>
              </div>

              {zones.length === 0 ? (
                <div className="empty-panel mt-6 min-h-[180px]">
                  <div className="text-lg font-semibold text-white">No zones loaded yet</div>
                  <p className="mt-2 max-w-md text-sm leading-6 text-slate-400">
                    Save Cloudflare credentials first. After domains load, pick one zone and deploy routing in one click.
                  </p>
                </div>
              ) : (
                <div className="mt-6 grid gap-4">
                  <div className="grid gap-2">
                    <label className="field-label" htmlFor="zone-select">Domain</label>
                    <select id="zone-select" className="field" value={selectedZone} onChange={(e) => setSelectedZone(e.target.value)}>
                      <option value="">Choose domain</option>
                      {zones.map((zone) => (
                        <option key={zone.id} value={zone.id}>{zone.name}</option>
                      ))}
                    </select>
                  </div>

                  <div className="grid gap-3 lg:grid-cols-2">
                    {zones.map((zone) => {
                      const isActive = selectedZone === zone.id
                      return (
                        <button
                          key={zone.id}
                          type="button"
                          onClick={() => setSelectedZone(zone.id)}
                          className={`rounded-2xl border px-4 py-4 text-left transition ${isActive ? 'border-blue-400/40 bg-blue-500/10' : 'border-white/10 bg-slate-950/50 hover:bg-white/5'}`}
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div>
                              <div className="text-sm font-semibold text-white">{zone.name}</div>
                              <div className="mt-1 text-xs text-slate-400">Account {zone.accountId}</div>
                            </div>
                            <span className={zone.selected ? 'badge-success' : isActive ? 'badge-info' : 'badge-muted'}>
                              {zone.selected ? 'Current' : isActive ? 'Selected' : zone.status || 'Available'}
                            </span>
                          </div>
                        </button>
                      )
                    })}
                  </div>

                  <button className="btn-success" onClick={() => provisionMutation.mutate()} disabled={!selectedZone || provisionMutation.isPending}>
                    {provisionMutation.isPending ? 'Deploying receiver…' : 'Enable routing and deploy worker'}
                  </button>

                  {provisionedZone && <div className="notice-success">Receiver provisioned. Dashboard ready for inbox traffic.</div>}
                </div>
              )}
            </section>

            {currentError && <div className="notice-error">{currentError.message}</div>}
          </div>
        </section>
      </div>
    </div>
  )
}
