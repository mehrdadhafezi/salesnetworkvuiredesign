import type { IconName } from '../../shared/ui/icons'
import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels for the Slice 8 reporting contract. Every uncertainty
 * state has its own label and tone; none is ever rendered as zero.
 */
export const valueStateLabel: Record<string, string> = {
  VALUE: 'مقدار ارزیابی‌شده', INCOMPLETE: 'ناقص (پوشش کامل نیست)', UNKNOWN: 'نامعلوم (بدون مقدار؛ صفر نیست)', NOT_EVALUATED: 'ارزیابی نشده (NOT FINAL)',
}
export const valueStateTone: Record<string, Tone> = { VALUE: 'green', INCOMPLETE: 'amber', UNKNOWN: 'slate', NOT_EVALUATED: 'violet' }
export const valueStateIcon: Record<string, IconName> = { VALUE: 'check', INCOMPLETE: 'alert', UNKNOWN: 'question', NOT_EVALUATED: 'ban' }

export const freshnessLabel: Record<string, string> = { FRESH: 'تازه', STALE: 'کهنه (قدیمی‌تر از قرارداد)', UNKNOWN: 'تازگی نامعلوم' }
export const freshnessTone: Record<string, Tone> = { FRESH: 'green', STALE: 'orange', UNKNOWN: 'slate' }

export const coverageLabel: Record<string, string> = {
  AVAILABLE: 'پوشش کامل', PARTIAL: 'پوشش ناقص', MISSING: 'منبع موجود نیست', UNKNOWN: 'پوشش نامعلوم', NOT_RECOVERABLE: 'قابل بازیابی نیست', STALE: 'منبع کهنه',
}
export const coverageTone: Record<string, Tone> = { AVAILABLE: 'green', PARTIAL: 'amber', MISSING: 'red', UNKNOWN: 'slate', NOT_RECOVERABLE: 'slate', STALE: 'orange' }

export const timeBasisLabel: Record<string, string> = {
  CURRENT_SNAPSHOT: 'تصویر فعلی (در زمان برش اجرا)', EVENT_RANGE: 'بازه رویداد (اولین مشاهده توسط ورود هدف)', HISTORICAL_AS_OF: 'تاریخی در لحظه (آن‌طور که ثبت شده بود)',
}

export const reportLabel: Record<string, string> = { cases: 'پرونده‌ها', custody: 'نگهداری', invoices: 'فاکتورها', payments: 'تلاش‌های پرداخت' }

export const sourceLabel: Record<string, string> = { cases: 'پرونده‌ها', custody: 'نگهداری', billing: 'صورت‌حساب', payments: 'پرداخت', customers: 'مشتریان' }

export const blockedDetail = 'گزارش‌گیری پشت پرچم ویژگی است (reporting.read_shadow · FEATURE_DISABLED).'

export const text = (v: string | null | undefined) => (v === null || v === undefined || v === '' ? '—' : v)
