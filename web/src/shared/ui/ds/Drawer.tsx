import { useEffect, useId, useLayoutEffect, useRef, type ReactNode } from 'react'
import { DsIcon } from './icons'

/*
 * Drawer / Sheet (SN-207 §9; prototype styles.css .drawer/.dr-*, crm-core.js openDrawer/closeDrawer).
 * - aside role=dialog, NON-modal, labelled by its title.
 * - 560px (wide: 660px) at the inline-end edge in RTL desktop; bottom sheet with grip ≤760px.
 * - Opening moves focus into the drawer; Esc (anywhere) or the close button closes it and
 *   restores focus to the element that opened it.
 * - When closed it stays mounted for the slide transition but is inert and aria-hidden.
 * The drawer carries no business action: footer actions are supplied by the caller.
 */
export function Drawer({ open, onClose, title, kicker = 'جزئیات', children, footer, footerHint, topExtra, wide = false, closeLabel = 'بستن', id }: {
  open: boolean
  onClose: () => void
  title: string
  /** Small label in the top bar (prototype dtop). */
  kicker?: string | undefined
  children: ReactNode
  /** Footer actions (prototype dfoot primary/secondary). */
  footer?: ReactNode | undefined
  footerHint?: string | undefined
  topExtra?: ReactNode | undefined
  /** 660px for review / result / finance-context kinds (SN-207 §20). */
  wide?: boolean | undefined
  closeLabel?: string | undefined
  id?: string | undefined
}) {
  const ref = useRef<HTMLElement>(null)
  const opener = useRef<HTMLElement | null>(null)
  const titleId = useId()
  const wasOpen = useRef(false)

  // Capture the opener before focus moves, and move focus in.
  useLayoutEffect(() => {
    if (open && !wasOpen.current) {
      const active = document.activeElement
      opener.current = active instanceof HTMLElement && !ref.current?.contains(active) ? active : null
      ref.current?.focus({ preventScroll: true })
    }
    if (!open && wasOpen.current) {
      const o = opener.current
      opener.current = null
      if (o && document.contains(o)) o.focus({ preventScroll: true })
    }
    wasOpen.current = open
  }, [open])

  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !e.defaultPrevented) { e.preventDefault(); onClose() }
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open, onClose])

  return (
    <>
      <div className="ds-scrim" data-open={open} aria-hidden="true" onClick={open ? onClose : undefined} />
      <aside
        ref={ref}
        id={id}
        className={`ds-drawer${wide ? ' ds-drawer-wide' : ''}`}
        data-open={open}
        role="dialog"
        aria-modal="false"
        aria-labelledby={titleId}
        aria-hidden={!open}
        inert={!open}
        tabIndex={-1}
      >
        <div className="ds-sheet-grip" aria-hidden="true" />
        <div className="ds-dr-top">
          <span>{kicker}</span>
          <span className="ds-grow" />
          {topExtra}
          <button type="button" className="btn btn-ghost btn-sm ds-btn-icon" aria-label={closeLabel} title={`${closeLabel} (Esc)`} onClick={onClose}>
            <DsIcon name="x" />
          </button>
        </div>
        <div className="ds-dr-head">
          <h2 id={titleId} className="ds-dr-title">{title}</h2>
        </div>
        <div className="ds-dr-body">{children}</div>
        {footer || footerHint ? (
          <div className="ds-dr-foot">
            {footerHint ? <span className="ds-dr-hint">{footerHint}</span> : null}
            <span className="ds-grow" />
            {footer}
          </div>
        ) : null}
      </aside>
    </>
  )
}

/** Drawer section (prototype .sec / .sec-h). */
export function DrawerSection({ title, aside, children }: { title: string; aside?: ReactNode | undefined; children: ReactNode }) {
  return (
    <section className="ds-sec">
      <div className="ds-sec-h"><h3>{title}</h3>{aside ? <span className="ds-sec-aside">{aside}</span> : null}</div>
      {children}
    </section>
  )
}
