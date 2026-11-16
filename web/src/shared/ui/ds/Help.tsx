import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from 'react'
import { DsIcon, type DsIconName } from './icons'
import { Popover } from './Menu'

/*
 * Help / contextual guidance primitives (SN-207 §16; prototype help.css .qhint, .hc-items, .terms; help-engine.js).
 * Content (role glossaries, tours, shortcut lists) is supplied by the caller; these components
 * only present it. Help never executes an action. The tour engine is deferred (see RM-E01 report).
 */

/** Contextual hint: an info button whose text is its accessible name and a hover/focus tooltip (dismissable with Esc). */
export function ContextHint({ text, label }: { text: string; label?: string | undefined }) {
  const [show, setShow] = useState(false)
  const tipId = useId()
  return (
    <span className="ds-hint-wrap">
      <button
        type="button"
        className="ds-qhint"
        aria-label={label ?? text}
        aria-describedby={label ? tipId : undefined}
        onMouseEnter={() => setShow(true)}
        onMouseLeave={() => setShow(false)}
        onFocus={() => setShow(true)}
        onBlur={() => setShow(false)}
        onKeyDown={e => { if (e.key === 'Escape' && show) { e.stopPropagation(); setShow(false) } }}
      >
        <DsIcon name="info" />
      </button>
      <span id={tipId} role="tooltip" className="ds-tooltip" hidden={!show}>{text}</span>
    </span>
  )
}

export interface GlossaryEntry { term: string; definition: string }

/** Glossary (prototype .terms): a description list. */
export function Glossary({ entries, label = 'واژه‌ها و مفاهیم' }: { entries: GlossaryEntry[]; label?: string | undefined }) {
  return (
    <dl className="ds-terms" aria-label={label}>
      {entries.map(e => (
        <div key={e.term} className="ds-term">
          <dt>{e.term}</dt>
          <dd>{e.definition}</dd>
        </div>
      ))}
    </dl>
  )
}

export interface ShortcutEntry { keys: string[]; description: string }

/** Keyboard shortcut help. Keys are rendered LTR inside <kbd>. */
export function ShortcutList({ entries }: { entries: ShortcutEntry[] }) {
  return (
    <ul className="ds-shortcuts">
      {entries.map(e => (
        <li key={e.description}>
          <span>{e.description}</span>
          <span className="ds-keys">{e.keys.map(k => <kbd key={k} className="ds-kbd">{k}</kbd>)}</span>
        </li>
      ))}
    </ul>
  )
}

export interface HelpTopic { id: string; title: string; summary: string; icon: DsIconName; onOpen: () => void }

/** Header help button + Help Center popover (prototype help-btn / hc-*). */
export function HelpCenterButton({ label = 'راهنما', title = 'مرکز راهنما', subtitle, topics, children }: {
  label?: string | undefined
  title?: string | undefined
  subtitle?: string | undefined
  topics: HelpTopic[]
  children?: ReactNode | undefined
}) {
  const [open, setOpen] = useState(false)
  const btn = useRef<HTMLButtonElement>(null)
  const onClose = useCallback(() => setOpen(false), [])
  const popId = useId()
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape' && !e.defaultPrevented) { e.preventDefault(); setOpen(false); btn.current?.focus() } }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [open])
  return (
    <>
      <button ref={btn} type="button" className="ds-icon-btn" aria-label={label} title={label} aria-haspopup="dialog" aria-expanded={open}
        aria-controls={open ? popId : undefined} onClick={() => setOpen(o => !o)}>
        <DsIcon name="help" />
      </button>
      <Popover id={popId} open={open} onClose={onClose} anchorRef={btn} label={title}>
        <div className="ds-hc">
          <div className="ds-hc-head"><h2>{title}</h2></div>
          {subtitle ? <p className="ds-hc-sub">{subtitle}</p> : null}
          <div className="ds-hc-list">
            {topics.map(t => (
              <button key={t.id} type="button" className="ds-hc-item" onClick={() => { setOpen(false); t.onOpen() }}>
                <span className="ds-hc-ico"><DsIcon name={t.icon} /></span>
                <span className="ds-hc-txt"><b>{t.title}</b><span>{t.summary}</span></span>
              </button>
            ))}
          </div>
          {children}
        </div>
      </Popover>
    </>
  )
}
