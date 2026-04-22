import { useMemo, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { api, type CloudflareStatus, type Zone } from '../lib/api'

function statusBadge(status?: string) {
  const value = status?.toLowerCase() ?? ''
  if (value.includes('active') || value.includes('configured') || value.includes('enabled')) {
    return 'badge-success'
  }
  if (value.includes('pending') || value.includes('setup') || value.includes('provision')) {
    return 'badge-warning'
  }
  return 'badge-muted'
}

function formatStatus(status: CloudflareStatus) {
  return [
    {
      label: 'Email routing',
      value: status.emailRoutingEnabled ? status.emailRoutingStatus || 'Enabled' : 'Not enabled',
      tone: status.emailRoutingEnabled ? 'badge-success' : 'badge-warning',
      description: status.emailRoutingEnabled ? 'Cloudflare accepted email routing for selected zone.' : 'Routing not configured yet.',
    },
    {
      label: 'Catch-all route',
      value: status.catchAllEnabled ? status.catchAllDestination || 'Configured' : 'Not configured',
      tone: status.catchAllEnabled ? 'badge-success' : 'badge-warning',
      description: status.catchAllEnabled ? 'Inbound mail forwards into worker destination.' : 'No catch-all destination found.',
    },
    {
      label: 'Worker script',
      value: status.workerScriptName || 'Unknown',
      tone: status.workerScriptName ? 'badge-info' : 'badge-muted',
      description: 'Worker script expected to receive and sign parsed messages.',
    },
  ]
}

export function SettingsCloudflare() {
  const [email, setEmail] = useState('')
  const [apiKey, setApiKey] = useState('')
  const statusQuery = useQuery({ queryKey: ['cloudflare-status'], queryFn: api.cloudflareStatus, retry: false })
  const zonesQuery = useQuery({ queryKey: ['zones'], queryFn: api.zones, retry: false })
  const credentialsMutation = useMutation({ mutationFn: () => api.saveCloudflareCredentials(email, apiKey) })
  const provisionMutation = useMutation({ mutationFn: (zoneId: string) => api.provisionZone(zoneId) })

  const zones = useMemo<Zone[]>(() => credentialsMutation.data?.zones ?? zonesQuery.data?.zones ?? [], [credentialsMutation.data?.zones, zonesQuery.data?.zones])
  const status = provisionMutation.data ?? statusQuery.data
  const currentError = (credentialsMutation.error || provisionMutation.error || statusQuery.error || zonesQuery.error) as Error | null
  const hasCachedConfiguration = zones.length > 0 || Boolean(status)

  return (
    <section className="surface px-6 py-6 sm:px-8 sm:py-8">
      <div className="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div>
          <div className="section-label">Cloudflare</div>
          <h2 className="mt-3 text-2xl font-semibold text-white">Routing and worker status</h2>
          <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-400">
            Credentials stay stored server-side after save. After refresh, this page reloads cached zones and selected-zone status even though secret fields stay blank.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <span className={status ? 'badge-success' : 'badge-muted'}>{status ? 'Status available' : 'No status yet'}</span>
          <span className={zones.length > 0 ? 'badge-info' : 'badge-muted'}>{zones.length > 0 ? `${zones.length} zones cached` : 'No cached zones'}</span>
        </div>
      </div>

      {hasCachedConfiguration && <div className="notice-info mt-6">Saved Cloudflare configuration detected. Secret inputs stay blank after refresh by design, but cached domains and selected zone state remain available.</div>}

      <div className="mt-8 grid gap-6 xl:grid-cols-[1.05fr_0.95fr]">
        <div className="grid gap-6">
          <section className="rounded-3xl border border-white/10 bg-white/5 p-5 sm:p-6">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="section-label">Credentials</div>
                <h3 className="mt-2 text-xl font-semibold text-white">Connect account</h3>
                <p className="mt-2 text-sm leading-6 text-slate-400">
                  Save or replace stored credentials, then refresh available zones for this account.
                </p>
              </div>
              <span className={credentialsMutation.isSuccess || hasCachedConfiguration ? 'badge-success' : 'badge-muted'}>{credentialsMutation.isSuccess || hasCachedConfiguration ? 'Configured' : 'Input required'}</span>
            </div>

            <div className="mt-6 grid gap-4 sm:grid-cols-2">
              <div className="grid gap-2">
                <label className="field-label" htmlFor="settings-cf-email">Cloudflare account email</label>
                <input id="settings-cf-email" className="field" placeholder="Enter only to save or replace" value={email} onChange={(e) => setEmail(e.target.value)} />
              </div>
              <div className="grid gap-2">
                <label className="field-label" htmlFor="settings-cf-key">Cloudflare Global API key</label>
                <input id="settings-cf-key" className="field" placeholder="Enter only to save or replace" value={apiKey} onChange={(e) => setApiKey(e.target.value)} />
              </div>
            </div>
            <div className="mt-4 flex flex-wrap gap-3">
              <button className="btn-primary" onClick={() => credentialsMutation.mutate()} disabled={credentialsMutation.isPending || !email || !apiKey}>
                {credentialsMutation.isPending ? 'Loading domains…' : 'Save and load domains'}
              </button>
              <button className="btn-secondary" onClick={() => {
                void zonesQuery.refetch()
                void statusQuery.refetch()
              }} disabled={zonesQuery.isFetching || statusQuery.isFetching}>
                {zonesQuery.isFetching || statusQuery.isFetching ? 'Refreshing…' : 'Refresh cached state'}
              </button>
            </div>
          </section>

          <section className="rounded-3xl border border-white/10 bg-white/5 p-5 sm:p-6">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="section-label">Zones</div>
                <h3 className="mt-2 text-xl font-semibold text-white">Provision target domain</h3>
                <p className="mt-2 text-sm leading-6 text-slate-400">
                  Choose zone to enable Email Routing, publish worker, and bind catch-all forwarding.
                </p>
              </div>
              <span className={zones.length > 0 ? 'badge-info' : 'badge-muted'}>{zones.length > 0 ? 'Ready to provision' : 'Load domains first'}</span>
            </div>

            {zones.length === 0 ? (
              <div className="empty-panel mt-6 min-h-[180px]">
                <div className="text-lg font-semibold text-white">No zones available</div>
                <p className="mt-2 max-w-md text-sm leading-6 text-slate-400">
                  If credentials were saved before, use refresh cached state. Otherwise save valid credentials to fetch domains.
                </p>
              </div>
            ) : (
              <div className="mt-6 grid gap-3">
                {zones.map((zone) => (
                  <div key={zone.id} className="rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-4">
                    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                      <div>
                        <div className="flex flex-wrap items-center gap-2">
                          <div className="text-sm font-semibold text-white">{zone.name}</div>
                          <span className={zone.selected ? 'badge-success' : statusBadge(zone.status)}>{zone.selected ? 'Current' : zone.status || 'Available'}</span>
                        </div>
                        <div className="mt-2 text-xs leading-5 text-slate-400">Account {zone.accountId}</div>
                      </div>
                      <button className="btn-secondary sm:min-w-40" onClick={() => provisionMutation.mutate(zone.id)} disabled={provisionMutation.isPending}>
                        {provisionMutation.isPending ? 'Provisioning…' : zone.selected ? 'Re-provision' : 'Provision zone'}
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </section>
        </div>

        <section className="rounded-3xl border border-white/10 bg-white/5 p-5 sm:p-6">
          <div className="section-label">Current status</div>
          <h3 className="mt-2 text-xl font-semibold text-white">Selected zone health</h3>
          <p className="mt-2 text-sm leading-6 text-slate-400">
            Readable snapshot of current routing state for selected zone.
          </p>

          {!status ? (
            <div className="empty-panel mt-6 min-h-[260px]">
              <div className="text-lg font-semibold text-white">No status loaded</div>
              <p className="mt-2 max-w-md text-sm leading-6 text-slate-400">
                If a zone is already provisioned, refresh cached state. If not, provision one from zone list to populate live status.
              </p>
            </div>
          ) : (
            <div className="mt-6 grid gap-4">
              <div className="rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-4">
                <div className="text-xs font-semibold uppercase tracking-[0.24em] text-slate-400">Selected zone</div>
                <div className="mt-3 text-lg font-semibold text-white">{status.zoneName || 'Unknown zone'}</div>
                <div className="mt-1 text-sm text-slate-400">Zone ID {status.zoneId || '—'}</div>
              </div>

              {formatStatus(status).map((item) => (
                <div key={item.label} className="rounded-2xl border border-white/10 bg-slate-950/50 px-4 py-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="text-sm font-semibold text-white">{item.label}</div>
                      <div className="mt-1 text-sm leading-6 text-slate-400">{item.description}</div>
                    </div>
                    <span className={item.tone}>{item.value}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </section>
      </div>

      {currentError && <div className="notice-error mt-6">{currentError.message}</div>}
      {!currentError && (credentialsMutation.isSuccess || provisionMutation.isSuccess) && (
        <div className="notice-success mt-6">Cloudflare configuration updated successfully.</div>
      )}
    </section>
  )
}
