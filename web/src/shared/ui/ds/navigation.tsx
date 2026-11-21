import {
  Fragment, useCallback, useLayoutEffect, useRef, useState,
  type CSSProperties, type KeyboardEvent as ReactKeyboardEvent, type MouseEvent, type ReactNode,
} from 'react'
import type { Tone } from '../primitives'
import { countLabel, countText, isKnownCount, type DsCount } from './format'
import { DsIcon, type DsIconName } from './icons'
import { Menu, type MenuEntry } from './Menu'

/*
 * RoleNavigation, MobileBottomNavigation and QueueTabs (SN-207 §2, §8; prototype
 * crm-core.js renderNav/fitNav, styles.css .nav/.bottom-nav/.queues).
 *
 * These components only PRESENT the destinations they are given. Which items a
 * user may see is decided by the caller from server-provided hints (RM-E02);
 * nothing here grants, infers or hides authority.
 */

export interface DsNavItem {
  id: string
  label: string
  href: string
  icon: DsIconName
  /** Count badge. Omit for none; 'unknown' renders «—», never 0. A known 0 shows no badge (prototype). */
  count?: DsCount | undefined
  /** Needs-action tone on the count badge. */
  alert?: boolean | undefined
  /** Items with different groups are separated by a divider. */
  group?: string | undefined
  /** Short label for the mobile bottom bar; items without it go into «بیشتر». */
  mobileLabel?: string | undefined
}

export type DsLinkRenderer = (item: DsNavItem, props: {
  className?: string | undefined
  'aria-current'?: 'page' | undefined
  onClick: (e: MouseEvent<HTMLAnchorElement>) => void
  hidden?: boolean | undefined
  children: ReactNode
  'data-id': string
}) => ReactNode

const defaultLink: DsLinkRenderer = (item, props) => <a href={item.href} {...props} />

function NavCount({ count, alert }: { count?: DsCount | undefined; alert?: boolean | undefined }) {
  if (count === undefined) return null
  if (isKnownCount(count) && count === 0) return null
  return (
    <span className={['ds-nav-count', alert && 'ds-count-alert', !isKnownCount(count) && 'ds-count-unknown'].filter(Boolean).join(' ')}>
      <span aria-hidden="true">{countText(count)}</span>
      <span className="sr">، {countLabel(count)}</span>
    </span>
  )
}

/**
 * Priority+ overflow (prototype fitNav): trailing items move into «بیشتر» when the
 * row does not fit; the current item is never moved. Pure function for testing.
 */
export function computeNavOverflow(input: {
  itemWidths: number[]
  separatorWidths: number[]
  available: number
  moreWidth: number
  gap: number
  currentIndex: number
}): number[] {
  const { itemWidths, separatorWidths, available, moreWidth, gap, currentIndex } = input
  const n = itemWidths.length
  const visible = new Array<boolean>(n).fill(true)
  const need = () => {
    let total = 0, count = 0
    for (let i = 0; i < n; i++) {
      if (!visible[i]) continue
      // A visible separator is its own flex item: its width AND one more flex gap.
      const sep = count > 0 ? separatorWidths[i] ?? 0 : 0
      total += (itemWidths[i] ?? 0) + sep + (sep > 0 ? gap : 0)
      count++
    }
    return total + gap * Math.max(0, count - 1)
  }
  if (need() <= available + 1) return []
  const room = available - moreWidth - gap
  for (let i = n - 1; i >= 0 && need() > room + 1; i--) {
    if (i === currentIndex) continue
    visible[i] = false
  }
  return visible.flatMap((v, i) => (v ? [] : [i]))
}

export function RoleNavigation({ items, currentId, label, onNavigate, renderLink = defaultLink, moreLabel = 'بیشتر' }: {
  items: DsNavItem[]
  currentId: string | undefined
  /** Accessible name of the nav landmark, e.g. «بخش‌های پنل فروشنده». */
  label: string
  /** Called on link click (may preventDefault and route) and on overflow-menu selection (no event). */
  onNavigate?: (item: DsNavItem, e?: MouseEvent<HTMLAnchorElement>) => void
  renderLink?: DsLinkRenderer | undefined
  moreLabel?: string | undefined
}) {
  const navRef = useRef<HTMLElement>(null)
  const moreRef = useRef<HTMLButtonElement>(null)
  const [hidden, setHidden] = useState<string[]>([])
  const [phase, setPhase] = useState<'measure' | 'done'>('measure')
  const [menuOpen, setMenuOpen] = useState(false)
  const [menuFocus, setMenuFocus] = useState<'first' | 'last'>('first')

  const itemKey = items.map(i => `${i.id}:${i.label}:${i.group ?? ''}:${String(i.count ?? '')}`).join('|')
  const [lastKey, setLastKey] = useState(`${itemKey}#${currentId}`)
  if (lastKey !== `${itemKey}#${currentId}`) { // re-measure when items or the current item change
    setLastKey(`${itemKey}#${currentId}`)
    setPhase('measure')
  }

  useLayoutEffect(() => {
    const nav = navRef.current
    if (!nav || typeof ResizeObserver === 'undefined') return
    let lastWidth = nav.clientWidth
    const ro = new ResizeObserver(() => {
      if (nav.clientWidth !== lastWidth) { lastWidth = nav.clientWidth; setPhase('measure') }
    })
    ro.observe(nav)
    // Item widths change when the web font swaps in; re-measure then (prototype nav-probe).
    let alive = true
    void document.fonts?.ready.then(() => { if (alive) setPhase('measure') })
    return () => { alive = false; ro.disconnect() }
  }, [])

  useLayoutEffect(() => {
    if (phase !== 'measure') return
    const nav = navRef.current
    if (!nav) return
    const links = Array.from(nav.querySelectorAll<HTMLElement>(':scope > [data-id]'))
    const seps = links.map(l => {
      const prev = l.previousElementSibling as HTMLElement | null
      return prev?.hasAttribute('data-sep') ? prev.getBoundingClientRect().width + 2 * parseFloat(getComputedStyle(prev).marginLeft || '0') : 0
    })
    const style = getComputedStyle(nav)
    const available = nav.clientWidth - (parseFloat(style.paddingLeft) || 0) - (parseFloat(style.paddingRight) || 0)
    const result = available > 0
      ? computeNavOverflow({
        itemWidths: links.map(l => l.getBoundingClientRect().width),
        separatorWidths: seps,
        available,
        moreWidth: moreRef.current?.getBoundingClientRect().width ?? 0,
        gap: parseFloat(style.columnGap) || 0,
        currentIndex: items.findIndex(i => i.id === currentId),
      })
      : []
    setHidden(result.flatMap(i => items[i]?.id ?? []))
    setPhase('done')
  }, [phase, items, currentId])

  const isHidden = (id: string) => phase === 'done' && hidden.includes(id)
  const overflowItems = items.filter(i => isHidden(i.id))
  const showMore = phase === 'measure' || overflowItems.length > 0
  const onCloseMenu = useCallback(() => setMenuOpen(false), [])

  const entries: MenuEntry[] = overflowItems.map(i => ({
    kind: 'item', id: i.id, label: i.label, icon: i.icon,
    badge: i.count !== undefined && !(isKnownCount(i.count) && i.count === 0) ? countText(i.count) : undefined,
    onSelect: () => { if (onNavigate) onNavigate(i); else window.location.assign(i.href) },
  }))

  const sepBefore = items.map((it, i) => i > 0 && it.group !== items[i - 1]?.group)
  return (
    <nav ref={navRef} className="ds-nav" aria-label={label}>
      {items.map((item, idx) => {
        const sep = sepBefore[idx]
        const h = isHidden(item.id)
        const current = item.id === currentId
        return (
          <Fragment key={item.id}>
            {sep ? <span className="ds-nav-sep" aria-hidden="true" data-sep hidden={h || undefined} /> : null}
            {renderLink(item, {
              className: 'ds-nav-link',
              'aria-current': current ? 'page' : undefined,
              hidden: h || undefined,
              'data-id': item.id,
              onClick: e => onNavigate?.(item, e),
              children: (
                <>
                  <DsIcon name={item.icon} />
                  <span>{item.label}</span>
                  <NavCount count={item.count} alert={item.alert} />
                </>
              ),
            })}
          </Fragment>
        )
      })}
      <button
        ref={moreRef}
        type="button"
        className="ds-nav-more"
        hidden={!showMore || undefined}
        style={phase === 'measure' ? { visibility: 'hidden' } : undefined}
        aria-haspopup="menu"
        aria-expanded={menuOpen}
        onClick={() => { setMenuFocus('first'); setMenuOpen(o => !o) }}
        onKeyDown={e => {
          if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); setMenuFocus(e.key === 'ArrowUp' ? 'last' : 'first'); setMenuOpen(true) }
        }}
      >
        <DsIcon name="more" />
        <span>{moreLabel}</span>
        {/* While measuring (button invisible) the badge holds the upper bound, items.length, so the reserved
            width is never smaller than the final badge (one digit while measuring vs two after the cut). */}
        <span className="ds-nav-count"><span aria-hidden="true">{countText(phase === 'measure' ? items.length : overflowItems.length)}</span><span className="sr">، {countLabel(overflowItems.length)} مورد</span></span>
      </button>
      <Menu open={menuOpen} onClose={onCloseMenu} anchorRef={moreRef} label={moreLabel} entries={entries} initialFocus={menuFocus} />
    </nav>
  )
}

/* ---------- MobileBottomNavigation (≤760px) ---------- */

export function MobileBottomNavigation({ items, currentId, label = 'ناوبری موبایل', onNavigate, renderLink = defaultLink, moreLabel = 'بیشتر', maxPrimary = 4 }: {
  items: DsNavItem[]
  currentId: string | undefined
  label?: string | undefined
  onNavigate?: (item: DsNavItem, e?: MouseEvent<HTMLAnchorElement>) => void
  renderLink?: DsLinkRenderer | undefined
  moreLabel?: string | undefined
  maxPrimary?: number | undefined
}) {
  const [open, setOpen] = useState(false)
  const moreRef = useRef<HTMLButtonElement>(null)
  const onClose = useCallback(() => setOpen(false), [])
  const primary = items.filter(i => i.mobileLabel).slice(0, maxPrimary)
  const rest = items.filter(i => !primary.includes(i))
  const moreCurrent = rest.some(i => i.id === currentId)
  const entries: MenuEntry[] = rest.map(i => ({
    kind: 'item', id: i.id, label: i.label, icon: i.icon, current: i.id === currentId,
    badge: i.count !== undefined && !(isKnownCount(i.count) && i.count === 0) ? countText(i.count) : undefined,
    onSelect: () => { if (onNavigate) onNavigate(i); else window.location.assign(i.href) },
  }))
  return (
    <nav className="ds-bottom-nav" aria-label={label} style={{ ['--cols' as string]: String(primary.length + (rest.length ? 1 : 0)) } as CSSProperties}>
      {primary.map(item => <Fragment key={item.id}>{renderLink(item, {
        className: 'ds-bn-link',
        'aria-current': item.id === currentId ? 'page' : undefined,
        'data-id': item.id,
        onClick: e => onNavigate?.(item, e),
        children: (
          <>
            <DsIcon name={item.icon} />
            {item.alert && item.count !== undefined && !(isKnownCount(item.count) && item.count === 0)
              ? <span className="ds-bn-badge"><span aria-hidden="true">{countText(item.count)}</span><span className="sr">، {countLabel(item.count)}</span></span>
              : null}
            <span>{item.mobileLabel}</span>
          </>
        ),
      })}</Fragment>)}
      {rest.length ? (
        <button
          ref={moreRef}
          type="button"
          className="ds-bn-link"
          aria-haspopup="menu"
          aria-expanded={open}
          aria-current={moreCurrent ? 'page' : undefined}
          onClick={() => setOpen(o => !o)}
        >
          <DsIcon name="grid" />
          <span>{moreLabel}</span>
        </button>
      ) : null}
      <Menu open={open} onClose={onClose} anchorRef={moreRef} label={moreLabel} entries={entries} />
    </nav>
  )
}

/* ---------- QueueTabs (prototype queues(): pressed-chip facets with counts) ---------- */

export interface DsQueue {
  id: string
  label: string
  icon?: DsIconName | undefined
  tone?: Tone | undefined
  /** Omit for none; 'unknown' renders «—», never 0. A known 0 is shown (it is a fact). */
  count?: DsCount | undefined
  /** Optional single-key shortcut hint (handled by the page, announced here). */
  shortcut?: string | undefined
}

function isRtl(el: HTMLElement): boolean {
  return (el.closest('[dir]')?.getAttribute('dir') ?? document.documentElement.dir) === 'rtl'
}

export function QueueTabs({ queues, value, onChange, label = 'صف‌ها' }: {
  queues: DsQueue[]
  value: string
  onChange: (id: string) => void
  label?: string | undefined
}) {
  const ref = useRef<HTMLDivElement>(null)
  const onKeyDown = (e: ReactKeyboardEvent<HTMLDivElement>) => {
    const list = Array.from(ref.current?.querySelectorAll<HTMLButtonElement>('button[data-queue]') ?? [])
    const i = list.indexOf(document.activeElement as HTMLButtonElement)
    if (i < 0) return
    const rtl = isRtl(e.currentTarget)
    const fwd = rtl ? 'ArrowLeft' : 'ArrowRight', back = rtl ? 'ArrowRight' : 'ArrowLeft'
    let n = -1
    if (e.key === fwd) n = (i + 1) % list.length
    else if (e.key === back) n = (i - 1 + list.length) % list.length
    else if (e.key === 'Home') n = 0
    else if (e.key === 'End') n = list.length - 1
    if (n < 0) return
    e.preventDefault()
    list[n]?.focus()
  }
  return (
    <div ref={ref} className="ds-queues" role="group" aria-label={label} onKeyDown={onKeyDown}>
      {queues.map(q => {
        const style = q.tone ? { ['--qc' as string]: `var(--${q.tone}-dot)`, ['--qbg' as string]: `var(--${q.tone}-bg)`, ['--qfg' as string]: `var(--${q.tone}-fg)` } as CSSProperties : undefined
        return (
          <button
            key={q.id}
            type="button"
            className="ds-q"
            data-queue={q.id}
            aria-pressed={q.id === value}
            aria-keyshortcuts={q.shortcut}
            style={style}
            onClick={() => onChange(q.id)}
          >
            {q.icon ? <span className="ds-q-ic"><DsIcon name={q.icon} /></span> : null}
            <span>{q.label}</span>
            {q.count !== undefined
              ? <span className={['ds-q-n', !isKnownCount(q.count) && 'ds-count-unknown'].filter(Boolean).join(' ')}><span aria-hidden="true">{countText(q.count)}</span><span className="sr">، {countLabel(q.count)}</span></span>
              : null}
          </button>
        )
      })}
    </div>
  )
}
