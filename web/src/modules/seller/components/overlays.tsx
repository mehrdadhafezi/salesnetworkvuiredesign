import { useEffect, useId, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react'
import { DsIcon, toFaDigits, type DsIconName } from '../../../shared/ui/ds'
import type { Tone } from '../../../shared/ui/primitives'
import { useSellerPreview } from '../state/SellerPreviewContext'
import { normalizeSearch } from '../state/selectors'

/*
 * Prototype overlays not covered by RM-E01: outcome chooser, toasts and the command palette.
 * Presentation-only patterns of the frozen Seller V1 prototype (app.js outcomeBtn(), toast(),
 * openPalette()). The palette searches SYNTHETIC fixtures in memory; it calls no API.
 */

export interface OutcomeChoice<T extends string> { id: T; label: string; tone: Tone; icon: DsIconName; hint: string; key: string; wide?: boolean | undefined }

/** Outcome buttons (aria-pressed toggle group). Number keys 1–n pick an option while the drawer has focus. */
export function OutcomeGrid<T extends string>({ options, value, onChange, label, cols = 4, hintFor }: {
  options: OutcomeChoice<T>[]
  value: T | null
  onChange: (v: T) => void
  label: string
  cols?: 3 | 4
  hintFor?: ((id: T) => string | null) | undefined
}) {
  return (
    <div className={`sl-outcomes sl-cols-${cols}`} role="group" aria-label={label}>
      {options.map(o => (
        <button key={o.id} type="button" className={`sl-outcome${o.wide ? ' sl-wide' : ''}`} data-outcome={o.id}
          style={{ ['--c' as string]: `var(--${o.tone}-dot)` }} aria-pressed={value === o.id} aria-keyshortcuts={o.key}
          onClick={() => onChange(o.id)}>
          <b><DsIcon name={o.icon} />{o.label}<span className="sl-k" aria-hidden="true">{toFaDigits(o.key)}</span></b>
          <small>{hintFor?.(o.id) ?? o.hint}</small>
        </button>
      ))}
    </div>
  )
}

export function Toasts() {
  const { toasts, dismiss } = useSellerPreview()
  return (
    <div className="sl-toasts" aria-live="polite" aria-relevant="additions">
      {toasts.map(t => (
        <div key={t.id} className={`sl-toast sl-toast-${t.kind}`} role={t.kind === 'error' ? 'alert' : 'status'}>
          <DsIcon name={t.kind === 'success' ? 'checkCircle' : t.kind === 'info' ? 'info' : 'alert'} />
          <span className="sl-toast-msg">{t.msg}</span>
          <button type="button" className="sl-toast-x" aria-label="بستن اعلان" onClick={() => dismiss(t.id)}><DsIcon name="x" /></button>
        </div>
      ))}
    </div>
  )
}

export interface PaletteItem { id: string; group: string; icon: DsIconName; label: string; meta?: string | undefined; keywords: string; run: () => void }

/** Command palette (Ctrl/⌘+K or /): modal dialog with a listbox; ↑↓ select, Enter run, Esc close and restore focus. */
export function CommandPalette({ open, onClose, items }: { open: boolean; onClose: () => void; items: PaletteItem[] }) {
  // Mounted only while open, so the query and selection always start clean.
  return open ? <PaletteDialog onClose={onClose} items={items} /> : null
}

function PaletteDialog({ onClose, items }: { onClose: () => void; items: PaletteItem[] }) {
  const [q, setQ] = useState('')
  const [sel, setSel] = useState(0)
  const input = useRef<HTMLInputElement>(null)
  const opener = useRef<HTMLElement | null>(null)
  const listId = useId()

  useEffect(() => {
    opener.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    input.current?.focus()
    return () => { const o = opener.current; if (o && document.contains(o)) o.focus() }
  }, [])

  const results = useMemo(() => {
    const nq = normalizeSearch(q)
    // Records only appear once something is typed (prototype palItems).
    return items.filter(i => (nq ? normalizeSearch(`${i.label} ${i.keywords}`).includes(nq) : i.group === 'رفتن به' || i.group === 'نمایش')).slice(0, 40)
  }, [items, q])
  const active = Math.min(sel, Math.max(0, results.length - 1))

  const run = (n: number) => { const it = results[n]; if (!it) return; onClose(); it.run() }
  const onKey = (e: ReactKeyboardEvent) => {
    if (e.key === 'Escape') { e.preventDefault(); e.stopPropagation(); onClose() }
    else if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); const n = results.length || 1; setSel((active + (e.key === 'ArrowDown' ? 1 : -1) + n) % n) }
    else if (e.key === 'Enter') { e.preventDefault(); run(active) }
    else if (e.key === 'Tab') { e.preventDefault(); input.current?.focus() }
  }
  let group = ''
  return (
    <div className="sl-palette-wrap" onKeyDown={onKey}>
      <div className="sl-palette-bd" aria-hidden="true" onClick={onClose} />
      <div className="sl-palette" role="dialog" aria-modal="true" aria-label="جستجو و فرمان‌ها">
        <div className="sl-pal-in">
          <DsIcon name="search" />
          <input ref={input} type="text" value={q} role="combobox" aria-expanded="true" aria-controls={listId} aria-autocomplete="list"
            aria-activedescendant={results[active] ? `${listId}-${active}` : undefined}
            aria-label="جستجو و فرمان‌ها" placeholder="نام یا شماره مشتری، کد فاکتور، یا یک فرمان…" autoComplete="off"
            onChange={e => { setQ(e.target.value); setSel(0) }} />
          <span className="ds-kbd">Esc</span>
        </div>
        <div className="sl-pal-list" role="listbox" id={listId} aria-label="نتیجه‌ها">
          {results.length ? results.map((it, n) => {
            const head = it.group !== group ? (group = it.group) : null
            return (
              <div key={it.id} role="presentation">
                {head ? <div className="sl-pal-group" role="presentation">{head}</div> : null}
                <div id={`${listId}-${n}`} role="option" aria-selected={n === active} className="sl-pal-item" tabIndex={-1}
                  onMouseMove={() => setSel(n)} onClick={() => run(n)}>
                  <DsIcon name={it.icon} /><span className="sl-grow">{it.label}</span>{it.meta ? <small>{it.meta}</small> : null}
                </div>
              </div>
            )
          }) : <div className="sl-pal-empty">نتیجه‌ای پیدا نشد</div>}
        </div>
        <div className="sl-pal-foot">
          <span><span className="ds-kbd">↑↓</span> انتخاب</span><span><span className="ds-kbd">Enter</span> اجرا</span>
          <span><span className="ds-kbd">/</span> یا <span className="ds-kbd">Ctrl K</span> باز کردن</span>
          <span>جستجو فقط روی داده ساختگی همین پیش‌نمایش انجام می‌شود.</span>
        </div>
      </div>
    </div>
  )
}
