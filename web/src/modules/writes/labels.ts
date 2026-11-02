import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels of the Slice 11 command contract. "Applied" always names
 * the target only: the legacy write-back is reported separately and never
 * assumed. UNKNOWN is shown as "being reconciled", never as "try again".
 */
export const outcomeLabel: Record<string, string> = {
  NO_ANSWER: 'جواب نداده', CALLBACK: 'تماس مجدد', NOT_PURCHASED: 'عدم خرید',
}

export const modeLabel: Record<string, string> = {
  TARGET_WRITE_DISABLED: 'ثبت در سامانه جدید غیرفعال است (فقط سامانه قدیمی)',
  SHADOW_COMPARE: 'حالت مقایسه: فقط ارزیابی و ثبت قصد؛ چیزی اعمال نمی‌شود',
  CONTROLLED_WRITEBACK: 'ثبت کنترل‌شده؛ سامانه قدیمی همچنان مرجع است',
}

export const writebackLabel: Record<string, string> = {
  LEGACY_WRITEBACK_PENDING: 'در انتظار ثبت در سامانه قدیمی',
  NOT_REQUESTED: 'ثبتی در سامانه قدیمی درخواست نشد',
  PENDING: 'در صف ثبت در سامانه قدیمی', WAITING_DEPENDENCY: 'منتظر ثبت درخواست اصلی در سامانه قدیمی',
  APPLIED: 'در سامانه قدیمی ثبت شد', UNKNOWN: 'نتیجه نامعلوم؛ در حال تطبیق (تکرار نکنید)',
  REJECTED: 'سامانه قدیمی رد کرد؛ نیازمند بررسی مالک دامنه', RECONCILIATION_REQUIRED: 'نیازمند تطبیق',
  COMPARED: 'مقایسه شد (بدون ثبت)',
}
export const writebackTone: Record<string, Tone> = {
  PENDING: 'amber', WAITING_DEPENDENCY: 'amber', APPLIED: 'green', UNKNOWN: 'slate', REJECTED: 'red', RECONCILIATION_REQUIRED: 'orange',
  COMPARED: 'blue', LEGACY_WRITEBACK_PENDING: 'amber', NOT_REQUESTED: 'neutral',
}

export const parityLabel: Record<string, string> = {
  MATCH: 'منطبق با سامانه قدیمی', MISMATCH: 'مغایر', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض', INCOMPLETE: 'ناقص',
}

export const hrTypeLabel: Record<string, string> = { transfer_seller: 'جابجایی فروشنده', terminate_seller: 'قطع همکاری فروشنده' }

export const hrStatusLabel: Record<string, string> = {
  pending_review: 'در انتظار تایید سلسله‌مراتبی', pending_hr: 'در انتظار منابع انسانی (اعمال نشده)', approved: 'تایید و اعمال شده',
  rejected: 'رد شده', cancelled: 'لغو شده', failed: 'خطا در اعمال',
}

export const blockedLabel: Record<string, string> = {
  record_duplicate_outcome: 'ثبت «تکراری»', record_pre_invoice_outcome: 'ثبت «پیش‌فاکتور»', issue_invoice: 'صدور فاکتور',
  return_case: 'برگشت پرونده', transfer_case: 'انتقال پرونده', allocate_case: 'تخصیص پرونده', mark_ready: 'آماده تبدیل',
  apply_final: 'اعمال نهایی منابع انسانی',
}

export const reasonText = (codes: readonly string[]) => (codes.length ? `کد: ${codes.join('، ')}` : '')
