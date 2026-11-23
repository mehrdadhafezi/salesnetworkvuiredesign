import type { Parity } from '../../shared/api'
import type { DsIconName } from '../../shared/ui/ds'

/**
 * RM-P05 MIS workspace vocabulary. Every label names an evidence state that
 * an existing contract already returns; none is an action, an eligibility
 * or a KPI. Labels for parity, findings and resolutions are reused from the
 * Slice 6 sales module (one vocabulary for one contract).
 */

export const derivationLabel: Record<string, string> = {
  DERIVED: 'نگهدار از رویدادها استنتاج شد', RELEASED: 'آخرین رویداد آزادسازی به منبع است', UNKNOWN: 'رویدادها نگهدار را ثابت نمی‌کنند',
  MULTIPLE: 'چند نگهدار هم‌زمان', NONE: 'رویداد نگهداری وجود ندارد',
}

export const presenceLabel: Record<string, string> = {
  PRESENT: 'در آخرین واردسازی کامل حاضر',
  ABSENT_IN_LATEST: 'در آخرین واردسازی کامل نیست (حذف یا آزاد نشده)',
}

export const contextLabel: Record<string, string> = {
  DISTRIBUTION: 'پخش', MIS: 'MIS', PAYMENT_WORKFLOW: 'گردش پرداخت', REFERRAL: 'ارجاع', CONVERSION: 'تبدیل', SELLER_UPDATE: 'به‌روزرسانی فروشنده', UNKNOWN: 'نامعلوم',
}

export const blockedSales = 'خواندن پرونده و نگهداری پشت پرچم ویژگی است (sales.read_shadow · FEATURE_DISABLED).'

export const MIS_NOTE = 'فضای کار MIS فقط خواندنی است: هیچ تخصیص، انتقال، برگشت، آزادسازی، بازتخصیص یا استفاده مجدد از اینجا انجام نمی‌شود. تلفن هرگز شناسه پرونده نیست.'

/**
 * Where an exception's count comes from. Every counter names the endpoint
 * and field it reads and the scope that field counts; a missing aggregate is
 * shown as unavailable, never as zero.
 */
export type CountSource =
  | { kind: 'parity'; key: Parity; scope: string }
  | { kind: 'finding'; key: string; scope: string }
  | { kind: 'coverage'; field: 'cases_with_unknown_state_mapping' | 'sourceless_cases'; scope: string }
  | { kind: 'coverage-absent'; scope: string }
  | { kind: 'coverage-map'; map: 'customer_link_states' | 'aliases_by_resolution'; key: string; scope: string }
  | { kind: 'none'; why: string }

/** The list the exception opens: an existing server filter. 'none' means no list contract exists. */
export type ExceptionList =
  | { read: 'custody'; params: Record<string, string> }
  | { read: 'cases'; params: Record<string, string> }
  | { read: 'none'; why: string }

export interface ExceptionDef {
  id: string
  group: 'custody' | 'identity' | 'case'
  label: string
  icon: DsIconName
  /** What the stored evidence says (definition, not a judgement). */
  meaning: string
  list: ExceptionList
  count: CountSource
}

const parity = (key: Parity, scope = 'پروجکشن‌های حاضر (PRESENT)'): CountSource => ({ kind: 'parity', key, scope })
const finding = (key: string): CountSource => ({ kind: 'finding', key, scope: 'تعداد یافته‌های حاضر، نه تعداد پرونده' })

/**
 * The exception queues of the workspace. Each one is an existing stored
 * finding or state, listed with an existing server filter. Nothing here
 * resolves, repairs, merges or re-derives anything.
 */
export const EXCEPTIONS: ExceptionDef[] = [
  { id: 'holder-conflict', group: 'custody', label: 'تعارض نگهدار', icon: 'alert',
    meaning: 'نگهدار ثبت‌شده در منبع با نگهدار استنتاجی از رویدادها متفاوت است.',
    list: { read: 'custody', params: { parity: 'CONFLICT', presence: 'PRESENT' } }, count: parity('CONFLICT') },
  { id: 'holder-multiple', group: 'custody', label: 'چند نگهدار هم‌زمان', icon: 'alert',
    meaning: 'رویدادهای هم‌ترتیب نگهدارهای متناقض نشان می‌دهند.',
    list: { read: 'custody', params: { parity: 'MULTIPLE_CURRENT', presence: 'PRESENT' } }, count: parity('MULTIPLE_CURRENT') },
  { id: 'holder-unknown', group: 'custody', label: 'نگهدار نامعلوم', icon: 'question',
    meaning: 'رویدادها نگهدار فعلی را ثابت نمی‌کنند (سرنخ‌های قدیمی گزارش تخصیص ندارند).',
    list: { read: 'custody', params: { parity: 'TARGET_UNKNOWN', presence: 'PRESENT' } }, count: parity('TARGET_UNKNOWN') },
  { id: 'holder-not-recoverable', group: 'custody', label: 'نگهداری قابل بازیابی نیست', icon: 'ban',
    meaning: 'هیچ شاهدی در منبع یا رویدادها برای نگهدار وجود ندارد.',
    list: { read: 'custody', params: { parity: 'NOT_RECOVERABLE', presence: 'PRESENT' } }, count: parity('NOT_RECOVERABLE') },
  { id: 'history-gap', group: 'custody', label: 'سابقه ناقص (نگهدار یکسان)', icon: 'history',
    meaning: 'نگهدار منبع و رویدادها یکی است اما زنجیره رویدادها شکاف دارد.',
    list: { read: 'custody', params: { parity: 'HISTORY_GAP', presence: 'PRESENT' } }, count: parity('HISTORY_GAP') },
  { id: 'missing-history', group: 'custody', label: 'سابقه رویداد وجود ندارد', icon: 'history',
    meaning: 'برای پرونده هیچ رویداد نگهداری ثبت نشده است (یافته MISSING_EVENT_HISTORY).',
    list: { read: 'custody', params: { finding_kind: 'MISSING_EVENT_HISTORY' } }, count: finding('MISSING_EVENT_HISTORY') },
  { id: 'gap-finding', group: 'custody', label: 'شکاف در زنجیره انتقال', icon: 'history',
    meaning: 'یافته HISTORY_GAP روی رویدادهای پرونده ثبت شده است.',
    list: { read: 'custody', params: { finding_kind: 'HISTORY_GAP' } }, count: finding('HISTORY_GAP') },
  { id: 'source-mismatch', group: 'custody', label: 'ناهمخوانی نگهدار منبع', icon: 'swap',
    meaning: 'یافته SOURCE_CURRENT_MISMATCH: نگهدار ثبت‌شده در منبع با آخرین رویداد نمی‌خواند.',
    list: { read: 'custody', params: { finding_kind: 'SOURCE_CURRENT_MISMATCH' } }, count: finding('SOURCE_CURRENT_MISMATCH') },
  { id: 'ordering', group: 'custody', label: 'تناقض ترتیب زمانی', icon: 'clock',
    meaning: 'یافته ORDERING_CONTRADICTION: ترتیب رویدادها با زمان‌های خام آن‌ها نمی‌خواند.',
    list: { read: 'custody', params: { finding_kind: 'ORDERING_CONTRADICTION' } }, count: finding('ORDERING_CONTRADICTION') },
  { id: 'missing-prior', group: 'custody', label: 'نگهدار قبلی ثبت نشده', icon: 'user',
    meaning: 'یافته MISSING_PRIOR_OWNER روی یک رویداد انتقال.',
    list: { read: 'custody', params: { finding_kind: 'MISSING_PRIOR_OWNER' } }, count: finding('MISSING_PRIOR_OWNER') },
  { id: 'missing-target', group: 'custody', label: 'نگهدار جدید ثبت نشده', icon: 'user',
    meaning: 'یافته MISSING_TARGET_OWNER روی یک رویداد انتقال.',
    list: { read: 'custody', params: { finding_kind: 'MISSING_TARGET_OWNER' } }, count: finding('MISSING_TARGET_OWNER') },
  { id: 'unknown-action', group: 'custody', label: 'کنش نگاشت‌نشده منبع', icon: 'question',
    meaning: 'یافته UNKNOWN_ACTION: کنش خام منبع در نگاشت تأییدشده نیست و خام حفظ شده است.',
    list: { read: 'custody', params: { finding_kind: 'UNKNOWN_ACTION' } }, count: finding('UNKNOWN_ACTION') },
  { id: 'unresolved-holder', group: 'identity', label: 'هویت نگهدار حل‌نشده', icon: 'user',
    meaning: 'کلید کاربر قدیمی نگهدار (استنتاجی یا ثبت‌شده) به حساب کاربری هدف وصل نشده است؛ کلید خام حفظ می‌شود و از تلفن یا نام حدس زده نمی‌شود.',
    list: { read: 'custody', params: { holder_link: 'UNRESOLVED' } },
    count: { kind: 'none', why: 'هیچ قرارداد تجمیعی برای این حالت وجود ندارد؛ فقط فهرست صفحه‌بندی‌شده در دسترس است.' } },
  { id: 'customer-conflict', group: 'identity', label: 'پیوند مشتری متعارض', icon: 'alert',
    meaning: 'شواهد پیوند پرونده به مشتری با هم تعارض دارند؛ هیچ مشتری انتخاب نشده است.',
    list: { read: 'cases', params: { customer_link_state: 'CONFLICT', presence: 'PRESENT' } },
    count: { kind: 'coverage-map', map: 'customer_link_states', key: 'CONFLICT', scope: 'پرونده‌های حاضر (PRESENT)' } },
  { id: 'customer-unknown', group: 'identity', label: 'پیوند مشتری نامعلوم', icon: 'question',
    meaning: 'هیچ مرجع صریحی به مشتری وجود ندارد؛ پیوند استنتاج نمی‌شود.',
    list: { read: 'cases', params: { customer_link_state: 'UNKNOWN', presence: 'PRESENT' } },
    count: { kind: 'coverage-map', map: 'customer_link_states', key: 'UNKNOWN', scope: 'پرونده‌های حاضر (PRESENT)' } },
  { id: 'customer-not-recoverable', group: 'identity', label: 'پیوند مشتری قابل بازیابی نیست', icon: 'ban',
    meaning: 'شاهد پیوند مشتری قابل بازیابی نیست.',
    list: { read: 'cases', params: { customer_link_state: 'NOT_RECOVERABLE', presence: 'PRESENT' } },
    count: { kind: 'coverage-map', map: 'customer_link_states', key: 'NOT_RECOVERABLE', scope: 'پرونده‌های حاضر (PRESENT)' } },
  { id: 'absent-latest', group: 'case', label: 'غایب در آخرین واردسازی', icon: 'layers',
    meaning: 'پرونده در آخرین واردسازی کامل منبع نبود. حذف، آزاد یا قابل استفاده مجدد نشده است؛ سابقه، انتساب و نگهداری آن حفظ می‌شود.',
    list: { read: 'cases', params: { presence: 'ABSENT_IN_LATEST' } }, count: { kind: 'coverage-absent', scope: 'همه پرونده‌ها منهای پرونده‌های حاضر' } },
  { id: 'unknown-state', group: 'case', label: 'وضعیت نگاشت‌نشده', icon: 'question',
    meaning: 'وضعیت خام منبع در نگاشت تأییدشده نیست؛ مقدار خام حفظ شده و پیش‌فرض گذاشته نشده است.',
    list: { read: 'cases', params: { state_mapping: 'UNKNOWN' } },
    count: { kind: 'coverage', field: 'cases_with_unknown_state_mapping', scope: 'همه پرونده‌ها (حاضر و غایب)' } },
  { id: 'sourceless', group: 'case', label: 'پرونده بدون ردیف منبع', icon: 'file',
    meaning: 'پرونده‌ای که ردیف منبع (مرجع ورود) ندارد.',
    list: { read: 'none', why: 'فهرست پرونده‌ها فقط پرونده‌های دارای منبع را برمی‌گرداند؛ قرارداد فهرستی برای این حالت وجود ندارد.' },
    count: { kind: 'coverage', field: 'sourceless_cases', scope: 'همه پرونده‌ها' } },
  { id: 'alias-conflict', group: 'case', label: 'پیوند متعارض بین نمایش‌ها', icon: 'swap',
    meaning: 'پیوند صریح بین دو نمایش منبع (مثلاً تبدیل سرنخ) متعارض است؛ ادغام انجام نمی‌شود (OPD-01).',
    list: { read: 'none', why: 'قرارداد فهرستی برای فیلتر پیوندها وجود ندارد؛ پیوندها در صفحه پرونده دیده می‌شوند.' },
    count: { kind: 'coverage-map', map: 'aliases_by_resolution', key: 'CONFLICT', scope: 'پیوندهای حاضر' } },
]

export const EXCEPTION_GROUPS: { id: ExceptionDef['group']; label: string }[] = [
  { id: 'custody', label: 'نگهداری و سابقه انتقال' },
  { id: 'identity', label: 'هویت حل‌نشده' },
  { id: 'case', label: 'منبع و حضور پرونده' },
]

export function exceptionById(id: string | null): ExceptionDef | undefined {
  return EXCEPTIONS.find(x => x.id === id)
}
