/*
 * RM-E04J — S1 contact-outcome draft (DEV/TEST preview only).
 *
 * A small, typed, frontend-only model of the seller's contact-outcome draft and its preview steps:
 *
 *   edit ──(validateOutcome ok)──▶ review ──(confirm preview)──▶ simulated
 *     ▲                              │                              │
 *     └────────── edit ──────────────┘                              │
 *   (discard / reset from any step → a clean edit step) ◀───────────┘
 *
 * The steps are PRESENTATION steps, not business statuses or workflow transitions. Nothing here is
 * sent, stored or persisted; a simulated result is never CRM history, has no server time, no event
 * id and no audit proof. Validation is the existing `validateOutcome` — no rule is duplicated here.
 * The real write (`cases.contact_outcome.record`) is NOT wired: Seller authority is blocked (D-42).
 */
import type { LeadOutcome } from '../types'
import { validateOutcome, type FieldErrors, type OutcomeFields } from './selectors'

/** Notices shown by the review and the simulated result. Both say «شبیه‌سازی» explicitly. */
export const SIMULATION_NOTICE = 'شبیه‌سازی — هیچ نتیجه‌ای در CRM ثبت یا ارسال نمی‌شود و وضعیت پرونده تغییر نمی‌کند.'
export const SIMULATED_RESULT_NOTICE = 'شبیه‌سازی انجام شد — چیزی در CRM ثبت نشد. این نتیجه فقط در همین پیش‌نمایش و تا بستن یا شروع دوباره دیده می‌شود.'

export type DraftStep = 'edit' | 'review' | 'simulated'
export type PayType = 'full' | 'staged'

/** Synthetic product choices of the existing S1 «موفق» form (unchanged labels). */
export const DRAFT_PRODUCTS = [
  { id: 'p1', label: 'اشتراک نمونه ۱۰ ستاره — ۱۰٬۰۰۰٬۰۰۰ تومان' },
  { id: 'p2', label: 'اشتراک نمونه نقره‌ای' },
] as const
export type DraftProduct = (typeof DRAFT_PRODUCTS)[number]['id']

export const PAY_TYPE_LABEL: Record<PayType, string> = { full: 'پرداخت کامل', staged: 'مرحله‌ای' }

export interface OutcomeDraft {
  step: DraftStep
  outcome: LeadOutcome | null
  fields: OutcomeFields
  product: DraftProduct
  payType: PayType
  errors: FieldErrors
}

export const EMPTY_FIELDS: OutcomeFields = { date: '۱۴۰۵/۰۷/۱۰', hour: '10', minute: '30', reason: '', amount: '' }

export function emptyDraft(outcome: LeadOutcome | null = null): OutcomeDraft {
  return { step: 'edit', outcome, fields: EMPTY_FIELDS, product: 'p1', payType: 'full', errors: {} }
}

export type DraftAction =
  | { type: 'pick'; outcome: LeadOutcome }
  | { type: 'field'; key: keyof OutcomeFields; value: string }
  | { type: 'product'; value: DraftProduct }
  | { type: 'payType'; value: PayType }
  /** Ask to review: runs the existing validation; only a valid draft reaches `review`. */
  | { type: 'review' }
  | { type: 'edit' }
  /** Confirm the PREVIEW (only from `review`): produces a simulated result, never a write. */
  | { type: 'confirm' }
  /** Discard the draft / reset the simulation: back to a clean edit step. */
  | { type: 'discard' }

export function draftReducer(d: OutcomeDraft, a: DraftAction): OutcomeDraft {
  switch (a.type) {
    case 'pick': return d.step === 'edit' ? { ...d, outcome: a.outcome, errors: {} } : d
    case 'field': return d.step === 'edit' ? { ...d, fields: { ...d.fields, [a.key]: a.value }, errors: { ...d.errors, [a.key]: undefined } } : d
    case 'product': return d.step === 'edit' ? { ...d, product: a.value } : d
    case 'payType': return d.step === 'edit' ? { ...d, payType: a.value } : d
    case 'review': {
      if (d.step !== 'edit' || !d.outcome) return d
      const errors = validateOutcome(d.outcome, d.fields)
      return Object.keys(errors).length ? { ...d, errors } : { ...d, errors: {}, step: 'review' }
    }
    case 'edit': return d.step === 'review' ? { ...d, step: 'edit' } : d
    // A second confirm (double click, Enter repeat) is a no-op: there is exactly one simulated result.
    case 'confirm': return d.step === 'review' ? { ...d, step: 'simulated' } : d
    case 'discard': return emptyDraft()
  }
}

export interface DraftLine { label: string; value: string }

/** The facts the review and the simulated result show: only what the seller entered in the existing form. */
export function draftLines(d: OutcomeDraft, ctx: { attemptsNext: number; fa: (s: string | number) => string }): DraftLine[] {
  const { fields: f, outcome } = d
  switch (outcome) {
    case 'no_answer': return [{ label: 'تلاش', value: `${ctx.fa(ctx.attemptsNext)} از ۳` }]
    case 'callback': return [
      { label: 'تاریخ تماس مجدد', value: f.date.trim() },
      { label: 'ساعت تهران', value: `${ctx.fa(f.hour.trim())}:${ctx.fa(f.minute.trim())}` },
      { label: 'یادآور', value: 'در پیش‌نمایش ساخته نمی‌شود' },
    ]
    case 'not_purchased': return [{ label: 'دلیل عدم خرید', value: f.reason.trim() }]
    case 'duplicate': return [{ label: 'ثبت‌کننده', value: 'شما (فقط در نسخه واقعی ذخیره می‌شود)' }]
    case 'success': return [
      { label: 'محصول', value: DRAFT_PRODUCTS.find(p => p.id === d.product)?.label ?? '—' },
      { label: 'نوع پرداخت', value: PAY_TYPE_LABEL[d.payType] },
      { label: 'پیش‌فاکتور', value: 'در پیش‌نمایش صادر یا پیامک نمی‌شود' },
    ]
    case null: return []
  }
}
