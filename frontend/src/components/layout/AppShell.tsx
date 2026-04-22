import { Link, Outlet } from 'react-router-dom'

export function AppShell() {
  return (
    <div className="min-h-screen text-slate-100">
      <header className="border-b border-slate-800 bg-slate-950/80 backdrop-blur">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
          <Link to="/inbox" className="text-lg font-semibold tracking-tight">Emaildash</Link>
          <nav className="flex gap-4 text-sm text-slate-300">
            <Link to="/inbox">Inbox</Link>
            <Link to="/settings">Settings</Link>
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-6 py-6">
        <Outlet />
      </main>
    </div>
  )
}
