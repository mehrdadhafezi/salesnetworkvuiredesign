import type { ReactNode } from 'react'
import { DsIcon } from './icons'

/*
 * Page layout primitives (SN-207 §5; prototype styles.css .hdr/.ws/.page-head/.panel/.toolbar).
 * Order: skip link → header → nav → main → drawer → bottom nav. One <h1> per view (PageHead).
 * Presentation only.
 */

export function DsAppShell({ header, nav, children, bottomNav, overlay, mainId = 'ws', skipLabel = 'پرش به محتوای اصلی' }: {
  header: ReactNode
  nav?: ReactNode | undefined
  children: ReactNode
  bottomNav?: ReactNode | undefined
  /** Drawer, toasts and other fixed layers. */
  overlay?: ReactNode | undefined
  mainId?: string | undefined
  skipLabel?: string | undefined
}) {
  return (
    <div className={`ds-app${bottomNav ? ' ds-has-bottom-nav' : ''}`}>
      <a className="skip-link" href={`#${mainId}`}>{skipLabel}</a>
      {header}
      {nav}
      <main id={mainId} className="ds-ws" tabIndex={-1}>{children}</main>
      {overlay}
      {bottomNav}
    </div>
  )
}

/** Responsive header: brand · role crumb · demo tag · search trigger · actions; ≤760px shows the page title. */
export function AppHeader({ brandMark, brandName, crumb, tag, mobileTitle, search, actions }: {
  brandMark: string
  brandName: string
  /** Current role / destination label (hidden ≤900px). */
  crumb?: string | undefined
  /** Environment tag, e.g. «نمونه نمایشی · داده ساختگی» (hidden ≤1200px). */
  tag?: string | undefined
  /** Title shown instead of the brand on phones. Not a heading (the page owns its h1). */
  mobileTitle?: string | undefined
  search?: { label: string; placeholder: string; shortcut?: string | undefined; onOpen: () => void } | undefined
  actions?: ReactNode | undefined
}) {
  return (
    <header className="ds-hdr">
      <div className="ds-brand">
        <span className="ds-brand-mark" aria-hidden="true">{brandMark}</span>
        <b>{brandName}</b>
        {crumb ? <span className="ds-crumb">{crumb}</span> : null}
        {tag ? <span className="ds-demo-tag">{tag}</span> : null}
      </div>
      {mobileTitle ? <div className="ds-m-title">{mobileTitle}</div> : null}
      {search ? (
        <button type="button" className="ds-search" aria-label={search.label} aria-haspopup="dialog" aria-keyshortcuts={search.shortcut ? 'Control+K' : undefined} onClick={search.onOpen}>
          <DsIcon name="search" />
          <span className="ds-ph">{search.placeholder}</span>
          {search.shortcut ? <span className="ds-kbd">{search.shortcut}</span> : null}
        </button>
      ) : <span className="ds-grow" />}
      <div className="ds-hdr-actions">
        {search ? (
          <button type="button" className="ds-icon-btn ds-search-btn" aria-label={search.label} aria-haspopup="dialog" onClick={search.onOpen}>
            <DsIcon name="search" />
          </button>
        ) : null}
        {actions}
      </div>
    </header>
  )
}

/** Header icon button with optional count dot (prototype .icon-btn/.dot-badge). */
export function HeaderIconButton({ label, icon, badge, onClick, pressed }: {
  label: string
  icon: Parameters<typeof DsIcon>[0]['name']
  /** Pre-formatted badge text; the accessible label must already include its meaning. */
  badge?: string | undefined
  onClick?: () => void
  pressed?: boolean | undefined
}) {
  return (
    <button type="button" className="ds-icon-btn" aria-label={label} title={label} onClick={onClick} aria-pressed={pressed}>
      <DsIcon name={icon} />
      {badge ? <span className="ds-dot-badge" aria-hidden="true">{badge}</span> : null}
    </button>
  )
}

export function PageHead({ title, sub, kpis, actions, meta }: {
  title: string
  sub?: ReactNode | undefined
  /** Usually a MetricGroup. */
  kpis?: ReactNode | undefined
  actions?: ReactNode | undefined
  /** Freshness / scope line under the head (prototype .head-meta). */
  meta?: ReactNode | undefined
}) {
  return (
    <>
      <div className="ds-page-head">
        <div>
          <h1>{title}</h1>
          {sub ? <div className="ds-sub">{sub}</div> : null}
        </div>
        <div className="ds-spacer" />
        {kpis}
        {actions ? <div className="ds-head-cta">{actions}</div> : null}
      </div>
      {meta ? <div className="ds-head-meta">{meta}</div> : null}
    </>
  )
}

/** Panel surface; give it a label when it is a meaningful region. */
export function Panel({ label, children, className }: { label?: string | undefined; children: ReactNode; className?: string | undefined }) {
  return <section className={`ds-panel${className ? ` ${className}` : ''}`} aria-label={label}>{children}</section>
}

export function PanelHeader({ title, icon, children }: { title: string; icon?: Parameters<typeof DsIcon>[0]['name'] | undefined; children?: ReactNode | undefined }) {
  return (
    <div className="ds-panel-h">
      <h2>{icon ? <DsIcon name={icon} /> : null}{title}</h2>
      {children}
    </div>
  )
}

/** Search + filter chips + right slot row above a list (not an ARIA toolbar: each control is its own tab stop). */
export function Toolbar({ children }: { children: ReactNode }) {
  return <div className="ds-toolbar">{children}</div>
}

/** Freshness line (prototype freshness()): «به‌روز …» or stale in amber with icon + text. */
export function Freshness({ text, stale = false, action }: { text: string; stale?: boolean | undefined; action?: ReactNode | undefined }) {
  return (
    <span className={`ds-fresh${stale ? ' ds-fresh-stale' : ''}`}>
      <DsIcon name={stale ? 'alert' : 'clock'} />
      <span>{text}</span>
      {action}
    </span>
  )
}
