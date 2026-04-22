import { Link, Outlet } from 'react-router-dom'

export function Settings() {
  return (
    <div className="grid gap-6 lg:grid-cols-[220px_1fr]">
      <aside className="rounded-2xl border border-slate-800 bg-slate-900/70 p-4">
        <div className="mb-3 text-sm font-semibold text-slate-300">Settings</div>
        <nav className="grid gap-2 text-sm">
          <Link to="password" className="rounded-xl bg-slate-800 px-3 py-2">Password</Link>
          <Link to="cloudflare" className="rounded-xl bg-slate-800 px-3 py-2">Cloudflare</Link>
        </nav>
      </aside>
      <Outlet />
    </div>
  )
}
