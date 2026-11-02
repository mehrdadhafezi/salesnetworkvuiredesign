import type { IconName } from '../../shared/ui/icons'
import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels for the Slice 10 Biawin / fulfillment READ shadow contract.
 * Every uncertainty state has its own label and tone; none is ever shown as
 * zero, active or complete by default. Raw legacy values are always shown
 * beside the mapped value.
 */
export const subStateLabel: Record<string, string> = {
  ACTIVE: 'فعال (طبق وضعیت ثبت‌شده قدیمی)', SUSPENDED: 'معلق (طبق وضعیت ثبت‌شده قدیمی)', UNKNOWN: 'نامعلوم (مقدار خام حفظ شد)', EMPTY: 'خالی در منبع',
}
export const subStateTone: Record<string, Tone> = { ACTIVE: 'green', SUSPENDED: 'amber', UNKNOWN: 'slate', EMPTY: 'neutral' }
export const subStateIcon: Record<string, IconName> = { ACTIVE: 'check', SUSPENDED: 'alert', UNKNOWN: 'question', EMPTY: 'inbox' }

export const linkLabel: Record<string, string> = {
  RESOLVED: 'پیوند اثبات‌شده', RELATED: 'مرتبط (نه یکسان)', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض', NOT_RECOVERABLE: 'قابل بازیابی نیست',
}
export const linkTone: Record<string, Tone> = { RESOLVED: 'green', RELATED: 'blue', UNKNOWN: 'slate', CONFLICT: 'orange', NOT_RECOVERABLE: 'slate' }

export const activationLabel: Record<string, string> = {
  LEGACY_ACTIVATION_RECORDED: 'زمان فعال‌سازی در منبع قدیمی ثبت شده', NOT_EVIDENCED: 'زمان فعال‌سازی ثبت نشده (از زمان ورود استنتاج نمی‌شود)',
}

export const kindLabel: Record<string, string> = {
  SERVICE_FOLLOWUP: 'پیگیری خدمت', EXECUTION: 'اجرای عملیات', UPGRADE_EXECUTION: 'اجرای ارتقا', SHIPPING: 'ارسال',
}

export const caseStatusLabel: Record<string, string> = {
  NOT_STARTED: 'شروع نشده', IN_PROGRESS: 'در جریان', AWAITING_PAYMENT: 'در انتظار پرداخت', AWAITING_PROVIDER: 'در انتظار تأمین‌کننده',
  PROVIDER_ACTION_RECORDED: 'اقدام تأمین‌کننده ثبت شده', OUTCOME_RECORDED: 'نتیجه ثبت شده', SHIPPED: 'ارسال شده', DELIVERED: 'تحویل شده',
  RETURNED: 'مرجوع شده', NOT_EXECUTED: 'اجرا نشده', CANCELLED: 'لغو شده', COMPLETED: 'تکمیل‌شده (فقط همین پرونده)', FAILED_AT_PROVIDER: 'ناموفق نزد تأمین‌کننده',
  UNKNOWN: 'نامعلوم (مقدار خام حفظ شد)', EMPTY: 'خالی در منبع',
}
export const caseStatusTone: Record<string, Tone> = {
  NOT_STARTED: 'neutral', IN_PROGRESS: 'blue', AWAITING_PAYMENT: 'amber', AWAITING_PROVIDER: 'amber', PROVIDER_ACTION_RECORDED: 'violet',
  OUTCOME_RECORDED: 'teal', SHIPPED: 'blue', DELIVERED: 'green', RETURNED: 'orange', NOT_EXECUTED: 'orange', CANCELLED: 'slate', COMPLETED: 'green',
  FAILED_AT_PROVIDER: 'red', UNKNOWN: 'slate', EMPTY: 'neutral',
}

export const roleLabel: Record<string, string> = { MANAGER: 'مدیر', EXPERT: 'کارشناس', HANDLER: 'مسئول' }

export const evidenceLabel: Record<string, string> = { EXECUTED: 'اجرا شد', NOT_EXECUTED: 'اجرا نشد', UNKNOWN: 'نامعلوم', NONE: 'بدون نتیجه' }

export const eventClassLabel: Record<string, string> = {
  SUBSCRIPTION_EVENT: 'رویداد اشتراک', ITEM_EVENT: 'رویداد آیتم', PROVIDER_EVENT_EVIDENCE: 'شاهد رویداد تأمین‌کننده (فراخوانی نیست)',
}

export const parityLabel: Record<string, string> = {
  MATCH: 'منطبق', MISMATCH: 'مغایر', UNKNOWN: 'نامعلوم (صفر نیست)', INCOMPLETE: 'ناقص',
}
export const parityTone: Record<string, Tone> = { MATCH: 'green', MISMATCH: 'red', UNKNOWN: 'slate', INCOMPLETE: 'amber' }
export const parityIcon: Record<string, IconName> = { MATCH: 'check', MISMATCH: 'x', UNKNOWN: 'question', INCOMPLETE: 'alert' }

export const resultLabel: Record<string, string> = {
  MATCHED: 'منطبق', MISSING_TARGET: 'در هدف نیست', EXTRA_TARGET: 'اضافه در هدف', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض',
  NOT_RECOVERABLE: 'قابل بازیابی نیست', INACTIVE_BOTH: 'در هر دو غیرفعال',
}
export const resultTone: Record<string, Tone> = {
  MATCHED: 'green', MISSING_TARGET: 'red', EXTRA_TARGET: 'red', UNKNOWN: 'slate', CONFLICT: 'orange', NOT_RECOVERABLE: 'slate', INACTIVE_BOTH: 'neutral',
}

export const coverageLabel: Record<string, string> = {
  AVAILABLE: 'پوشش کامل', PARTIAL: 'پوشش ناقص', MISSING: 'منبع موجود نیست', UNKNOWN: 'پوشش نامعلوم', NOT_RECOVERABLE: 'قابل بازیابی نیست', STALE: 'منبع کهنه',
}
export const coverageTone: Record<string, Tone> = { AVAILABLE: 'green', PARTIAL: 'amber', MISSING: 'red', UNKNOWN: 'slate', NOT_RECOVERABLE: 'slate', STALE: 'orange' }

export const freshnessLabel: Record<string, string> = { FRESH: 'تازه (۲۴ ساعت اخیر)', STALE: 'کهنه (قدیمی‌تر از ۲۴ ساعت)' }

export const blockedDetail = 'نمای خواندنی بیاوین و اجرای خدمت پشت پرچم ویژگی است (biawin_fulfillment.read_shadow · FEATURE_DISABLED).'

export const text = (v: string | null | undefined) => (v === null || v === undefined || v === '' ? '—' : v)
