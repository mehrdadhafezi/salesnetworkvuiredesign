import { csrfToken, expectActionDenied, loginUI, sessionGet, sessionHeaders, type Envelope } from './fixtures'
import {
  createRequest, decide, expect, expectContractInvalid, expectFeatureDisabled, expectNoRequestControls, hrReads, idemKey, rawPost,
  requestReads, T, terminate, test, transfer, unknownRequest,
} from './hr-fixtures'

// R7-6 class A — PRODUCTION-EQUIVALENT: real browser → real SPA → real
// crm-api (R1 sessions, production subject resolver) → disposable PostgreSQL
// with the SYNTHETIC Slice 11 organisation and two pending HR requests the
// orchestrator created through the domain service before the browser ran.
//
// What the shipped product does today: supervisor A holds sellers in scope
// and the senior is the recorded reviewer of A's request, but no role bundle
// carries an HR read or HR request capability and no ACTIVE grant exists
// (D-42; D-34 and R5-SEC-01 are open). Every HR read and command is refused
// at the capability gate (403 UNAUTHORIZED / ACTION_DENIED) before scope,
// hierarchy, version, idempotency or DEF-HR-01 rules run; those rules are
// proven by the backend suites. The orchestrator then proves no HR request,
// event, hierarchy, employment, identity grant or other module row changed.

const PAGES: [string, string][] = [['/hr', T.workforce], ['/hr/structure', T.structure], ['/hr/exceptions', T.exceptions]]

interface SessionBody {
  data: { authenticated: boolean; session?: { user_id: string } | null; features?: { hr_read: boolean; hr_request_write: boolean } }
}

test.describe('HR reads (flags ON in the disposable database)', () => {
  test('supervisor: navigation hints follow the flags only; every HR page shows the no-access state and no data', async ({ page, hr, blocked }) => {
    void blocked
    await loginUI(page, hr.supervisor.login, hr.supervisor.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.authenticated).toBe(true)
    expect(s.data.session?.user_id).toBe(hr.supervisor.userId)
    // Navigation hints follow the flags; they grant nothing.
    expect(s.data.features?.hr_read).toBe(true)
    expect(s.data.features?.hr_request_write).toBe(true)
    await page.getByRole('link', { name: T.workforce, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.workforce, exact: true })).toBeVisible()
    for (const [path, heading] of PAGES) {
      await page.goto(path)
      await expect(page.getByRole('heading', { name: heading, exact: true }), path).toBeVisible()
      await expect(page.getByText(T.denied).first(), path).toBeVisible()
      await expect(page.getByRole('table'), path).toHaveCount(0)
      await expect(page.locator('a[href^="/hr/people/"]'), path).toHaveCount(0)
    }
    await page.goto(`/hr/people/${hr.personId}`)
    await expect(page.getByText(T.denied).first()).toBeVisible()
    for (const path of hrReads(hr)) await expectActionDenied(await sessionGet(page, path))
  })

  test('filters and pagination parameters do not change the refusal', async ({ page, hr, blocked }) => {
    void blocked
    await loginUI(page, hr.supervisor.login, hr.supervisor.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    await page.goto('/hr?q=Synthetic&employment_status=ACTIVE&has_findings=true&sort=-display_name')
    await expect(page.getByText(T.denied).first()).toBeVisible()
    await expect(page.getByRole('button', { name: 'صفحه بعد' })).toHaveCount(0)
    for (const q of ['?limit=1', '?cursor=bm90LWEtY3Vyc29y', '?kind=CYCLE', '?severity=CONFLICT']) {
      const r = await sessionGet(page, `/api/v1/organization/${q.startsWith('?kind') || q.startsWith('?severity') ? 'hierarchy-findings' : 'persons'}${q}`)
      expect([400, 403], q).toContain(r.status())
      if (r.status() === 403) await expectActionDenied(r)
    }
  })
})

test.describe('HR requests (flags ON, CONTROLLED_WRITEBACK mode)', () => {
  test('the list, a seeded request, another reviewer\'s request and an unknown id all get the same refusal and no control', async ({ page, hr, blocked }) => {
    void blocked
    await loginUI(page, hr.senior.login, hr.senior.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    await page.getByRole('link', { name: T.requestsNav, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.requests, exact: true })).toBeVisible()
    await expect(page.getByText(T.denied).first()).toBeVisible()
    await expectNoRequestControls(page)
    for (const id of [hr.requestTerminate, hr.requestTransfer, unknownRequest]) {
      await page.goto(`/hr/requests/${id}`)
      await expect(page.getByRole('heading', { name: T.request, exact: true }), id).toBeVisible()
      await expect(page.getByText(T.denied).first(), id).toBeVisible()
      await expectNoRequestControls(page)
    }
    for (const path of requestReads(hr)) await expectActionDenied(await sessionGet(page, path))
  })

  test('create: in-scope, out-of-scope, ineligible destination and no-op transfer are refused identically before any rule', async ({ page, hr, blocked }) => {
    void blocked
    await loginUI(page, hr.supervisor.login, hr.supervisor.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const csrf = await csrfToken(page)
    const bodies = {
      'terminate in scope': terminate(hr.sellerInScope),
      'terminate out of scope': terminate(hr.sellerOutOfScope),
      'transfer to an eligible supervisor': transfer(hr.sellerInScope, hr.targetSupervisor),
      'transfer to a seller (ineligible destination)': transfer(hr.sellerInScope, hr.sellerOutOfScope),
      'transfer to the current parent (no-op)': transfer(hr.sellerInScope, hr.supervisor.userId),
      'request about oneself': terminate(hr.supervisor.userId),
    }
    for (const [label, body] of Object.entries(bodies)) {
      const r = await createRequest(page, hr, body, idemKey('create'), csrf)
      await expectActionDenied(r)
      expect(((await r.json()) as Envelope).outcome, label).toBe('UNAUTHORIZED')
    }
    // A retry with the same key and body is refused the same way (nothing was stored to replay).
    const key = idemKey('retry')
    await expectActionDenied(await createRequest(page, hr, terminate(hr.sellerInScope), key, csrf))
    await expectActionDenied(await createRequest(page, hr, terminate(hr.sellerInScope), key, csrf))
  })

  test('decide: the recorded reviewer, a wrong reviewer, a stale version, a replay and an unknown request all get the capability refusal', async ({ page, hr, blocked }) => {
    void blocked
    await loginUI(page, hr.senior.login, hr.senior.password) // the recorded reviewer of the terminate request
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const csrf = await csrfToken(page)
    await expectActionDenied(await decide(page, hr, hr.requestTerminate, 'approve', idemKey('approve'), csrf))
    await expectActionDenied(await decide(page, hr, hr.requestTerminate, 'reject', idemKey('reject'), csrf))
    await expectActionDenied(await decide(page, hr, hr.requestTransfer, 'approve', idemKey('wrong-reviewer'), csrf))
    await expectActionDenied(await decide(page, hr, hr.requestTerminate, 'approve', idemKey('stale'), csrf, '"7"'))
    await expectActionDenied(await decide(page, hr, unknownRequest, 'approve', idemKey('unknown'), csrf))
    const key = idemKey('replay')
    await expectActionDenied(await decide(page, hr, hr.requestTerminate, 'approve', key, csrf))
    await expectActionDenied(await decide(page, hr, hr.requestTerminate, 'approve', key, csrf))
    // The requester cannot decide either.
    await page.getByRole('button', { name: T.logout }).click()
    await expect(page.getByRole('button', { name: T.login, exact: true })).toBeVisible()
    await loginUI(page, hr.supervisor.login, hr.supervisor.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    await expectActionDenied(await decide(page, hr, hr.requestTerminate, 'approve', idemKey('requester'), await csrfToken(page)))
  })

  test('CSRF and contract checks run first; anonymous callers are refused', async ({ page, hr, blocked }) => {
    void blocked
    // Anonymous: no session cookie, no ambient authority.
    for (const path of [...hrReads(hr), ...requestReads(hr)]) await expectActionDenied(await page.request.get(path))
    await expectActionDenied(await createRequest(page, hr, terminate(hr.sellerInScope), idemKey('anon'), null))
    await expectActionDenied(await decide(page, hr, hr.requestTerminate, 'approve', idemKey('anon-decide'), null))

    await loginUI(page, hr.supervisor.login, hr.supervisor.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    // With a session but no CSRF token: refused by the cross-site guard (problem+json).
    for (const r of [
      await createRequest(page, hr, terminate(hr.sellerInScope), idemKey('no-csrf'), null),
      await decide(page, hr, hr.requestTerminate, 'approve', idemKey('no-csrf-decide'), null),
    ]) {
      expect(r.status()).toBe(403)
      expect(r.headers()['content-type']).toContain('application/problem+json')
      expect(((await r.json()) as { reason_code: string }).reason_code).toBe('ACTION_DENIED')
    }
    const csrf = await csrfToken(page)
    const valid = JSON.stringify(terminate(hr.sellerInScope))
    await expectContractInvalid(await rawPost(page, hr, '/api/v1/hr/requests', csrf, {}, valid), 'create without Idempotency-Key')
    await expectContractInvalid(await rawPost(page, hr, '/api/v1/hr/requests', csrf, { 'Idempotency-Key': idemKey('arr') }, '[]'), 'create body is an array')
    await expectContractInvalid(await rawPost(page, hr, '/api/v1/hr/requests', csrf, { 'Idempotency-Key': idemKey('extra') },
      JSON.stringify({ ...terminate(hr.sellerInScope), status: 'approved' })), 'create with a field outside the command')
    await expectContractInvalid(await rawPost(page, hr, `/api/v1/hr/requests/${hr.requestTerminate}/decisions`, csrf,
      { 'Idempotency-Key': idemKey('apply') }, JSON.stringify({ decision: 'apply' })), 'decision "apply" is not a contract value')
    await expectContractInvalid(await rawPost(page, hr, '/api/v1/hr/requests/not-a-uuid/decisions', csrf,
      { 'Idempotency-Key': idemKey('bad-id'), 'If-Match': '"1"' }, JSON.stringify({ decision: 'approve' })), 'malformed request id')
  })

  test('logout invalidates the session; the old cookie is refused everywhere', async ({ page, browser, hr, blocked }) => {
    void blocked
    await loginUI(page, hr.senior.login, hr.senior.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const old = (await page.context().cookies()).find(c => c.name === '__Host-crm_session')
    expect(old, 'the real login set the session cookie').toBeTruthy()
    const csrf = await csrfToken(page)
    await page.getByRole('button', { name: T.logout }).click()
    await expect(page.getByRole('button', { name: T.login, exact: true })).toBeVisible()
    const replay = await browser.newContext({ baseURL: hr.baseURL })
    const cookie = { Cookie: `${old?.name}=${old?.value}` }
    const s = (await (await replay.request.get('/api/v1/session', { headers: cookie })).json()) as SessionBody
    expect(s.data.authenticated).toBe(false)
    for (const path of [...hrReads(hr), ...requestReads(hr)]) await expectActionDenied(await replay.request.get(path, { headers: cookie }))
    await expectActionDenied(await replay.request.post(`/api/v1/hr/requests/${hr.requestTerminate}/decisions`, {
      headers: { ...cookie, 'X-CSRF-Token': csrf, Origin: hr.origin, 'Idempotency-Key': idemKey('after-logout'), 'If-Match': '"1"' },
      data: { decision: 'approve' },
    }))
    await replay.close()
  })

  test('a session past its absolute lifetime is refused by the server and the open page returns to login', async ({ page, browser, hr, blocked }) => {
    void blocked
    // crm-api runs with CRM_AUTH_SESSION_IDLE / _ABSOLUTE = 1m; no session row is edited.
    test.setTimeout(150_000)
    await loginUI(page, hr.senior.login, hr.senior.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const loggedInAt = Date.now()
    const cookie = (await page.context().cookies()).find(c => c.name === '__Host-crm_session')
    expect(cookie).toBeTruthy()
    await page.goto(`/hr/requests/${hr.requestTerminate}`)
    await expect(page.getByRole('heading', { name: T.request, exact: true })).toBeVisible()
    const probe = await browser.newContext({ baseURL: hr.baseURL })
    const headers = { Cookie: `${cookie?.name}=${cookie?.value}` }
    await expect.poll(async () => {
      const s = (await (await probe.request.get('/api/v1/session', { headers })).json()) as SessionBody
      return s.data.authenticated
    }, { timeout: 120_000, intervals: [5_000] }).toBe(false)
    expect(Date.now() - loggedInAt, 'expired by the configured lifetime, not earlier').toBeGreaterThanOrEqual(55_000)
    await expectActionDenied(await probe.request.get(`/api/v1/hr/requests/${hr.requestTerminate}`, { headers }))
    await probe.close()
    await page.reload()
    await expect(page.getByRole('button', { name: T.login, exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: T.logout })).toHaveCount(0)
    void sessionHeaders
  })
})

test.describe('HR flags OFF @flags-off', () => {
  test('reads and commands answer the disabled contract; the UI shows the not-enabled state and hides navigation', async ({ page, hr, blocked }) => {
    void blocked
    await loginUI(page, hr.senior.login, hr.senior.password)
    await expect(page.getByRole('button', { name: T.logout })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.features?.hr_read).toBe(false)
    expect(s.data.features?.hr_request_write).toBe(false)
    await expect(page.getByRole('link', { name: T.workforce, exact: true })).toHaveCount(0)
    await expect(page.getByRole('link', { name: T.requestsNav, exact: true })).toHaveCount(0)
    for (const [path, heading] of [...PAGES, ['/hr/requests', T.requests] as [string, string]]) {
      await page.goto(path)
      await expect(page.getByRole('heading', { name: heading, exact: true }), path).toBeVisible()
      await expect(page.getByText(T.disabled).first(), path).toBeVisible()
    }
    for (const path of [...hrReads(hr), ...requestReads(hr)]) await expectFeatureDisabled(await sessionGet(page, path), path)
    const csrf = await csrfToken(page)
    for (const r of [
      await createRequest(page, hr, terminate(hr.sellerInScope), idemKey('off-create'), csrf),
      await decide(page, hr, hr.requestTerminate, 'approve', idemKey('off-decide'), csrf),
    ]) {
      await expectFeatureDisabled(r, 'command')
      const c = ((await r.json()) as { command?: { applied: boolean; rollout_mode: string; command_id?: string } }).command
      expect(c?.applied).toBe(false)
      expect(c?.rollout_mode).toBe('TARGET_WRITE_DISABLED')
      expect(c?.command_id).toBeUndefined()
    }
  })
})
