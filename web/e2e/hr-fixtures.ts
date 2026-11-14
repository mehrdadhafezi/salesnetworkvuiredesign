import type { APIResponse, Page } from '@playwright/test'
import { expect, guarded, required, sessionHeaders } from './fixtures'

/** A synthetic login provided by the orchestrator (password random per run). */
export interface HRLogin { login: string; password: string; userId: string }

/** Values of the R7-6 production-equivalent stack (TestBrowserHRFullStack). */
export interface HRStack {
  baseURL: string
  origin: string
  supervisor: HRLogin
  senior: HRLogin
  sellerInScope: string
  sellerOutOfScope: string
  targetSupervisor: string
  requestTerminate: string
  requestTransfer: string
  personId: string
}

export function hrStack(): HRStack {
  const baseURL = required('E2E_BASE_URL')
  return {
    baseURL,
    origin: new URL(baseURL).origin,
    supervisor: { login: required('E2E_HR_SUPERVISOR_LOGIN'), password: required('E2E_HR_SUPERVISOR_PASSWORD'), userId: required('E2E_HR_SUPERVISOR_USER_ID') },
    senior: { login: required('E2E_HR_SENIOR_LOGIN'), password: required('E2E_HR_SENIOR_PASSWORD'), userId: required('E2E_HR_SENIOR_USER_ID') },
    sellerInScope: required('E2E_HR_SELLER_IN_SCOPE'),
    sellerOutOfScope: required('E2E_HR_SELLER_OUT_OF_SCOPE'),
    targetSupervisor: required('E2E_HR_TARGET_SUPERVISOR'),
    requestTerminate: required('E2E_HR_REQUEST_TERMINATE'),
    requestTransfer: required('E2E_HR_REQUEST_TRANSFER'),
    personId: required('E2E_HR_PERSON_ID'),
  }
}

export const test = guarded.extend<{ hr: HRStack }>({
  hr: async ({ baseURL }, provide) => {
    const s = hrStack()
    expect(baseURL, 'baseURL comes from E2E_BASE_URL').toBe(s.baseURL)
    await provide(s)
  },
})

export { expect }

/** Page texts (Persian UI). */
export const T = {
  denied: 'دسترسی به این بخش ندارید',
  disabled: 'این بخش هنوز فعال نشده است',
  workforce: 'نیروها',
  structure: 'ساختار سازمانی',
  exceptions: 'استثناهای سلسله‌مراتب',
  requests: 'درخواست‌های جابجایی / قطع همکاری',
  request: 'درخواست منابع انسانی',
  requestsNav: 'درخواست‌های منابع انسانی',
  createForm: 'ثبت درخواست منابع انسانی',
  approve: 'تایید مرحله',
  reject: 'رد درخواست',
  logout: 'خروج',
  login: 'ورود',
}

/** A well-formed request id that names no request. */
export const unknownRequest = '00000000-0000-4000-8000-0000000000e6'

/** Every HR read the suite calls (organisation shadow + HR requests). */
export function hrReads(s: HRStack): string[] {
  return [
    '/api/v1/organization/persons',
    '/api/v1/organization/persons?q=Synthetic&limit=5',
    `/api/v1/organization/persons/${s.personId}`,
    '/api/v1/organization/positions',
    '/api/v1/organization/levels',
    '/api/v1/organization/org-units',
    '/api/v1/organization/assignments',
    '/api/v1/organization/hierarchy-findings',
  ]
}

export function requestReads(s: HRStack): string[] {
  return ['/api/v1/hr/requests', `/api/v1/hr/requests/${s.requestTerminate}`, `/api/v1/hr/requests/${s.requestTransfer}`,
    `/api/v1/hr/requests/${unknownRequest}`]
}

async function unsafeHeaders(page: Page, s: HRStack, csrf: string | null): Promise<Record<string, string>> {
  const h: Record<string, string> = { ...(await sessionHeaders(page)), Origin: s.origin }
  if (csrf !== null) h['X-CSRF-Token'] = csrf
  return h
}

export type CreateBody = { request_type: 'transfer_seller' | 'terminate_seller'; subject_user_id: string; reason: string; target_parent_user_id?: string }

export function terminate(subject: string): CreateBody {
  return { request_type: 'terminate_seller', subject_user_id: subject, reason: 'synthetic e2e reason' }
}

export function transfer(subject: string, target: string): CreateBody {
  return { request_type: 'transfer_seller', subject_user_id: subject, target_parent_user_id: target, reason: 'synthetic e2e reason' }
}

/** POST /api/v1/hr/requests with the browser's own session cookie. */
export async function createRequest(page: Page, s: HRStack, body: CreateBody, key: string, csrf: string | null): Promise<APIResponse> {
  return page.request.post('/api/v1/hr/requests', { headers: { ...(await unsafeHeaders(page, s, csrf)), 'Idempotency-Key': key }, data: body })
}

/** POST /api/v1/hr/requests/{id}/decisions. */
export async function decide(page: Page, s: HRStack, id: string, decision: string, key: string, csrf: string | null, ifMatch = '"1"'):
Promise<APIResponse> {
  return page.request.post(`/api/v1/hr/requests/${id}/decisions`, {
    headers: { ...(await unsafeHeaders(page, s, csrf)), 'Idempotency-Key': key, 'If-Match': ifMatch },
    data: { decision },
  })
}

/** A raw POST with exactly the given extra headers and body. */
export async function rawPost(page: Page, s: HRStack, path: string, csrf: string | null, headers: Record<string, string>, body: string):
Promise<APIResponse> {
  return page.request.post(path, { headers: { ...(await unsafeHeaders(page, s, csrf)), 'Content-Type': 'application/json', ...headers }, data: body })
}

/** Unique, well-formed idempotency key. */
export function idemKey(label: string): string {
  return `e2e-hr-${label}-${Date.now()}-${Math.floor(Math.random() * 1e6)}`
}

/** The read / command disabled contract: 200 BLOCKED / FEATURE_DISABLED. */
export async function expectFeatureDisabled(r: APIResponse, label: string) {
  expect(r.status(), label).toBe(200)
  const body = (await r.json()) as { outcome: string; reasons: string[] }
  expect(body.outcome, label).toBe('BLOCKED')
  expect(body.reasons, label).toEqual(['FEATURE_DISABLED'])
}

/** The contract check (400 problem+json, CONTRACT_INVALID). */
export async function expectContractInvalid(r: APIResponse, label: string) {
  expect(r.status(), label).toBe(400)
  expect(r.headers()['content-type'], label).toContain('application/problem+json')
  expect(((await r.json()) as { reason_code: string }).reason_code, label).toBe('CONTRACT_INVALID')
}

/** No create form, decision control or confirmation is offered. */
export async function expectNoRequestControls(page: Page) {
  await expect(page.getByRole('form', { name: T.createForm })).toHaveCount(0)
  await expect(page.getByRole('button', { name: T.approve })).toHaveCount(0)
  await expect(page.getByRole('button', { name: T.reject })).toHaveCount(0)
  await expect(page.getByRole('dialog')).toHaveCount(0)
}
