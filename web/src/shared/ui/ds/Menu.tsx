import {
  useCallback, useEffect, useId, useLayoutEffect, useMemo, useRef, useState,
  type KeyboardEvent as ReactKeyboardEvent, type ReactNode, type RefObject,
} from 'react'
import { DsIcon, type DsIconName } from './icons'

/*
 * Menu / Popover primitives (prototype .menu, openMenu/closeMenu/placeMenu).
 * - Menu: role=menu, roving focus (↑/↓/Home/End), Esc and Tab close, focus returns to the trigger.
 * - Disabled items stay visible with a visible reason (SN-207 §1.2: disabled/explained,
 *   never hidden-and-implied). A disabled item never fires onSelect.
 * - Popover: anchored non-modal dialog for non-menu content (help, preferences).
 * Rendered inline (no portal) so theme tokens of the surrounding scope apply.
 * Presentation only: an item's onSelect is supplied by the caller; this file grants nothing.
 */

export type MenuEntry =
  | {
    kind: 'item'; id: string; label: string; icon?: DsIconName | undefined; onSelect?: (() => void) | undefined
    disabled?: boolean | undefined; disabledReason?: string | undefined; danger?: boolean | undefined
    badge?: string | undefined; current?: boolean | undefined
  }
  | { kind: 'separator'; id: string }
  | { kind: 'heading'; id: string; label: string }

function useAnchoredPosition(open: boolean, anchorRef: RefObject<HTMLElement | null>, panelRef: RefObject<HTMLElement | null>) {
  useLayoutEffect(() => {
    if (!open) return
    const place = () => {
      const a = anchorRef.current, m = panelRef.current
      if (!a || !m) return
      const r = a.getBoundingClientRect()
      // A long menu (e.g. a role navigation overflow) must stay reachable: it never extends
      // past the viewport; it scrolls inside (ds.css .ds-menu overflow-y) instead.
      m.style.maxHeight = ''
      const below = window.innerHeight - (r.bottom + 6) - 8
      const above = r.top - 6 - 8
      const fitsBelow = m.offsetHeight <= below
      const useAbove = !fitsBelow && above > below
      const room = Math.max(120, useAbove ? above : below)
      if (m.offsetHeight > room) m.style.maxHeight = `${room}px`
      const top = useAbove ? Math.max(8, r.top - 6 - Math.min(m.offsetHeight, room)) : r.bottom + 6
      m.style.top = `${top}px`
      m.style.left = `${Math.max(8, Math.min(r.left, window.innerWidth - m.offsetWidth - 8))}px`
    }
    place()
    window.addEventListener('resize', place)
    window.addEventListener('scroll', place, true)
    return () => { window.removeEventListener('resize', place); window.removeEventListener('scroll', place, true) }
  }, [open, anchorRef, panelRef])
}

function useOutsideClose(open: boolean, onClose: () => void, refs: RefObject<HTMLElement | null>[]) {
  useEffect(() => {
    if (!open) return
    const onDown = (e: PointerEvent) => {
      const t = e.target as Node
      if (refs.some(r => r.current?.contains(t))) return
      onClose()
    }
    document.addEventListener('pointerdown', onDown)
    return () => document.removeEventListener('pointerdown', onDown)
  }, [open, onClose, refs])
}

export function Menu({ id, open, onClose, anchorRef, label, entries, initialFocus = 'first' }: {
  id?: string | undefined
  open: boolean
  onClose: (reason: 'escape' | 'select' | 'outside' | 'tab') => void
  anchorRef: RefObject<HTMLElement | null>
  label: string
  entries: MenuEntry[]
  initialFocus?: 'first' | 'last' | undefined
}) {
  const ref = useRef<HTMLDivElement>(null)
  const autoId = useId()
  useAnchoredPosition(open, anchorRef, ref)
  const outsideClose = useCallback(() => onClose('outside'), [onClose])
  const outsideRefs = useMemo<RefObject<HTMLElement | null>[]>(() => [ref, anchorRef], [anchorRef])
  useOutsideClose(open, outsideClose, outsideRefs)

  const items = () => Array.from(ref.current?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])

  useEffect(() => {
    if (!open) return
    const list = items()
    const target = initialFocus === 'last' ? list[list.length - 1] : list[0]
    target?.focus()
  }, [open, initialFocus])

  if (!open) return null

  const close = (reason: 'escape' | 'select' | 'tab') => {
    onClose(reason)
    if (reason !== 'tab') anchorRef.current?.focus()
  }

  const onKeyDown = (e: ReactKeyboardEvent) => {
    const list = items()
    const i = list.indexOf(document.activeElement as HTMLElement)
    const move = (n: number) => { e.preventDefault(); list[(n + list.length) % list.length]?.focus() }
    switch (e.key) {
      case 'ArrowDown': move(i + 1); break
      case 'ArrowUp': move(i - 1); break
      case 'Home': move(0); break
      case 'End': move(list.length - 1); break
      case 'Escape': e.preventDefault(); e.stopPropagation(); close('escape'); break
      case 'Tab': close('tab'); break
    }
  }

  return (
    <div ref={ref} id={id ?? autoId} className="ds-menu" role="menu" aria-label={label} onKeyDown={onKeyDown}>
      {entries.map(entry => {
        if (entry.kind === 'separator') return <hr key={entry.id} role="separator" />
        if (entry.kind === 'heading') return <div key={entry.id} className="ds-mh" role="presentation">{entry.label}</div>
        const reasonId = entry.disabled && entry.disabledReason ? `${autoId}-${entry.id}-why` : undefined
        return (
          <button
            key={entry.id}
            type="button"
            role="menuitem"
            tabIndex={-1}
            className={['ds-mi', entry.danger && 'ds-mi-danger', entry.disabled && 'ds-mi-off'].filter(Boolean).join(' ')}
            aria-disabled={entry.disabled || undefined}
            aria-describedby={reasonId}
            aria-current={entry.current ? 'page' : undefined}
            data-id={entry.id}
            onClick={() => {
              if (entry.disabled) return
              entry.onSelect?.()
              close('select')
            }}
          >
            {entry.icon ? <DsIcon name={entry.icon} /> : null}
            <span className="ds-mi-text">
              <span>{entry.label}</span>
              {reasonId ? <small id={reasonId} className="ds-mi-why">{entry.disabledReason}</small> : null}
            </span>
            {entry.badge ? <span className="ds-mi-b">{entry.badge}</span> : null}
          </button>
        )
      })}
    </div>
  )
}

/** Trigger button + Menu. Enter/Space/↓ opens on the first item, ↑ on the last. */
export function MenuButton({ label, entries, children, className = 'btn', menuLabel, icon }: {
  /** Accessible name of the trigger. */
  label: string
  entries: MenuEntry[]
  children?: ReactNode | undefined
  className?: string | undefined
  menuLabel?: string | undefined
  icon?: DsIconName | undefined
}) {
  const [open, setOpen] = useState(false)
  const [focusAt, setFocusAt] = useState<'first' | 'last'>('first')
  const btn = useRef<HTMLButtonElement>(null)
  const menuId = useId()
  const onClose = useCallback(() => setOpen(false), [])
  return (
    <>
      <button
        ref={btn}
        type="button"
        className={className}
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => { setFocusAt('first'); setOpen(o => !o) }}
        onKeyDown={e => {
          if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
            e.preventDefault()
            setFocusAt(e.key === 'ArrowUp' ? 'last' : 'first')
            setOpen(true)
          }
        }}
      >
        {icon ? <DsIcon name={icon} /> : null}
        {children}
      </button>
      <Menu id={menuId} open={open} onClose={onClose} anchorRef={btn} label={menuLabel ?? label} entries={entries} initialFocus={focusAt} />
    </>
  )
}

/** Anchored non-modal dialog. Esc / outside click close; focus moves in on open and back to the trigger on Esc. */
export function Popover({ open, onClose, anchorRef, label, children, id }: {
  open: boolean
  onClose: () => void
  anchorRef: RefObject<HTMLElement | null>
  label: string
  children: ReactNode
  id?: string | undefined
}) {
  const ref = useRef<HTMLDivElement>(null)
  useAnchoredPosition(open, anchorRef, ref)
  const outsideRefs = useMemo<RefObject<HTMLElement | null>[]>(() => [ref, anchorRef], [anchorRef])
  useOutsideClose(open, onClose, outsideRefs)
  useEffect(() => { if (open) ref.current?.focus() }, [open])
  if (!open) return null
  return (
    <div
      ref={ref}
      id={id}
      className="ds-menu ds-popover"
      role="dialog"
      aria-modal="false"
      aria-label={label}
      tabIndex={-1}
      onKeyDown={e => {
        if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); onClose(); anchorRef.current?.focus() }
      }}
    >
      {children}
    </div>
  )
}
