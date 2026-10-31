import { Suspense, useState } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router'
import { useSession } from '../shared/auth/currentSession'
import { Icon } from '../shared/ui/icons'
import { StateBlock } from '../shared/ui/primitives'

export interface NavItem {
  to: string
  label: string
  dev?: boolean
  /** Shown only when the server session reports this feature enabled (navigation hint only). */
  feature?: 'hr_read'
}

/** AppShell (SN-207 §2, §5): skip link → header → nav → main#ws. */
export function Shell({ nav }: { nav: NavItem[] }) {
  const [open, setOpen] = useState(false)
  const location = useLocation()
  const session = useSession()
  const features = session.status === 'ready' ? session.session.features : undefined
  const visible = nav.filter(item => !item.feature || features?.[item.feature] === true)
  const [lastPath, setLastPath] = useState(location.pathname)
  if (lastPath !== location.pathname) { // close the mobile menu after navigation
    setLastPath(location.pathname)
    setOpen(false)
  }
  return (
    <>
      <a className="skip-link" href="#ws">پرش به محتوای اصلی</a>
      <header className="hdr">
        <div className="brand">
          <span className="brand-mark" aria-hidden="true">C</span>
          <b>CRM</b>
        </div>
        <span className="grow" />
        <Identity />
        <button type="button" className="btn btn-ghost btn-sm nav-toggle" aria-expanded={open} aria-controls="main-nav"
          onClick={() => setOpen(o => !o)}>
          <Icon name="menu" /><span>منو</span>
        </button>
      </header>
      <nav id="main-nav" className="nav" aria-label="ناوبری اصلی" data-open={open}>
        {visible.map(item => (
          <NavLink key={item.to} to={item.to} end>
            {item.label}
            {item.dev ? <span className="demo-tag">DEV</span> : null}
          </NavLink>
        ))}
      </nav>
      <main id="ws" className="ws" tabIndex={-1}>
        <Suspense fallback={<StateBlock kind="loading" title="در حال بارگذاری…" />}>
          <Outlet />
        </Suspense>
      </main>
    </>
  )
}

function Identity() {
  const state = useSession()
  if (state.status === 'loading') return <span className="who muted">…</span>
  if (state.status === 'failed') return <span className="who muted">نشست نامشخص</span>
  const s = state.session
  if (s.authenticated && s.principal) {
    return (
      <span className="who">
        <span className="demo-tag">DEV · ساختگی</span>
        <span className="name">{s.principal.display_name}</span>
      </span>
    )
  }
  return <span className="who muted">{s.auth_mode === 'dev_stub' ? 'بدون هویت ساختگی' : 'ورود هنوز فعال نیست'}</span>
}
