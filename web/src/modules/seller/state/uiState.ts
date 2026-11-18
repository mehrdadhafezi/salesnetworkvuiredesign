/*
 * RM-E04C / A7 — Seller UI state model (presentation only).
 *
 * One pure module that decides how a Seller screen presents a server answer. It follows the shared UI
 * state model of docs/security/rm-e04c-seller-screen-designs.md §0 and uses only the existing contract
 * vocabulary (`Outcome`, `ReasonCode` from the generated OpenAPI types). It invents no business state,
 * reads no data and grants nothing: the server decides scope; this module only decides what the screen
 * may show for the answer it got.
 *
 * In the DEV/TEST-only preview the answers come from SYNTHETIC_ANSWERS (selected with `?sim=`). Nothing
 * here performs a request.
 */
import type { Outcome, ReasonCode } from '../../../shared/api/types'

export type { Outcome, ReasonCode }

/** The twelve A7 presentation states (screen-design §0). Not business statuses. */
export type SellerUiState =
  | 'success' | 'empty' | 'partial' | 'count_absent' | 'field_denied' | 'unknown'
  | 'feature_disabled' | 'action_denied' | 'row_scope_denied' | 'not_found' | 'session_expired' | 'failure'

export const SELLER_UI_STATES: readonly SellerUiState[] = [
  'success', 'empty', 'partial', 'count_absent', 'field_denied', 'unknown',
  'feature_disabled', 'action_denied', 'row_scope_denied', 'not_found', 'session_expired', 'failure',
]

/** A server answer as the envelope / HTTP layer would expose it. `status: 0` = network failure (no response). */
export interface ServerAnswer {
  status: number
  outcome?: Outcome | undefined
  reasons?: readonly ReasonCode[] | undefined
  correlationId?: string | undefined
  /** Items on the page, when the answer is a list. */
  itemCount?: number | undefined
  /** `count.state` of a scoped list (API proposal `ScopedCount`). */
  count?: 'EXACT' | 'ABSENT' | undefined
}

/**
 * Answer → presentation state. Precedence: transport / authorization first, then the envelope.
 * Anything unrecognised fails closed to `failure` (never to success or to "empty").
 */
export function classifyAnswer(a: ServerAnswer): SellerUiState {
  const has = (r: ReasonCode) => a.reasons?.includes(r) ?? false
  if (a.status === 401) return 'session_expired'
  if (a.status === 403) return has('ROW_SCOPE_DENIED') ? 'row_scope_denied' : 'action_denied'
  if (a.status === 404) return 'not_found'
  if (a.status !== 200) return 'failure'
  if (a.outcome === 'BLOCKED' && has('FEATURE_DISABLED')) return 'feature_disabled'
  if (a.outcome === 'UNAUTHORIZED') return has('ROW_SCOPE_DENIED') ? 'row_scope_denied' : 'action_denied'
  if (a.outcome === 'FAILED' || a.outcome === 'CONFLICT' || a.outcome === 'BLOCKED') return 'failure'
  if (a.outcome === 'PARTIAL' || has('COVERAGE_INCOMPLETE')) return 'partial'
  if (has('FIELD_DENIED')) return 'field_denied'
  if (a.outcome === 'UNKNOWN' || has('OUTCOME_UNKNOWN')) return 'unknown'
  if (a.count === 'ABSENT') return 'count_absent'
  if (a.itemCount === 0 && a.count === 'EXACT') return 'empty'
  if (a.outcome === 'SUCCESS') return 'success'
  return 'failure'
}

export interface StatePolicy {
  /** The screen body (lists, drawers, amounts) must not render at all: nothing stale or sensitive remains. */
  blocksContent: boolean
  /** A retry is offered. Only safe, idempotent reads are retried; only transport / server failures qualify. */
  retry: boolean
  /** Counts render «—» (never 0). */
  countsAbsent: boolean
  /** Amounts that depend on incomplete facts (remaining, payable, totals) are withheld («—»). */
  dependentAmountsWithheld: boolean
  /** Customer identity fields render «محدود» with no partial value. */
  fieldsRestricted: boolean
  /** Status / stage mappings render «نامعلوم». */
  statusUnknown: boolean
  /** Every detail lookup answers the generic «یافت نشد» (missing ≡ out of scope). */
  detailNotFound: boolean
}

const BASE: StatePolicy = {
  blocksContent: false, retry: false, countsAbsent: false, dependentAmountsWithheld: false,
  fieldsRestricted: false, statusUnknown: false, detailNotFound: false,
}

export const STATE_POLICY: Record<SellerUiState, StatePolicy> = {
  success: BASE,
  empty: BASE,
  partial: { ...BASE, dependentAmountsWithheld: true },
  count_absent: { ...BASE, countsAbsent: true },
  field_denied: { ...BASE, fieldsRestricted: true },
  unknown: { ...BASE, statusUnknown: true },
  feature_disabled: { ...BASE, blocksContent: true },
  action_denied: { ...BASE, blocksContent: true },
  row_scope_denied: { ...BASE, blocksContent: true },
  not_found: { ...BASE, detailNotFound: true },
  session_expired: { ...BASE, blocksContent: true },
  failure: { ...BASE, blocksContent: true, retry: true },
}

/** Synthetic answers used by the preview, one per state. Correlation ids are visibly synthetic. */
export const SYNTHETIC_ANSWERS: Record<SellerUiState, ServerAnswer> = {
  success: { status: 200, outcome: 'SUCCESS', reasons: [], itemCount: 10, count: 'EXACT' },
  empty: { status: 200, outcome: 'SUCCESS', reasons: [], itemCount: 0, count: 'EXACT' },
  partial: { status: 200, outcome: 'PARTIAL', reasons: ['COVERAGE_INCOMPLETE'], itemCount: 10, count: 'EXACT' },
  count_absent: { status: 200, outcome: 'SUCCESS', reasons: [], itemCount: 10, count: 'ABSENT' },
  field_denied: { status: 200, outcome: 'SUCCESS', reasons: ['FIELD_DENIED'], itemCount: 10, count: 'EXACT' },
  unknown: { status: 200, outcome: 'SUCCESS', reasons: ['OUTCOME_UNKNOWN'], itemCount: 10, count: 'EXACT' },
  feature_disabled: { status: 200, outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'] },
  action_denied: { status: 403, outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlationId: 'syn-corr-a7-403a' },
  row_scope_denied: { status: 403, outcome: 'UNAUTHORIZED', reasons: ['ROW_SCOPE_DENIED'], correlationId: 'syn-corr-a7-403r' },
  not_found: { status: 404, correlationId: 'syn-corr-a7-404' },
  session_expired: { status: 401 },
  failure: { status: 500, correlationId: 'syn-corr-a7-500' },
}

/* ---------- Preview simulator (`?sim=`) ---------- */

/**
 * Preview demo states. `none`, `loading`, `empty`, `error`, `offline` are the RM-E03 prototype states
 * and keep their URLs; the others are A7. `loading` / `offline` are transport presentations with no answer yet.
 */
export type Sim =
  | 'none' | 'loading' | 'empty' | 'error' | 'offline'
  | 'partial' | 'count_absent' | 'field_denied' | 'unknown'
  | 'feature_disabled' | 'action_denied' | 'row_scope_denied' | 'not_found' | 'session_expired'

export const SIMS: readonly Sim[] = [
  'none', 'loading', 'empty', 'error', 'offline', 'partial', 'count_absent', 'field_denied', 'unknown',
  'feature_disabled', 'action_denied', 'row_scope_denied', 'not_found', 'session_expired',
]

export const SIM_LABEL: Record<Sim, string> = {
  none: 'عادی', loading: 'در حال بارگذاری', empty: 'خالی', error: 'خطای سرور', offline: 'آفلاین',
  partial: 'داده ناقص (PARTIAL)', count_absent: 'تعداد نامعلوم', field_denied: 'فیلد محدود', unknown: 'وضعیت نامعلوم',
  feature_disabled: 'قابلیت خاموش', action_denied: 'بدون مجوز', row_scope_denied: 'هویت تأییدنشده',
  not_found: 'یافت نشد (جزئیات)', session_expired: 'نشست منقضی',
}

/** Preview state → the answer state it stands for. `loading` / `offline` have no answer and present as success-in-waiting. */
export function stateOfSim(sim: Sim): SellerUiState {
  switch (sim) {
    case 'none': case 'loading': case 'offline': return 'success'
    case 'error': return 'failure'
    default: return sim
  }
}

export function answerOfSim(sim: Sim): ServerAnswer {
  return SYNTHETIC_ANSWERS[stateOfSim(sim)]
}
