import type { APIResponse, Page } from '@playwright/test'
import { expect, guarded, required, sessionHeaders } from './fixtures'

/** Values provided by the Go orchestrator (TestBrowserPaymentsFullStack). */
export interface PaymentsStack {
  baseURL: string
  origin: string
  sellerX: { login: string; password: string; userId: string }
  sellerY: { login: string; password: string; userId: string }
  /** Synthetic invoice 3001, whose seller is seller X. */
  invoiceOfX: string
}

export function paymentsStack(): PaymentsStack {
  const baseURL = required('E2E_BASE_URL')
  return {
    baseURL,
    origin: new URL(baseURL).origin,
    sellerX: { login: required('E2E_SELLER_X_LOGIN'), password: required('E2E_SELLER_X_PASSWORD'), userId: required('E2E_SELLER_X_USER_ID') },
    sellerY: { login: required('E2E_SELLER_Y_LOGIN'), password: required('E2E_SELLER_Y_PASSWORD'), userId: required('E2E_SELLER_Y_USER_ID') },
    invoiceOfX: required('E2E_INVOICE_OF_X'),
  }
}

/** The R7-2 payments stack, behind the shared network guard. */
export const test = guarded.extend<{ pay: PaymentsStack }>({
  pay: async ({ baseURL }, provide) => {
    const s = paymentsStack()
    expect(baseURL, 'baseURL comes from E2E_BASE_URL').toBe(s.baseURL)
    await provide(s)
  },
})

export { expect }

/** A small synthetic PDF receipt body (signature + marker text; no real document). */
export function syntheticReceipt(n: number): string {
  return `%PDF-1.4\n% synthetic e2e receipt ${n}\n`
}

/** A well-formed but never-staged receipt reference. */
export const syntheticReceiptRef = { upload_id: '00000000-0000-4000-8000-0000000000e2', content_sha256: 'e2'.repeat(32) }

/** Headers of an unsafe request made with the browser's own session. */
async function unsafeHeaders(page: Page, s: PaymentsStack, csrf: string | null): Promise<Record<string, string>> {
  const h: Record<string, string> = { ...(await sessionHeaders(page)), Origin: s.origin }
  if (csrf !== null) h['X-CSRF-Token'] = csrf
  return h
}

/** Stages one receipt file (POST /api/v1/payments/uploads). csrf null omits the token. */
export async function stageUpload(page: Page, s: PaymentsStack, body: string, csrf: string | null): Promise<APIResponse> {
  return page.request.post('/api/v1/payments/uploads', {
    headers: { ...(await unsafeHeaders(page, s, csrf)), 'Content-Type': 'application/octet-stream' },
    data: body,
  })
}

/** Submits one receipt-mode batch for the invoice's stage 1. */
export async function submitBatch(page: Page, s: PaymentsStack, invoiceId: string, key: string, csrf: string, version = '0'): Promise<APIResponse> {
  return page.request.post(`/api/v1/payments/invoices/${invoiceId}/batches`, {
    headers: { ...(await unsafeHeaders(page, s, csrf)), 'Idempotency-Key': key, 'If-Match': version },
    data: { stage_no: 1, entry_mode: 'RECEIPT', items: [{ item_key: 'e2e-item-1', amount: 400, unit: 'IRT', receipts: [syntheticReceiptRef] }] },
  })
}

/** Every GET this suite reads for one invoice (billing read + payments write surfaces). */
export function invoiceReads(invoiceId: string): string[] {
  return [
    `/api/v1/invoices/${invoiceId}`,
    `/api/v1/invoices/${invoiceId}/items`,
    `/api/v1/invoices/${invoiceId}/stages`,
    `/api/v1/invoices/${invoiceId}/payments`,
    `/api/v1/invoices/${invoiceId}/receipts`,
    `/api/v1/invoices/${invoiceId}/totals`,
    `/api/v1/payments/invoices/${invoiceId}/submission`,
  ]
}

/** Collection reads of the billing, payments and finance surfaces. */
export const collectionReads = [
  '/api/v1/invoices',
  '/api/v1/payments/attempts',
  '/api/v1/payments/evidence',
  '/api/v1/payments/batches/00000000-0000-4000-8000-0000000000e3',
  '/api/v1/finance/review/attempts',
]

/** The disabled contract of every flag-gated surface: 200, BLOCKED / FEATURE_DISABLED. */
export async function expectFeatureDisabled(r: APIResponse) {
  expect(r.status()).toBe(200)
  const body = (await r.json()) as { outcome: string; reasons: string[] }
  expect(body.outcome).toBe('BLOCKED')
  expect(body.reasons).toEqual(['FEATURE_DISABLED'])
}
