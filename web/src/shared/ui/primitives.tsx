import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Icon, type IconName } from './icons'

/** SN-207 semantic tones (styles.css .t-*). */
export type Tone = 'neutral' | 'teal' | 'green' | 'amber' | 'orange' | 'red' | 'blue' | 'violet' | 'slate'

type ButtonVariant = 'default' | 'primary' | 'ghost'

export function Button({ variant = 'default', small = false, className, ...rest }:
  ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant; small?: boolean }) {
  const cls = ['btn', variant === 'primary' && 'btn-primary', variant === 'ghost' && 'btn-ghost', small && 'btn-sm', className]
    .filter(Boolean).join(' ')
  return <button type="button" {...rest} className={cls} />
}

/** Status pill: always label + icon + tone, never colour alone. */
export function StatusPill({ tone, icon, label }: { tone: Tone; icon: IconName; label: string }) {
  return (
    <span className={`pill t-${tone}`}>
      <Icon name={icon} />
      {label}
    </span>
  )
}

export function Card({ title, children }: { title?: string; children: ReactNode }) {
  return (
    <section className="card" aria-label={title}>
      {title ? <h2>{title}</h2> : null}
      {children}
    </section>
  )
}

export function PageHeader({ title, subtitle, children }: { title: string; subtitle?: string; children?: ReactNode }) {
  return (
    <div className="page-head">
      <div>
        <h1>{title}</h1>
        {subtitle ? <p className="sub">{subtitle}</p> : null}
      </div>
      {children}
    </div>
  )
}

/** SN-207 §7 system states used by the shell. */
export type StateKind = 'loading' | 'empty' | 'error' | 'unknown' | 'locked'

const stateIcons: Record<StateKind, IconName> = {
  loading: 'inbox',
  empty: 'inbox',
  error: 'x',
  unknown: 'question',
  locked: 'lock',
}

export function StateBlock({ kind, title, detail, action }:
  { kind: StateKind; title: string; detail?: string; action?: ReactNode }) {
  // Errors interrupt (alert); every other state is a polite status update.
  const role = kind === 'error' ? 'alert' : 'status'
  return (
    <div className={`state state-${kind}`} role={role} aria-busy={kind === 'loading' || undefined}>
      {kind === 'loading'
        ? <span className="skeleton" aria-hidden="true" />
        : <span className="state-ico"><Icon name={stateIcons[kind]} /></span>}
      <h2>{title}</h2>
      {detail ? <p>{detail}</p> : null}
      {action}
    </div>
  )
}
