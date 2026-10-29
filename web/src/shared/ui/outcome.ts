import type { Outcome } from '../api'
import type { IconName } from './icons'
import type { Tone } from './primitives'

/**
 * Presentation of the seven frozen outcomes (SN-207 §7 colour meaning).
 * `satisfies Record<Outcome, …>` makes the map fail to compile if the
 * contract gains or loses an outcome; nothing is collapsed: UNKNOWN is not
 * FAILED and PARTIAL is not SUCCESS.
 */
export const outcomePresentation = {
  SUCCESS: { tone: 'green', icon: 'check', label: 'موفق' },
  PARTIAL: { tone: 'orange', icon: 'split', label: 'بخشی' },
  FAILED: { tone: 'red', icon: 'x', label: 'ناموفق' },
  UNKNOWN: { tone: 'amber', icon: 'question', label: 'نامعلوم — پیش از تلاش دوباره تطبیق دهید' },
  CONFLICT: { tone: 'orange', icon: 'alert', label: 'تعارض' },
  BLOCKED: { tone: 'red', icon: 'ban', label: 'مسدود' },
  UNAUTHORIZED: { tone: 'slate', icon: 'lock', label: 'مجاز نیست' },
} as const satisfies Record<Outcome, { tone: Tone; icon: IconName; label: string }>
