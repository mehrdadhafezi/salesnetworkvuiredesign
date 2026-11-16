import { Suspense, useCallback, type ReactNode } from 'react'
import { Link, Outlet, useLocation, useNavigate } from 'react-router'
import { useSession, useSessionActions } from '../shared/auth/currentSession'
import { LoginPage } from '../shared/auth/LoginPage'
import { AppHeader, DsAppShell, MobileBottomNavigation, RoleNavigation, type DsLinkRenderer, type DsNavItem } from '../shared/ui/ds'
import { Button, StateBlock } from '../shared/ui/primitives'
import { destinationForPath, type NavAccess } from './navigation/access'
import { SAFE_DESTINATIONS } from './navigation/destinations'
import { NavigationAccessProvider, useNavigationAccess } from './navigation/NavigationAccess'
import { ROLE_CONTRACTS } from './navigation/roles'

/** DEV-only tool links (never in production builds; see routes.tsx). */
export interface DevNavItem { to: string; label: string }

export const NAV_LABEL = 'ناوبری اصلی'

/**
 * RM-E02 AppShell on the RM-E01 design system: skip link → header → role
 * navigation → main#ws → mobile bottom navigation. Business destinations come
 * only from the fail-closed resolver (server session features AND server
 * effective-access decisions); home and system status are always available.
 * The server still authorises every request: hiding a link is presentation.
 */
export function Shell({ devNav = [] }: { devNav?: DevNavItem[] }) {
  return (
    <NavigationAccessProvider>
      <ShellFrame devNav={devNav} />
    </NavigationAccessProvider>
  )
}

/** Navigation items: safe entry → authorised destinations (role-mapped first) → safe status → DEV tools. */
export function navItems(access: NavAccess, devNav: DevNavItem[]): DsNavItem[] {
  const { home, system } = SAFE_DESTINATIONS
  const entries = access.status === 'GRANTED' ? access.entries : []
  return [
    { id: home.id, href: home.to, label: home.label, mobileLabel: home.short, icon: home.icon, group: 'entry' },
    ...entries.map(e => ({ id: e.id, href: e.to, label: e.label, mobileLabel: e.short, icon: e.icon, group: e.placement })),
    { id: system.id, href: system.to, label: system.label, mobileLabel: system.short, icon: system.icon, group: 'safe' },
    ...devNav.map(d => ({ id: `dev:${d.to}`, href: d.to, label: `${d.label} · DEV`, icon: 'grid' as const, group: 'dev' })),
  ]
}

const renderLink: DsLinkRenderer = (item, props) => <Link to={item.href} {...props} />

function ShellFrame({ devNav }: { devNav: DevNavItem[] }) {
  const access = useNavigationAccess()
  const location = useLocation()
  const navigate = useNavigate()
  const items = navItems(access, devNav)
  const current = destinationForPath(items.map(i => ({ id: i.id, to: i.href, label: i.label })), location.pathname)
  // Link clicks route through <Link>; overflow-menu selections arrive without an event.
  const onNavigate = useCallback((item: DsNavItem, e?: unknown) => { if (!e) void navigate(item.href) }, [navigate])
  const crumb = access.status === 'GRANTED' && access.role.kind === 'SINGLE' ? `نقش: ${ROLE_CONTRACTS[access.role.role].label}` : undefined
  return (
    <DsAppShell
      header={<AppHeader brandMark="C" brandName="CRM" crumb={crumb} mobileTitle={current?.label ?? 'CRM'} actions={<Identity />} />}
      nav={<RoleNavigation items={items} currentId={current?.id} label={NAV_LABEL} onNavigate={onNavigate} renderLink={renderLink} />}
      bottomNav={<MobileBottomNavigation items={items} currentId={current?.id} onNavigate={onNavigate} renderLink={renderLink} />}
    >
      <Gate>
        <Suspense fallback={<StateBlock kind="loading" title="در حال بارگذاری…" />}>
          <Outlet />
        </Suspense>
      </Gate>
    </DsAppShell>
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
