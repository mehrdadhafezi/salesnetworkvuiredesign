import type { ReactNode } from 'react'
import {
  DsIcon, MenuButton, StatePanel, StatusIndicator, TonePill, formatFaNumber, toFaDigits, type DsIconName, type MenuEntry,
} from '../../../shared/ui/ds'
import type { Tone } from '../../../shared/ui/primitives'
import { INVOICE_STATUS, LEAD_STATUS } from '../labels'
import type { Page } from '../state/selectors'
import { useSellerPreview } from '../state/SellerPreviewContext'
import { Restricted } from './states'
import type { Invoice, Lead, LeadEvent, LeadNext, MoneyUnit } from '../types'

/*
 * Small Seller-domain presentation pieces of the frozen prototype (prototype/app.js money(),
 * pill(), nextInfo(), tfoot(), toolbar chips, timeline, notes). Built on RM-E01 primitives;
 * every value is synthetic and no element performs a business action.
 */

export const UNIT_LABEL: Record<MoneyUnit, string> = { IRT: 'تومان', IRR: 'ریال' }

/** Amount with its unit (SN-207 §4). null = unknown → «—» with an accessible reason, never 0. */
export function Money({ value, strong = false, unit = 'IRT' }: { value: number | null; strong?: boolean; unit?: MoneyUnit | undefined }) {
  if (value === null) return <span className="sl-amount sl-unknown"><span aria-hidden="true">—</span><span className="sr">مبلغ نامعلوم</span></span>
  return (
    <span className={`sl-amount${strong ? ' sl-strong' : ''}`} data-unit={unit}>
      <span className="num">{formatFaNumber(value)}</span><small>{UNIT_LABEL[unit]}</small>
    </span>
  )
}

export function Fa({ children }: { children: string | number }) {
  return <>{toFaDigits(children)}</>
}

/** A status the answer could not map onto the known vocabulary (A7 `unknown`): «نامعلوم», never a guess. */
function useStatusUnknown(): boolean {
  return useSellerPreview().ui.policy.statusUnknown
}

export function LeadStatusPill({ lead }: { lead: Pick<Lead, 'status'> }) {
  const p = LEAD_STATUS[lead.status]
  if (useStatusUnknown()) return <StatusIndicator state="unknown" />
  return <TonePill tone={p.tone} icon={p.icon} label={p.label} />
}

export function InvoiceStatusPill({ inv }: { inv: Pick<Invoice, 'status'> }) {
  const p = INVOICE_STATUS[inv.status]
  if (useStatusUnknown()) return <StatusIndicator state="unknown" />
  return <TonePill tone={p.tone} icon={p.icon} label={p.label} />
}

/** Any module status pill under the A7 unknown-mapping state. */
export function StatusPill({ tone, icon, label }: { tone: Tone; icon: DsIconName; label: string }) {
  if (useStatusUnknown()) return <StatusIndicator state="unknown" />
  return <TonePill tone={tone} icon={icon} label={label} />
}

/** Customer display name under the field policy (A7 `field_denied` → «محدود», no partial value). */
export function useCustomerName(): (name: string) => string {
  const restricted = useSellerPreview().ui.policy.fieldsRestricted
  return (name: string) => (restricted ? 'مشتری (محدود)' : name)
}

const NEXT_ICON: Record<LeadNext['kind'], DsIconName> = { callback: 'clock', attempts: 'phoneOff', archive: 'alert', invoice: 'file' }

/** Next step: amber = needs attention soon, red = due now / at risk (prototype nextInfo()). */
export function NextStep({ next }: { next: LeadNext | null }) {
  if (!next) return <span className="muted">—</span>
  const cls = next.due ? 'sl-due' : next.warn ? 'sl-warn' : ''
  return <span className={`sl-next ${cls}`}><DsIcon name={NEXT_ICON[next.kind]} />{next.text}</span>
}

export function nextChipClass(next: LeadNext | null): string {
  if (!next) return ''
  return next.due ? 'sl-due' : next.warn ? 'sl-warn' : ''
}

/** Name + discovery-only phone with a copy shortcut (copy is a demo toast; nothing touches the clipboard). */
export function Who({ name, phone, sub, copy = true }: { name: string; phone?: string | undefined; sub?: ReactNode; copy?: boolean }) {
  const { toast, ui } = useSellerPreview()
  if (ui.policy.fieldsRestricted) {
    return (
      <div className="sl-who">
        <Restricted what="نام مشتری" />
        {phone ? <span className="sl-phone"><Restricted what="شماره مشتری" /></span> : sub ? <span className="sl-phone">{sub}</span> : null}
      </div>
    )
  }
  return (
    <div className="sl-who">
      <b>{name}</b>
      {phone ? (
        <span className="sl-phone">
          <span className="mono">{toFaDigits(phone)}</span>
          {copy ? (
            <button type="button" className="sl-copy" aria-label={`کپی شماره ${name}`} title="کپی شماره"
              onClick={e => { e.stopPropagation(); toast(`نمایشی: شماره ${toFaDigits(phone)} کپی نشد — پیش‌نمایش به کلیپ‌بورد دست نمی‌زند.`) }}>
              <DsIcon name="copy" />
            </button>
          ) : null}
        </span>
      ) : sub ? <span className="sl-phone">{sub}</span> : null}
    </div>
  )
}

export function Tag({ children, teal = false }: { children: ReactNode; teal?: boolean }) {
  return <span className={`sl-tag${teal ? ' sl-teal' : ''}`}>{children}</span>
}

export function Timeline({ events, label }: { events: LeadEvent[]; label: string }) {
  if (!events.length) return <p className="muted">رویدادی ثبت نشده است.</p>
  return (
    <ul className="sl-timeline" aria-label={label}>
      {events.map((e, i) => <li key={i}>{e.text}<span>{e.meta}</span></li>)}
    </ul>
  )
}

export function Note({ tone, icon, children }: { tone: 'warn' | 'info' | 'danger'; icon: DsIconName; children: ReactNode }) {
  return <div className={`sl-note sl-note-${tone}`}><DsIcon name={icon} /><span>{children}</span></div>
}

export function FactChip({ icon, children, cls = '' }: { icon: DsIconName; children: ReactNode; cls?: string }) {
  return <span className={`sl-fact${cls ? ` ${cls}` : ''}`}><DsIcon name={icon} />{children}</span>
}

/** Ownership facts (SN-207 §6): fixed vocabulary, one cell per fact, never merged into «مالک». */
export function OwnershipFacts({ rows }: { rows: { term: string; value: ReactNode }[] }) {
  return (
    <dl className="sl-own">
      {rows.map(r => <div key={r.term}><dt>{r.term}</dt><dd>{r.value}</dd></div>)}
    </dl>
  )
}

/** Pagination footer (prototype tfoot()). RTL: «›» is previous, «‹» is next. */
export function TableFoot({ page, onPage, perPage }: { page: Page<unknown>; onPage: (p: number) => void; perPage: number }) {
  const absent = useSellerPreview().ui.policy.countsAbsent
  return (
    <div className="sl-tfoot">
      {absent
        ? <span>نمایش <b className="num">{toFaDigits(page.total ? `${page.from}–${page.to}` : '0')}</b> از <b className="num" aria-hidden="true">—</b><span className="sr">تعداد کل نامعلوم</span></span>
        : <span>نمایش <b className="num">{toFaDigits(page.total ? `${page.from}–${page.to}` : '0')}</b> از <b className="num">{toFaDigits(page.total)}</b></span>}
      <nav className="sl-pager" aria-label="صفحه‌بندی">
        <button type="button" aria-label="صفحه قبلی" disabled={page.page <= 1} onClick={() => onPage(page.page - 1)}>›</button>
        {Array.from({ length: page.pages }, (_, i) => i + 1).map(n => (
          <button key={n} type="button" aria-current={n === page.page ? 'page' : undefined} aria-label={`صفحه ${toFaDigits(n)}`} onClick={() => onPage(n)}>{toFaDigits(n)}</button>
        ))}
        <button type="button" aria-label="صفحه بعدی" disabled={page.page >= page.pages} onClick={() => onPage(page.page + 1)}>‹</button>
      </nav>
      <span className="sl-desk">{toFaDigits(perPage)} در صفحه</span>
    </div>
  )
}

/** Toolbar filter chip that opens a local option menu; the chip shows the active choice. */
export function FilterChip<T extends string>({ label, icon = 'filter', value, options, onChange }: {
  label: string
  icon?: DsIconName
  value: T | null
  options: { id: T; label: string }[]
  onChange: (v: T | null) => void
}) {
  const current = options.find(o => o.id === value)
  const entries: MenuEntry[] = [
    { kind: 'item', id: '__all', label: 'همه', current: value === null, onSelect: () => onChange(null) },
    { kind: 'separator', id: '__s' },
    ...options.map<MenuEntry>(o => ({ kind: 'item', id: o.id, label: o.label, current: o.id === value, onSelect: () => onChange(o.id) })),
  ]
  return (
    <MenuButton label={current ? `${label}: ${current.label}` : label} className={`ds-chip-btn${current ? ' sl-chip-on' : ''}`} entries={entries}>
      <DsIcon name={icon} /><span>{current ? `${label}: ${current.label}` : label}</span>
    </MenuButton>
  )
}

export function SearchField({ id, label, placeholder, value, onChange }: { id: string; label: string; placeholder: string; value: string; onChange: (v: string) => void }) {
  return (
    <label className="sl-field" htmlFor={id}>
      <span className="sr">{label}</span>
      <DsIcon name="filter" />
      <input id={id} className="input" type="search" placeholder={placeholder} value={value} aria-keyshortcuts="F"
        onChange={e => onChange(e.target.value)}
        onKeyDown={e => { if (e.key === 'Escape' && value) { e.preventDefault(); e.stopPropagation(); onChange('') } }} />
    </label>
  )
}

/**
 * Shared list-area state (prototype applyTableState): loading / empty / noresult. Returns null when the
 * list itself should render. Blocking answers (denied, disabled, expired, failure) never reach here:
 * ScreenFrame renders them instead of the screen body. An exact empty answer arrives as zero rows.
 */
export function useListState({ count, emptyTitle, emptyDetail, noResult, onClear }: {
  count: number
  emptyTitle: string
  emptyDetail: string
  /** True when filters/search are active, so an empty list is "no result" rather than "nothing exists". */
  noResult: boolean
  onClear?: (() => void) | undefined
  /** Kept for call-site compatibility; retry now lives in ScreenFrame (failure only). */
  onRetry?: (() => void) | undefined
}): ReactNode {
  const { sim } = useSellerPreview()
  if (sim === 'loading') return <StatePanel kind="loading" title="در حال بارگذاری فهرست" />
  if (count > 0) return null
  if (noResult) return <StatePanel kind="noresult" title="نتیجه‌ای پیدا نشد" detail="جستجو یا فیلترها همه موارد را کنار گذاشتند." actions={onClear ? <button type="button" className="btn btn-soft" onClick={onClear}>پاک کردن جستجو و فیلترها</button> : undefined} />
  return <StatePanel kind="empty" title={emptyTitle} detail={emptyDetail} />
}

export function toneVars(tone: Tone): Record<string, string> {
  return { '--c': `var(--${tone}-dot)` }
}
