import type { ReactNode } from 'react'
import { Link } from 'react-router'
import type { ResolutionState } from '../api'
import { StatusPill } from '../ui/primitives'
import { relationIcon, relationLabel, relationTone, unlinkedNote } from './labels'

const known = (s: string): s is ResolutionState => s in relationLabel

/** Relationship state: label + icon + tone (never colour alone). */
export function RelationPill({ state }: { state: string }) {
  if (!known(state)) return <StatusPill tone="slate" icon="question" label={state} />
  return <StatusPill tone={relationTone[state]} icon={relationIcon[state]} label={relationLabel[state]} />
}

/**
 * One relationship to another native record. A link is rendered only for a
 * RESOLVED state that carries the target's native id; every other state is
 * shown with its basis and no navigation. The target id is never derived
 * here (no phone, name, legacy key or cross-module inference).
 */
export function RelationLink({ state, id, to, children, basis }:
  { state: string; id: string | null | undefined; to: (id: string) => string; children: ReactNode; basis?: string }) {
  const navigable = state === 'RESOLVED' && typeof id === 'string' && id !== ''
  return (
    <span className="rel-link">
      <RelationPill state={state} />
      {navigable
        ? <> <Link to={to(id)}>{children}</Link></>
        : <span className="muted"> {unlinkedNote}</span>}
      {basis ? <span className="muted"> — {basis}</span> : null}
    </span>
  )
}
