import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels of the Slice 12 payments contract. "Submitted" names the
 * target only: it never means legacy accepted the payment, never means a
 * Finance review happened and never means the stage was paid.
 */
export const unitLabel: Record<string, string> = { IRT: 'تومان', IRR: 'ریال' }

export const receiptStateLabel: Record<string, string> = {
  QUARANTINED: 'در قرنطینه (نهایی نشده)', FINALIZING: 'در حال نهایی‌سازی', READY: 'آماده و تاییدشده (هش بررسی شد)',
  STORAGE_ERROR: 'خطای ذخیره‌سازی؛ در حال تطبیق', NOT_RECOVERABLE: 'غیرقابل بازیابی؛ نیازمند بررسی مالی',
}
export const receiptStateTone: Record<string, Tone> = {
  QUARANTINED: 'amber', FINALIZING: 'amber', READY: 'green', STORAGE_ERROR: 'orange', NOT_RECOVERABLE: 'red',
}

export const evidenceStateLabel: Record<string, string> = {
  PENDING_FINALISATION: 'شواهد در انتظار نهایی‌سازی', READY: 'همه شواهد آماده', NOT_RECOVERABLE: 'شواهد غیرقابل بازیابی',
}
export const evidenceStateTone: Record<string, Tone> = { PENDING_FINALISATION: 'amber', READY: 'green', NOT_RECOVERABLE: 'red' }

export const uploadStateLabel: Record<string, string> = {
  UNBOUND: 'بارگذاری شده؛ هنوز به پرداختی متصل نیست (سند محسوب نمی‌شود)', BOUND: 'متصل به یک پرداخت', PURGED: 'منقضی و پاک شد',
}

export const attemptStateLabel: Record<string, string> = { PENDING_REVIEW: 'در انتظار بررسی مالی (بررسی نشده)' }

export const batchWritebackLabel: Record<string, string> = {
  AWAITING_EVIDENCE_FINALISATION: 'پس از نهایی شدن شواهد به سامانه قدیمی ارسال می‌شود',
  NOT_REQUESTED: 'چیزی به سامانه قدیمی ارسال نشد',
}

export const blockedPaymentLabel: Record<string, string> = {
  submit_payment_info_without_receipt: 'ثبت اطلاعات واریز بدون فیش',
  submit_payment_for_another_seller: 'ثبت پرداخت برای فروشنده دیگر / به نیابت',
  pay_online_gateway: 'پرداخت آنلاین (درگاه)',
  finance_approve_payment: 'تایید مالی پرداخت',
  finance_reject_payment: 'رد مالی پرداخت',
  issue_next_payment_stage: 'صدور مرحله بعدی پرداخت',
  operations_upgrade_payment: 'پرداخت ارتقای عملیات',
  refund_payment: 'استرداد وجه',
  issue_invoice: 'صدور فاکتور',
  return_case: 'برگشت پرونده',
}

export const gateActionLabel: Record<string, string> = { 'finance.attempt.approve': 'تایید مالی', 'finance.attempt.reject': 'رد مالی' }

export const reconciliationReasonLabel: Record<string, string> = {
  STORAGE_ERROR: 'خطای ذخیره‌سازی', NOT_RECOVERABLE: 'غیرقابل بازیابی', RECOVERED: 'بازیابی شد',
}
export const nextActionLabel: Record<string, string> = {
  RETRY_FROM_QUARANTINE: 'تلاش دوباره از قرنطینه (خودکار)', DOMAIN_OWNER_REVIEW: 'بررسی توسط مالک دامنه (مالی)', NO_AUTOMATIC_ACTION: 'بدون اقدام خودکار',
}

/** Integer money with its explicit unit; never converted or rounded. */
export function money(amount: number | null | undefined, unit: string | null | undefined): string {
  if (amount === null || amount === undefined || !unit) return 'قابل ارزیابی دقیق نیست'
  return `${amount.toLocaleString('fa-IR')} ${unitLabel[unit] ?? unit}`
}

export const codes = (xs: readonly string[] | undefined) => (xs && xs.length ? xs.join('، ') : '—')
