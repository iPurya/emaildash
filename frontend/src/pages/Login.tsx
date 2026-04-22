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
    <div className="mx-auto max-w-md rounded-3xl border border-slate-800 bg-slate-900/80 p-8">
      <h1 className="text-3xl font-semibold text-white">Welcome back</h1>
      <p className="mt-2 text-slate-300">Single password. No signup noise.</p>
      <div className="mt-6 grid gap-3">
        <input className="rounded-xl border border-slate-700 bg-slate-950 px-4 py-3" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Password" />
        <button className="rounded-xl bg-blue-600 px-4 py-3 font-medium" onClick={() => mutation.mutate()} disabled={mutation.isPending}>Log in</button>
      </div>
      {mutation.error && <div className="mt-4 text-sm text-red-300">{(mutation.error as Error).message}</div>}
    </div>
  )
}
