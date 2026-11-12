import type { APIResponse, Page } from '@playwright/test'
import { expect, guarded, required, sessionHeaders } from './fixtures'

/** Values provided by the Go orchestrator (TestBrowserFulfillmentFullStack). */
export interface OpsStack {
  baseURL: string
  origin: string
  /** CURRENT supervisor and executor (catalog scope) of services A and P. */
  userX: { login: string; password: string; userId: string }
  /** CURRENT supervisor and executor of service B. */
  userY: { login: string; password: string; userId: string }
  /** QUEUED v1 case of service A (no provider). */
  caseA: string
  /** QUEUED v1 case of service B (out of X's scope). */
  caseB: string
  /** QUEUED v1 case of service P (BIAWIN provider; completion needs a CONFIRMED operation). */
  caseP: string
  /** PREPARED, never dispatched provider operation of case P. */
  operation: string
  serviceA: string
  serviceB: string
  serviceP: string
}

export function opsStack(): OpsStack {
  const baseURL = required('E2E_BASE_URL')
  return {
    baseURL,
    origin: new URL(baseURL).origin,
    userX: { login: required('E2E_SELLER_X_LOGIN'), password: required('E2E_SELLER_X_PASSWORD'), userId: required('E2E_SELLER_X_USER_ID') },
    userY: { login: required('E2E_SELLER_Y_LOGIN'), password: required('E2E_SELLER_Y_PASSWORD'), userId: required('E2E_SELLER_Y_USER_ID') },
    caseA: required('E2E_WORK_CASE_A'),
    caseB: required('E2E_WORK_CASE_B'),
    caseP: required('E2E_WORK_CASE_PROVIDER'),
    operation: required('E2E_PROVIDER_OPERATION'),
    serviceA: required('E2E_WORK_SERVICE_A'),
    serviceB: required('E2E_WORK_SERVICE_B'),
    serviceP: required('E2E_WORK_SERVICE_P'),
  }
}

/** The R7-3 operations stack, behind the shared network guard. */
export const test = guarded.extend<{ ops: OpsStack }>({
  ops: async ({ baseURL }, provide) => {
    const s = opsStack()
    expect(baseURL, 'baseURL comes from E2E_BASE_URL').toBe(s.baseURL)
    await provide(s)
  },
})

export { expect }

/** A well-formed case id that no work case carries. */
export const unknownCase = '00000000-0000-4000-8000-0000000000f3'

/** A synthetic evidence fingerprint (no file exists or is uploaded). */
export const syntheticSha = 'f3'.repeat(32)

/** The four work-case commands with a well-formed body each. */
export type WorkCommand = 'assignment' | 'operational-actions' | 'evidence-records' | 'outcome'

export function commandBody(cmd: WorkCommand, assignee: string): Record<string, unknown> {
  switch (cmd) {
    case 'assignment': return { assignee_user_id: assignee }
    case 'operational-actions': return { action: 'NOTE', note: 'synthetic e2e note' }
    case 'evidence-records': return { evidence_type: 'service-form', label: 'synthetic e2e evidence', content_sha256: syntheticSha }
    case 'outcome': return { outcome: 'EXECUTED' }
  }
}

export const workCommands: WorkCommand[] = ['assignment', 'operational-actions', 'evidence-records', 'outcome']

/** Headers of an unsafe request made with the browser's own session. csrf null omits the token. */
async function unsafeHeaders(page: Page, s: OpsStack, csrf: string | null): Promise<Record<string, string>> {
  const h: Record<string, string> = { ...(await sessionHeaders(page)), Origin: s.origin }
  if (csrf !== null) h['X-CSRF-Token'] = csrf
  return h
}

/** Sends one work-case command with an Idempotency-Key and If-Match. */
export async function workCommand(page: Page, s: OpsStack, caseId: string, cmd: WorkCommand, key: string, csrf: string | null,
  ifMatch = '"1"', body: Record<string, unknown> = commandBody(cmd, s.userX.userId)): Promise<APIResponse> {
  return page.request.post(`/api/v1/fulfillment/work-cases/${caseId}/${cmd}`, {
    headers: { ...(await unsafeHeaders(page, s, csrf)), 'Idempotency-Key': key, 'If-Match': ifMatch },
    data: body,
  })
}

/** Records human reconciliation evidence for one provider operation. */
export async function resolveOperation(page: Page, s: OpsStack, operationId: string, key: string, csrf: string | null): Promise<APIResponse> {
  return page.request.post(`/api/v1/integration/provider-operations/${operationId}/resolution`, {
    headers: { ...(await unsafeHeaders(page, s, csrf)), 'Idempotency-Key': key, 'If-Match': '"1"' },
    data: { present: true, external_ref: 'synthetic-e2e-ref', evidence_note: 'synthetic e2e reconciliation evidence' },
  })
}

/** Every work-case and operations GET this suite reads for one case. */
export function caseReads(caseId: string): string[] {
  return [`/api/v1/fulfillment/work-cases/${caseId}`]
}

/** Collection reads of the Slice 14 operations surface. */
export function collectionReads(s: OpsStack): string[] {
  return [
    '/api/v1/fulfillment/work-cases',
    `/api/v1/fulfillment/work-cases?service_key=${s.serviceA}`,
    '/api/v1/fulfillment/work-cases?state=QUEUED&limit=1',
    '/api/v1/biawin/activation-decisions',
    '/api/v1/integration/provider-operations',
    '/api/v1/integration/provider-operations?state=PREPARED',
  ]
}

/** The disabled contract of every flag-gated surface: 200, BLOCKED / FEATURE_DISABLED. */
export async function expectFeatureDisabled(r: APIResponse) {
  expect(r.status()).toBe(200)
  const body = (await r.json()) as { outcome: string; reasons: string[] }
  expect(body.outcome).toBe('BLOCKED')
  expect(body.reasons).toEqual(['FEATURE_DISABLED'])
}

/** The disabled contract of a work-case command: not applied, TARGET_WRITE_DISABLED, nothing recorded. */
export async function expectCommandDisabled(r: APIResponse, label: string) {
  await expectFeatureDisabled(r)
  const c = ((await r.json()) as { command?: { applied: boolean; rollout_mode: string; blocked_by: string[]; command_id?: string; version?: number; replayed?: boolean } }).command
  expect(c, label).toBeDefined()
  expect(c?.applied, label).toBe(false)
  expect(c?.rollout_mode, label).toBe('TARGET_WRITE_DISABLED')
  expect(c?.blocked_by, label).toEqual([])
  expect(c?.command_id, label).toBeUndefined()
  expect(c?.version, label).toBeUndefined()
  expect(c?.replayed, label).toBe(false)
}

/**
 * Sends a work-case command with exactly the given extra headers and raw
 * body (the session cookie, Origin and, when given, the CSRF token are
 * added). Used for contract-invalid requests the typed helper cannot build.
 */
export async function rawCommand(page: Page, s: OpsStack, caseId: string, cmd: WorkCommand, csrf: string | null,
  headers: Record<string, string>, body: string): Promise<APIResponse> {
  return page.request.post(`/api/v1/fulfillment/work-cases/${caseId}/${cmd}`, {
    headers: { ...(await unsafeHeaders(page, s, csrf)), 'Content-Type': 'application/json', ...headers },
    data: body,
  })
}

/** The contract check (400 problem+json, CONTRACT_INVALID); returns the case-independent part of the body. */
export async function expectContractInvalid(r: APIResponse, label: string): Promise<string> {
  expect(r.status(), label).toBe(400)
  expect(r.headers()['content-type'], label).toContain('application/problem+json')
  const p = (await r.json()) as { type: string; title: string; status: number; reason_code: string; detail?: string }
  expect(p.reason_code, label).toBe('CONTRACT_INVALID')
  return JSON.stringify([p.type, p.title, p.status, p.reason_code, p.detail ?? null])
}

/** Unique, well-formed idempotency key (8–128 of [A-Za-z0-9_-]). */
export function idemKey(label: string): string {
  return `e2e-ops-${label}-${Date.now()}-${Math.floor(Math.random() * 1e6)}`
}
