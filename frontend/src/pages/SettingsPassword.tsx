import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../lib/api'

export function SettingsPassword() {
  const [oldPassword, setOldPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const mutation = useMutation({
    mutationFn: () => api.changePassword(oldPassword, newPassword),
    onSuccess: () => {
      setOldPassword('')
      setNewPassword('')
    },
  })

  return (
    <section className="surface px-6 py-6 sm:px-8 sm:py-8">
      <div className="section-label">Password</div>
      <h2 className="mt-3 text-2xl font-semibold text-white">Update dashboard password</h2>
      <p className="mt-3 max-w-2xl text-sm leading-6 text-slate-400">
        Rotate password used for dashboard access. Change succeeds only when current password matches active stored hash.
      </p>

      <div className="mt-8 grid gap-4 sm:max-w-xl">
        <div className="grid gap-2">
          <label className="field-label" htmlFor="current-password">Current password</label>
          <input id="current-password" className="field" placeholder="Enter current password" type="password" value={oldPassword} onChange={(e) => setOldPassword(e.target.value)} />
        </div>
        <div className="grid gap-2">
          <label className="field-label" htmlFor="new-password">New password</label>
          <input id="new-password" className="field" placeholder="Choose new password" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} />
        </div>
        <button className="btn-primary sm:w-fit sm:min-w-44" onClick={() => mutation.mutate()} disabled={mutation.isPending || !oldPassword || !newPassword}>
          {mutation.isPending ? 'Updating…' : 'Update password'}
        </button>
      </div>

      {mutation.isSuccess && <div className="notice-success mt-6">Password updated successfully.</div>}
      {mutation.isError && <div className="notice-error mt-6">{(mutation.error as Error).message}</div>}
      {!mutation.isSuccess && !mutation.isError && <div className="notice-info mt-6">No minimum length enforced. Use whatever password you prefer.</div>}
    </section>
  )
}
