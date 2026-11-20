/*
 * RM-E04K — Seller S1 read boundary (DEV/TEST-only preview).
 *
 * The S1 screen reads its data only through `S1ReadSource`. Today the one implementation is the
 * deterministic SYNTHETIC source below, built over the RM-E03 fixtures. A later, separately authorized
 * PR may add a transport-backed source behind the same interface without rewriting the screen.
 *
 * What this is NOT:
 * - not a transport DTO / OpenAPI contract: the shapes here are PREVIEW PRESENTATION TYPES. The proposed
 *   Seller operations (`listMyCases`, `getMyCase`, `listMyCaseHistory`, `getMyCaseContactOutcomes`,
 *   RM-E04C API proposal) are NOT approved and are not called, mirrored or implemented here;
 * - not an authorization layer: the synthetic source never decides scope from a role label, a phone, a
 *   name or the hierarchy. Which rows it returns is the fixture set; which answer it gives is the preview
 *   scenario (`?sim=`). Real row scope can only be established by a future, approved backend.
 *
 * The synthetic source never fetches, never touches browser storage and never writes.
 */
import { INVOICES, LEADS } from '../fixtures/sellerFixtures'
import { STATE_POLICY, SYNTHETIC_ANSWERS, answerOfSim, classifyAnswer, type SellerUiState, type ServerAnswer, type Sim } from '../state/uiState'
import type { Invoice, Lead, LeadEvent } from '../types'

/* ---------- Preview presentation types (NOT authoritative transport DTOs) ---------- */

/** One read's result: the answer as the envelope / HTTP layer would expose it, its A7 state, and the data it may show. */
export interface S1Read<T> {
  answer: ServerAnswer
  /** A7 presentation state of `answer` (`classifyAnswer`). */
  state: SellerUiState
  /** null whenever the state may not show data (blocking, not found, failure). Never an empty stand-in. */
  data: T | null
}

/** List read. The synthetic source answers the whole synthetic set as one page (no cursor contract exists). */
export interface S1CaseList {
  rows: readonly Lead[]
  /** Synthetic demonstration only: the preview has no approved pagination contract, so there is never a next page. */
  nextPage: null
}

/** Detail read. Invoices come only from the explicit synthetic case→invoice references (never a phone join). */
export interface S1CaseDetail {
  record: Lead
  invoices: readonly Invoice[]
}

/** Contact / activity history read. `complete: false` = the answer said it is truncated (PARTIAL). */
export interface S1CaseHistory {
  caseId: string
  events: readonly LeadEvent[]
  complete: boolean
}

/**
 * Contact-outcome history read. The fixtures hold no separate contact-outcome records, and deriving them
 * from the lead status would invent history, so the synthetic source answers `available: false`.
 */
export interface S1ContactOutcomes {
  caseId: string
  available: false
}

export interface S1ReadOptions {
  /** Aborting resolves nothing: the pending read rejects with an AbortError and its answer is never shown. */
  signal?: AbortSignal | undefined
}

/** The S1 read boundary. Every method is a safe, idempotent read; there is no write here by design. */
export interface S1ReadSource {
  /** Stable identity of this source (a new scenario = a new source = new reads). */
  readonly id: string
  listCases(opts?: S1ReadOptions): Promise<S1Read<S1CaseList>>
  getCase(caseId: string, opts?: S1ReadOptions): Promise<S1Read<S1CaseDetail>>
  listCaseHistory(caseId: string, opts?: S1ReadOptions): Promise<S1Read<S1CaseHistory>>
  listCaseContactOutcomes(caseId: string, opts?: S1ReadOptions): Promise<S1Read<S1ContactOutcomes>>
}

/** The answer may show data: anything blocking, not-found or failed carries none. */
export function mayShowData(state: SellerUiState): boolean {
  return !STATE_POLICY[state].blocksContent && state !== 'not_found'
}

/** Authorization / session answers: once seen, the screen must not show protected content again for this source. */
export function isAuthorizationBlock(state: SellerUiState): boolean {
  return state === 'action_denied' || state === 'row_scope_denied' || state === 'session_expired' || state === 'feature_disabled'
}

export function abortError(): Error {
  const e = new Error('The synthetic read was aborted')
  e.name = 'AbortError'
  return e
}

export function isAbortError(e: unknown): boolean {
  return e instanceof Error && e.name === 'AbortError'
}

/* ---------- Synthetic source ---------- */

/** Synthetic latency so the loading presentation is observable in the DEV preview; 0 under Vitest. */
export const SYNTHETIC_LATENCY_MS = import.meta.env.MODE === 'test' ? 0 : 450

/** Synthetic demonstration of truncation: a PARTIAL history answer carries only the newest event. */
const PARTIAL_HISTORY_EVENTS = 1

export interface SyntheticSourceOptions {
  /** The preview scenario (`?sim=`). It selects the answer; it is not a permission input. */
  scenario: Sim
  latencyMs?: number | undefined
  /** Injected for tests; defaults to the fixtures. */
  rows?: readonly Lead[] | undefined
}

let nextSourceId = 1

/**
 * Deterministic synthetic S1 source. DEV/TEST only (the module is reachable only from the DEV-guarded
 * /dev/seller route, and this factory refuses to run in a production build).
 */
export function createSyntheticS1Source({ scenario, latencyMs = SYNTHETIC_LATENCY_MS, rows = LEADS }: SyntheticSourceOptions): S1ReadSource {
  if (!import.meta.env.DEV) throw new Error('The synthetic Seller read source is DEV/TEST only.')
  const id = `syn-${scenario}-${nextSourceId++}`
  // `not_found` is a detail-only scenario: the list itself answers normally.
  const listAnswer = scenario === 'not_found' ? SYNTHETIC_ANSWERS.success : answerOfSim(scenario)
  const fieldDenied = scenario === 'field_denied'

  /** Settles after the synthetic latency; `loading` never settles (until aborted); `offline` is a network failure. */
  function respond<T>(answer: ServerAnswer, data: () => T, opts: S1ReadOptions | undefined): Promise<S1Read<T>> {
    return new Promise((resolve, reject) => {
      const signal = opts?.signal
      if (signal?.aborted) { reject(abortError()); return }
      const final: ServerAnswer = scenario === 'offline' ? { status: 0 } : answer
      // `loading` is the pending demo: no answer ever arrives (until the read is aborted).
      const timer = scenario === 'loading' ? undefined : setTimeout(() => {
        signal?.removeEventListener('abort', onAbort)
        const state = classifyAnswer(final)
        resolve({ answer: final, state, data: mayShowData(state) ? data() : null })
      }, latencyMs)
      function onAbort() { if (timer !== undefined) clearTimeout(timer); reject(abortError()) }
      signal?.addEventListener('abort', onAbort, { once: true })
    })
  }

  /** FIELD_DENIED: the answer omits the identity fields; they are blanked here, never partially kept. */
  const shape = (l: Lead): Lead => (fieldDenied ? { ...l, name: '', phone: '' } : l)
  const shapeInvoice = (i: Invoice): Invoice => (fieldDenied ? { ...i, customer: '', phone: '' } : i)
  const visible = (): readonly Lead[] => (scenario === 'empty' ? [] : rows)
  /** Missing and out-of-scope look the same: one generic 404, no hint which. */
  const notFound: ServerAnswer = { status: 404, correlationId: 'syn-corr-e04k-404' }
  const blocking = STATE_POLICY[classifyAnswer(listAnswer)].blocksContent

  /** The answer for a single-record read: the list's blocking/transport answer, else 404 or the record answer. */
  function recordAnswer(caseId: string, ok: ServerAnswer): { answer: ServerAnswer; record: Lead | undefined } {
    if (blocking) return { answer: listAnswer, record: undefined }
    const record = scenario === 'not_found' ? undefined : visible().find(l => String(l.id) === caseId)
    return record ? { answer: ok, record } : { answer: notFound, record: undefined }
  }

  const success: ServerAnswer = { status: 200, outcome: 'SUCCESS', reasons: fieldDenied ? ['FIELD_DENIED'] : [] }

  return {
    id,
    listCases(opts) {
      const items = visible()
      const answer: ServerAnswer = blocking ? listAnswer : { ...listAnswer, itemCount: items.length }
      return respond(answer, () => ({ rows: items.map(shape), nextPage: null }), opts)
    },
    getCase(caseId, opts) {
      const r = recordAnswer(caseId, success)
      const record = r.record
      return respond(r.answer, () => {
        if (!record) throw new Error('unreachable: no record for a data-bearing answer')
        const refs = record.invoiceRefs ?? []
        return { record: shape(record), invoices: INVOICES.filter(i => refs.includes(i.code)).map(shapeInvoice) }
      }, opts)
    },
    listCaseHistory(caseId, opts) {
      const partial = scenario === 'partial'
      const r = recordAnswer(caseId, partial ? { status: 200, outcome: 'PARTIAL', reasons: ['COVERAGE_INCOMPLETE'] } : success)
      const record = r.record
      return respond(r.answer, () => {
        if (!record) throw new Error('unreachable: no record for a data-bearing answer')
        return { caseId, events: partial ? record.history.slice(0, PARTIAL_HISTORY_EVENTS) : record.history, complete: !partial }
      }, opts)
    },
    listCaseContactOutcomes(caseId, opts) {
      // No synthetic contact-outcome records exist: answer UNKNOWN with no data, never an invented history.
      const r = recordAnswer(caseId, { status: 200, outcome: 'UNKNOWN', reasons: ['OUTCOME_UNKNOWN'] })
      return respond(r.answer, () => ({ caseId, available: false as const }), opts)
    },
  }
}

/* ---------- Request ordering ---------- */

/**
 * Request-generation guard. `begin()` aborts the previous read and starts a new generation; only the
 * newest, un-aborted generation may apply its result. `cancel()` (close / unmount) makes every pending
 * read stale. This is what keeps a late answer for case A, an earlier query or a pre-denial success from
 * ever reaching the screen.
 */
export class ReadSequencer {
  private generation = 0
  private controller: AbortController | null = null

  begin(): { generation: number; signal: AbortSignal } {
    this.controller?.abort()
    this.controller = new AbortController()
    this.generation += 1
    return { generation: this.generation, signal: this.controller.signal }
  }

  isCurrent(generation: number): boolean {
    return generation === this.generation && this.controller !== null && !this.controller.signal.aborted
  }

  cancel(): void {
    this.controller?.abort()
    this.controller = null
    this.generation += 1
  }
}
