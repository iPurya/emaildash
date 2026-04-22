import { NavLink, Outlet, useLocation } from 'react-router-dom'

const navItems = [
  { to: '/inbox', label: 'Inbox', description: 'Review received mail' },
  { to: '/settings', label: 'Settings', description: 'Manage auth and Cloudflare' },
]

function pageMeta(pathname: string) {
  if (pathname.startsWith('/settings/cloudflare')) {
    return {
      eyebrow: 'Settings',
      title: 'Cloudflare connection',
      subtitle: 'Stored credentials stay server-side. Cached zones and selected routing status stay visible here after refresh.',
    }
  }
  if (pathname.startsWith('/settings/password')) {
    return {
      eyebrow: 'Settings',
      title: 'Security settings',
      subtitle: 'Rotate dashboard password and recover cleanly when sessions truly expire.',
    }
  }
  if (pathname.startsWith('/settings')) {
    return {
      eyebrow: 'Settings',
      title: 'Workspace settings',
      subtitle: 'Manage app security and Cloudflare provisioning from one predictable place.',
    }
  }
  return {
    eyebrow: 'Inbox',
    title: 'Inbound email dashboard',
    subtitle: 'Track recipients, auto-refresh new mail every 2 seconds, and inspect full content without hopping between screens.',
  }
}

export function AppShell() {
  const location = useLocation()
  const meta = pageMeta(location.pathname)

  return (
    <div className="min-h-screen text-slate-100">
      <div className="page-wrap py-6 sm:py-8">
        <div className="grid gap-6 xl:grid-cols-[280px_minmax(0,1fr)] xl:items-start">
          <aside className="surface px-5 py-5 sm:px-6 xl:sticky xl:top-8 xl:min-h-[calc(100vh-4rem)]">
            <div className="space-y-3">
              <div className="badge-info">Emaildash</div>
              <div>
                <div className="text-2xl font-semibold tracking-tight text-white">Operational inbox</div>
                <p className="mt-2 text-sm leading-6 text-slate-300">
                  Production mail capture, Cloudflare routing, and review tools in one workspace.
                </p>
              </div>
            </div>

            <nav className="mt-8 grid gap-3">
              {navItems.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) =>
                    `rounded-2xl border px-4 py-4 transition ${isActive ? 'border-blue-400/40 bg-blue-500/10 text-white shadow-lg shadow-blue-950/30' : 'border-white/10 bg-white/5 text-slate-300 hover:bg-white/10 hover:text-white'}`
                  }
                >
                  {({ isActive }) => (
                    <div>
                      <div className="flex items-center justify-between gap-3">
                        <span className="text-sm font-semibold">{item.label}</span>
                        <span className={isActive ? 'badge-info' : 'badge-muted'}>{isActive ? 'Open' : 'Go'}</span>
                      </div>
                      <div className="mt-1 text-xs leading-5 text-slate-400">{item.description}</div>
                    </div>
                  )}
                </NavLink>
              ))}
            </nav>
          </aside>

          <div>
            <section className="px-1 pb-8 pt-1 sm:px-2">
              <div className="page-eyebrow">{meta.eyebrow}</div>
              <h1 className="page-title mt-3">{meta.title}</h1>
              <p className="page-subtitle">{meta.subtitle}</p>
            </section>

            <main>
              <Outlet />
            </main>
          </div>
        </div>
      </div>
    </div>
  )
}
