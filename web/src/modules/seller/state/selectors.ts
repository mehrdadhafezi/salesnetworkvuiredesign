/*
 * Pure, local presentation logic for the RM-E03 Seller preview: search, queue/filter, sort,
 * pagination, derived invoice/wallet figures and form validation over SYNTHETIC fixtures.
 * No function here reads or writes CRM data. Unknown values stay `null` and are never
 * coerced to zero.
 */
import type {
  BehaviorItem, Conversion, ConversionQueue, Invoice, InvoiceQueue, InvoiceStatus, Lead, LeadOutcome, LeadQueue, LeadStatus,
  MoneyUnit, RepeatItem, WalletChannel, WalletTx, WalletTxState,
} from '../types'

/** Unit of an invoice or wallet line; the RM-E03 fixtures predate units and are IRT. */
export function unitOf(x: { unit?: MoneyUnit | undefined }): MoneyUnit {
  return x.unit ?? 'IRT'
}

/** Stable unit order for display (IRT first, as the fixtures are mostly IRT). */
export const UNIT_ORDER: readonly MoneyUnit[] = ['IRT', 'IRR']

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

export function leadCounts(leads: readonly Lead[]): Record<LeadQueue, number> {
  const out: Record<LeadQueue, number> = { none: 0, no_answer: 0, callback: 0, not_purchased: 0, duplicate: 0, invoiced: 0, all: leads.length }
  for (const l of leads) out[l.status] += 1
  return out
}

/** `identity=false` (A7 FIELD_DENIED): name and phone are not searchable, so a search cannot confirm a withheld value. */
export function filterLeads(leads: readonly Lead[], f: LeadFilter, identity = true): Lead[] {
  const list = leads.filter(l =>
    (f.queue === 'all' || l.status === f.queue)
    && matches(f.q, ...(identity ? [l.name, l.phone] : []), l.city)
    && (f.source === null || l.source === f.source)
    && (f.prob === null || (f.prob === '—' ? l.prob === '' : l.prob === f.prob))
    && (f.range === 'all' || l.assignedDays <= Number(f.range)))
  return [...list].sort((a, b) => (f.sort === 'newest' ? a.assignedDays - b.assignedDays : b.assignedDays - a.assignedDays))
}

/** Share of assigned leads that already carry a status (prototype «نرخ پیگیری»). null when there are no leads. */
export function followUpRate(leads: readonly Lead[]): number | null {
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
  /** A7 PARTIAL: depends on incomplete facts, so it is not shown (not the same as unknown). */
  | { kind: 'withheld' }

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

export function invoiceCounts(invoices: readonly Invoice[]): Record<InvoiceQueue, number> {
  const out: Record<InvoiceQueue, number> = { all: invoices.length, pre_invoice: 0, staged: 0, finance_review: 0, approved: 0, rejected: 0, cancelled: 0 }
  for (const i of invoices) out[i.status] += 1
  return out
}

export function filterInvoices(invoices: readonly Invoice[], queue: InvoiceQueue, q: string, product: string | null, identity = true): Invoice[] {
  return invoices.filter(i => (queue === 'all' || i.status === queue) && matches(q, i.code, ...(identity ? [i.customer, i.phone] : [])) && (product === null || i.product === product))
}

/**
 * Remaining receivable across non-cancelled invoices, one sum PER UNIT (never added across units),
 * and whether any part is unknown (then no sum is final).
 */
export function collectibleTotal(invoices: readonly Invoice[]): { byUnit: { unit: MoneyUnit; value: number }[]; hasUnknown: boolean } {
  const sums = new Map<MoneyUnit, number>()
  let hasUnknown = false
  for (const i of invoices) {
    const a = invoiceAmounts(i)
    if (a.remaining !== null) sums.set(unitOf(i), (sums.get(unitOf(i)) ?? 0) + a.remaining)
    if (a.unknown > 0) hasUnknown = true
  }
  return { byUnit: UNIT_ORDER.filter(u => sums.has(u)).map(u => ({ unit: u, value: sums.get(u) ?? 0 })), hasUnknown }
}

export const INVOICE_STATUSES: InvoiceStatus[] = ['pre_invoice', 'staged', 'finance_review', 'approved', 'rejected', 'cancelled']

/* ---------- Conversions / repeat / behavior ---------- */

export function filterConversions(list: readonly Conversion[], queue: ConversionQueue, q: string, identity = true): Conversion[] {
  return list.filter(c => (queue === 'all' || c.stage === queue) && matches(q, ...(identity ? [c.customer, c.phone] : []), c.invoice, String(c.id)))
}

export function filterRepeat(list: readonly RepeatItem[], queue: 'open' | 'done', q: string, identity = true): RepeatItem[] {
  return list.filter(r => (queue === 'open' ? !r.done : r.done) && matches(q, ...(identity ? [r.customer, r.phone] : []), r.invoice))
}

export function filterBehavior(list: readonly BehaviorItem[], q: string, onlyHot: boolean, identity = true): BehaviorItem[] {
  return list.filter(b => (!onlyHot || b.hot) && matches(q, ...(identity ? [b.name, b.phone] : []), b.invoice))
}

/* ---------- Wallet ---------- */

export interface WalletUnitTotals { unit: MoneyUnit; confirmed: number; pending: number }

export interface WalletTotals {
  /** One entry per unit present; units are never added together. */
  byUnit: WalletUnitTotals[]
  /** Lines whose amount or state is unknown — excluded from every sum, so totals are incomplete. */
  unknownCount: number
}

export function walletTotals(tx: readonly WalletTx[]): WalletTotals {
  const sums = new Map<MoneyUnit, WalletUnitTotals>()
  let unknownCount = 0
  for (const t of tx) {
    if (t.state === 'unknown' || t.amount === null) { unknownCount += 1; continue }
    const u = unitOf(t)
    const cur = sums.get(u) ?? { unit: u, confirmed: 0, pending: 0 }
    if (t.state === 'confirmed') cur.confirmed += t.amount
    else cur.pending += t.amount
    sums.set(u, cur)
  }
  return { byUnit: UNIT_ORDER.flatMap(u => { const v = sums.get(u); return v ? [v] : [] }), unknownCount }
}

export function filterWallet(tx: readonly WalletTx[], channel: 'all' | WalletChannel, state: 'all' | WalletTxState): WalletTx[] {
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
