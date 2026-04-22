import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../lib/api'

export function SettingsPassword() {
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const mutation = useMutation({ mutationFn: () => api.changePassword(oldPassword, newPassword) })

  return (
    <div className="rounded-2xl border border-slate-800 bg-slate-900/70 p-6">
      <h2 className="text-xl font-semibold text-white">Change password</h2>
      <div className="mt-4 grid gap-3">
        <input className="rounded-xl border border-slate-700 bg-slate-950 px-4 py-3" placeholder="Current password" type="password" value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} />
        <input className="rounded-xl border border-slate-700 bg-slate-950 px-4 py-3" placeholder="New password" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
        <button className="rounded-xl bg-blue-600 px-4 py-3 font-medium" onClick={() => mutation.mutate()} disabled={mutation.isPending}>Update password</button>
      </div>
      {mutation.isSuccess && <div className="mt-3 text-sm text-emerald-300">Password updated.</div>}
      {mutation.error && <div className="mt-3 text-sm text-red-300">{(mutation.error as Error).message}</div>}
    </div>
  )
}
