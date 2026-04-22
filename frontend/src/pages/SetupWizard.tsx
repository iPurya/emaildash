import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api, type Zone } from '../lib/api'

type Props = { onReady: () => void }

export function SetupWizard({ onReady }: Props) {
  const [password, setPassword] = useState('')
  const [email, setEmail] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [zones, setZones] = useState<Zone[]>([])
  const [selectedZone, setSelectedZone] = useState('')

  const initMutation = useMutation({ mutationFn: () => api.initialize(password) })
  const credentialsMutation = useMutation({ mutationFn: () => api.saveCloudflareCredentials(email, apiKey), onSuccess: (data) => setZones(data.zones) })
  const provisionMutation = useMutation({ mutationFn: () => api.provisionZone(selectedZone), onSuccess: onReady })

  return (
    <div className="mx-auto grid max-w-3xl gap-6 rounded-3xl border border-slate-800 bg-slate-900/80 p-8 shadow-2xl shadow-slate-950/40">
      <div>
        <h1 className="text-3xl font-semibold text-white">Zero-hassle inbound email.</h1>
        <p className="mt-2 text-slate-300">Create password, connect Cloudflare, pick domain, deploy receiver.</p>
      </div>
      <div className="grid gap-3">
        <label className="text-sm text-slate-300">Dashboard password</label>
        <input className="rounded-xl border border-slate-700 bg-slate-950 px-4 py-3" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        <button className="rounded-xl bg-blue-600 px-4 py-3 font-medium" onClick={() => initMutation.mutate()} disabled={initMutation.isPending}>Initialize app</button>
      </div>
      <div className="grid gap-3 border-t border-slate-800 pt-6">
        <label className="text-sm text-slate-300">Cloudflare account email</label>
        <input className="rounded-xl border border-slate-700 bg-slate-950 px-4 py-3" value={email} onChange={(e) => setEmail(e.target.value)} />
        <label className="text-sm text-slate-300">Cloudflare Global API Key</label>
        <input className="rounded-xl border border-slate-700 bg-slate-950 px-4 py-3" value={apiKey} onChange={(e) => setApiKey(e.target.value)} />
        <button className="rounded-xl bg-slate-100 px-4 py-3 font-medium text-slate-950" onClick={() => credentialsMutation.mutate()} disabled={credentialsMutation.isPending}>Load domains</button>
      </div>
      {zones.length > 0 && (
        <div className="grid gap-3 border-t border-slate-800 pt-6">
          <label className="text-sm text-slate-300">Select domain</label>
          <select className="rounded-xl border border-slate-700 bg-slate-950 px-4 py-3" value={selectedZone} onChange={(e) => setSelectedZone(e.target.value)}>
            <option value="">Choose domain</option>
            {zones.map((zone) => <option key={zone.id} value={zone.id}>{zone.name}</option>)}
          </select>
          <button className="rounded-xl bg-emerald-600 px-4 py-3 font-medium" onClick={() => provisionMutation.mutate()} disabled={!selectedZone || provisionMutation.isPending}>Enable routing and deploy worker</button>
        </div>
      )}
      {(initMutation.error || credentialsMutation.error || provisionMutation.error) && (
        <div className="rounded-xl border border-red-500/40 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {(initMutation.error || credentialsMutation.error || provisionMutation.error as Error).message}
        </div>
      )}
    </div>
  )
}
