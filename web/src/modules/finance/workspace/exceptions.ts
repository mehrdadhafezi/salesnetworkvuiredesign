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

/**
 * What one counted or listed row is. Counts and lists keep their own unit:
 * a receipt count is never shown as a number of payments, a history-row
 * count never as a number of attempts.
 */
export type Unit = 'invoice' | 'attempt' | 'submission' | 'receipt' | 'receipt-ref' | 'history-row' | 'amount'

export const UNIT_LABEL: Record<Unit, string> = {
  invoice: 'فاکتور', attempt: 'تلاش پرداخت قدیمی', submission: 'پرداخت ثبت‌شده', receipt: 'فیش سامانه جدید',
  'receipt-ref': 'ارجاع رسید قدیمی', 'history-row': 'ردیف سابقه', amount: 'شاهد مبلغ',
}

/** The unit of each list read (one row = one of these). */
export const LIST_UNIT: Record<Exclude<ListRead['read'], 'none'>, Unit> = { attempts: 'attempt', invoices: 'invoice', review: 'submission' }

export interface ExceptionDef {
  id: string
  group: 'relationships' | 'states' | 'evidence' | 'provenance'
  label: string
  icon: DsIconName
  meaning: string
  count: CountSource
  /** Unit of the counter (meaningful when count.kind is not 'none'). */
  countUnit: Unit
  list: ListRead
  /**
   * Required when the counter and the list count different things: states
   * that the two numbers are different sets and no one-to-one is implied.
   */
  cardinality?: string
}

/** The unit a queue's list returns, or undefined when it has no list contract. */
export function listUnit(x: ExceptionDef): Unit | undefined {
  return x.list.read === 'none' ? undefined : LIST_UNIT[x.list.read]
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
    count: C.attemptsWithoutInvoice, countUnit: 'attempt', list: { read: 'attempts', params: { invoice_link_state: 'UNKNOWN', ...present } } },
  { id: 'invoice-customer-conflict', group: 'relationships', label: 'پیوند مشتری فاکتور: تعارض', icon: 'alert',
    meaning: 'فاکتورهایی که پیوند مشتری آن‌ها CONFLICT است؛ هیچ مشتری‌ای انتخاب یا استنتاج نمی‌شود.',
    count: C.customerLink('CONFLICT'), countUnit: 'invoice', list: { read: 'invoices', params: { customer_link_state: 'CONFLICT', ...present } } },
  { id: 'invoice-customer-unknown', group: 'relationships', label: 'پیوند مشتری فاکتور: نامعلوم', icon: 'question',
    meaning: 'فاکتورهایی بدون پیوند مشتری قابل اثبات (UNKNOWN).',
    count: C.customerLink('UNKNOWN'), countUnit: 'invoice', list: { read: 'invoices', params: { customer_link_state: 'UNKNOWN', ...present } } },
  { id: 'invoice-customer-nr', group: 'relationships', label: 'پیوند مشتری فاکتور: غیرقابل بازیابی', icon: 'ban',
    meaning: 'پیوند مشتری از منبع قابل بازیابی نیست (NOT_RECOVERABLE).',
    count: C.customerLink('NOT_RECOVERABLE'), countUnit: 'invoice', list: { read: 'invoices', params: { customer_link_state: 'NOT_RECOVERABLE', ...present } } },
  { id: 'invoice-case-conflict', group: 'relationships', label: 'پیوند پرونده فاکتور: تعارض', icon: 'alert',
    meaning: 'فاکتورهایی که پیوند پرونده آن‌ها CONFLICT است.',
    count: C.caseLink('CONFLICT'), countUnit: 'invoice', list: { read: 'invoices', params: { case_link_state: 'CONFLICT', ...present } } },
  { id: 'invoice-case-unknown', group: 'relationships', label: 'پیوند پرونده فاکتور: نامعلوم', icon: 'question',
    meaning: 'فاکتورهایی بدون پیوند پرونده قابل اثبات (UNKNOWN).',
    count: C.caseLink('UNKNOWN'), countUnit: 'invoice', list: { read: 'invoices', params: { case_link_state: 'UNKNOWN', ...present } } },
  { id: 'invoice-case-nr', group: 'relationships', label: 'پیوند پرونده فاکتور: غیرقابل بازیابی', icon: 'ban',
    meaning: 'پیوند پرونده از منبع قابل بازیابی نیست (NOT_RECOVERABLE).',
    count: C.caseLink('NOT_RECOVERABLE'), countUnit: 'invoice', list: { read: 'invoices', params: { case_link_state: 'NOT_RECOVERABLE', ...present } } },

  { id: 'attempt-unknown-state', group: 'states', label: 'وضعیت خام ناشناخته تلاش', icon: 'question',
    meaning: 'مقدار خام وضعیت تلاش در واژگان نگاشت نیست (state_mapping = UNKNOWN)؛ مقدار خام حفظ شده و حدس زده نمی‌شود.',
    count: C.attemptMapping('UNKNOWN'), countUnit: 'attempt', list: { read: 'attempts', params: { state_mapping: 'UNKNOWN', ...present } } },
  { id: 'attempt-empty-state', group: 'states', label: 'وضعیت خالی تلاش', icon: 'question',
    meaning: 'وضعیت خام تلاش خالی است (state_mapping = EMPTY).',
    count: C.attemptMapping('EMPTY'), countUnit: 'attempt', list: { read: 'attempts', params: { state_mapping: 'EMPTY', ...present } } },
  { id: 'attempt-reconciliation', group: 'states', label: 'تلاش نیازمند تطبیق', icon: 'swap',
    meaning: 'وضعیت نگاشت‌شده RECONCILIATION_REQUIRED (از جمله reconcile_error قدیمی). اینجا هیچ تطبیقی انجام نمی‌شود.',
    count: C.attemptState('RECONCILIATION_REQUIRED'), countUnit: 'attempt', list: { read: 'attempts', params: { state: 'RECONCILIATION_REQUIRED', ...present } } },
  { id: 'attempt-rejected', group: 'states', label: 'تلاش ردشده', icon: 'ban',
    meaning: 'وضعیت نگاشت‌شده REJECTED در سامانه قدیمی؛ رد قدیمی، تصمیم بررسی مالی سامانه جدید نیست.',
    count: C.attemptState('REJECTED'), countUnit: 'attempt', list: { read: 'attempts', params: { state: 'REJECTED', ...present } } },
  { id: 'attempt-cancelled', group: 'states', label: 'تلاش لغوشده', icon: 'ban',
    meaning: 'وضعیت نگاشت‌شده CANCELLED (از جمله لغو در درگاه)؛ لغو به معنای استرداد وجه نیست.',
    count: C.attemptState('CANCELLED'), countUnit: 'attempt', list: { read: 'attempts', params: { state: 'CANCELLED', ...present } } },
  { id: 'attempt-regression', group: 'states', label: 'بازگشت وضعیت در سابقه تلاش', icon: 'repeat',
    meaning: 'تلاش‌هایی که در سابقه افزایشی آن‌ها دست‌کم یک ردیف «بازگشت از پذیرفته‌شده» ثبت شده است. شمارنده تعداد ردیف سابقه است، نه تلاش.',
    count: C.regressions, countUnit: 'history-row', list: { read: 'attempts', params: { has_state_regression: 'true' } },
    cardinality: 'شمارنده تعداد ردیف‌های بازگشت در سابقه است و فهرست تلاش‌ها را نشان می‌دهد؛ یک تلاش ممکن است چند ردیف داشته باشد، پس دو عدد برابر نیستند.' },
  { id: 'attempt-unknown-provenance', group: 'states', label: 'منشأ تأیید نامعلوم', icon: 'question',
    meaning: 'منشأ تأیید تلاش قابل تعیین نیست (approval_provenance = UNKNOWN).',
    count: C.attemptProvenance('UNKNOWN'), countUnit: 'attempt', list: { read: 'attempts', params: { approval_provenance: 'UNKNOWN', ...present } } },
  { id: 'attempt-unknown-channel', group: 'states', label: 'کانال پرداخت نامعلوم', icon: 'question',
    meaning: 'کانال تلاش (درگاه / کارت به کارت) قابل تعیین نیست.',
    count: C.attemptChannel('UNKNOWN'), countUnit: 'attempt', list: { read: 'attempts', params: { channel: 'UNKNOWN', ...present } } },

  { id: 'review-evidence-pending', group: 'evidence', label: 'پرداخت ثبت‌شده: فیش در انتظار نهایی‌سازی', icon: 'inbox',
    meaning: 'پرداخت‌های ثبت‌شده در سامانه جدید که همه فیش‌هایشان هنوز آماده نیست (evidence_state = PENDING_FINALISATION)؛ تصمیم مالی طبق D-43 مسدود است.',
    count: C.reviewQueue, countUnit: 'submission', list: { read: 'review', params: { evidence_state: 'PENDING_FINALISATION' } } },
  { id: 'review-evidence-nr', group: 'evidence', label: 'پرداخت ثبت‌شده: فیش غیرقابل بازیابی', icon: 'alert',
    meaning: 'پرداخت‌های ثبت‌شده‌ای که شواهدشان غیرقابل بازیابی است (evidence_state = NOT_RECOVERABLE)؛ مالک دامنه باید بررسی کند.',
    count: C.reviewQueue, countUnit: 'submission', list: { read: 'review', params: { evidence_state: 'NOT_RECOVERABLE' } } },
  { id: 'legacy-receipt-missing', group: 'evidence', label: 'ارجاع رسید قدیمی: مفقود', icon: 'inbox',
    meaning: 'ارجاع‌های رسید قدیمی با وضعیت MISSING. فهرست سراسری ارجاع رسید در قرارداد نیست؛ رسیدهای هر فاکتور در جزئیات همان فاکتور دیده می‌شوند.',
    count: C.receiptsMissing, countUnit: 'receipt-ref', list: { read: 'none', why: 'ارجاع‌های رسید فقط به تفکیک فاکتور (GET /api/v1/invoices/{id}/receipts) خوانده می‌شوند؛ فهرست سراسری با فیلتر وضعیت در قرارداد وجود ندارد.' } },
  { id: 'legacy-receipt-unknown', group: 'evidence', label: 'ارجاع رسید قدیمی: نامعلوم', icon: 'question',
    meaning: 'ارجاع‌های رسید قدیمی با وضعیت UNKNOWN.',
    count: C.receiptsUnknown, countUnit: 'receipt-ref', list: { read: 'none', why: 'ارجاع‌های رسید فقط به تفکیک فاکتور خوانده می‌شوند؛ فهرست سراسری با فیلتر وضعیت در قرارداد وجود ندارد.' } },
  { id: 'target-receipt-nr', group: 'evidence', label: 'فیش سامانه جدید: غیرقابل بازیابی (شمار فیش)', icon: 'alert',
    meaning: 'تعداد فیش‌هایی که پس از خطای ذخیره‌سازی قابل بازیابی نیستند. واحد این شمارنده «فیش» است، نه پرداخت؛ یک پرداخت ثبت‌شده ممکن است چند فیش داشته باشد.',
    count: C.targetReceipt('NOT_RECOVERABLE'), countUnit: 'receipt',
    list: { read: 'none', why: 'فهرست فیش‌ها با فیلتر وضعیت در قرارداد نیست (GET /api/v1/payments/evidence فقط شمارش و موارد تطبیق را برمی‌گرداند). پرداخت‌های ثبت‌شده با شواهد غیرقابل بازیابی مجموعه دیگری‌اند و در صف «پرداخت ثبت‌شده: فیش غیرقابل بازیابی» فهرست می‌شوند؛ تناظر یک‌به‌یک ادعا نمی‌شود.' } },

  { id: 'invoice-unknown-status', group: 'provenance', label: 'فاکتور با وضعیت خام ناشناخته', icon: 'question',
    meaning: 'فاکتورهایی که دست‌کم یک فیلد وضعیت خام نگاشت‌نشده دارند (has_unknown_status = true).',
    count: C.unknownInvoiceStatus, countUnit: 'invoice', list: { read: 'invoices', params: { has_unknown_status: 'true', ...present } } },
  { id: 'invoice-absent', group: 'provenance', label: 'فاکتور غایب در آخرین واردسازی کامل', icon: 'alert',
    meaning: 'فاکتورهایی که در آخرین دسته کامل دیده نشدند (ABSENT_IN_LATEST)؛ حذف، لغو یا تسویه نشده‌اند.',
    count: C.absent, countUnit: 'invoice', list: { read: 'invoices', params: { presence: 'ABSENT_IN_LATEST' } } },
  { id: 'attempt-absent', group: 'provenance', label: 'تلاش غایب در آخرین واردسازی کامل', icon: 'alert',
    meaning: 'تلاش‌هایی که در آخرین دسته کامل دیده نشدند؛ حذف یا برگشت نشده‌اند.',
    count: C.absent, countUnit: 'attempt', list: { read: 'attempts', params: { presence: 'ABSENT_IN_LATEST' } } },
  { id: 'invoice-no-items', group: 'provenance', label: 'فاکتور بدون قلم', icon: 'file',
    meaning: 'فاکتورهای حاضری که هیچ قلمی برایشان وارد نشده است.',
    count: C.invoicesWithoutItems, countUnit: 'invoice', list: { read: 'none', why: 'فیلتر فهرست فاکتور برای «بدون قلم» در قرارداد نیست؛ فقط شمارنده تجمیعی موجود است.' } },
  { id: 'invoice-no-stages', group: 'provenance', label: 'فاکتور بدون مرحله پرداخت', icon: 'file',
    meaning: 'فاکتورهای حاضری که هیچ مرحله پرداختی برایشان وارد نشده است.',
    count: C.invoicesWithoutStages, countUnit: 'invoice', list: { read: 'none', why: 'فیلتر فهرست فاکتور برای «بدون مرحله» در قرارداد نیست؛ فقط شمارنده تجمیعی موجود است.' } },
  { id: 'amount-unrepresentable', group: 'provenance', label: 'مبلغ تلاش غیرقابل نمایش دقیق', icon: 'alert',
    meaning: 'مبالغ خام تلاش‌های حاضر که بدون گرد کردن قابل نمایش نیستند. مبالغ فاکتور جداگانه شمرده می‌شوند (billing.unrepresentable_amounts) و با این عدد جمع نمی‌شوند.',
    count: C.attemptsUnrepresentable, countUnit: 'amount', list: { read: 'none', why: 'فیلتر فهرست برای مبلغ غیرقابل نمایش در قرارداد نیست؛ مبلغ خام در جزئیات فاکتور و تلاش دیده می‌شود.' } },
  { id: 'amount-conflict', group: 'provenance', label: 'تعارض یا ناهم‌خوانی مبالغ فاکتور', icon: 'swap',
    meaning: 'طبقه تطبیق مبالغ (MISMATCH / CONFLICT / INCOMPLETE) فقط به تفکیک هر فاکتور و واحد محاسبه می‌شود.',
    count: { kind: 'none', why: 'طبقه تطبیق فقط در GET /api/v1/invoices/{id}/totals برای یک فاکتور خوانده می‌شود؛ شمارنده یا فیلتر سراسری وجود ندارد.' },
    countUnit: 'invoice', list: { read: 'none', why: 'برای دیدن طبقه تطبیق، جزئیات هر فاکتور (بخش «مبالغ») را باز کنید؛ هیچ ناهم‌خوانی‌ای اینجا استنتاج نمی‌شود.' } },
]

export function exceptionById(id: string | null): ExceptionDef | undefined {
  return EXCEPTIONS.find(x => x.id === id)
}

/** The destination ids whose capability a queue's list read needs (presentation gating only). */
export const LIST_DESTINATION: Record<Exclude<ListRead['read'], 'none'>, string> = {
  attempts: 'billing.attempts', invoices: 'billing.invoices', review: 'finance.review',
}
