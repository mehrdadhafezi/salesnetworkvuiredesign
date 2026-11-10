import { test as base, expect, type APIResponse, type Page } from '@playwright/test'

/** Values provided by the Go orchestrator for one disposable stack. */
export interface Stack {
  baseURL: string
  sellerX: { login: string; password: string; userId: string }
  sellerY: { login: string; password: string; userId: string }
  caseHeldByX: string
  caseHeldByY: string
}

function required(name: string): string {
  const v = process.env[name]
  if (!v) throw new Error(`${name} is not set: run the suite through the Go orchestrator (go test -tags e2e)`)
  return v
}

export function stack(): Stack {
  return {
    baseURL: required('E2E_BASE_URL'),
    sellerX: { login: required('E2E_SELLER_X_LOGIN'), password: required('E2E_SELLER_X_PASSWORD'), userId: required('E2E_SELLER_X_USER_ID') },
    sellerY: { login: required('E2E_SELLER_Y_LOGIN'), password: required('E2E_SELLER_Y_PASSWORD'), userId: required('E2E_SELLER_Y_USER_ID') },
    caseHeldByX: required('E2E_CASE_HELD_BY_X'),
    caseHeldByY: required('E2E_CASE_HELD_BY_Y'),
  }
}

function originOf(url: string): string {
  try {
    return new URL(url).origin
  } catch {
    return ''
  }
}

/**
 * Network guard: every request whose origin is not the local stack is
 * aborted and recorded, and the test fails if any was attempted.
 */
export const test = base.extend<{ stack: Stack; blocked: string[] }>({
  stack: async ({ baseURL }, provide) => {
    const s = stack()
    expect(baseURL, 'baseURL comes from E2E_BASE_URL').toBe(s.baseURL)
    await provide(s)
  },
  blocked: async ({ context, stack }, provide) => {
    const allowed = new URL(stack.baseURL).origin
    const blocked: string[] = []
    await context.route('**/*', async route => {
      const url = route.request().url()
      if (originOf(url) === allowed || url.startsWith('data:') || url.startsWith('blob:')) {
        await route.continue()
        return
      }
      blocked.push(url)
      await route.abort('blockedbyclient')
    })
    await provide(blocked)
    expect(blocked, 'unexpected non-local requests').toEqual([])
  },
})

export { expect }

/** Logs in through the real login form. */
export async function loginUI(page: Page, login: string, password: string) {
  await page.goto('/')
  await page.getByRole('textbox', { name: 'نام کاربری' }).fill(login)
  await page.getByLabel('گذرواژه').fill(password)
  await page.getByRole('button', { name: 'ورود', exact: true }).click()
}

/**
 * Headers carrying the browser's own session cookie. Playwright's request
 * context does not replay the Secure __Host- cookie over plain-HTTP loopback,
 * so the cookie the real login set in the browser is attached explicitly.
 * It is held in memory only and never logged.
 */
export async function sessionHeaders(page: Page): Promise<Record<string, string>> {
  const jar = await page.context().cookies()
  const c = jar.find(k => k.name === '__Host-crm_session')
  return c ? { Cookie: `${c.name}=${c.value}` } : {}
}

/** GET with the browser's session cookie. */
export async function sessionGet(page: Page, path: string): Promise<APIResponse> {
  return page.request.get(path, { headers: await sessionHeaders(page) })
}

/** The CSRF token of the browser's current session (identity only). */
export async function csrfToken(page: Page): Promise<string> {
  const r = await sessionGet(page, '/api/v1/session')
  expect(r.status()).toBe(200)
  const body = (await r.json()) as { data?: { session?: { csrf_token?: string } | null } }
  const token = body.data?.session?.csrf_token
  expect(token, 'an authenticated session carries a CSRF token').toBeTruthy()
  return token ?? ''
}

/** Calls the contact-outcome command with the browser's session cookie. */
export async function recordOutcome(page: Page, s: Stack, caseId: string, key: string, csrf: string): Promise<APIResponse> {
  return page.request.post(`/api/v1/cases/${caseId}/contact-outcomes`, {
    headers: { ...(await sessionHeaders(page)), 'Idempotency-Key': key, 'X-CSRF-Token': csrf, 'If-Match': '"1"', Origin: new URL(s.baseURL).origin },
    data: { outcome: 'NO_ANSWER' },
  })
}

/** The result envelope of a command or refusal. */
export interface Envelope {
  outcome: string
  reasons: string[]
  command?: { applied: boolean; rollout_mode: string; writeback_state: string }
}

/** Asserts the capability refusal of the production subject resolver (D-42). */
export async function expectActionDenied(r: APIResponse) {
  expect(r.status()).toBe(403)
  const body = (await r.json()) as Envelope
  expect(body.outcome).toBe('UNAUTHORIZED')
  expect(body.reasons).toEqual(['ACTION_DENIED'])
  expect(body.command).toBeUndefined()
}

