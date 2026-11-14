import { csrfToken, loginUI, sessionGet, sessionHeaders } from './fixtures'
import { allReads, expect, expectDenied, expectDisabled, expectNoData, NAV, PAGES, readOnly, required, T, unknownId, type RowIds } from './reporting-biawin-fixtures'

// R7-7 class A — PRODUCTION-EQUIVALENT: real browser → real SPA → real
// crm-api (R1 sessions, production subject resolver) → disposable PostgreSQL
// seeded with the R7-5 integrated SYNTHETIC dataset (one reporting projection
// run, one Biawin import run with its parity check).
//
// What the shipped product does today: no role bundle carries a reporting /
// MIS or Biawin read capability and no ACTIVE grant exists (D-42; D-34 and
// R5-SEC-01 are open), so every read is refused at the capability gate
// (403 UNAUTHORIZED / ACTION_DENIED) although the data exists. The
// orchestrator then proves no row of any domain schema changed.

const ids: RowIds = {
  caseId: required('E2E_RB_CASE_ID'),
  invoiceId: required('E2E_RB_INVOICE_ID'),
  attemptId: required('E2E_RB_ATTEMPT_ID'),
  subscription: required('E2E_RB_SUBSCRIPTION'),
  reportingRun: required('E2E_RB_RP_RUN_ID'),
  biawinRun: required('E2E_RB_BW_RUN_ID'),
}
const userA = { login: required('E2E_RB_A_LOGIN'), password: required('E2E_RB_A_PASSWORD'), userId: required('E2E_RB_A_USER_ID') }
const userB = { login: required('E2E_RB_B_LOGIN'), password: required('E2E_RB_B_PASSWORD'), userId: required('E2E_RB_B_USER_ID') }
const baseURL = required('E2E_BASE_URL')
const origin = new URL(baseURL).origin

interface SessionBody {
  data: { authenticated: boolean; session?: { user_id: string } | null; features?: { reporting_read: boolean; biawin_fulfillment_read: boolean } }
}

const test = readOnly

test.describe('reporting / MIS and Biawin reads (flags ON in the disposable database)', () => {
  test('navigation hints follow the flags only; every page shows the no-access state and no data', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    await loginUI(page, userA.login, userA.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.authenticated).toBe(true)
    expect(s.data.session?.user_id).toBe(userA.userId)
    // The hints follow the flags; they grant nothing.
    expect(s.data.features?.reporting_read).toBe(true)
    expect(s.data.features?.biawin_fulfillment_read).toBe(true)
    for (const name of NAV) await expect(page.getByRole('link', { name, exact: true }), name).toBeVisible()
    await page.getByRole('link', { name: T.mis, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.mis, exact: true })).toBeVisible()
    for (const [path, heading] of [...PAGES, [`/biawin/subscriptions/${ids.subscription}`, T.subscription] as [string, string]]) {
      await page.goto(path)
      await expect(page.getByRole('heading', { name: heading, exact: true }), path).toBeVisible()
      await expectNoData(page, T.denied, path)
    }
    for (const path of allReads(ids)) await expectDenied(await sessionGet(page, path), path)
  })

  test('filters, pinned runs, row detail and pagination parameters do not change the refusal', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    await loginUI(page, userB.login, userB.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    for (const path of [
      `/reports/payments?state=ACCEPTED&run_id=${ids.reportingRun}&row=${ids.attemptId}`,
      `/reports/cases?state_mapping=UNKNOWN&cursor=bm90LWEtY3Vyc29y`,
      `/biawin/subscriptions?invoice_link_state=UNKNOWN&run_id=${ids.biawinRun}`,
      `/biawin/parity?result=UNKNOWN&run_id=${ids.biawinRun}`,
    ]) {
      await page.goto(path)
      await expectNoData(page, T.denied, path)
      await expect(page.getByRole('button', { name: T.nextPage }), path).toHaveCount(0)
    }
    for (const q of [
      `/api/v1/reports/payments?state=ACCEPTED&run_id=${ids.reportingRun}`, '/api/v1/reports/cases?limit=1', '/api/v1/reports/invoices?unit=IRT',
      `/api/v1/reports/cases/${unknownId}`, `/api/v1/biawin/subscriptions?state=ACTIVE&limit=1&run_id=${ids.biawinRun}`,
      `/api/v1/biawin/subscriptions/${unknownId}`, '/api/v1/biawin/parity?result=MATCHED', '/api/v1/mis/reconciliation?limit=1',
    ]) await expectDenied(await sessionGet(page, q), q)
  })

  test('no write exists on the read surfaces: unsafe methods are refused, with or without CSRF', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    // Anonymous: no session cookie, no ambient authority.
    for (const path of allReads(ids)) await expectDenied(await page.request.get(path), `anonymous ${path}`)
    await loginUI(page, userA.login, userA.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const session = await sessionHeaders(page)
    for (const path of ['/api/v1/biawin/subscriptions', `/api/v1/biawin/subscriptions/${ids.subscription}`, '/api/v1/reports/payments', '/api/v1/mis/overview']) {
      // Without a CSRF token: refused by the cross-site guard before routing.
      const noCsrf = await page.request.post(path, { headers: { ...session, Origin: origin, 'Content-Type': 'application/json' }, data: '{}' })
      expect(noCsrf.status(), `POST ${path} without CSRF`).toBe(403)
      expect(noCsrf.headers()['content-type'], path).toContain('application/problem+json')
      expect(((await noCsrf.json()) as { reason_code: string }).reason_code, path).toBe('ACTION_DENIED')
    }
    const csrf = await csrfToken(page)
    for (const [method, path] of [['POST', '/api/v1/biawin/subscriptions'], ['PUT', `/api/v1/biawin/subscriptions/${ids.subscription}`],
      ['DELETE', `/api/v1/reports/payments/${ids.attemptId}`], ['PATCH', '/api/v1/mis/overview']] as [string, string][]) {
      // With a valid session and CSRF token: no such operation exists.
      const r = await page.request.fetch(path, { method, headers: { ...session, Origin: origin, 'X-CSRF-Token': csrf, 'Content-Type': 'application/json' }, data: '{}' })
      expect([404, 405], `${method} ${path}`).toContain(r.status())
    }
  })

  test('logout invalidates the session; the old cookie is refused everywhere', async ({ page, browser, blocked, writes }) => {
    void blocked
    void writes
    await loginUI(page, userA.login, userA.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const old = (await page.context().cookies()).find(c => c.name === '__Host-crm_session')
    expect(old, 'the real login set the session cookie').toBeTruthy()
    await page.getByRole('button', { name: T.logout }).click()
    await expect(page.getByRole('button', { name: T.login, exact: true })).toBeVisible()
    await page.goto('/mis')
    await expect(page.getByRole('button', { name: T.login, exact: true })).toBeVisible()
    const replay = await browser.newContext({ baseURL })
    const cookie = { Cookie: `${old?.name}=${old?.value}` }
    const s = (await (await replay.request.get('/api/v1/session', { headers: cookie })).json()) as SessionBody
    expect(s.data.authenticated).toBe(false)
    for (const path of allReads(ids)) await expectDenied(await replay.request.get(path, { headers: cookie }), `after logout ${path}`)
    await replay.close()
  })

  test('a session past its absolute lifetime is refused by the server and the open page returns to login', async ({ page, browser, blocked, writes }) => {
    void blocked
    void writes
    // crm-api runs with CRM_AUTH_SESSION_IDLE / _ABSOLUTE = 1m; no session row is edited.
    test.setTimeout(150_000)
    await loginUI(page, userB.login, userB.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const loggedInAt = Date.now()
    const cookie = (await page.context().cookies()).find(c => c.name === '__Host-crm_session')
    expect(cookie).toBeTruthy()
    await page.goto('/biawin/parity')
    await expect(page.getByRole('heading', { name: T.parity, exact: true })).toBeVisible()
    const probe = await browser.newContext({ baseURL })
    const headers = { Cookie: `${cookie?.name}=${cookie?.value}` }
    await expect.poll(async () => {
      const s = (await (await probe.request.get('/api/v1/session', { headers })).json()) as SessionBody
      return s.data.authenticated
    }, { timeout: 120_000, intervals: [5_000] }).toBe(false)
    expect(Date.now() - loggedInAt, 'expired by the configured lifetime, not earlier').toBeGreaterThanOrEqual(55_000)
    await expectDenied(await probe.request.get('/api/v1/biawin/parity', { headers }), 'expired parity')
    await expectDenied(await probe.request.get('/api/v1/mis/overview', { headers }), 'expired overview')
    await probe.close()
    await page.reload()
    await expect(page.getByRole('button', { name: T.login, exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: T.logout })).toHaveCount(0)
  })
})

test.describe('reporting / MIS and Biawin flags OFF @flags-off', () => {
  test('reads answer the disabled contract; the UI shows the not-enabled state and hides navigation', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    await loginUI(page, userA.login, userA.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.features?.reporting_read).toBe(false)
    expect(s.data.features?.biawin_fulfillment_read).toBe(false)
    for (const name of NAV) await expect(page.getByRole('link', { name, exact: true }), name).toHaveCount(0)
    for (const [path, heading] of [...PAGES, [`/biawin/subscriptions/${ids.subscription}`, T.subscription] as [string, string]]) {
      await page.goto(path)
      await expect(page.getByRole('heading', { name: heading, exact: true }), path).toBeVisible()
      await expectNoData(page, T.disabled, path)
    }
    for (const path of allReads(ids)) await expectDisabled(await sessionGet(page, path), path)
  })
})
