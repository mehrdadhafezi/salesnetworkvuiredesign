import { Link, useLocation } from 'react-router'
import { destinationForPath } from '../../../app/navigation/access'
import { useNavigationAccess } from '../../../app/navigation/NavigationAccess'
import { ROLE_CONTRACTS } from '../../../app/navigation/roles'
import '../mis.css'
import { StatusPill } from '../../../shared/ui/primitives'
import { MIS_NOTE } from '../labels'

/**
 * Workspace sections. `via` is the destinations.ts id whose server feature
 * switch and capability decision gate the section: the inventory reads
 * GET /api/v1/cases (sales.cases: sales_read + cases.case.read); custody and
 * exceptions read GET /api/v1/custody/assignments (sales.custody: sales_read +
 * custody.custody.read). No second permission list and no new global
 * destination: a seller who may read cases is not offered an MIS link.
 */
const SECTIONS = [
  { id: 'mis.today', via: 'mis.today', to: '/mis', label: 'امروز MIS' },
  { id: 'mis.cases', via: 'sales.cases', to: '/mis/cases', label: 'موجودی پرونده‌ها' },
  { id: 'mis.custody', via: 'sales.custody', to: '/mis/custody', label: 'نگهداری و انتساب' },
  { id: 'mis.exceptions', via: 'sales.custody', to: '/mis/exceptions', label: 'استثناها و تطبیق' },
  { id: 'mis.quality', via: 'mis.quality', to: '/mis/source-quality', label: 'کیفیت منبع' },
  { id: 'mis.reconciliation', via: 'mis.reconciliation', to: '/mis/reconciliation', label: 'تطبیق و پوشش' },
] as const

/** Ids of the MIS sections the shell's fail-closed resolver allows (presentation only). */
export function useMisSections(): Set<string> {
  const access = useNavigationAccess()
  const granted = new Set(access.status === 'GRANTED' ? access.entries.map(e => e.id) : [])
  return new Set(SECTIONS.filter(s => granted.has(s.via)).map(s => s.id))
}

export const MIS_NAV_LABEL = 'بخش‌های فضای کار MIS'

/**
 * RM-P05 MIS workspace section navigation. A section is listed only when
 * the shell's fail-closed resolver lists its destination (server feature
 * switch AND an ALLOWED effective-access decision); the current section
 * always stays visible. Presentation only: every page renders the server's
 * own answer, including denied and disabled states.
 */
export function MISSectionNav({ note = true }: { note?: boolean }) {
  const granted = useMisSections()
  const { pathname } = useLocation()
  const current = destinationForPath([...SECTIONS], pathname)
  const items = SECTIONS.filter(s => s.id === current?.id || granted.has(s.id))
  const design = ROLE_CONTRACTS.mis.design.status
  return (
    <div className="mis-ws">
      <nav className="mis-ws-nav" aria-label={MIS_NAV_LABEL}>
        <ul>
          {items.map(s => (
            <li key={s.id}>
              <Link to={s.to} aria-current={s.id === current?.id ? 'page' : undefined}>{s.label}</Link>
            </li>
          ))}
        </ul>
      </nav>
      <p className="mis-ws-meta muted">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
        <StatusPill tone="slate" icon="alert" label={design === 'FROZEN' ? 'طراحی MIS: منجمد' : 'طراحی MIS: آماده بازبینی (منجمد نشده)'} />
        <span>اجرای نقش MIS با حساب واقعی تأیید زنده نشده است؛ نقش MIS امروز هیچ قابلیتی ندارد.</span>
      </p>
      {note ? <p className="muted mis-note" role="note">{MIS_NOTE}</p> : null}
    </div>
  )
}
