/*
 * RM-E03 Seller V1 preview — presentation types only.
 * These mirror the frozen Seller Design V1 prototype (prototype/data.js, prototype/app.js) so the
 * React screens can be compared with it. They are NOT API contracts, business statuses or workflow
 * definitions: nothing here is read from or written to the CRM, and no key grants anything.
 */
import type { Tone } from '../../shared/ui/primitives'
import type { DsIconName } from '../../shared/ui/ds'
import type { Unit } from '../../shared/api/types'

/** Money unit, mirrored from the contract `Unit` (IRR · IRT). Amounts in different units are never added together. */
export type MoneyUnit = Unit

export type SellerSection = 'leads' | 'conversions' | 'repeat' | 'invoices' | 'behavior' | 'request' | 'wallet'

export const SELLER_SECTIONS: readonly SellerSection[] = ['leads', 'conversions', 'repeat', 'invoices', 'behavior', 'request', 'wallet']

/** Lead queue keys exactly as the prototype names them (prototype/app.js LEAD, QUEUE_ORDER). */
export type LeadStatus = 'none' | 'no_answer' | 'callback' | 'not_purchased' | 'duplicate' | 'invoiced'
export type LeadQueue = LeadStatus | 'all'

export type LeadSource = 'MIS' | 'درخواستی' | 'ارتباط مجدد'
export type LeadProbability = 'ضعیف' | 'متوسط' | 'بالا' | '۱۰۰٪' | ''

export interface LeadNext {
  kind: 'callback' | 'attempts' | 'archive' | 'invoice'
  text: string
  warn: boolean
  /** Prototype marks "today" callbacks and pending archive as due (red). */
  due?: boolean | undefined
}

export interface LeadEvent { text: string; meta: string }

export interface Lead {
  id: number
  name: string
  /** Synthetic discovery data only — never an identity or join key (AGENTS.md Identity). */
  phone: string
  city: string
  province: string
  source: LeadSource
  assigned: string
  assignedAgo: string
  /** Days since assignment; drives the local date filter only. */
  assignedDays: number
  status: LeadStatus
  attempts: number
  prob: LeadProbability
  note: string
  next: LeadNext | null
  lastActivity: string
  history: LeadEvent[]
  /**
   * Explicit synthetic case→invoice references. The prototype matched invoices by phone; the
   * preview never does (AGENTS.md: phone is discovery data, never a join).
   */
  invoiceRefs?: string[] | undefined
}

export type LeadOutcome = 'no_answer' | 'callback' | 'not_purchased' | 'duplicate' | 'success'

/* ---------- Invoices ---------- */

export type InvoiceStatus = 'pre_invoice' | 'staged' | 'finance_review' | 'approved' | 'rejected' | 'cancelled'
export type InvoiceQueue = InvoiceStatus | 'all'

/**
 * Stage presentation state. A submitted payment (seller sent a receipt) is NOT a financially
 * confirmed payment: only `confirmed` counts as paid. `unknown` = the submission result never
 * came back (Outcome Unknown) and must be reconciled before any resend.
 */
export type StageState = 'confirmed' | 'submitted' | 'rejected' | 'due' | 'upcoming' | 'unknown' | 'void'

export interface InvoiceStage { n: number; amount: number; state: StageState; note?: string | undefined }

export type InvoiceNextAction = 'send_link' | 'register_payment' | 'resend_finance' | null

export interface InvoiceEvent { text: string; meta: string }

export interface Invoice {
  code: string
  customer: string
  phone: string
  product: string
  status: InvoiceStatus
  stages: InvoiceStage[]
  next: InvoiceNextAction
  rejectReason?: string | undefined
  history: InvoiceEvent[]
  /** Unit of every stage amount of this invoice; absent = IRT (the RM-E03 fixtures). */
  unit?: MoneyUnit | undefined
}

/* ---------- Conversions ---------- */

export type ConversionQueue = 'all' | 'follow' | 'link' | 'done'
export type ConversionStage = 'new' | 'follow' | 'link' | 'done'

export interface Conversion {
  id: number
  invoice: string
  customer: string
  phone: string
  option: string
  /** null = remaining amount unknown (never rendered as 0). */
  remaining: number | null
  stage: ConversionStage
  stageLabel: string
  result: string
  resultTone: Tone
  resultIcon: DsIconName
  lastActivity: string
  history: LeadEvent[]
}

/* ---------- Repeat action ---------- */

export interface RepeatItem {
  id: number
  invoice: string
  customer: string
  phone: string
  stageLabel: string
  amount: number
  /** Original seller (issuer) — historical attribution, never rewritten by the handoff. */
  originalSeller: string
  /** Current custody of the next stage. */
  currentCustody: string
  nextActor: string
  due: string
  dueTone: 'due' | 'warn' | ''
  done: boolean
  history: LeadEvent[]
}

/* ---------- Behavior ---------- */

export interface BehaviorItem {
  id: string
  name: string
  phone: string
  invoice: string
  last: string
  when: string
  /** null = event count not available (rendered «—», never 0). */
  count: number | null
  hot: boolean
  events: LeadEvent[]
}

/* ---------- Extra-number request ---------- */

export interface ExtraRequest {
  id: string
  phone: string
  name: string
  when: string
  status: 'pending' | 'approved' | 'rejected'
  timeline: LeadEvent[]
}

/* ---------- Wallet ---------- */

/** Posting state of a wallet line. Only `confirmed` is counted in confirmed totals. */
export type WalletTxState = 'confirmed' | 'pending' | 'unknown'
export type WalletChannel = 'online' | 'card'

export interface WalletTx {
  id: string
  date: string
  channel: WalletChannel
  /** null = amount not known yet (never 0). */
  amount: number | null
  /** Sale (invoice) the commission line refers to; the sale amount is not the commission amount. */
  invoice: string
  rule: string
  state: WalletTxState
  /** Absent = IRT. */
  unit?: MoneyUnit | undefined
}

export interface WalletFixture {
  lastCredit: string
  tx: WalletTx[]
}
