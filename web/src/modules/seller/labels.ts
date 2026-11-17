/*
 * Presentation dictionaries of the frozen Seller V1 prototype (prototype/app.js LEAD, INV,
 * invActions, leadOutcomes, NAV, QUEUE_ORDER). Labels, tones and icons are reused verbatim
 * for visual parity. They are display vocabulary only — not business statuses, transitions
 * or permissions, and nothing here is sent anywhere.
 */
import type { DsIconName, DsNavItem } from '../../shared/ui/ds'
import type { Tone } from '../../shared/ui/primitives'
import type { InvoiceNextAction, InvoiceStatus, LeadOutcome, LeadStatus, SellerSection, StageState, WalletTxState } from './types'

export interface Presentation { label: string; tone: Tone; icon: DsIconName }

export const LEAD_STATUS: Record<LeadStatus, Presentation & { key: string }> = {
  none: { label: 'بدون وضعیت', tone: 'neutral', icon: 'dashed', key: '1' },
  no_answer: { label: 'جواب نداده', tone: 'amber', icon: 'phoneOff', key: '2' },
  callback: { label: 'تماس مجدد', tone: 'blue', icon: 'clock', key: '3' },
  not_purchased: { label: 'عدم خرید', tone: 'red', icon: 'xCircle', key: '4' },
  duplicate: { label: 'تکراری', tone: 'slate', icon: 'copy', key: '5' },
  invoiced: { label: 'پیش‌فاکتور', tone: 'violet', icon: 'file', key: '6' },
}

export const LEAD_QUEUE_ORDER = ['none', 'no_answer', 'callback', 'not_purchased', 'duplicate', 'invoiced', 'all'] as const

/** Statuses whose row offers «مشاهده» instead of «ثبت نتیجه» (prototype `closed`). */
export const CLOSED_LEAD_STATUSES: readonly LeadStatus[] = ['invoiced', 'not_purchased', 'duplicate']

export const INVOICE_STATUS: Record<InvoiceStatus, Presentation> = {
  pre_invoice: { label: 'پیش‌فاکتور', tone: 'violet', icon: 'file' },
  staged: { label: 'مرحله‌ای', tone: 'teal', icon: 'layers' },
  finance_review: { label: 'در بررسی مالی', tone: 'orange', icon: 'clock' },
  approved: { label: 'تایید شده مالی', tone: 'green', icon: 'checkCircle' },
  rejected: { label: 'رد شده', tone: 'red', icon: 'alert' },
  cancelled: { label: 'لغو شده', tone: 'slate', icon: 'ban' },
}

/** Queue labels (prototype uses the short «تایید شده» in the queue chip). */
export const INVOICE_QUEUE_LABEL: Record<InvoiceStatus, string> = {
  pre_invoice: 'پیش‌فاکتور', staged: 'مرحله‌ای', finance_review: 'در بررسی مالی', approved: 'تایید شده', rejected: 'رد شده', cancelled: 'لغو شده',
}

export const INVOICE_ACTION: Record<Exclude<InvoiceNextAction, null>, { label: string; icon: DsIconName }> = {
  send_link: { label: 'ارسال لینک پرداخت', icon: 'send' },
  register_payment: { label: 'ثبت واریز و فیش', icon: 'file' },
  resend_finance: { label: 'ارسال مجدد به مالی', icon: 'repeat' },
}

/** Stage presentation: submitted ≠ confirmed; unknown is its own state. */
export const STAGE_STATE: Record<StageState, Presentation> = {
  confirmed: { label: 'تایید مالی شده', tone: 'green', icon: 'checkCircle' },
  submitted: { label: 'فیش ارسال‌شده — در انتظار تایید مالی', tone: 'orange', icon: 'clock' },
  rejected: { label: 'رد شده — نیاز به ارسال مجدد', tone: 'red', icon: 'alert' },
  due: { label: 'مرحله جاری — قابل پرداخت', tone: 'blue', icon: 'clock' },
  upcoming: { label: 'بعدی', tone: 'neutral', icon: 'dashed' },
  unknown: { label: 'نتیجه ارسال نامعلوم', tone: 'amber', icon: 'question' },
  void: { label: 'لغو شده', tone: 'slate', icon: 'ban' },
}

export const WALLET_STATE: Record<WalletTxState, Presentation> = {
  confirmed: { label: 'تایید شده', tone: 'green', icon: 'checkCircle' },
  pending: { label: 'در انتظار تایید', tone: 'orange', icon: 'clock' },
  unknown: { label: 'نامعلوم', tone: 'amber', icon: 'question' },
}

export interface OutcomeOption { id: LeadOutcome; label: string; tone: Tone; icon: DsIconName; hint: string; key: string; wide?: boolean }

export const LEAD_OUTCOMES: OutcomeOption[] = [
  { id: 'no_answer', label: 'جواب نداده', tone: 'amber', icon: 'phoneOff', hint: '', key: '1' },
  { id: 'callback', label: 'تماس مجدد', tone: 'blue', icon: 'clock', hint: 'تاریخ و ساعت لازم', key: '2' },
  { id: 'not_purchased', label: 'عدم خرید', tone: 'red', icon: 'xCircle', hint: 'دلیل الزامی', key: '3' },
  { id: 'duplicate', label: 'تکراری', tone: 'slate', icon: 'copy', hint: 'قبلاً ثبت شده', key: '4' },
  { id: 'success', label: 'موفق — صدور پیش‌فاکتور', tone: 'green', icon: 'checkCircle', hint: 'فرم پیش‌فاکتور همین‌جا باز می‌شود', key: '5', wide: true },
]

export type CaseOutcome = 'no_answer' | 'follow_up' | 'customer_declined' | 'payment_link'

export const CASE_OUTCOMES: { id: CaseOutcome; label: string; tone: Tone; icon: DsIconName; hint: string; key: string; wide?: boolean }[] = [
  { id: 'no_answer', label: 'جواب نداده', tone: 'amber', icon: 'phoneOff', hint: '', key: '1' },
  { id: 'follow_up', label: 'تماس مجدد', tone: 'blue', icon: 'clock', hint: 'تاریخ و ساعت لازم', key: '2' },
  { id: 'customer_declined', label: 'انصراف', tone: 'red', icon: 'xCircle', hint: 'علت الزامی', key: '3' },
  { id: 'payment_link', label: 'ادامه — ارسال لینک پرداخت', tone: 'green', icon: 'send', hint: 'محصول نهایی و مبلغ مرحله', key: '4', wide: true },
]

export const SECTION_LABEL: Record<SellerSection, string> = {
  leads: 'شماره‌های من',
  conversions: 'آماده‌های تبدیل',
  repeat: 'نیاز به اقدام مجدد',
  invoices: 'فاکتورها',
  behavior: 'رفتار مشتریان',
  request: 'درخواست اضافه کردن شماره',
  wallet: 'کیف پول من',
}

/** Navigation labels and grouping from prototype/app.js NAV (tab labels are shorter for the last two). */
export function sellerNavItems(base: string, counts: { leadsNone: number; conversions: number; invoicesAction: number }): DsNavItem[] {
  return [
    { id: 'leads', label: 'شماره‌های من', href: `${base}/leads`, icon: 'users', count: counts.leadsNone, alert: true, mobileLabel: 'شماره‌ها' },
    { id: 'conversions', label: 'آماده‌های تبدیل', href: `${base}/conversions`, icon: 'swap', count: counts.conversions, mobileLabel: 'تبدیل' },
    { id: 'repeat', label: 'نیاز به اقدام مجدد', href: `${base}/repeat`, icon: 'repeat' },
    { id: 'invoices', label: 'فاکتورها', href: `${base}/invoices`, icon: 'receipt', count: counts.invoicesAction, mobileLabel: 'فاکتورها' },
    { id: 'behavior', label: 'رفتار مشتریان', href: `${base}/behavior`, icon: 'activity', group: 'more' },
    { id: 'request', label: 'درخواست شماره', href: `${base}/request`, icon: 'plus', group: 'more' },
    { id: 'wallet', label: 'کیف پول', href: `${base}/wallet`, icon: 'wallet', group: 'more', mobileLabel: 'کیف پول' },
  ]
}

/** Every simulated action ends with this suffix: the preview never claims a server action happened. */
export const DEMO_SUFFIX = '(نمایشی — چیزی ثبت یا ارسال نشد)'
