import { useMutation, useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'

export function SettingsCloudflare() {
  const statusQuery = useQuery({ queryKey: ['cloudflare-status'], queryFn: api.cloudflareStatus })
  const zonesQuery = useQuery({ queryKey: ['zones'], queryFn: api.zones })
  const provisionMutation = useMutation({ mutationFn: (zoneId: string) => api.provisionZone(zoneId) })

  return (
    <div className="rounded-2xl border border-slate-800 bg-slate-900/70 p-6">
      <h2 className="text-xl font-semibold text-white">Cloudflare status</h2>
      <pre className="mt-4 overflow-auto rounded-xl bg-slate-950 p-4 text-xs text-slate-300">{JSON.stringify(statusQuery.data ?? {}, null, 2)}</pre>
      <div className="mt-6 space-y-2">
        {(zonesQuery.data?.zones ?? []).map((zone) => (
          <button key={zone.id} className="block w-full rounded-xl bg-slate-800 px-4 py-3 text-left" onClick={() => provisionMutation.mutate(zone.id)}>
            {zone.name}
          </button>
        ))}
      </div>
      {provisionMutation.error && <div className="mt-3 text-sm text-red-300">{(provisionMutation.error as Error).message}</div>}
    </div>
  )
}
