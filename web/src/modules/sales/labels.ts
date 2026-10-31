import type { Facet, Parity, WorkflowType } from '../../shared/api'
import type { IconName } from '../../shared/ui/icons'
import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels for the Slice 6 contract enums. Ownership states are
 * evidence states, never actions; inferred and proven holders never share a
 * label or a tone.
 */
export const workflowLabel = { LEGACY_LEAD: 'سرنخ قدیمی', DISTRIBUTION_ITEM: 'آیتم پخش' } as const satisfies Record<WorkflowType, string>

/** Current-holder presentation per parity: proven, derived, unknown, conflict, not recoverable. */
export const parityLabel = {
  MATCH: 'نگهدار تأییدشده (منبع و رویدادها یکسان)',
  HISTORY_GAP: 'نگهدار یکسان، سابقه ناقص',
  LEGACY_UNKNOWN: 'فقط استنتاج از رویدادها (منبع نگهدار ندارد)',
  TARGET_UNKNOWN: 'نامعلوم (رویدادها نگهدار را ثابت نمی‌کنند)',
  CONFLICT: 'تعارض: منبع و رویدادها متفاوت‌اند',
  MULTIPLE_CURRENT: 'تعارض: چند نگهدار هم‌زمان',
  NOT_RECOVERABLE: 'قابل بازیابی نیست',
} as const satisfies Record<Parity, string>

export const parityTone: Record<Parity, Tone> = {
  MATCH: 'green', HISTORY_GAP: 'amber', LEGACY_UNKNOWN: 'blue', TARGET_UNKNOWN: 'amber', CONFLICT: 'orange', MULTIPLE_CURRENT: 'orange', NOT_RECOVERABLE: 'slate',
}

export const parityIcon: Record<Parity, IconName> = {
  MATCH: 'check', HISTORY_GAP: 'alert', LEGACY_UNKNOWN: 'split', TARGET_UNKNOWN: 'question', CONFLICT: 'alert', MULTIPLE_CURRENT: 'alert', NOT_RECOVERABLE: 'ban',
}

export const facetLabel = { ORIGINAL_OWNER: 'مالک اولیه', ORIGINAL_SELLER: 'فروشنده اولیه', CREDIT_OWNER: 'صاحب اعتبار (کمیسیون)' } as const satisfies Record<Facet, string>

export const resolutionLabel: Record<string, string> = {
  RESOLVED: 'ثبت‌شده با شاهد', RELATED: 'مرتبط', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض', NOT_RECOVERABLE: 'قابل بازیابی نیست',
}

export const resolutionTone: Record<string, Tone> = { RESOLVED: 'green', RELATED: 'blue', UNKNOWN: 'amber', CONFLICT: 'orange', NOT_RECOVERABLE: 'slate' }

export const eventKindLabel: Record<string, string> = {
  HOLDER_ASSIGNED: 'تغییر نگهدار', HOLDER_RELEASED: 'آزادسازی به منبع', NO_HOLDER_CHANGE: 'بدون تغییر نگهدار', UNKNOWN: 'اقدام ناشناخته (مقدار خام حفظ شد)',
}

export const findingLabel: Record<string, string> = {
  MISSING_PRIOR_OWNER: 'نگهدار قبلی ثبت نشده', MISSING_TARGET_OWNER: 'نگهدار جدید ثبت نشده', HISTORY_GAP: 'شکاف در سابقه',
  MULTIPLE_CURRENT: 'چند نگهدار هم‌زمان', ORDERING_CONTRADICTION: 'تناقض ترتیب زمانی', MISSING_EVENT_HISTORY: 'سابقه رویداد وجود ندارد',
  SOURCE_CURRENT_MISMATCH: 'نگهدار منبع با رویدادها نمی‌خواند', UNKNOWN_ACTION: 'اقدام نگاشت‌نشده',
}

export const blockedDetail = 'خواندن فروش پشت پرچم ویژگی است (sales.read_shadow · FEATURE_DISABLED).'

export const text = (v: string | null | undefined) => (v === null || v === undefined || v === '' ? '—' : v)
