import type { DsIconName } from '../../../shared/ui/ds'
import { C, type CountSource } from './reports'

/**
 * RM-P07 Finance exception queues. Every queue is a state already stored
 * by an existing contract and listed with an existing (or RM-P07 read-only)
 * server filter; nothing is derived client-side, nothing is resolved, and
 * no mismatch is invented. A queue without a list contract says so.
 *
 * Inspected before writing (main ce4a15a):
 * - GET /api/v1/payments/attempts filters: channel, state, state_mapping,
 *   approval_provenance, presence, invoice_link_state (+ RM-P07
 *   has_state_regression). payments.attempt.read.
 * - GET /api/v1/invoices filters: presence, customer_link_state,
 *   case_link_state, has_unknown_status. billing.invoice.read.
 * - GET /api/v1/finance/review/attempts filters: invoice_id (+ RM-P07
 *   evidence_state). finance.review.read. Only PENDING_REVIEW rows exist:
 *   no reviewed, accepted or received submission state exists.
 * - Legacy receipt references have no list endpoint (per invoice only);
 *   per-invoice amount classes (MISMATCH / CONFLICT) have no list filter.
 */

export type ListRead =
  | { read: 'attempts'; params: Record<string, string> }
  | { read: 'invoices'; params: Record<string, string> }
  | { read: 'review'; params: Record<string, string> }
  | { read: 'none'; why: string }

export interface ExceptionDef {
  id: string
  group: 'relationships' | 'states' | 'evidence' | 'provenance'
  label: string
  icon: DsIconName
  meaning: string
  count: CountSource
  list: ListRead
}

export const EXCEPTION_GROUPS: { id: ExceptionDef['group']; label: string }[] = [
  { id: 'relationships', label: 'پیوندهای حل‌نشده' },
  { id: 'states', label: 'وضعیت‌های نامعلوم، ردشده یا نیازمند تطبیق' },
  { id: 'evidence', label: 'شواهد رسید و فیش' },
  { id: 'provenance', label: 'منشأ و پوشش ناقص' },
]

// Lists that match a present-only aggregate add presence=PRESENT, so the
// rows listed are exactly the rows counted.
const present = { presence: 'PRESENT' }

export const EXCEPTIONS: ExceptionDef[] = [
  { id: 'attempt-no-invoice', group: 'relationships', label: 'تلاش پرداخت بدون فاکتور حل‌شده', icon: 'split',
    meaning: 'تلاش قدیمی که ارجاع فاکتورش به فاکتور واردشده‌ای حل نشد (invoice_link_state = UNKNOWN). با شماره تلفن یا مبلغ به فاکتوری وصل نمی‌شود.',
    count: C.attemptsWithoutInvoice, list: { read: 'attempts', params: { invoice_link_state: 'UNKNOWN', ...present } } },
  { id: 'invoice-customer-conflict', group: 'relationships', label: 'پیوند مشتری فاکتور: تعارض', icon: 'alert',
    meaning: 'فاکتورهایی که پیوند مشتری آن‌ها CONFLICT است؛ هیچ مشتری‌ای انتخاب یا استنتاج نمی‌شود.',
    count: C.customerLink('CONFLICT'), list: { read: 'invoices', params: { customer_link_state: 'CONFLICT', ...present } } },
  { id: 'invoice-customer-unknown', group: 'relationships', label: 'پیوند مشتری فاکتور: نامعلوم', icon: 'question',
    meaning: 'فاکتورهایی بدون پیوند مشتری قابل اثبات (UNKNOWN).',
    count: C.customerLink('UNKNOWN'), list: { read: 'invoices', params: { customer_link_state: 'UNKNOWN', ...present } } },
  { id: 'invoice-customer-nr', group: 'relationships', label: 'پیوند مشتری فاکتور: غیرقابل بازیابی', icon: 'ban',
    meaning: 'پیوند مشتری از منبع قابل بازیابی نیست (NOT_RECOVERABLE).',
    count: C.customerLink('NOT_RECOVERABLE'), list: { read: 'invoices', params: { customer_link_state: 'NOT_RECOVERABLE', ...present } } },
  { id: 'invoice-case-conflict', group: 'relationships', label: 'پیوند پرونده فاکتور: تعارض', icon: 'alert',
    meaning: 'فاکتورهایی که پیوند پرونده آن‌ها CONFLICT است.',
    count: C.caseLink('CONFLICT'), list: { read: 'invoices', params: { case_link_state: 'CONFLICT', ...present } } },
  { id: 'invoice-case-unknown', group: 'relationships', label: 'پیوند پرونده فاکتور: نامعلوم', icon: 'question',
    meaning: 'فاکتورهایی بدون پیوند پرونده قابل اثبات (UNKNOWN).',
    count: C.caseLink('UNKNOWN'), list: { read: 'invoices', params: { case_link_state: 'UNKNOWN', ...present } } },
  { id: 'invoice-case-nr', group: 'relationships', label: 'پیوند پرونده فاکتور: غیرقابل بازیابی', icon: 'ban',
    meaning: 'پیوند پرونده از منبع قابل بازیابی نیست (NOT_RECOVERABLE).',
    count: C.caseLink('NOT_RECOVERABLE'), list: { read: 'invoices', params: { case_link_state: 'NOT_RECOVERABLE', ...present } } },

  { id: 'attempt-unknown-state', group: 'states', label: 'وضعیت خام ناشناخته تلاش', icon: 'question',
    meaning: 'مقدار خام وضعیت تلاش در واژگان نگاشت نیست (state_mapping = UNKNOWN)؛ مقدار خام حفظ شده و حدس زده نمی‌شود.',
    count: C.attemptMapping('UNKNOWN'), list: { read: 'attempts', params: { state_mapping: 'UNKNOWN', ...present } } },
  { id: 'attempt-empty-state', group: 'states', label: 'وضعیت خالی تلاش', icon: 'question',
    meaning: 'وضعیت خام تلاش خالی است (state_mapping = EMPTY).',
    count: C.attemptMapping('EMPTY'), list: { read: 'attempts', params: { state_mapping: 'EMPTY', ...present } } },
  { id: 'attempt-reconciliation', group: 'states', label: 'تلاش نیازمند تطبیق', icon: 'swap',
    meaning: 'وضعیت نگاشت‌شده RECONCILIATION_REQUIRED (از جمله reconcile_error قدیمی). اینجا هیچ تطبیقی انجام نمی‌شود.',
    count: C.attemptState('RECONCILIATION_REQUIRED'), list: { read: 'attempts', params: { state: 'RECONCILIATION_REQUIRED', ...present } } },
  { id: 'attempt-rejected', group: 'states', label: 'تلاش ردشده', icon: 'ban',
    meaning: 'وضعیت نگاشت‌شده REJECTED در سامانه قدیمی؛ رد قدیمی، تصمیم بررسی مالی سامانه جدید نیست.',
    count: C.attemptState('REJECTED'), list: { read: 'attempts', params: { state: 'REJECTED', ...present } } },
  { id: 'attempt-cancelled', group: 'states', label: 'تلاش لغوشده', icon: 'ban',
    meaning: 'وضعیت نگاشت‌شده CANCELLED (از جمله لغو در درگاه)؛ لغو به معنای استرداد وجه نیست.',
    count: C.attemptState('CANCELLED'), list: { read: 'attempts', params: { state: 'CANCELLED', ...present } } },
  { id: 'attempt-regression', group: 'states', label: 'بازگشت وضعیت در سابقه تلاش', icon: 'repeat',
    meaning: 'تلاش‌هایی که در سابقه افزایشی آن‌ها دست‌کم یک ردیف «بازگشت از پذیرفته‌شده» ثبت شده است. شمارنده تعداد ردیف سابقه است، نه تلاش.',
    count: C.regressions, list: { read: 'attempts', params: { has_state_regression: 'true' } } },
  { id: 'attempt-unknown-provenance', group: 'states', label: 'منشأ تأیید نامعلوم', icon: 'question',
    meaning: 'منشأ تأیید تلاش قابل تعیین نیست (approval_provenance = UNKNOWN).',
    count: C.attemptProvenance('UNKNOWN'), list: { read: 'attempts', params: { approval_provenance: 'UNKNOWN', ...present } } },
  { id: 'attempt-unknown-channel', group: 'states', label: 'کانال پرداخت نامعلوم', icon: 'question',
    meaning: 'کانال تلاش (درگاه / کارت به کارت) قابل تعیین نیست.',
    count: C.attemptChannel('UNKNOWN'), list: { read: 'attempts', params: { channel: 'UNKNOWN', ...present } } },

  { id: 'review-evidence-pending', group: 'evidence', label: 'پرداخت ثبت‌شده: فیش در انتظار نهایی‌سازی', icon: 'inbox',
    meaning: 'پرداخت‌های ثبت‌شده در سامانه جدید که همه فیش‌هایشان هنوز آماده نیست (evidence_state = PENDING_FINALISATION)؛ تصمیم مالی طبق D-43 مسدود است.',
    count: C.reviewQueue, list: { read: 'review', params: { evidence_state: 'PENDING_FINALISATION' } } },
  { id: 'review-evidence-nr', group: 'evidence', label: 'پرداخت ثبت‌شده: فیش غیرقابل بازیابی', icon: 'alert',
    meaning: 'پرداخت‌های ثبت‌شده‌ای که شواهدشان غیرقابل بازیابی است (evidence_state = NOT_RECOVERABLE)؛ مالک دامنه باید بررسی کند.',
    count: C.reviewQueue, list: { read: 'review', params: { evidence_state: 'NOT_RECOVERABLE' } } },
  { id: 'legacy-receipt-missing', group: 'evidence', label: 'ارجاع رسید قدیمی: مفقود', icon: 'inbox',
    meaning: 'ارجاع‌های رسید قدیمی با وضعیت MISSING. فهرست سراسری ارجاع رسید در قرارداد نیست؛ رسیدهای هر فاکتور در جزئیات همان فاکتور دیده می‌شوند.',
    count: C.receiptsMissing, list: { read: 'none', why: 'ارجاع‌های رسید فقط به تفکیک فاکتور (GET /api/v1/invoices/{id}/receipts) خوانده می‌شوند؛ فهرست سراسری با فیلتر وضعیت وجود ندارد.' } },
  { id: 'legacy-receipt-unknown', group: 'evidence', label: 'ارجاع رسید قدیمی: نامعلوم', icon: 'question',
    meaning: 'ارجاع‌های رسید قدیمی با وضعیت UNKNOWN.',
    count: C.receiptsUnknown, list: { read: 'none', why: 'ارجاع‌های رسید فقط به تفکیک فاکتور خوانده می‌شوند؛ فهرست سراسری با فیلتر وضعیت وجود ندارد.' } },
  { id: 'target-receipt-nr', group: 'evidence', label: 'فیش سامانه جدید: غیرقابل بازیابی', icon: 'alert',
    meaning: 'فیش‌هایی که پس از خطای ذخیره‌سازی قابل بازیابی نیستند. تطبیق‌ها در بخش شواهد رسید فهرست شده‌اند؛ اقدام اصلاحی وجود ندارد.',
    count: C.targetReceipt('NOT_RECOVERABLE'), list: { read: 'review', params: { evidence_state: 'NOT_RECOVERABLE' } } },

  { id: 'invoice-unknown-status', group: 'provenance', label: 'فاکتور با وضعیت خام ناشناخته', icon: 'question',
    meaning: 'فاکتورهایی که دست‌کم یک فیلد وضعیت خام نگاشت‌نشده دارند (has_unknown_status = true).',
    count: C.unknownInvoiceStatus, list: { read: 'invoices', params: { has_unknown_status: 'true', ...present } } },
  { id: 'invoice-absent', group: 'provenance', label: 'فاکتور غایب در آخرین واردسازی کامل', icon: 'alert',
    meaning: 'فاکتورهایی که در آخرین دسته کامل دیده نشدند (ABSENT_IN_LATEST)؛ حذف، لغو یا تسویه نشده‌اند.',
    count: C.absent, list: { read: 'invoices', params: { presence: 'ABSENT_IN_LATEST' } } },
  { id: 'attempt-absent', group: 'provenance', label: 'تلاش غایب در آخرین واردسازی کامل', icon: 'alert',
    meaning: 'تلاش‌هایی که در آخرین دسته کامل دیده نشدند؛ حذف یا برگشت نشده‌اند.',
    count: C.absent, list: { read: 'attempts', params: { presence: 'ABSENT_IN_LATEST' } } },
  { id: 'invoice-no-items', group: 'provenance', label: 'فاکتور بدون قلم', icon: 'file',
    meaning: 'فاکتورهای حاضری که هیچ قلمی برایشان وارد نشده است.',
    count: C.invoicesWithoutItems, list: { read: 'none', why: 'فیلتر فهرست فاکتور برای «بدون قلم» در قرارداد نیست؛ فقط شمارنده تجمیعی موجود است.' } },
  { id: 'invoice-no-stages', group: 'provenance', label: 'فاکتور بدون مرحله پرداخت', icon: 'file',
    meaning: 'فاکتورهای حاضری که هیچ مرحله پرداختی برایشان وارد نشده است.',
    count: C.invoicesWithoutStages, list: { read: 'none', why: 'فیلتر فهرست فاکتور برای «بدون مرحله» در قرارداد نیست؛ فقط شمارنده تجمیعی موجود است.' } },
  { id: 'amount-unrepresentable', group: 'provenance', label: 'مبلغ غیرقابل نمایش دقیق', icon: 'alert',
    meaning: 'مبالغ خامی که بدون گرد کردن قابل نمایش نیستند (فاکتور و تلاش جدا شمرده می‌شوند و جمع نمی‌شوند).',
    count: C.attemptsUnrepresentable, list: { read: 'none', why: 'فیلتر فهرست برای مبلغ غیرقابل نمایش در قرارداد نیست؛ مبلغ خام در جزئیات فاکتور و تلاش دیده می‌شود.' } },
  { id: 'amount-conflict', group: 'provenance', label: 'تعارض یا ناهم‌خوانی مبالغ فاکتور', icon: 'swap',
    meaning: 'طبقه تطبیق مبالغ (MISMATCH / CONFLICT / INCOMPLETE) فقط به تفکیک هر فاکتور و واحد محاسبه می‌شود.',
    count: { kind: 'none', why: 'طبقه تطبیق فقط در GET /api/v1/invoices/{id}/totals برای یک فاکتور خوانده می‌شود؛ شمارنده یا فیلتر سراسری وجود ندارد.' },
    list: { read: 'none', why: 'برای دیدن طبقه تطبیق، جزئیات هر فاکتور (بخش «مبالغ») را باز کنید؛ هیچ ناهم‌خوانی‌ای اینجا استنتاج نمی‌شود.' } },
]

export function exceptionById(id: string | null): ExceptionDef | undefined {
  return EXCEPTIONS.find(x => x.id === id)
}

/** The destination ids whose capability a queue's list read needs (presentation gating only). */
export const LIST_DESTINATION: Record<Exclude<ListRead['read'], 'none'>, string> = {
  attempts: 'billing.attempts', invoices: 'billing.invoices', review: 'finance.review',
}
