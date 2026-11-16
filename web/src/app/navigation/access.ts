import type { EffectiveAccess, SessionContext } from '../../shared/api'
import { DESTINATIONS, type ContractCapability, type Destination } from './destinations'
import { isRoleKey, roleDestinationIds, type RoleKey } from './roles'

/**
 * RM-E02 fail-closed navigation resolver (pure). Inputs are server answers only:
 * the session (`GET /api/v1/session`: authentication + feature switches) and the
 * effective-access evaluation (`GET /api/v1/authorization/effective-access`:
 * per-capability ALLOWED decisions from the same evaluator as allowed_actions).
 * Nothing else — no localStorage, query parameter, role guess, dev identity,
 * `capability_hints` or UI state — can add a destination.
 *
 * The result is a presentation of what the server already decided. It is not
 * a security boundary: every page request is still authorised by the server.
 */

/** Presentation-only role context, derived from the server's matched_roles. Never a permission. */
export type RoleContext =
  /** Every allowed decision matched exactly one known role. */
  | { kind: 'SINGLE'; role: RoleKey }
  /** More than one role (or grant basis) matched: no approved context-selection policy exists (D-42 / RM-B02). */
  | { kind: 'MULTIPLE'; roles: string[] }
  /** No known role matched (e.g. a synthetic test grant): the SPA does not name a role. */
  | { kind: 'UNKNOWN'; roles: string[] }

export type Evaluation =
  | { state: 'IDLE' }
  | { state: 'LOADING' }
  | { state: 'FAILED' }
  | { state: 'READY'; data: EffectiveAccess }

export interface NavEntry extends Destination {
  /** 'role': mapped by the role contract; 'other': authorised but outside the role contract. */
  placement: 'role' | 'other'
}

export type NavAccess =
  /** Session or evaluation still loading: no business link, no data. */
  | { status: 'LOADING' }
  /** Session unknown (failed) or not authenticated: only safe routes. */
  | { status: 'ANONYMOUS' }
  /** The evaluation request failed: fail closed (UNKNOWN is never "allowed"). */
  | { status: 'FAILED' }
  /** Session and evaluation disagree about authentication: fail closed. */
  | { status: 'CONFLICT' }
  /** The server could not evaluate (every decision BLOCKED, e.g. FEATURE_DISABLED). */
  | { status: 'BLOCKED'; reasons: string[] }
  /** Evaluated: no destination is both allowed and enabled. */
  | { status: 'NONE'; reason: 'NO_GRANT' | 'NOT_ENABLED' }
  | { status: 'GRANTED'; entries: NavEntry[]; role: RoleContext }

function allowedSet(ev: EffectiveAccess): Set<ContractCapability> {
  const out = new Set<ContractCapability>()
  for (const d of ev.decisions) if (d.allowed === true && d.outcome === 'SUCCESS') out.add(d.capability)
  return out
}

function roleContext(ev: EffectiveAccess): RoleContext {
  const roles = [...new Set(ev.decisions.filter(d => d.allowed === true && d.outcome === 'SUCCESS').flatMap(d => d.matched_roles))].sort()
  const first = roles[0]
  if (roles.length === 1 && first !== undefined && isRoleKey(first)) return { kind: 'SINGLE', role: first }
  if (roles.length > 1) return { kind: 'MULTIPLE', roles }
  return { kind: 'UNKNOWN', roles }
}

export function resolveNavigation(session: { status: 'loading' } | { status: 'failed' } | { status: 'ready'; session: SessionContext }, evaluation: Evaluation): NavAccess {
  if (session.status === 'loading') return { status: 'LOADING' }
  if (session.status === 'failed') return { status: 'ANONYMOUS' }
  const s = session.session
  if (!s.authenticated) return { status: 'ANONYMOUS' }
  if (evaluation.state === 'IDLE' || evaluation.state === 'LOADING') return { status: 'LOADING' }
  if (evaluation.state === 'FAILED') return { status: 'FAILED' }
  const ev = evaluation.data
  if (!ev.authenticated) return { status: 'CONFLICT' }
  if (ev.decisions.length === 0) return { status: 'NONE', reason: 'NO_GRANT' }
  if (ev.decisions.every(d => d.outcome === 'BLOCKED')) {
    return { status: 'BLOCKED', reasons: [...new Set(ev.decisions.map(d => d.reason_code ?? 'UNKNOWN'))].sort() }
  }
  const allowed = allowedSet(ev)
  const features = s.features
  const contract = DESTINATIONS.filter(d => d.capability.kind === 'CONTRACT' && allowed.has(d.capability.capability))
  const visible = contract.filter(d => features?.[d.feature] === true)
  if (visible.length === 0) return { status: 'NONE', reason: contract.length === 0 ? 'NO_GRANT' : 'NOT_ENABLED' }
  const role = roleContext(ev)
  if (role.kind !== 'SINGLE') return { status: 'GRANTED', role, entries: visible.map(d => ({ ...d, placement: 'other' })) }
  const order = roleDestinationIds(role.role)
  const mapped = order.flatMap(id => visible.filter(d => d.id === id)).map(d => ({ ...d, placement: 'role' as const }))
  const rest = visible.filter(d => !order.includes(d.id)).map(d => ({ ...d, placement: 'other' as const }))
  return { status: 'GRANTED', role, entries: [...mapped, ...rest] }
}

/** The destination that owns a pathname (longest matching prefix), for aria-current. */
export function destinationForPath<T extends { to: string }>(entries: readonly T[], pathname: string): T | undefined {
  let best: T | undefined
  for (const e of entries) {
    const hit = e.to === '/' ? pathname === '/' : pathname === e.to || pathname.startsWith(`${e.to}/`)
    if (hit && (!best || e.to.length > best.to.length)) best = e
  }
  return best
}
