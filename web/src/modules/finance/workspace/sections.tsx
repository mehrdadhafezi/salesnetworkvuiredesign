import { Link, useLocation } from 'react-router'
import { destinationForPath } from '../../../app/navigation/access'
import { useNavigationAccess } from '../../../app/navigation/NavigationAccess'
import { ROLE_CONTRACTS } from '../../../app/navigation/roles'
import { StatusPill } from '../../../shared/ui/primitives'
import './workspace.css'

/**
 * RM-P07 Finance workspace sections. `via` lists the destinations.ts ids
 * whose server feature switch AND ALLOWED effective-access decision gate the
 * section (any one suffices; every page still renders the server's own
 * answer per read). No new global destination and no second permission list:
 *   - invoices        → billing.invoices (billing_read + billing.invoice.read)
 *   - legacy attempts → billing.attempts (billing_read + payments.attempt.read)
 *   - review queue    → finance.review   (finance_review_read + finance.review.read)
 *   - receipts        → billing.report (billing.reconciliation.read) or payments.evidence (payments.evidence.read)
 *   - exceptions      → any of invoices / attempts / review (each queue reads its own module)
 * The landing is listed whenever any section is, because each of its blocks
 * reads independently and shows its own denied / disabled state.
 */
export const FINANCE_SECTIONS = [
  { id: 'finance.home', via: ['billing.invoices', 'billing.attempts', 'finance.review', 'billing.report', 'payments.evidence', 'finance.dashboard'], to: '/finance', label: 'میز کار مالی' },
  { id: 'finance.invoices', via: ['billing.invoices'], to: '/billing/invoices', label: 'فاکتورها' },
  { id: 'finance.attempts', via: ['billing.attempts'], to: '/billing/attempts', label: 'تلاش‌های پرداخت (قدیمی)' },
  { id: 'finance.review', via: ['finance.review'], to: '/finance/review', label: 'صف بررسی و پرداخت‌های ثبت‌شده' },
  { id: 'finance.receipts', via: ['billing.report', 'payments.evidence'], to: '/finance/receipts', label: 'شواهد رسید' },
  { id: 'finance.exceptions', via: ['billing.invoices', 'billing.attempts', 'finance.review'], to: '/finance/exceptions', label: 'استثناها و شواهد حل‌نشده' },
] as const

export type FinanceSectionId = (typeof FINANCE_SECTIONS)[number]['id']

/** Destination ids the shell's fail-closed resolver allows (presentation only). */
export function useGrantedDestinations(): Set<string> {
  const access = useNavigationAccess()
  return new Set(access.status === 'GRANTED' ? access.entries.map(e => e.id) : [])
}

/** Finance workspace sections whose gating destination is allowed (presentation only). */
export function useFinanceSections(): Set<string> {
  const granted = useGrantedDestinations()
  return new Set(FINANCE_SECTIONS.filter(s => s.via.some(v => granted.has(v))).map(s => s.id))
}

export const FINANCE_NAV_LABEL = 'بخش‌های میز کار مالی'

export const FINANCE_NOTE = 'فقط خواندنی. ثبت پرداخت، فیش بارگذاری‌شده یا پذیرش قدیمی به معنای وصول قطعی وجه، تسویه یا ثبت حسابداری نیست؛ مبالغ با واحد خود نمایش داده می‌شوند و هرگز با هم جمع یا تبدیل نمی‌شوند.'

/**
 * RM-P07 section navigation. A section is listed only when one of its
 * destinations is allowed by the shell's resolver; the current section
 * always stays visible. The Finance role is not live verified and its
 * design is not frozen; both facts are stated on every workspace page.
 */
export function FinanceSectionNav({ note = true }: { note?: boolean }) {
  const granted = useFinanceSections()
  const { pathname } = useLocation()
  const current = destinationForPath([...FINANCE_SECTIONS], pathname)
  const items = FINANCE_SECTIONS.filter(s => s.id === current?.id || granted.has(s.id))
  const design = ROLE_CONTRACTS.finance.design.status
  return (
    <div className="fw-ws">
      <nav className="fw-ws-nav" aria-label={FINANCE_NAV_LABEL}>
        <ul>
          {items.map(s => (
            <li key={s.id}><Link to={s.to} aria-current={s.id === current?.id ? 'page' : undefined}>{s.label}</Link></li>
          ))}
        </ul>
      </nav>
      <p className="fw-ws-meta muted">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
        <StatusPill tone="slate" icon="alert" label={design === 'FROZEN' ? 'طراحی مالی: منجمد' : 'طراحی مالی: آماده بازبینی (منجمد نشده)'} />
        <span>اجرای نقش مالی با حساب واقعی تأیید زنده نشده است؛ آزمون‌های مجوز ساختگی، شاهد تولید نیستند.</span>
      </p>
      {note ? <p className="muted fw-note" role="note">{FINANCE_NOTE}</p> : null}
    </div>
  )
}
