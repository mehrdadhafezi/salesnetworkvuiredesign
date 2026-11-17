/*
 * Pure, local presentation logic for the RM-E03 Seller preview: search, queue/filter, sort,
 * pagination, derived invoice/wallet figures and form validation over SYNTHETIC fixtures.
 * No function here reads or writes CRM data. Unknown values stay `null` and are never
 * coerced to zero.
 */
import type {
  BehaviorItem, Conversion, ConversionQueue, Invoice, InvoiceQueue, InvoiceStatus, Lead, LeadOutcome, LeadQueue, LeadStatus,
  RepeatItem, WalletChannel, WalletTx, WalletTxState,
} from '../types'

/** Persian / Arabic-Indic digits → ASCII, lower-case, trimmed (prototype palette norm()). */
export function normalizeSearch(s: string): string {
  return s
    .replace(/[۰-۹]/g, d => String('۰۱۲۳۴۵۶۷۸۹'.indexOf(d)))
    .replace(/[٠-٩]/g, d => String('٠١٢٣٤٥٦٧٨٩'.indexOf(d)))
    .replace(/[\s-]+/g, ' ')
    .trim()
    .toLowerCase()
}

function matches(q: string, ...fields: string[]): boolean {
  const nq = normalizeSearch(q)
  if (!nq) return true
  return fields.some(f => normalizeSearch(f).includes(nq))
}

/* ---------- Leads ---------- */

export type DateRange = 'all' | '7' | '30'
export type LeadSort = 'newest' | 'oldest'

export interface LeadFilter {
  queue: LeadQueue
  q: string
  source: string | null
  prob: string | null
  range: DateRange
  sort: LeadSort
}

export const DEFAULT_LEAD_FILTER: LeadFilter = { queue: 'none', q: '', source: null, prob: null, range: 'all', sort: 'newest' }

export function leadCounts(leads: Lead[]): Record<LeadQueue, number> {
  const out: Record<LeadQueue, number> = { none: 0, no_answer: 0, callback: 0, not_purchased: 0, duplicate: 0, invoiced: 0, all: leads.length }
  for (const l of leads) out[l.status] += 1
  return out
}

export function filterLeads(leads: Lead[], f: LeadFilter): Lead[] {
  const list = leads.filter(l =>
    (f.queue === 'all' || l.status === f.queue)
    && matches(f.q, l.name, l.phone, l.city)
    && (f.source === null || l.source === f.source)
    && (f.prob === null || (f.prob === '—' ? l.prob === '' : l.prob === f.prob))
    && (f.range === 'all' || l.assignedDays <= Number(f.range)))
  return [...list].sort((a, b) => (f.sort === 'newest' ? a.assignedDays - b.assignedDays : b.assignedDays - a.assignedDays))
}

/** Share of assigned leads that already carry a status (prototype «نرخ پیگیری»). null when there are no leads. */
export function followUpRate(leads: Lead[]): number | null {
  if (!leads.length) return null
  return Math.round(((leads.length - leadCounts(leads).none) / leads.length) * 100)
}

/* ---------- Pagination ---------- */

export interface Page<T> { items: T[]; page: number; pages: number; from: number; to: number; total: number }

export function paginate<T>(list: T[], page: number, size: number): Page<T> {
  const pages = Math.max(1, Math.ceil(list.length / size))
  const p = Math.min(Math.max(1, page), pages)
  const start = (p - 1) * size
  const items = list.slice(start, start + size)
  return { items, page: p, pages, from: items.length ? start + 1 : 0, to: start + items.length, total: list.length }
}

/* ---------- Invoices ---------- */

export type Payable =
  | { kind: 'amount'; value: number; stage: number }
  | { kind: 'settled' }
  | { kind: 'awaiting_finance' }
  | { kind: 'unknown' }
  | { kind: 'none' }

export interface InvoiceAmounts {
  total: number
  /** Financially confirmed only. */
  confirmed: number
  /** Receipts submitted by the seller, not yet confirmed by finance. Not counted as paid. */
  submitted: number
  /** Stage amounts whose submission outcome is unknown. Not counted as paid or as remaining-certain. */
  unknown: number
  /** total − confirmed; null when the invoice is cancelled (no receivable to show). */
  remaining: number | null
  payable: Payable
}

export function invoiceAmounts(inv: Invoice): InvoiceAmounts {
  const sum = (pred: (s: Invoice['stages'][number]) => boolean) => inv.stages.filter(pred).reduce((t, s) => t + s.amount, 0)
  const total = sum(() => true)
  const confirmed = sum(s => s.state === 'confirmed')
  const submitted = sum(s => s.state === 'submitted')
  const unknown = sum(s => s.state === 'unknown')
  if (inv.status === 'cancelled') return { total, confirmed, submitted, unknown, remaining: null, payable: { kind: 'none' } }
  const current = inv.stages.find(s => s.state === 'due' || s.state === 'rejected')
  let payable: Payable
  if (current) payable = { kind: 'amount', value: current.amount, stage: current.n }
  else if (inv.stages.some(s => s.state === 'unknown')) payable = { kind: 'unknown' }
  else if (inv.stages.some(s => s.state === 'submitted')) payable = { kind: 'awaiting_finance' }
  else if (confirmed >= total) payable = { kind: 'settled' }
  else payable = { kind: 'none' }
  return { total, confirmed, submitted, unknown, remaining: total - confirmed, payable }
}

export function invoiceCounts(invoices: Invoice[]): Record<InvoiceQueue, number> {
  const out: Record<InvoiceQueue, number> = { all: invoices.length, pre_invoice: 0, staged: 0, finance_review: 0, approved: 0, rejected: 0, cancelled: 0 }
  for (const i of invoices) out[i.status] += 1
  return out
}

export function filterInvoices(invoices: Invoice[], queue: InvoiceQueue, q: string, product: string | null): Invoice[] {
  return invoices.filter(i => (queue === 'all' || i.status === queue) && matches(q, i.code, i.customer, i.phone) && (product === null || i.product === product))
}

/** Sum of remaining receivable across non-cancelled invoices; also reports whether any part is unknown. */
export function collectibleTotal(invoices: Invoice[]): { value: number; hasUnknown: boolean } {
  let value = 0, hasUnknown = false
  for (const i of invoices) {
    const a = invoiceAmounts(i)
    if (a.remaining !== null) value += a.remaining
    if (a.unknown > 0) hasUnknown = true
  }
  return { value, hasUnknown }
}

export const INVOICE_STATUSES: InvoiceStatus[] = ['pre_invoice', 'staged', 'finance_review', 'approved', 'rejected', 'cancelled']

/* ---------- Conversions / repeat / behavior ---------- */

export function filterConversions(list: Conversion[], queue: ConversionQueue, q: string): Conversion[] {
  return list.filter(c => (queue === 'all' || c.stage === queue) && matches(q, c.customer, c.phone, c.invoice, String(c.id)))
}

export function filterRepeat(list: RepeatItem[], queue: 'open' | 'done', q: string): RepeatItem[] {
  return list.filter(r => (queue === 'open' ? !r.done : r.done) && matches(q, r.customer, r.phone, r.invoice))
}

export function filterBehavior(list: BehaviorItem[], q: string, onlyHot: boolean): BehaviorItem[] {
  return list.filter(b => (!onlyHot || b.hot) && matches(q, b.name, b.phone, b.invoice))
}

/* ---------- Wallet ---------- */

export interface WalletTotals {
  confirmed: number
  pending: number
  /** Lines whose amount or state is unknown — excluded from both sums, so totals are incomplete. */
  unknownCount: number
}

export function walletTotals(tx: WalletTx[]): WalletTotals {
  let confirmed = 0, pending = 0, unknownCount = 0
  for (const t of tx) {
    if (t.state === 'unknown' || t.amount === null) { unknownCount += 1; continue }
    if (t.state === 'confirmed') confirmed += t.amount
    else pending += t.amount
  }
  return { confirmed, pending, unknownCount }
}

export function filterWallet(tx: WalletTx[], channel: 'all' | WalletChannel, state: 'all' | WalletTxState): WalletTx[] {
  return tx.filter(t => (channel === 'all' || t.channel === channel) && (state === 'all' || t.state === state))
}

/* ---------- Validation (presentation only; mirrors the prototype's required fields) ---------- */

export interface OutcomeFields { date: string; hour: string; minute: string; reason: string; amount: string }

export type FieldErrors = Partial<Record<keyof OutcomeFields | 'phone' | 'name' | 'note', string>>

export const REQUIRED_TEXT = 'این فیلد الزامی است.'
export const REQUIRED_REASON = 'نوشتن دلیل برای این نتیجه الزامی است.'

/** callback / follow_up need date + time; not_purchased / customer_declined need a reason (prototype REQUIRED). */
export function validateOutcome(outcome: LeadOutcome | 'follow_up' | 'customer_declined' | 'payment_link' | null, f: OutcomeFields): FieldErrors {
  const e: FieldErrors = {}
  if (outcome === 'callback' || outcome === 'follow_up') {
    if (!f.date.trim()) e.date = REQUIRED_TEXT
    if (!f.hour.trim()) e.hour = REQUIRED_TEXT
    if (!f.minute.trim()) e.minute = REQUIRED_TEXT
  }
  if (outcome === 'not_purchased' || outcome === 'customer_declined') {
    if (!f.reason.trim()) e.reason = REQUIRED_REASON
  }
  return e
}

export const PHONE_REQUIRED = 'شماره موبایل مشتری الزامی است.'
export const PHONE_INVALID = 'شماره باید ۱۱ رقم و با ۰۹ شروع شود.'
export const NOTE_TOO_LONG = 'یادداشت حداکثر ۳۰۰ نویسه است.'

export function validateExtraRequest(v: { phone: string; name: string; note: string }): FieldErrors {
  const e: FieldErrors = {}
  const p = normalizeSearch(v.phone).replace(/\s/g, '')
  if (!p) e.phone = PHONE_REQUIRED
  else if (!/^09\d{9}$/.test(p)) e.phone = PHONE_INVALID
  if (v.note.length > 300) e.note = NOTE_TOO_LONG
  return e
}

export function isLeadStatus(v: string | null): v is LeadStatus {
  return v === 'none' || v === 'no_answer' || v === 'callback' || v === 'not_purchased' || v === 'duplicate' || v === 'invoiced'
}
