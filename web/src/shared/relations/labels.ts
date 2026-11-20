import type { ResolutionState } from '../api'
import type { IconName } from '../ui/icons'
import type { Tone } from '../ui/primitives'

/**
 * RM-P01 relationship presentation. A relationship state is evidence about a
 * link between two native records; only RESOLVED is a confirmed, navigable
 * link. Every other state is shown, never hidden, and never linked.
 */
export const relationLabel = {
  RESOLVED: 'پیوند تأییدشده',
  RELATED: 'مرتبط (تأیید نشده)',
  UNKNOWN: 'نامعلوم',
  CONFLICT: 'تعارض',
  NOT_RECOVERABLE: 'قابل بازیابی نیست',
} as const satisfies Record<ResolutionState, string>

export const relationTone: Record<ResolutionState, Tone> = { RESOLVED: 'green', RELATED: 'blue', UNKNOWN: 'amber', CONFLICT: 'orange', NOT_RECOVERABLE: 'slate' }

export const relationIcon: Record<ResolutionState, IconName> = { RESOLVED: 'check', RELATED: 'split', UNKNOWN: 'question', CONFLICT: 'alert', NOT_RECOVERABLE: 'ban' }

/** Why an unconfirmed link is never navigable (shown beside it). */
export const unlinkedNote = 'پیوند تأیید نشده است؛ پیمایش ارائه نمی‌شود.'

export const casesBlockedDetail = 'خواندن فروش پشت پرچم ویژگی است (sales.read_shadow · FEATURE_DISABLED).'
export const invoicesBlockedDetail = 'خواندن صورت‌حساب و پرداخت پشت پرچم ویژگی است (billing_payments.read_shadow · FEATURE_DISABLED).'

export const text = (v: string | null | undefined) => (v === null || v === undefined || v === '' ? '—' : v)
