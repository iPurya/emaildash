import { NavLink, Outlet } from 'react-router-dom'

const items = [
  { to: 'password', label: 'Password', description: 'Update dashboard access' },
  { to: 'cloudflare', label: 'Cloudflare', description: 'Connect domains and routing' },
]

export function Settings() {
  return (
    <div className="grid gap-6 xl:grid-cols-[280px_1fr]">
      <aside className="surface px-5 py-5 sm:px-6">
        <div className="section-label">Settings</div>
        <h2 className="mt-3 text-xl font-semibold text-white">Configuration areas</h2>
        <p className="mt-2 text-sm leading-6 text-slate-400">
          Security controls and Cloudflare provisioning live here. Pick area to update.
        </p>

        <nav className="mt-6 grid gap-3 text-sm">
          {items.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              className={({ isActive }) =>
                `rounded-2xl border px-4 py-4 transition ${isActive ? 'border-blue-400/40 bg-blue-500/10 text-white' : 'border-white/10 bg-white/5 text-slate-300 hover:bg-white/10 hover:text-white'}`
              }
            >
              {({ isActive }) => (
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="font-semibold">{item.label}</div>
                    <div className="mt-1 text-xs leading-5 text-slate-400">{item.description}</div>
                  </div>
                  <span className={isActive ? 'badge-info' : 'badge-muted'}>{isActive ? 'Open' : 'View'}</span>
                </div>
              )}
            </NavLink>
          ))}
        </nav>
      </aside>
      <Outlet />
    </div>
  )
}
