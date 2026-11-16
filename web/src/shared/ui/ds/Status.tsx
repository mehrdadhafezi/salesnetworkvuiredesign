import type { ReactNode } from 'react'
import type { Tone } from '../primitives'
import { countLabel, countText, type DsCount } from './format'
import { DsIcon, type DsIconName } from './icons'

/*
 * Status and state indicators (SN-207 §1.3, §1.4, §7, §15).
 * Every state = visible label + icon + tone; never colour alone.
 * UNKNOWN, CONFLICT, INCOMPLETE, NOT_EVALUATED, STALE and PARTIAL stay distinct
 * and never render as success, "none" or zero.
 * These are PRESENTATION states only: they are not business statuses and
 * carry no workflow meaning. Domain status dictionaries stay with their modules.
 */

export type DataState = 'unknown' | 'conflict' | 'incomplete' | 'not_evaluated' | 'stale' | 'partial'

interface StatePresentation { tone: Tone; icon: DsIconName; label: string; dashed?: boolean | undefined }

/** SN-207 §7 colour meaning: amber = attention/unknown · orange = conflict/partial · slate = inert/undefined. */
export const dataStatePresentation = {
  unknown: { tone: 'amber', icon: 'question', label: 'نامعلوم', dashed: true },
  conflict: { tone: 'orange', icon: 'swap', label: 'تعارض' },
  incomplete: { tone: 'amber', icon: 'layers', label: 'ناقص' },
  not_evaluated: { tone: 'slate', icon: 'dashed', label: 'ارزیابی نشده' },
  stale: { tone: 'amber', icon: 'clock', label: 'قدیمی' },
  partial: { tone: 'orange', icon: 'split', label: 'بخشی' },
} as const satisfies Record<DataState, StatePresentation>

/** Inline state pill. `label` may refine the wording; the kind (icon + tone) never changes. */
export function StatusIndicator({ state, label, detail }: { state: DataState; label?: string | undefined; detail?: string | undefined }) {
  const p: StatePresentation = dataStatePresentation[state]
  return (
    <span className={`pill t-${p.tone} ds-state-pill${p.dashed ? ' ds-dashed' : ''}`} data-state={state}>
      <DsIcon name={p.icon} />
      <span>{label ?? p.label}</span>
      {detail ? <span className="ds-pill-detail">· {detail}</span> : null}
    </span>
  )
}

/** Generic tone pill (label + icon + tone) for module dictionaries. */
export function TonePill({ tone, icon, label }: { tone: Tone; icon: DsIconName; label: string }) {
  return (
    <span className={`pill t-${tone}`}>
      <DsIcon name={icon} />
      <span>{label}</span>
    </span>
  )
}

/* ---------- StatePanel (stateBlock) ---------- */

export type PanelStateKind =
  | 'loading' | 'empty' | 'noresult' | 'error' | 'locked' | 'stale' | 'incomplete'
  | 'conflict' | 'partial' | 'retry' | 'unknown' | 'offline' | 'info'

const panelIcon: Record<Exclude<PanelStateKind, 'loading'>, DsIconName> = {
  empty: 'check', noresult: 'search', error: 'alert', locked: 'lock', stale: 'clock', incomplete: 'layers',
  conflict: 'swap', partial: 'split', retry: 'refresh', unknown: 'question', offline: 'wifiOff', info: 'info',
}

/**
 * Full-area system state (prototype stateBlock + skeleton loading).
 * error / conflict interrupt (role=alert); everything else is a polite status.
 */
export function StatePanel({ kind, title, detail, actions, headingLevel = 2 }: {
  kind: PanelStateKind
  title: string
  detail?: ReactNode | undefined
  actions?: ReactNode | undefined
  headingLevel?: 2 | 3 | undefined
}) {
  const role = kind === 'error' || kind === 'conflict' ? 'alert' : 'status'
  const H = headingLevel === 3 ? 'h3' : 'h2'
  if (kind === 'loading') {
    return (
      <div className="ds-state ds-state-loading" role="status" aria-busy="true" data-kind={kind}>
        <span className="ds-sk ds-sk-line" style={{ width: '40%' }} aria-hidden="true" />
        <span className="ds-sk ds-sk-line" style={{ width: '65%' }} aria-hidden="true" />
        <span className="ds-sk ds-sk-line" style={{ width: '52%' }} aria-hidden="true" />
        <span className="sr">{title}</span>
      </div>
    )
  }
  return (
    <div className={`ds-state ds-state-${kind}`} role={role} data-kind={kind}>
      <span className="ds-state-ico"><DsIcon name={panelIcon[kind]} /></span>
      <H className="ds-state-title">{title}</H>
      {detail ? <p className="ds-state-detail">{detail}</p> : null}
      {actions ? <div className="ds-state-acts">{actions}</div> : null}
    </div>
  )
}

/* ---------- Banner (ws-banner) ---------- */

export type BannerKind = 'stale' | 'incomplete' | 'locked' | 'conflict' | 'unknown' | 'offline' | 'info'

const bannerIcon: Record<BannerKind, DsIconName> = {
  stale: 'clock', incomplete: 'layers', locked: 'lock', conflict: 'swap', unknown: 'question', offline: 'wifiOff', info: 'info',
}

export function Banner({ kind, children, action }: { kind: BannerKind; children: ReactNode; action?: ReactNode | undefined }) {
  return (
    <div className={`ds-banner ds-banner-${kind}`} role={kind === 'conflict' ? 'alert' : 'status'} data-kind={kind}>
      <DsIcon name={bannerIcon[kind]} />
      <span className="ds-banner-text">{children}</span>
      {action}
    </div>
  )
}

/* ---------- Counts and metrics ---------- */

/** Count badge: Persian digits; unknown renders «—» with an accessible "unknown" name, never 0. */
export function CountBadge({ count, alert = false, className }: { count: DsCount; alert?: boolean | undefined; className?: string | undefined }) {
  const unknown = countText(count) === '—'
  const cls = ['ds-count', alert && 'ds-count-alert', unknown && 'ds-count-unknown', className].filter(Boolean).join(' ')
  return (
    <span className={cls}>
      <span aria-hidden="true">{countText(count)}</span>
      <span className="sr">{countLabel(count)}</span>
    </span>
  )
}

/** Metric value states (prototype kpiHtml): unavailable / undefined / unknown render «—» + reason, never zero. */
export type MetricState = 'ok' | 'stale' | 'unavailable' | 'undefined' | 'unknown'

const metricStateLabel: Record<Exclude<MetricState, 'ok'>, string> = {
  stale: 'قدیمی', unavailable: 'نامعتبر', undefined: 'تعریف نشده', unknown: 'نامعلوم',
}

export function Metric({ label, value, unit, state = 'ok', accent = 'neutral', meaning, tone }: {
  label: string
  /** Pre-formatted display value. Ignored unless state is 'ok' or 'stale'. */
  value?: string | undefined
  unit?: string | undefined
  state?: MetricState | undefined
  accent?: Tone | undefined
  /** Meaning / time basis shown as secondary text (MetricMeta content is supplied by the caller). */
  meaning?: string | undefined
  tone?: 'warn' | 'good' | 'bad' | undefined
}) {
  const hasValue = (state === 'ok' || state === 'stale') && value !== undefined && value !== ''
  const cls = ['ds-kpi', `ds-kpi-${state}`, tone && `ds-kpi-${tone}`].filter(Boolean).join(' ')
  return (
    <div className={cls} style={{ ['--k' as string]: `var(--${accent}-dot)` }} data-state={state}>
      <span className="ds-kpi-label">{label}</span>
      <b className="ds-kpi-value">
        {hasValue
          ? <><span className="num">{value}</span>{unit ? <small>{unit}</small> : null}</>
          : <span className="ds-na">—</span>}
        {state !== 'ok' ? <small className="ds-kstate">{metricStateLabel[state]}</small> : null}
        {!hasValue && state === 'ok' ? <small className="ds-kstate">{metricStateLabel.unknown}</small> : null}
      </b>
      {meaning ? <span className="ds-kpi-meaning">{meaning}</span> : null}
    </div>
  )
}

export function MetricGroup({ label, children }: { label: string; children: ReactNode }) {
  return <div className="ds-kpis" role="group" aria-label={label}>{children}</div>
}
