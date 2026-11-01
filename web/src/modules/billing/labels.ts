import type { ApprovalProvenance, PaymentChannel, TargetMapping, TotalClass } from '../../shared/api'
import type { IconName } from '../../shared/ui/icons'
import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels for the Slice 7 contract enums. Invoice status fields,
 * payment stage, payment attempt, gateway and receipt states each have their
 * own vocabulary and are never shown as one generic status. No label names
 * a canonical invoice status (TAD #20) or a canonical total (TAD #19).
 */
export const stateFieldLabel: Record<string, string> = {
  MAIN: 'وضعیت اصلی فاکتور (خام)', INVOICE: 'وضعیت فاکتور (خام)', PAYMENT: 'وضعیت پرداخت فاکتور (خام)',
  WORKFLOW: 'گردش کار پرداخت (خام)', RETURN: 'برگشت مالی (خام)', LEGACY_DISPLAY: 'نمایش قدیمی (مشتق، فقط شاهد)',
}

export const vocabularyLabel: Record<string, string> = {
  KNOWN_VOCABULARY: 'در واژگان شناخته‌شده', UNKNOWN: 'ناشناخته (مقدار خام حفظ شد)', EMPTY: 'خالی', LEGACY_DERIVED: 'مشتق قدیمی (تصمیم‌گیر نیست)',
}
export const vocabularyTone: Record<string, Tone> = { KNOWN_VOCABULARY: 'blue', UNKNOWN: 'amber', EMPTY: 'slate', LEGACY_DERIVED: 'violet' }

export const mappingLabel = { MAPPED: 'نگاشت صریح', UNKNOWN: 'ناشناخته (خام حفظ شد)', EMPTY: 'خالی' } as const satisfies Record<TargetMapping, string>

export const stageStateLabel: Record<string, string> = {
  PENDING: 'مرحله: در انتظار', AWAITING_REVIEW: 'مرحله: در انتظار بررسی', ACCEPTED: 'مرحله: پذیرفته‌شده (ثبت قدیمی)', REJECTED: 'مرحله: ردشده', CANCELLED: 'مرحله: لغوشده',
}

export const attemptStateLabel: Record<string, string> = {
  PENDING: 'تلاش: در انتظار', ACCEPTED: 'تلاش: پذیرفته‌شده (نه بررسی مالی، نه تسویه)', REJECTED: 'تلاش: ردشده', CANCELLED: 'تلاش: لغوشده',
  RECONCILIATION_REQUIRED: 'تلاش: نیازمند تطبیق',
}

export const gatewayStateLabel: Record<string, string> = { INITIATED: 'درگاه: آغازشده', VERIFIED: 'درگاه: تأییدشده توسط درگاه', CANCELLED_AT_GATEWAY: 'درگاه: لغو در درگاه' }

export const receiptStateLabel: Record<string, string> = {
  REFERENCE_ONLY: 'فقط ارجاع (فایل بررسی نشده)', AVAILABLE: 'در دسترس', MISSING: 'مفقود', NOT_RECOVERABLE: 'قابل بازیابی نیست', UNKNOWN: 'نامعلوم',
}

export const channelLabel = { GATEWAY: 'درگاه', MANUAL_CARD: 'کارت به کارت (دستی)', UNKNOWN: 'نامعلوم' } as const satisfies Record<PaymentChannel, string>

export const provenanceLabel = {
  GATEWAY_AUTO: 'پذیرش خودکار درگاه (بررسی مالی نیست)', LEGACY_MANUAL_APPROVAL: 'تأیید دستی قدیمی', NOT_APPROVED: 'تأیید نشده', UNKNOWN: 'نامعلوم',
} as const satisfies Record<ApprovalProvenance, string>

export const classLabel = {
  MATCH: 'هم‌خوان', MISMATCH: 'ناهم‌خوان', UNKNOWN: 'نامعلوم (شاهد کافی نیست)', INCOMPLETE: 'ناقص', CONFLICT: 'تعارض در مبالغ اعلامی',
} as const satisfies Record<TotalClass, string>
export const classTone: Record<TotalClass, Tone> = { MATCH: 'green', MISMATCH: 'orange', UNKNOWN: 'slate', INCOMPLETE: 'amber', CONFLICT: 'red' }
export const classIcon: Record<TotalClass, IconName> = { MATCH: 'check', MISMATCH: 'alert', UNKNOWN: 'question', INCOMPLETE: 'alert', CONFLICT: 'alert' }

export const sumStateLabel: Record<string, string> = { COMPLETE: 'کامل', INCOMPLETE: 'ناقص', NONE: 'بدون سهم (صفر نیست)', OVERFLOW: 'سرریز (مقداری ادعا نمی‌شود)' }

export const unitLabel: Record<string, string> = { IRT: 'IRT (تومان)', IRR: 'IRR (ریال)', UNATTRIBUTED: 'بدون واحد قابل انتساب' }

export const declaredLabel: Record<string, string> = {
  FINAL_TOTAL: 'جمع نهایی اعلامی', PAYMENT_TOTAL_AMOUNT: 'جمع پرداخت اعلامی', ORIGINAL_TOTAL: 'جمع اولیه', DISCOUNT_TOTAL: 'تخفیف',
  PRODUCT_PRICE: 'قیمت محصول', CURRENT_DUE_AMOUNT: 'مبلغ سررسید فعلی', PAID_TOTAL_AMOUNT: 'پرداخت‌شده (اعلامی)', REMAINING_AMOUNT: 'باقی‌مانده (اعلامی)',
}

export const blockedDetail = 'خواندن صورت‌حساب و پرداخت پشت پرچم ویژگی است (billing_payments.read_shadow · FEATURE_DISABLED).'

export const text = (v: string | null | undefined) => (v === null || v === undefined || v === '' ? '—' : v)
