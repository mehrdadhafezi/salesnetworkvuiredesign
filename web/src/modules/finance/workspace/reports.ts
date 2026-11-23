import { useApi, type BillingReconciliationReport as BillingReport, type EvidenceDiagnostics } from '../../../shared/api'
import { settle, useRead, type ReadState } from '../../../shared/read/load'

export interface FinanceReports {
  /** GET /api/v1/billing/reconciliation-report — billing.reconciliation.read (flag billing_payments.read_shadow). */
  billing: ReadState<BillingReport>
  /** GET /api/v1/payments/evidence — payments.evidence.read (flag payments.manual_write). */
  evidence: ReadState<EvidenceDiagnostics>
}

/**
 * The two existing aggregates the workspace counters read. Each is one
 * request per page view, authorized on its own; a denied, disabled or
 * failed aggregate leaves its counters "unavailable", never zero.
 */
export function useFinanceReports(): FinanceReports {
  const api = useApi()
  const billing = useRead('fw-billing-report', async () => settle(await api.GET('/api/v1/billing/reconciliation-report'), e => e.data))
  const evidence = useRead('fw-payment-evidence', async () => settle(await api.GET('/api/v1/payments/evidence'), e => e.data))
  return { billing, evidence }
}

/**
 * A counter's source. 'billing' and 'evidence' name a field of an existing
 * aggregate; 'none' means no authoritative count exists in any contract
 * (shown as «در دسترس نیست» with the reason, never computed client-side).
 */
export type CountSource =
  | { kind: 'billing'; path: string; scope: string; get: (r: BillingReport) => number | null | undefined; note?: string }
  | { kind: 'evidence'; path: string; scope: string; get: (r: EvidenceDiagnostics) => number | null | undefined; note?: string }
  | { kind: 'none'; why: string }

const BILLING_SRC = 'GET /api/v1/billing/reconciliation-report'
const EVIDENCE_SRC = 'GET /api/v1/payments/evidence'

export function countOf(c: CountSource, r: FinanceReports): number | null {
  if (c.kind === 'billing') return r.billing.status === 'ok' ? (c.get(r.billing.data) ?? null) : null
  if (c.kind === 'evidence') return r.evidence.status === 'ok' ? (c.get(r.evidence.data) ?? null) : null
  return null
}

/** Why a counter has no value (denied / disabled / failed source, or no contract). */
export function unavailableWhy(c: CountSource, r: FinanceReports): string | undefined {
  if (c.kind === 'none') return c.why
  const s = c.kind === 'billing' ? r.billing : r.evidence
  const what = c.kind === 'billing' ? 'گزارش تطبیق صورت‌حساب' : 'شواهد پرداخت'
  switch (s.status) {
    case 'ok': return undefined
    case 'loading': return 'در حال دریافت'
    case 'denied': return `دسترسی به ${what} ندارید`
    case 'blocked': return `${what} فعال نشده است`
    case 'failed': return `دریافت ${what} ناموفق بود`
  }
}

export function sourceOf(c: CountSource): string {
  if (c.kind === 'none') return 'بدون منبع قراردادی'
  return `${c.kind === 'billing' ? BILLING_SRC : EVIDENCE_SRC} · ${c.path}`
}

/**
 * Map counts (CountMap) are sparse: a key absent from a successfully read
 * map means the aggregate counted none of that value (its own contract),
 * which is the only case a 0 is shown.
 */
export const key = (m: Record<string, number> | undefined, k: string) => (m ? (m[k] ?? 0) : undefined)

export const C = {
  invoicesPresent: { kind: 'billing', path: 'billing.invoices_present', scope: 'فاکتورهای حاضر در آخرین واردسازی', get: r => r.billing.invoices_present },
  invoicesWithoutItems: { kind: 'billing', path: 'billing.invoices_without_items', scope: 'فاکتورهای حاضر', get: r => r.billing.invoices_without_items },
  invoicesWithoutStages: { kind: 'billing', path: 'billing.invoices_without_stages', scope: 'فاکتورهای حاضر', get: r => r.billing.invoices_without_stages },
  billingUnrepresentable: { kind: 'billing', path: 'billing.unrepresentable_amounts', scope: 'شواهد مبلغ فاکتور', get: r => r.billing.unrepresentable_amounts },
  customerLink: (k: string) => ({ kind: 'billing', path: `billing.customer_link_states.${k}`, scope: 'فاکتورهای حاضر', get: (r: BillingReport) => key(r.billing.customer_link_states, k) }) as CountSource,
  caseLink: (k: string) => ({ kind: 'billing', path: `billing.case_link_states.${k}`, scope: 'فاکتورهای حاضر', get: (r: BillingReport) => key(r.billing.case_link_states, k) }) as CountSource,
  attemptsPresent: { kind: 'billing', path: 'payments.attempts_present', scope: 'تلاش‌های حاضر', get: r => r.payments.attempts_present },
  attemptsWithoutInvoice: { kind: 'billing', path: 'payments.attempts_without_invoice', scope: 'تلاش‌های حاضر', get: r => r.payments.attempts_without_invoice },
  attemptState: (k: string) => ({ kind: 'billing', path: `payments.attempt_states.${k}`, scope: 'تلاش‌های حاضر', get: (r: BillingReport) => key(r.payments.attempt_states, k) }) as CountSource,
  attemptMapping: (k: string) => ({ kind: 'billing', path: `payments.attempt_state_mapping.${k}`, scope: 'تلاش‌های حاضر', get: (r: BillingReport) => key(r.payments.attempt_state_mapping, k) }) as CountSource,
  attemptProvenance: (k: string) => ({ kind: 'billing', path: `payments.approval_provenance.${k}`, scope: 'تلاش‌های حاضر', get: (r: BillingReport) => key(r.payments.approval_provenance, k) }) as CountSource,
  attemptChannel: (k: string) => ({ kind: 'billing', path: `payments.attempts_by_channel.${k}`, scope: 'تلاش‌های حاضر', get: (r: BillingReport) => key(r.payments.attempts_by_channel, k) }) as CountSource,
  approvedRaw: { kind: 'billing', path: 'payments.approved_raw_count', scope: 'تلاش‌های حاضر', get: r => r.payments.approved_raw_count },
  regressions: { kind: 'billing', path: 'payments.attempt_state_regressions', scope: 'همه ردیف‌های سابقه (تعداد ردیف، نه تلاش)', get: r => r.payments.attempt_state_regressions,
    note: 'شمار ردیف‌های بازگشت در سابقه است؛ یک تلاش ممکن است چند ردیف داشته باشد.' },
  attemptsUnrepresentable: { kind: 'billing', path: 'payments.unrepresentable_amounts', scope: 'تلاش‌های حاضر', get: r => r.payments.unrepresentable_amounts },
  receiptsReferenced: { kind: 'billing', path: 'payments.receipts_referenced', scope: 'ارجاع‌های رسید حاضر', get: r => r.payments.receipts_referenced },
  receiptsUnknown: { kind: 'billing', path: 'payments.receipts_unknown', scope: 'ارجاع‌های رسید حاضر', get: r => r.payments.receipts_unknown },
  receiptsMissing: { kind: 'billing', path: 'payments.receipts_missing', scope: 'ارجاع‌های رسید حاضر', get: r => r.payments.receipts_missing },
  targetReceipt: (k: string) => ({ kind: 'evidence', path: `receipts_by_state.${k}`, scope: 'فیش‌های سامانه جدید (تعداد فیش، نه پرداخت)', get: (r: EvidenceDiagnostics) => key(r.receipts_by_state, k) }) as CountSource,
  reviewQueue: { kind: 'none', why: 'صف بررسی صفحه‌بندی کلیدی دارد و شمار کل برنمی‌گرداند؛ عددی ساخته نمی‌شود.' },
  absent: { kind: 'none', why: 'گزارش تطبیق فقط ردیف‌های حاضر را می‌شمارد؛ شمار غایب‌ها در هیچ قراردادی نیست.' },
  unknownInvoiceStatus: { kind: 'none', why: 'گزارش تطبیق مقادیر خام ناشناخته را به تفکیک فیلد می‌شمارد، نه تعداد فاکتور؛ جمع آن‌ها تعداد فاکتور نیست.' },
} satisfies Record<string, CountSource | ((k: string) => CountSource)>
