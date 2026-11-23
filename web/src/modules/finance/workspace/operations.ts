/**
 * RM-P07 financial operations and why none is offered in the workspace.
 *
 * Inspected before writing (main ce4a15a):
 * - The API router (`internal/platform/apiserver/server.go`, ContractMethods)
 *   exposes GET only for /finance/review/attempts and its item; there is no
 *   approve / reject route. The review evaluator
 *   (`internal/finance/review`) never opens a gate while D-41, OPD-08 and
 *   OPD-09 are open, and D-43 (every receipt READY) is shown per attempt.
 * - No posting, settlement, reversal, repayment, refund, payment retry,
 *   bank-account or reconciliation-write route exists for billing,
 *   payments or the finance shadow ledger (finance reads are GET only; the
 *   shadow rebuild is an offline job).
 * - The only financial write is the owning seller's manual payment batch
 *   (`POST /api/v1/payments/invoices/{id}/batches`, flag
 *   payments.manual_write, submitter-only row scope). It is a seller
 *   surface, never a Finance one, and it is not composed here.
 *
 * Every entry is therefore UNAVAILABLE with its grounded reason. No button,
 * link or disabled placeholder is rendered for any of them.
 */

export interface FinancialOperation {
  id: string
  label: string
  reason: string
  blockedBy: string[]
}

const NO_ROUTE = 'هیچ مسیر یا قرارداد API برای این عملیات وجود ندارد.'

export const FINANCIAL_OPERATIONS: FinancialOperation[] = [
  { id: 'approve', label: 'تأیید مالی پرداخت', reason: `${NO_ROUTE} ارزیاب تصمیم هرگز دروازه را باز نمی‌کند؛ ثبت‌کننده هرگز بررسی‌کننده مستقل نیست.`, blockedBy: ['D-41', 'OPD-08', 'OPD-09', 'D-43'] },
  { id: 'reject', label: 'رد مالی پرداخت', reason: `${NO_ROUTE} رد قدیمی تلاش‌ها فقط شاهد است و تصمیم سامانه جدید نیست.`, blockedBy: ['D-41', 'OPD-08', 'D-43'] },
  { id: 'post', label: 'ثبت در دفتر / تسویه', reason: `${NO_ROUTE} دفتر سایه فقط با کار آفلاین بازسازی می‌شود و هیچ ثبتی از رابط انجام نمی‌شود.`, blockedBy: ['D-37', 'D-38', 'D-41'] },
  { id: 'reverse', label: 'برگشت / اصلاح مالی', reason: `${NO_ROUTE} اصلاح فقط به‌صورت ردیف مرتبط جدید تعریف شده و هنوز مصوب نیست.`, blockedBy: ['D-38', 'OPD-08'] },
  { id: 'refund', label: 'استرداد وجه', reason: `${NO_ROUTE} لغو یا رد به معنای استرداد نیست؛ سیاست اثبات و اجرای استرداد باز است.`, blockedBy: ['OPD-04'] },
  { id: 'repay', label: 'پرداخت مجدد / صدور مرحله بعد', reason: `${NO_ROUTE} صدور مرحله بعدی پرداخت در هیچ قراردادی مجاز نیست.`, blockedBy: ['OPD-09', 'D-41'] },
  { id: 'retry', label: 'تلاش دوباره پرداخت', reason: `${NO_ROUTE} نتیجه نامعلوم باید پیش از هر تلاش دوباره تطبیق شود؛ تلاش کور ممنوع است.`, blockedBy: ['GATE-0 Outcome Unknown', 'FIN-IDEMPOTENCY'] },
  { id: 'reconcile', label: 'اصلاح یا تطبیق داده مالی', reason: `${NO_ROUTE} بررسی از اختیار اصلاح جداست؛ هیچ پیوند یا مبلغی بازنویسی نمی‌شود.`, blockedBy: ['FINANCE-PRODUCT-SPEC §13', 'OPD-06'] },
  { id: 'bank', label: 'تغییر حساب بانکی', reason: `${NO_ROUTE} داده بانکی در قلمرو این میز کار نیست.`, blockedBy: ['FINANCE-PRODUCT-SPEC §17'] },
]
