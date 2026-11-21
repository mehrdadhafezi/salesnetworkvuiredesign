import type { Evaluation } from '../../../app/navigation/access'
import { useEffectiveAccess } from '../../../app/navigation/NavigationAccess'
import type { ContractCapability } from '../../../app/navigation/destinations'

/**
 * Presentation-only capability lookup over the server's effective-access
 * evaluation of the CURRENT principal. 'unknown' (still loading, failed or
 * anonymous) is never treated as allowed. Hiding a section is not security:
 * every read is still authorised by the server, which may refuse anyway.
 */
export type CapabilityAnswer = 'allowed' | 'denied' | 'blocked' | 'unknown'

export function capabilityAnswer(ev: Evaluation, cap: ContractCapability): CapabilityAnswer {
  if (ev.state !== 'READY') return 'unknown'
  const d = ev.data.decisions.find(x => x.capability === cap)
  if (!d) return 'unknown'
  if (d.allowed && d.outcome === 'SUCCESS') return 'allowed'
  return d.outcome === 'BLOCKED' ? 'blocked' : 'denied'
}

export function useCapabilities<K extends ContractCapability>(caps: readonly K[]): Record<K, CapabilityAnswer> {
  const ev = useEffectiveAccess()
  return Object.fromEntries(caps.map(c => [c, capabilityAnswer(ev, c)])) as Record<K, CapabilityAnswer>
}

/** Resolving a user to a person reads the employment user link: both reads must be allowed. */
export function useCanResolveUsers(): boolean {
  const a = useCapabilities(['org.person.read', 'org.employment.read'] as const)
  return a['org.person.read'] === 'allowed' && a['org.employment.read'] === 'allowed'
}
