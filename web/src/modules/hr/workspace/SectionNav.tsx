import { Link, useLocation } from 'react-router'
import { destinationForPath } from '../../../app/navigation/access'
import { useNavigationAccess } from '../../../app/navigation/NavigationAccess'
import { ROLE_CONTRACTS } from '../../../app/navigation/roles'
import { useSession } from '../../../shared/auth/currentSession'
import { StatusPill } from '../../../shared/ui/primitives'

/** Workspace sections in HR IA order (HR-DESIGN-SPEC §2). Ids are destinations.ts ids. */
const SECTIONS = [
  { id: 'hr.workforce', to: '/hr', label: 'نیروها' },
  { id: 'hr.structure', to: '/hr/structure', label: 'ساختار سازمانی' },
  { id: 'hr.assignments', to: '/hr/assignments', label: 'انتساب‌ها و گزارش‌دهی' },
  { id: 'hr.positions', to: '/hr/positions', label: 'سمت‌ها و سطوح' },
  { id: 'hr.requests', to: '/hr/requests', label: 'درخواست‌ها' },
  { id: 'hr.exceptions', to: '/hr/exceptions', label: 'استثناها' },
] as const

const ACCESS = { id: 'hr.access', to: '/hr/access', label: 'دسترسی من' } as const

export const HR_NAV_LABEL = 'بخش‌های فضای کار منابع انسانی'

/**
 * RM-P04 HR workspace section navigation. A section is listed only when the
 * shell's fail-closed resolver lists its destination (server feature switch
 * AND an ALLOWED effective-access decision). "دسترسی من" shows the caller's
 * own evaluation and is offered to every authenticated principal. The
 * current section stays visible so the user always knows where they are.
 * Presentation only: every page renders the server's own answer.
 */
export function HRSectionNav() {
  const access = useNavigationAccess()
  const session = useSession()
  const { pathname } = useLocation()
  const granted = new Set(access.status === 'GRANTED' ? access.entries.map(e => e.id) : [])
  const authenticated = session.status === 'ready' && session.session.authenticated
  const all = [...SECTIONS, ACCESS]
  const current = destinationForPath(all, pathname)
  const items = all.filter(s => s.id === current?.id || (s.id === ACCESS.id ? authenticated : granted.has(s.id)))
  const design = ROLE_CONTRACTS.hr.design.status
  return (
    <div className="hr-ws">
      <nav className="hr-ws-nav" aria-label={HR_NAV_LABEL}>
        <ul>
          {items.map(s => (
            <li key={s.id}>
              <Link to={s.to} aria-current={s.id === current?.id ? 'page' : undefined}>{s.label}</Link>
            </li>
          ))}
        </ul>
      </nav>
      <p className="hr-ws-meta muted">
        <StatusPill tone="slate" icon="alert" label={design === 'FROZEN' ? 'طراحی منابع انسانی: منجمد' : 'طراحی منابع انسانی: آماده بازبینی (منجمد نشده)'} />
        <span> اجرای نقش منابع انسانی با حساب واقعی هنوز تأیید زنده نشده است.</span>
      </p>
    </div>
  )
}
