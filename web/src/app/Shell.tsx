import { Suspense, useState, type ReactNode } from 'react'
import { NavLink, Outlet, useLocation } from 'react-router'
import { useSession, useSessionActions } from '../shared/auth/currentSession'
import { LoginPage } from '../shared/auth/LoginPage'
import { Icon } from '../shared/ui/icons'
import { Button, StateBlock } from '../shared/ui/primitives'

export interface NavItem {
  to: string
  label: string
  dev?: boolean
  /** Shown only when the server session reports this feature enabled (navigation hint only). */
  feature?: 'hr_read' | 'customers_identity' | 'sales_read' | 'billing_read' | 'reporting_read' | 'finance_shadow_read' | 'biawin_fulfillment_read' | 'hr_request_write' | 'writeback_read' | 'payment_submission' | 'finance_review_read' | 'commission_read' | 'fulfillment_work'
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
        <Gate>
          <Suspense fallback={<StateBlock kind="loading" title="در حال بارگذاری…" />}>
            <Outlet />
          </Suspense>
        </Gate>
      </main>
    </>
  )
}

/**
 * R1 protected routes: in session mode every page needs an authenticated
 * session; otherwise the login page is shown (with an expiry notice when
 * the server reported the presented session invalid). The server still
 * authorises every request; this only avoids rendering pages that would
 * be denied.
 */
function Gate({ children }: { children: ReactNode }) {
  const state = useSession()
  const { refresh } = useSessionActions()
  if (state.status === 'loading') return <StateBlock kind="loading" title="در حال بررسی نشست…" />
  if (state.status === 'failed') {
    return <StateBlock kind="error" title="وضعیت نشست مشخص نشد" detail="ارتباط با سرور برقرار نشد."
      action={<Button onClick={refresh}>تلاش دوباره</Button>} />
  }
  const s = state.session
  if (s.auth_mode === 'session' && !s.authenticated) {
    return <LoginPage expired={s.session_state === 'invalid'} enabled={s.login_enabled === true} />
  }
  return <>{children}</>
}

function Identity() {
  const state = useSession()
  const { logout } = useSessionActions()
  if (state.status === 'loading') return <span className="who muted">…</span>
  if (state.status === 'failed') return <span className="who muted">نشست نامشخص</span>
  const s = state.session
  if (s.auth_mode === 'session') {
    if (!s.authenticated || !s.session) return <span className="who muted">وارد نشده‌اید</span>
    return (
      <span className="who">
        <span className="name">{s.session.display_name || 'کاربر'}</span>
        <Button small onClick={() => { void logout() }}>خروج</Button>
      </span>
    )
  }
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
