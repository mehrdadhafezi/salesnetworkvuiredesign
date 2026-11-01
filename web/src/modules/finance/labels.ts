import type { IconName } from '../../shared/ui/icons'
import type { Tone } from '../../shared/ui/primitives'

/**
 * Persian labels for the Slice 9 finance SHADOW contract. Every uncertainty
 * state has its own label and tone; none is ever rendered as zero. Accrual
 * and cash are labelled side by side with the same weight (D-41 open).
 */
export const basisLabel: Record<string, string> = {
  ACCRUAL_SHADOW: 'مبنای تعهدی (سایه؛ نهایی نیست)', CASH_SHADOW: 'مبنای نقدی (سایه؛ نهایی نیست)', WALLET_MIRROR_SHADOW: 'آینه کیف پول قدیمی (سایه)',
}

export const reconLabel: Record<string, string> = {
  MATCH: 'منطبق', MISMATCH: 'مغایر', UNKNOWN: 'نامعلوم (صفر نیست)', INCOMPLETE: 'ناقص', CONFLICT: 'تعارض', NOT_RECOVERABLE: 'قابل بازیابی نیست',
  NOT_COMPARABLE: 'رقم قدیمی قابل مقایسه ندارد',
}
export const reconTone: Record<string, Tone> = {
  MATCH: 'green', MISMATCH: 'red', UNKNOWN: 'slate', INCOMPLETE: 'amber', CONFLICT: 'orange', NOT_RECOVERABLE: 'slate', NOT_COMPARABLE: 'neutral',
}
export const reconIcon: Record<string, IconName> = {
  MATCH: 'check', MISMATCH: 'x', UNKNOWN: 'question', INCOMPLETE: 'alert', CONFLICT: 'split', NOT_RECOVERABLE: 'ban', NOT_COMPARABLE: 'inbox',
}

export const entryStatusLabel: Record<string, string> = {
  CONSISTENT: 'سازگار', CONFLICT: 'تعارض (تکراری احتمالی؛ حذف نشد)', UNRESOLVED_CORRECTION: 'اصلاح اثبات‌نشده (پیوندی ساخته نشد)',
}

export const coverageLabel: Record<string, string> = {
  AVAILABLE: 'پوشش کامل', PARTIAL: 'پوشش ناقص', MISSING: 'منبع موجود نیست', UNKNOWN: 'پوشش نامعلوم', NOT_RECOVERABLE: 'قابل بازیابی نیست', STALE: 'منبع کهنه',
}
export const coverageTone: Record<string, Tone> = { AVAILABLE: 'green', PARTIAL: 'amber', MISSING: 'red', UNKNOWN: 'slate', NOT_RECOVERABLE: 'slate', STALE: 'orange' }

export const freshnessLabel: Record<string, string> = { FRESH: 'تازه', STALE: 'کهنه (قدیمی‌تر از ۲۴ ساعت)', UNKNOWN: 'تازگی نامعلوم' }

export const sourceLabel: Record<string, string> = {
  billing: 'صورت‌حساب', payments: 'پرداخت', legacy_wallets: 'کیف پول‌های قدیمی', legacy_wallet_transactions: 'تراکنش‌های کیف پول قدیمی',
}

export const categoryLabel: Record<string, string> = {
  customer_receivable: 'دریافتنی مشتری (به ازای فاکتور)', revenue_by_product_kind: 'درآمد (نوع محصول نامشخص)', cash_bank_clearing: 'تسویه نقد/بانک (به ازای منشأ تأیید)',
  commission_payable: 'پورسانت پرداختنی (کیف پول کارکنان)', legacy_wallet_unclassified: 'کیف پول قدیمی طبقه‌بندی‌نشده', legacy_wallet_contra: 'حساب مقابل آینه کیف پول',
}

export const figureLabel: Record<string, string> = { WALLET_BALANCE: 'مانده کیف پول قدیمی', INVOICE_REMAINING: 'مانده فاکتور قدیمی', INVOICE_PAID: 'پرداخت‌شده فاکتور قدیمی' }

export const blockedDetail = 'دفتر سایه مالی پشت پرچم ویژگی است (finance.shadow_ledger_read · FEATURE_DISABLED).'

export const text = (v: string | null | undefined) => (v === null || v === undefined || v === '' ? '—' : v)
