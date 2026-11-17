/*
 * RM-E03 SYNTHETIC fixtures for the DEV/TEST-only Seller V1 preview.
 *
 * - Structure and wording follow prototype/data.js (frozen Seller Design V1); the values are new
 *   and fictional. Nothing here is imported from window.MOCK, the CRM API, WordPress or any database.
 * - Names are visibly synthetic («مشتری نمونه …»); phones use the unallocated 0900-000 range;
 *   invoice codes carry a TST- prefix. None of these identify a real person, case or invoice.
 * - Amounts, balances and commission lines are invented for layout only. They are not financial
 *   facts, entitlements or rules.
 * - This module is reachable only from the DEV-guarded /dev/seller route; the production bundle
 *   check in web-ci greps for SELLER_PREVIEW_MARKER and fails if it ever appears.
 */
import { toFaDigits } from '../../../shared/ui/ds/format'
import type {
  BehaviorItem, Conversion, ExtraRequest, Invoice, Lead, LeadProbability, LeadSource, LeadStatus, RepeatItem, WalletFixture,
} from '../types'

export const SELLER_PREVIEW_MARKER = 'RM-E03-SYNTHETIC-SELLER-FIXTURE'

export const PREVIEW_USER = { name: 'فروشنده نمونه', id: 'SYN-0001', role: 'فروشنده (ساختگی)', initials: 'ف‌ن' } as const

const phone = (n: number) => `0900000${String(n).padStart(4, '0')}`

const baseHistory = (assigned: string) => [
  { text: 'تخصیص به شما', meta: `سیستم توزیع (نمونه) · ${assigned}` },
  { text: 'ورود شماره از منبع', meta: 'دسته ورودی نمونه #SYN-88' },
]

/* The first ten leads mirror the prototype rows one-for-one (status, source, next step) with fictional values. */
const handLeads: Lead[] = [
  { id: 900101, name: 'مشتری نمونه الف', phone: phone(101), city: 'تهران', province: 'تهران', source: 'MIS', assigned: 'امروز ۲۲:۵۴', assignedAgo: 'امروز', assignedDays: 0, status: 'none', attempts: 0, prob: '۱۰۰٪', note: '', next: null, lastActivity: '—', invoiceRefs: ['TST-1007'], history: baseHistory('امروز ۲۲:۵۴') },
  { id: 900102, name: 'مشتری نمونه ب', phone: phone(102), city: 'کرج', province: 'البرز', source: 'MIS', assigned: '۱۴۰۵/۰۵/۱۹', assignedAgo: '۲ ماه پیش', assignedDays: 60, status: 'none', attempts: 0, prob: '', note: '', next: null, lastActivity: '—', history: baseHistory('۱۴۰۵/۰۵/۱۹') },
  { id: 900103, name: 'مشتری نمونه پ', phone: phone(103), city: 'قم', province: 'قم', source: 'MIS', assigned: '۱۴۰۵/۰۵/۱۹', assignedAgo: '۲ ماه پیش', assignedDays: 60, status: 'none', attempts: 0, prob: '', note: '', next: null, lastActivity: '—', history: baseHistory('۱۴۰۵/۰۵/۱۹') },
  { id: 900104, name: 'مشتری نمونه ت', phone: phone(104), city: 'اصفهان', province: 'اصفهان', source: 'درخواستی', assigned: '۱۴۰۵/۰۷/۰۵', assignedAgo: '۳ روز پیش', assignedDays: 3, status: 'no_answer', attempts: 2, prob: 'متوسط', note: 'دو بار زنگ خورد، جواب نداد (نمونه)', next: { kind: 'attempts', text: 'بی‌پاسخ ۲ از ۳', warn: true }, lastActivity: '۲ روز پیش', invoiceRefs: ['TST-1003'], history: [{ text: 'تماس بی‌پاسخ (۲ از ۳)', meta: 'فروشنده نمونه · ۲ روز پیش' }, ...baseHistory('۱۴۰۵/۰۷/۰۵')] },
  { id: 900105, name: 'مشتری نمونه ث', phone: phone(105), city: 'شیراز', province: 'فارس', source: 'MIS', assigned: '۱۴۰۵/۰۷/۰۲', assignedAgo: '۶ روز پیش', assignedDays: 6, status: 'callback', attempts: 0, prob: 'بالا', note: 'بعد از ساعت ۱۰ صبح تماس بگیرید (نمونه)', next: { kind: 'callback', text: 'فردا ۱۰:۳۰', warn: false }, lastActivity: 'دیروز', invoiceRefs: ['TST-1004'], history: [{ text: 'تماس مجدد برای فردا ۱۰:۳۰', meta: 'فروشنده نمونه · دیروز' }, ...baseHistory('۱۴۰۵/۰۷/۰۲')] },
  { id: 900106, name: 'مشتری نمونه ج', phone: phone(106), city: 'مشهد', province: 'خراسان رضوی', source: 'ارتباط مجدد', assigned: '۱۴۰۵/۰۶/۲۸', assignedAgo: '۱۰ روز پیش', assignedDays: 10, status: 'callback', attempts: 0, prob: 'متوسط', note: '', next: { kind: 'callback', text: 'امروز ۱۶:۰۰', warn: true, due: true }, lastActivity: 'دیروز', invoiceRefs: ['TST-1005'], history: [{ text: 'تماس مجدد برای امروز ۱۶:۰۰', meta: 'فروشنده نمونه · دیروز' }, ...baseHistory('۱۴۰۵/۰۶/۲۸')] },
  { id: 900107, name: 'مشتری نمونه چ', phone: phone(107), city: 'تبریز', province: 'آذربایجان شرقی', source: 'MIS', assigned: '۱۴۰۵/۰۶/۲۰', assignedAgo: '۱۸ روز پیش', assignedDays: 18, status: 'no_answer', attempts: 3, prob: 'ضعیف', note: '', next: { kind: 'archive', text: 'بایگانی تا ۲ روز دیگر', warn: true, due: true }, lastActivity: '۲ روز پیش', history: [{ text: 'تماس بی‌پاسخ (۳ از ۳)', meta: 'فروشنده نمونه · ۲ روز پیش' }, ...baseHistory('۱۴۰۵/۰۶/۲۰')] },
  { id: 900108, name: 'مشتری نمونه ح', phone: phone(108), city: 'رشت', province: 'گیلان', source: 'MIS', assigned: '۱۴۰۵/۰۶/۱۵', assignedAgo: '۲۳ روز پیش', assignedDays: 23, status: 'invoiced', attempts: 0, prob: '۱۰۰٪', note: 'اشتراک نمونه ۱۰ ستاره', next: { kind: 'invoice', text: 'منتظر پرداخت', warn: false }, lastActivity: '۲۳ روز پیش', invoiceRefs: ['TST-1006'], history: [{ text: 'صدور پیش‌فاکتور TST-1006', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۱۵' }, ...baseHistory('۱۴۰۵/۰۶/۱۵')] },
  { id: 900109, name: 'مشتری نمونه خ', phone: phone(109), city: 'اهواز', province: 'خوزستان', source: 'MIS', assigned: '۱۴۰۵/۰۶/۱۰', assignedAgo: '۲۸ روز پیش', assignedDays: 28, status: 'not_purchased', attempts: 0, prob: 'ضعیف', note: 'قیمت را بالا دانست (نمونه)', next: null, lastActivity: '۲۸ روز پیش', history: [{ text: 'ثبت عدم خرید', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۱۰' }, ...baseHistory('۱۴۰۵/۰۶/۱۰')] },
  { id: 900110, name: 'مشتری نمونه د', phone: phone(110), city: 'تهران', province: 'تهران', source: 'MIS', assigned: '۱۴۰۵/۰۶/۰۸', assignedAgo: '۱ ماه پیش', assignedDays: 31, status: 'duplicate', attempts: 0, prob: '', note: 'قبلاً توسط همکار ثبت شده (نمونه)', next: null, lastActivity: '۱ ماه پیش', history: [{ text: 'علامت تکراری', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۰۸' }, ...baseHistory('۱۴۰۵/۰۶/۰۸')] },
]

/* Fifteen generated rows so pagination, search and filters have something to work on. */
const LETTERS = ['ذ', 'ر', 'ز', 'ژ', 'س', 'ش', 'ص', 'ض', 'ط', 'ظ', 'ع', 'غ', 'ف', 'ق', 'ک']
const CITIES: [string, string][] = [['تهران', 'تهران'], ['کرج', 'البرز'], ['یزد', 'یزد'], ['کرمان', 'کرمان'], ['ساری', 'مازندران']]
const STATUSES: LeadStatus[] = ['none', 'no_answer', 'callback', 'none', 'not_purchased', 'none', 'no_answer', 'invoiced', 'none', 'callback', 'duplicate', 'none', 'no_answer', 'none', 'callback']
const SOURCES: LeadSource[] = ['MIS', 'MIS', 'درخواستی', 'MIS', 'ارتباط مجدد']
const PROBS: LeadProbability[] = ['', 'ضعیف', 'متوسط', 'بالا', '']

const genLeads: Lead[] = LETTERS.map((letter, i) => {
  const status = STATUSES[i] ?? 'none'
  const [city, province] = CITIES[i % CITIES.length] ?? ['تهران', 'تهران']
  const days = 4 + i * 3
  const assigned = `${toFaDigits(days)} روز پیش`
  const next = status === 'no_answer' ? { kind: 'attempts' as const, text: 'بی‌پاسخ ۱ از ۳', warn: false }
    : status === 'callback' ? { kind: 'callback' as const, text: 'پس‌فردا ۱۱:۰۰', warn: false }
      : status === 'invoiced' ? { kind: 'invoice' as const, text: 'منتظر پرداخت', warn: false }
        : null
  return {
    id: 900111 + i, name: `مشتری نمونه ${letter}`, phone: phone(111 + i), city, province,
    source: SOURCES[i % SOURCES.length] ?? 'MIS', assigned, assignedAgo: assigned, assignedDays: days,
    status, attempts: status === 'no_answer' ? 1 : 0, prob: PROBS[i % PROBS.length] ?? '', note: '', next,
    lastActivity: status === 'none' ? '—' : assigned, history: baseHistory(assigned),
  }
})

export const LEADS: Lead[] = [...handLeads, ...genLeads]

const stage = (n: number, amount: number, state: Invoice['stages'][number]['state'], note?: string) => ({ n, amount, state, note })

export const INVOICES: Invoice[] = [
  { code: 'TST-1001', customer: 'مشتری نمونه ر', phone: phone(201), product: 'اشتراک نمونه ۱۰ ستاره', status: 'pre_invoice', next: 'send_link', stages: [stage(1, 10_000_000, 'due')], history: [{ text: 'صدور پیش‌فاکتور', meta: 'فروشنده نمونه · ۱۴۰۵/۰۷/۰۱' }] },
  { code: 'TST-1002', customer: 'مشتری نمونه ز', phone: phone(202), product: 'اعتبارسنجی نمونه ۲۵۰ هزار', status: 'approved', next: null, stages: [stage(1, 250_000, 'confirmed')], history: [{ text: 'تایید مالی', meta: 'کارشناس مالی (نمونه) · ۱۴۰۵/۰۶/۱۸' }, { text: 'صدور پیش‌فاکتور', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۱۵' }] },
  { code: 'TST-1003', customer: 'مشتری نمونه ت', phone: phone(104), product: 'اشتراک نمونه نقره‌ای', status: 'staged', next: 'register_payment', stages: [stage(1, 10_000_000, 'confirmed'), stage(2, 19_000_000, 'due'), stage(3, 19_000_000, 'upcoming')], history: [{ text: 'تایید مالی مرحله ۱', meta: 'کارشناس مالی (نمونه) · ۱۴۰۵/۰۷/۰۳' }, { text: 'صدور پیش‌فاکتور', meta: 'فروشنده نمونه · ۱۴۰۵/۰۷/۰۱' }] },
  { code: 'TST-1004', customer: 'مشتری نمونه ث', phone: phone(105), product: 'اشتراک نمونه طلایی', status: 'finance_review', next: null, stages: [stage(1, 125_000_000, 'submitted', 'فیش ارسال شد؛ هنوز تایید مالی نشده')], history: [{ text: 'ارسال فیش برای بررسی مالی', meta: 'فروشنده نمونه · دیروز' }, { text: 'صدور پیش‌فاکتور', meta: 'فروشنده نمونه · ۱۴۰۵/۰۷/۰۲' }] },
  { code: 'TST-1005', customer: 'مشتری نمونه ج', phone: phone(106), product: 'اشتراک نمونه برنزی', status: 'rejected', next: 'resend_finance', rejectReason: 'فیش خوانا نیست (نمونه)', stages: [stage(1, 9_250_000, 'rejected', 'فیش رد شد'), stage(2, 9_250_000, 'upcoming')], history: [{ text: 'رد توسط مالی', meta: 'کارشناس مالی (نمونه) · دیروز ۱۴:۲۰' }, { text: 'ارسال فیش برای بررسی', meta: 'فروشنده نمونه · ۲ روز پیش' }, { text: 'صدور پیش‌فاکتور', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۲۸' }] },
  { code: 'TST-1006', customer: 'مشتری نمونه ح', phone: phone(108), product: 'اشتراک نمونه ۱۰ ستاره', status: 'approved', next: null, stages: [stage(1, 10_000_000, 'confirmed')], history: [{ text: 'تایید مالی', meta: 'کارشناس مالی (نمونه) · ۱۴۰۵/۰۶/۱۸' }, { text: 'صدور پیش‌فاکتور', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۱۵' }] },
  { code: 'TST-1007', customer: 'مشتری نمونه الف', phone: phone(101), product: 'اشتراک نمونه نقره‌ای + پکیج', status: 'staged', next: null, stages: [stage(1, 32_000_000, 'confirmed'), stage(2, 32_000_000, 'unknown', 'پاسخ ارسال فیش نرسید؛ ممکن است ثبت شده باشد')], history: [{ text: 'ارسال فیش مرحله ۲ — نتیجه نامعلوم', meta: 'فروشنده نمونه · امروز' }, { text: 'تایید مالی مرحله ۱', meta: 'کارشناس مالی (نمونه) · ۱۴۰۵/۰۶/۲۲' }, { text: 'صدور پیش‌فاکتور', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۲۰' }] },
  { code: 'TST-1008', customer: 'مشتری نمونه چ', phone: phone(107), product: 'اعتبارسنجی نمونه ۲۵۰ هزار', status: 'cancelled', next: null, stages: [stage(1, 250_000, 'void')], history: [{ text: 'لغو پیش‌فاکتور', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۲۱' }] },
]

export const CONVERSIONS: Conversion[] = [
  { id: 104, invoice: 'TST-1002', customer: 'مشتری نمونه ز', phone: phone(202), option: 'اشتراک نمونه نقره‌ای', remaining: 10_000_000, stage: 'follow', stageLabel: 'تماس انجام شد', result: 'مشتری تأیید کرد', resultTone: 'green', resultIcon: 'checkCircle', lastActivity: 'دیروز', history: [{ text: 'مشتری تأیید کرد', meta: 'دیروز' }, { text: 'تخصیص به شما', meta: '۱۴۰۵/۰۶/۲۰' }] },
  { id: 97, invoice: 'TST-1006', customer: 'مشتری نمونه ح', phone: phone(108), option: 'اشتراک نمونه طلایی', remaining: 42_000_000, stage: 'link', stageLabel: 'لینک پرداخت ارسال شد', result: 'آماده ارسال لینک', resultTone: 'teal', resultIcon: 'send', lastActivity: '۲ روز پیش', history: [{ text: 'لینک پرداخت ارسال شد', meta: '۲ روز پیش' }, { text: 'تخصیص به شما', meta: '۱۴۰۵/۰۶/۱۸' }] },
  { id: 91, invoice: 'TST-1007', customer: 'مشتری نمونه الف', phone: phone(101), option: '—', remaining: null, stage: 'new', stageLabel: 'تخصیص به تبدیل‌کننده', result: 'جدید', resultTone: 'neutral', resultIcon: 'dashed', lastActivity: '—', history: [{ text: 'تخصیص به شما', meta: '۱۴۰۵/۰۶/۲۲' }] },
]

export const REPEAT_ITEMS: RepeatItem[] = [
  { id: 301, invoice: 'TST-1003', customer: 'مشتری نمونه ت', phone: phone(104), stageLabel: 'مرحله ۲ از ۳', amount: 19_000_000, originalSeller: 'فروشنده نمونه دیگر · SYN-0002', currentCustody: 'شما (فروشنده نمونه)', nextActor: 'شما (فروشنده نمونه)', due: 'امروز', dueTone: 'due', done: false, history: [{ text: 'ارجاع مرحله ۲ به شما', meta: 'سیستم (نمونه) · امروز ۰۹:۱۰' }, { text: 'تایید مالی مرحله ۱', meta: 'کارشناس مالی (نمونه) · ۱۴۰۵/۰۷/۰۳' }, { text: 'صدور فاکتور', meta: 'فروشنده نمونه دیگر · ۱۴۰۵/۰۷/۰۱' }] },
  { id: 302, invoice: 'TST-1009', customer: 'مشتری نمونه ص', phone: phone(303), stageLabel: 'مرحله ۳ از ۳', amount: 12_500_000, originalSeller: 'فروشنده نمونه سوم · SYN-0003', currentCustody: 'شما (فروشنده نمونه)', nextActor: 'شما (فروشنده نمونه)', due: '۱۴۰۵/۰۷/۱۲', dueTone: 'warn', done: false, history: [{ text: 'ارجاع مرحله ۳ به شما', meta: 'سیستم (نمونه) · دیروز' }, { text: 'صدور فاکتور', meta: 'فروشنده نمونه سوم · ۱۴۰۵/۰۶/۰۹' }] },
  { id: 303, invoice: 'TST-1010', customer: 'مشتری نمونه ض', phone: phone(304), stageLabel: 'مرحله ۲ از ۲', amount: 8_000_000, originalSeller: 'فروشنده نمونه دیگر · SYN-0002', currentCustody: 'شما (فروشنده نمونه)', nextActor: '—', due: 'انجام شد', dueTone: '', done: true, history: [{ text: 'پرداخت مرحله ۲ ثبت شد', meta: 'فروشنده نمونه · ۱۴۰۵/۰۷/۰۲' }, { text: 'ارجاع مرحله ۲ به شما', meta: 'سیستم (نمونه) · ۱۴۰۵/۰۶/۳۰' }] },
]

export const BEHAVIOR: BehaviorItem[] = [
  { id: 'b1', name: 'مشتری نمونه ر', phone: phone(201), invoice: 'TST-1001', last: 'باز کردن لینک فاکتور', when: '۱۰ دقیقه پیش', count: 7, hot: true, events: [{ text: 'باز کردن لینک فاکتور', meta: '۱۰ دقیقه پیش' }, { text: 'مشاهده اطلاعات محصول', meta: '۲ ساعت پیش' }, { text: 'ارسال پیامک پیش‌فاکتور', meta: '۳ روز پیش' }] },
  { id: 'b2', name: 'مشتری نمونه ت', phone: phone(104), invoice: 'TST-1003', last: 'کلیک روی گردونه', when: '۱ ساعت پیش', count: 4, hot: true, events: [{ text: 'کلیک روی گردونه', meta: '۱ ساعت پیش' }, { text: 'باز کردن لینک فاکتور', meta: 'دیروز' }] },
  { id: 'b3', name: 'مشتری نمونه الف', phone: phone(101), invoice: 'TST-1007', last: 'پرداخت مرحله ۲', when: 'دیروز', count: 12, hot: false, events: [{ text: 'پرداخت مرحله ۲ (نتیجه نامعلوم)', meta: 'دیروز' }, { text: 'باز کردن لینک فاکتور', meta: '۲ روز پیش' }] },
  { id: 'b4', name: 'مشتری نمونه ج', phone: phone(106), invoice: 'TST-1005', last: 'مشاهده اطلاعات محصول', when: '۳ روز پیش', count: 3, hot: false, events: [{ text: 'مشاهده اطلاعات محصول', meta: '۳ روز پیش' }] },
  { id: 'b5', name: 'مشتری نمونه ح', phone: phone(108), invoice: 'TST-1006', last: 'پرداخت موفق', when: '۵ روز پیش', count: 9, hot: false, events: [{ text: 'پرداخت موفق', meta: '۵ روز پیش' }, { text: 'باز کردن لینک فاکتور', meta: '۶ روز پیش' }] },
  { id: 'b6', name: 'مشتری نمونه ز', phone: phone(202), invoice: 'TST-1002', last: 'باز کردن لینک فاکتور', when: '۶ روز پیش', count: null, hot: false, events: [{ text: 'باز کردن لینک فاکتور', meta: '۶ روز پیش' }, { text: 'شمار رویدادهای قدیمی‌تر در دسترس نیست', meta: 'پوشش ناقص (نمونه)' }] },
]

export const EXTRA_REQUESTS: ExtraRequest[] = [
  { id: 'r1', phone: phone(401), name: 'مشتری نمونه ط', when: '۱۴۰۵/۰۷/۰۶', status: 'pending', timeline: [{ text: 'ثبت درخواست', meta: 'فروشنده نمونه · ۱۴۰۵/۰۷/۰۶' }, { text: 'در انتظار بررسی مدیر', meta: 'نمونه' }] },
  { id: 'r2', phone: phone(402), name: '—', when: '۱۴۰۵/۰۶/۲۹', status: 'approved', timeline: [{ text: 'تایید شد', meta: 'مدیر فروش (نمونه) · ۱۴۰۵/۰۶/۳۰' }, { text: 'ثبت درخواست', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۲۹' }] },
  { id: 'r3', phone: phone(403), name: 'مشتری نمونه ظ', when: '۱۴۰۵/۰۶/۱۲', status: 'rejected', timeline: [{ text: 'رد شد', meta: 'مدیر فروش (نمونه) · ۱۴۰۵/۰۶/۱۳' }, { text: 'ثبت درخواست', meta: 'فروشنده نمونه · ۱۴۰۵/۰۶/۱۲' }] },
]

export const WALLET: WalletFixture = {
  lastCredit: '۱۴۰۵/۰۶/۱۷',
  tx: [
    { id: 'w1', date: '۱۴۰۵/۰۶/۱۷ ۱۴:۰۱', channel: 'card', amount: 7_500, invoice: 'TST-1002', rule: 'قاعده نمونه #۶', state: 'confirmed' },
    { id: 'w2', date: '۱۴۰۵/۰۶/۱۰ ۱۶:۰۹', channel: 'card', amount: 7_500, invoice: 'TST-1006', rule: 'قاعده نمونه #۶', state: 'confirmed' },
    { id: 'w3', date: '۱۴۰۵/۰۶/۰۵ ۲۲:۲۵', channel: 'online', amount: 1_130_000, invoice: 'TST-1003', rule: 'قاعده نمونه #۴', state: 'confirmed' },
    { id: 'w4', date: '۱۴۰۵/۰۵/۲۳ ۱۸:۵۶', channel: 'card', amount: 7_500, invoice: 'TST-1011', rule: 'قاعده نمونه #۶', state: 'confirmed' },
    { id: 'w5', date: '۱۴۰۵/۰۵/۲۳ ۰۲:۰۴', channel: 'online', amount: 300_000, invoice: 'TST-1012', rule: 'قاعده نمونه #۴', state: 'pending' },
    { id: 'w6', date: '۱۴۰۵/۰۵/۲۲ ۲۳:۱۵', channel: 'card', amount: 480, invoice: 'TST-1013', rule: 'قاعده نمونه #۶', state: 'confirmed' },
    { id: 'w7', date: '۱۴۰۵/۰۵/۲۲ ۲۲:۴۴', channel: 'online', amount: null, invoice: 'TST-1007', rule: 'قاعده نمونه #۴', state: 'unknown' },
    { id: 'w8', date: '۱۴۰۵/۰۵/۱۹ ۲۲:۲۷', channel: 'online', amount: 300_000, invoice: 'TST-1014', rule: 'قاعده نمونه #۴', state: 'pending' },
  ],
}
