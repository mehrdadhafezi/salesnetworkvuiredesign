import { csrfToken, expectActionDenied, loginUI, sessionGet, sessionHeaders, type Envelope } from './fixtures'
import {
  caseReads, collectionReads, commandBody, expect, expectCommandDisabled, expectContractInvalid, expectFeatureDisabled, idemKey, rawCommand,
  resolveOperation, test, unknownCase, workCommand, workCommands,
} from './fulfillment-fixtures'

// R7-3: real browser → real SPA → real crm-api (R1 sessions, production
// subject resolver) → disposable PostgreSQL with SYNTHETIC work cases created
// through the production path (catalog recording → SHADOW activation → relay
// → fulfillment intake) and one PREPARED, never-dispatched provider operation.
//
// What the shipped product does today: user X is the CURRENT supervisor and
// executor (catalog scope) of services A and P, and Y of service B, but no
// role bundle carries a fulfillment capability and no ACTIVE grant exists
// (D-42; D-34 and R5-SEC-01 are open). Every work-case read and command,
// every provider-operation read and every reconciliation is therefore refused
// at the capability gate (403 UNAUTHORIZED / ACTION_DENIED), before row scope,
// state, version, idempotency, evidence or provider-confirmation rules run.
// The positive "assign → action → evidence → outcome" lifecycle is asserted
// as REFUSED here, never skipped and never reported as executed. The
// orchestrator then proves no fulfillment, integration, biawin, catalog,
// billing, payments or finance row changed.

const DENIED = 'دسترسی به این بخش ندارید'
const DISABLED = 'این بخش هنوز فعال نشده است'
const QUEUE = 'صف کار اجرای خدمت'
const CASE = 'پرونده اجرای خدمت'
const PROVIDERS = 'عملیات ارائه‌دهنده خارجی'
const ACTIVATIONS = 'تصمیم‌های فعال‌سازی اشتراک'
const NAV = [QUEUE, ACTIVATIONS, PROVIDERS]

interface SessionBody {
  data: { authenticated: boolean; session?: { user_id: string } | null; features?: { fulfillment_work: boolean } }
}

/** No command form, confirmation or submit control is offered on a case page. */
async function expectNoCommandControls(page: import('@playwright/test').Page) {
  for (const form of ['تخصیص پرونده', 'ثبت اقدام عملیاتی', 'ثبت مدرک', 'ثبت نتیجه']) {
    await expect(page.getByRole('form', { name: form })).toHaveCount(0)
  }
  await expect(page.getByRole('dialog', { name: 'تایید دستور' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'تایید و ارسال' })).toHaveCount(0)
}

test.describe('R1 login, navigation and session on the operations stack', () => {
  test('X logs in, sees the operations navigation hint, and logout invalidates the session', async ({ page, context, browser, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.authenticated).toBe(true)
    expect(s.data.session?.user_id).toBe(ops.userX.userId)
    // The navigation hint follows fulfillment.work_read only; it grants nothing.
    expect(s.data.features?.fulfillment_work).toBe(true)
    for (const label of NAV) await expect(page.getByRole('link', { name: label, exact: true })).toBeVisible()

    // Navigation reaches the queue through the real link.
    await page.getByRole('link', { name: QUEUE, exact: true }).click()
    await expect(page).toHaveURL(/\/operations\/work$/)
    await expect(page.getByRole('heading', { name: QUEUE, exact: true })).toBeVisible()
    await expect(page.getByText(DENIED).first()).toBeVisible()

    const old = (await context.cookies()).find(c => c.name === '__Host-crm_session')
    expect(old, 'the real login set the session cookie').toBeTruthy()
    await page.getByRole('button', { name: 'خروج' }).click()
    await expect(page.getByRole('button', { name: 'ورود', exact: true })).toBeVisible()

    // The old cookie no longer authenticates; anonymous callers are refused.
    const replay = await browser.newContext({ baseURL: ops.baseURL })
    const cookie = { Cookie: `${old?.name}=${old?.value}` }
    const after = (await (await replay.request.get('/api/v1/session', { headers: cookie })).json()) as SessionBody
    expect(after.data.authenticated).toBe(false)
    for (const path of ['/api/v1/fulfillment/work-cases', `/api/v1/fulfillment/work-cases/${ops.caseA}`, '/api/v1/integration/provider-operations']) {
      const r = await replay.request.get(path, { headers: cookie })
      expect(r.status(), path).toBe(403)
      const body = (await r.json()) as Envelope
      expect(body.outcome, path).toBe('UNAUTHORIZED')
      expect(body.reasons, path).toEqual(['ACTION_DENIED'])
    }
    await replay.close()
  })

  test('a session revoked server-side (expired for the browser) returns the open page to the login form', async ({ page, browser, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    await page.goto(`/operations/work/${ops.caseA}`)
    await expect(page.getByRole('heading', { name: CASE, exact: true })).toBeVisible()

    // End the session from another client (same cookie, its CSRF token), as
    // an expiry or a revocation would; the browser still holds the cookie.
    const csrf = await csrfToken(page)
    const other = await browser.newContext({ baseURL: ops.baseURL })
    const out = await other.request.post('/api/v1/auth/logout', {
      headers: { ...(await sessionHeaders(page)), 'X-CSRF-Token': csrf, Origin: ops.origin },
    })
    expect(out.status()).toBe(200)
    await other.close()

    await page.reload()
    await expect(page.getByRole('button', { name: 'ورود', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'خروج' })).toHaveCount(0)
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.authenticated).toBe(false)
    await expectActionDenied(await sessionGet(page, `/api/v1/fulfillment/work-cases/${ops.caseA}`))
  })

  test('anonymous callers are refused on every read and command', async ({ page, ops, blocked }) => {
    void blocked
    for (const path of [...collectionReads(ops), ...caseReads(ops.caseA)]) await expectActionDenied(await page.request.get(path))
    // Anonymous unsafe requests carry no ambient authority: the cross-site
    // guard leaves them to the handlers, which deny them at the capability gate.
    for (const cmd of workCommands) {
      await expectActionDenied(await workCommand(page, ops, ops.caseA, cmd, idemKey(`anon-${cmd}`), null))
    }
    await expectActionDenied(await resolveOperation(page, ops, ops.operation, idemKey('anon-resolve'), null))
  })
})

test.describe('work-case reads (flags ON in the disposable database)', () => {
  test('queue: the UI shows the no-access state and no row; every filter is refused the same way', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    await page.goto(`/operations/work?state=QUEUED&service_key=${ops.serviceA}`)
    await expect(page.getByRole('heading', { name: QUEUE, exact: true })).toBeVisible()
    await expect(page.getByText(DENIED).first()).toBeVisible()
    await expect(page.locator('a[href^="/operations/work/"]')).toHaveCount(0)
    await expect(page.getByRole('table')).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'صفحه بعد' })).toHaveCount(0)
    for (const path of collectionReads(ops)) await expectActionDenied(await sessionGet(page, path))
  })

  test('in-scope, out-of-scope and unknown cases get the same refusal: no data, no existence oracle, no command control', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    for (const id of [ops.caseA, ops.caseP, ops.caseB, unknownCase]) {
      await page.goto(`/operations/work/${id}`)
      await expect(page.getByRole('heading', { name: CASE, exact: true }), id).toBeVisible()
      await expect(page.getByText(DENIED).first(), id).toBeVisible()
      await expect(page.getByText(id)).toHaveCount(0)
      await expect(page.getByRole('list', { name: 'تاریخچه پرونده' })).toHaveCount(0)
      await expect(page.getByRole('list', { name: 'دستورهای مسدود' })).toHaveCount(0)
      await expectNoCommandControls(page)
      const r = await sessionGet(page, `/api/v1/fulfillment/work-cases/${id}`)
      await expectActionDenied(r)
      const body = (await r.json()) as { case?: unknown; history?: unknown; evidence?: unknown; blocked_commands?: unknown }
      expect(body.case).toBeUndefined()
      expect(body.history).toBeUndefined()
      expect(body.evidence).toBeUndefined()
      expect(body.blocked_commands).toBeUndefined()
    }
  })

  test("Y (supervisor and executor of service B) is refused on Y's own case too", async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userY.login, ops.userY.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    await page.goto(`/operations/work/${ops.caseB}`)
    await expect(page.getByText(DENIED).first()).toBeVisible()
    await expectNoCommandControls(page)
    await expectActionDenied(await sessionGet(page, `/api/v1/fulfillment/work-cases/${ops.caseB}`))
    await expectActionDenied(await sessionGet(page, `/api/v1/fulfillment/work-cases?service_key=${ops.serviceB}`))
  })
})

test.describe('work-case commands (flags ON, CONTROLLED mode)', () => {
  test('assign, action, evidence and outcome on X\'s in-scope case are refused at the capability gate', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    for (const cmd of workCommands) await expectActionDenied(await workCommand(page, ops, ops.caseA, cmd, idemKey(cmd), csrf))
    // NOT_EXECUTED with a reason is refused the same way.
    await expectActionDenied(await workCommand(page, ops, ops.caseA, 'outcome', idemKey('not-exec'), csrf, '"1"',
      { outcome: 'NOT_EXECUTED', reason: 'synthetic e2e reason' }))
  })

  test('completion of the provider-gated case is refused before the provider-confirmation rule', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    // The case's only provider operation is PREPARED (never dispatched, never
    // CONFIRMED); the PROVIDER_CONFIRMATION block is unreachable without a grant.
    await expectActionDenied(await workCommand(page, ops, ops.caseP, 'outcome', idemKey('prov-complete'), csrf))
    await expectActionDenied(await workCommand(page, ops, ops.caseP, 'evidence-records', idemKey('prov-evidence'), csrf))
  })

  test('out-of-scope assignment, unknown case and the other user are refused identically', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    let csrf = await csrfToken(page)
    // X assigning Y's case (X supervises only A and P) and X assigning Y as executor of A.
    await expectActionDenied(await workCommand(page, ops, ops.caseB, 'assignment', idemKey('oos-assign'), csrf))
    await expectActionDenied(await workCommand(page, ops, ops.caseA, 'assignment', idemKey('ineligible'), csrf, '"1"', { assignee_user_id: ops.userY.userId }))
    for (const cmd of workCommands) await expectActionDenied(await workCommand(page, ops, unknownCase, cmd, idemKey(`unknown-${cmd}`), csrf))
    await page.getByRole('button', { name: 'خروج' }).click()
    await expect(page.getByRole('button', { name: 'ورود', exact: true })).toBeVisible()

    await loginUI(page, ops.userY.login, ops.userY.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    csrf = await csrfToken(page)
    for (const cmd of workCommands) await expectActionDenied(await workCommand(page, ops, ops.caseA, cmd, idemKey(`other-${cmd}`), csrf))
  })

  test('replayed idempotency keys, a reused key with another body and a stale If-Match create nothing', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    const key = idemKey('replay')
    // Refused before the idempotency store and the version check: every call
    // answers the same refusal, and none is recorded.
    await expectActionDenied(await workCommand(page, ops, ops.caseA, 'operational-actions', key, csrf))
    await expectActionDenied(await workCommand(page, ops, ops.caseA, 'operational-actions', key, csrf))
    await expectActionDenied(await workCommand(page, ops, ops.caseA, 'operational-actions', key, csrf, '"1"', { action: 'NO_ANSWER' }))
    await expectActionDenied(await workCommand(page, ops, ops.caseA, 'operational-actions', idemKey('stale'), csrf, '"7"'))
    // The orchestrator asserts no command record, history, assignment or evidence row exists.
  })

  test('a command without the session CSRF token is rejected before any handler', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    for (const r of [
      await workCommand(page, ops, ops.caseA, 'assignment', idemKey('no-csrf'), null),
      await resolveOperation(page, ops, ops.operation, idemKey('no-csrf-resolve'), null),
    ]) {
      expect(r.status()).toBe(403)
      expect(r.headers()['content-type']).toContain('application/problem+json')
      expect(((await r.json()) as { reason_code: string }).reason_code).toBe('ACTION_DENIED')
    }
  })

  test('contract-invalid commands get one 400 answer, identical for in-scope, out-of-scope and unknown cases (no existence oracle)', async ({ page, ops, blocked }) => {
    void blocked
    const valid = JSON.stringify(commandBody('assignment', ops.userX.userId))
    // Anonymous (no session cookie yet): the contract check runs before authority.
    await expectContractInvalid(await rawCommand(page, ops, ops.caseA, 'assignment', null, { 'If-Match': '"1"' }, valid), 'anonymous, no Idempotency-Key')
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    const shapes: [string, Record<string, string>, string][] = [
      ['no Idempotency-Key', { 'If-Match': '"1"' }, valid],
      ['no If-Match', { 'Idempotency-Key': idemKey('no-if-match') }, valid],
      ['body is not one JSON object', { 'Idempotency-Key': idemKey('array-body'), 'If-Match': '"1"' }, '[]'],
      ['field outside the command scope', { 'Idempotency-Key': idemKey('extra-field'), 'If-Match': '"1"' },
        JSON.stringify({ ...commandBody('assignment', ops.userX.userId), state: 'COMPLETED' })],
    ]
    for (const [label, headers, body] of shapes) {
      const answers = new Set<string>()
      for (const id of [ops.caseA, ops.caseB, unknownCase]) {
        answers.add(await expectContractInvalid(await rawCommand(page, ops, id, 'assignment', csrf, headers, body), `${label} ${id}`))
      }
      expect(answers.size, `${label}: one answer for every case`).toBe(1)
    }
    // A malformed case id is a contract error too.
    await expectContractInvalid(await rawCommand(page, ops, 'not-a-case-id', 'assignment', csrf,
      { 'Idempotency-Key': idemKey('bad-id'), 'If-Match': '"1"' }, valid), 'malformed case id')
    // With a session but no CSRF token, the cross-site guard refuses first
    // (403 problem+json), before the contract check.
    const noCsrf = await rawCommand(page, ops, ops.caseA, 'assignment', null, { 'If-Match': '"1"' }, valid)
    expect(noCsrf.status()).toBe(403)
    expect(noCsrf.headers()['content-type']).toContain('application/problem+json')
    expect(((await noCsrf.json()) as { reason_code: string }).reason_code).toBe('ACTION_DENIED')
    // A well-formed request with an unparsable If-Match version reaches the
    // capability gate first: the refusal, not a version error.
    await expectActionDenied(await workCommand(page, ops, ops.caseA, 'assignment', idemKey('bad-if-match'), csrf, 'not-a-version'))
    // The orchestrator asserts nothing was recorded.
  })

  test('a session past its absolute lifetime is refused by the server and the open page returns to login', async ({ page, browser, ops, blocked }) => {
    void blocked
    // The orchestrator starts crm-api with CRM_AUTH_SESSION_IDLE / _ABSOLUTE = 1m:
    // this waits for the server to expire the real session; no row is edited.
    test.setTimeout(150_000)
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const loggedInAt = Date.now()
    const cookie = (await page.context().cookies()).find(c => c.name === '__Host-crm_session')
    expect(cookie, 'the real login set the session cookie').toBeTruthy()
    await page.goto(`/operations/work/${ops.caseA}`)
    await expect(page.getByRole('heading', { name: CASE, exact: true })).toBeVisible()

    // Probe with the same cookie value from a separate client, so a
    // browser-side cookie expiry cannot fake the result.
    const probe = await browser.newContext({ baseURL: ops.baseURL })
    const headers = { Cookie: `${cookie?.name}=${cookie?.value}` }
    await expect.poll(async () => {
      const s = (await (await probe.request.get('/api/v1/session', { headers })).json()) as SessionBody
      return s.data.authenticated
    }, { timeout: 120_000, intervals: [5_000] }).toBe(false)
    expect(Date.now() - loggedInAt, 'expired by the configured lifetime, not earlier').toBeGreaterThanOrEqual(55_000)
    await expectActionDenied(await probe.request.get(`/api/v1/fulfillment/work-cases/${ops.caseA}`, { headers }))
    await probe.close()

    await page.reload()
    await expect(page.getByRole('button', { name: 'ورود', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'خروج' })).toHaveCount(0)
  })
})

test.describe('provider operations and activation decisions (flags ON)', () => {
  test('provider ledger and activation decisions: no-access state in the UI, refusal in the API, no reconcile control', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    for (const [route, title] of [['/operations/provider-operations', PROVIDERS], ['/operations/activations', ACTIVATIONS]] as const) {
      await page.goto(route)
      await expect(page.getByRole('heading', { name: title, exact: true }), route).toBeVisible()
      await expect(page.getByText(DENIED).first(), route).toBeVisible()
      await expect(page.getByRole('table'), route).toHaveCount(0)
    }
    await expect(page.getByText(ops.operation)).toHaveCount(0)
    await expectActionDenied(await sessionGet(page, '/api/v1/integration/provider-operations'))
    await expectActionDenied(await sessionGet(page, '/api/v1/biawin/activation-decisions'))
  })

  test('reconciling the PREPARED operation is refused; a replay of the same key is refused the same way', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    const key = idemKey('resolve')
    await expectActionDenied(await resolveOperation(page, ops, ops.operation, key, csrf))
    await expectActionDenied(await resolveOperation(page, ops, ops.operation, key, csrf))
    await expectActionDenied(await resolveOperation(page, ops, unknownCase, idemKey('resolve-unknown'), csrf))
  })
})

test.describe('operations surfaces with every Slice 14 flag OFF @flags-off', () => {
  test('no navigation hint; every page shows the disabled state and no command control @flags-off', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.features?.fulfillment_work).toBe(false)
    for (const label of NAV) await expect(page.getByRole('link', { name: label, exact: true })).toHaveCount(0)
    for (const route of ['/operations/work', `/operations/work/${ops.caseA}`, '/operations/provider-operations', '/operations/activations']) {
      await page.goto(route)
      await expect(page.getByText(DISABLED).first(), route).toBeVisible()
      await expect(page.getByText(DENIED), route).toHaveCount(0)
    }
    await expectNoCommandControls(page)
  })

  test('every read answers BLOCKED / FEATURE_DISABLED with no data, for the session and for anonymous @flags-off', async ({ page, ops, blocked }) => {
    void blocked
    // The flag gate runs before authority: anonymous gets the disabled contract too.
    for (const path of [...collectionReads(ops), ...caseReads(ops.caseA), ...caseReads(unknownCase)]) await expectFeatureDisabled(await page.request.get(path))
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    for (const path of [...collectionReads(ops), ...caseReads(ops.caseA), ...caseReads(ops.caseB), ...caseReads(unknownCase)]) {
      const r = await sessionGet(page, path)
      await expectFeatureDisabled(r)
      const body = (await r.json()) as { items?: unknown; case?: unknown; history?: unknown; allowed_actions?: unknown }
      expect(body.items, path).toBeUndefined()
      expect(body.case, path).toBeUndefined()
      expect(body.history, path).toBeUndefined()
      expect(body.allowed_actions, path).toBeUndefined()
    }
  })

  test('commands answer BLOCKED / FEATURE_DISABLED, not applied, TARGET_WRITE_DISABLED; replay changes nothing @flags-off', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    const key = idemKey('off')
    for (const [id, cmd] of [
      ...workCommands.map(c => [ops.caseA, c] as const), [ops.caseA, 'operational-actions'] as const, [ops.caseP, 'outcome'] as const,
      [unknownCase, 'assignment'] as const,
    ]) {
      await expectCommandDisabled(await workCommand(page, ops, id, cmd, cmd === 'operational-actions' ? key : idemKey(`off-${cmd}`), csrf), `${cmd} ${id}`)
    }
    const res = await resolveOperation(page, ops, ops.operation, idemKey('off-resolve'), csrf)
    await expectFeatureDisabled(res)
    expect(((await res.json()) as { operation?: unknown }).operation).toBeUndefined()
  })

  test('a contract-invalid command is still answered 400 by the contract check, which runs before the flag gate @flags-off', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    await expectContractInvalid(await rawCommand(page, ops, ops.caseA, 'operational-actions', csrf, { 'If-Match': '"1"' },
      JSON.stringify(commandBody('operational-actions', ops.userX.userId))), 'no Idempotency-Key, flags off')
  })
})

test.describe('reads ON, commands OFF, write-back flag ON alone, reconciliation OFF @flags-mixed', () => {
  test('the navigation hint follows the read flag; every page shows the no-access state, not the disabled state @flags-mixed', async ({ page, ops, blocked }) => {
    void blocked
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.features?.fulfillment_work).toBe(true)
    for (const label of NAV) await expect(page.getByRole('link', { name: label, exact: true })).toBeVisible()
    for (const route of ['/operations/work', `/operations/work/${ops.caseA}`, '/operations/provider-operations', '/operations/activations']) {
      await page.goto(route)
      await expect(page.getByText(DENIED).first(), route).toBeVisible()
      await expect(page.getByText(DISABLED), route).toHaveCount(0)
    }
    await expectNoCommandControls(page)
  })

  test('reads are refused by the capability gate (403), for the session and for anonymous @flags-mixed', async ({ page, ops, blocked }) => {
    void blocked
    for (const path of [...collectionReads(ops), ...caseReads(ops.caseA)]) await expectActionDenied(await page.request.get(path))
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    for (const path of [...collectionReads(ops), ...caseReads(ops.caseA), ...caseReads(ops.caseB), ...caseReads(unknownCase)]) {
      await expectActionDenied(await sessionGet(page, path))
    }
  })

  test('commands answer the disabled contract (200) while reads are denied (403); the write-back flag alone enables nothing @flags-mixed', async ({ page, ops, blocked }) => {
    void blocked
    // Anonymous: the command flag is OFF, so the flag gate answers before authority.
    await expectCommandDisabled(await workCommand(page, ops, ops.caseA, 'assignment', idemKey('mixed-anon'), null), 'anonymous assignment')
    await loginUI(page, ops.userX.login, ops.userX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    for (const id of [ops.caseA, ops.caseB, ops.caseP, unknownCase]) {
      for (const cmd of workCommands) await expectCommandDisabled(await workCommand(page, ops, id, cmd, idemKey(`mixed-${cmd}`), csrf), `${cmd} ${id}`)
    }
    // Same case, same moment: the read is denied, the command is disabled. Never interchangeable.
    await expectActionDenied(await sessionGet(page, `/api/v1/fulfillment/work-cases/${ops.caseA}`))
    const res = await resolveOperation(page, ops, ops.operation, idemKey('mixed-resolve'), csrf)
    await expectFeatureDisabled(res)
    expect(((await res.json()) as { operation?: unknown }).operation).toBeUndefined()
  })
})

test.describe('isolation on the operations stack', () => {
  test('the network guard aborts and reports any non-local request from a work-case page', async ({ page, ops, blocked }) => {
    await page.goto(`/operations/work/${ops.caseA}`)
    await page.evaluate(() => {
      const link = document.createElement('link')
      link.rel = 'stylesheet'
      link.href = 'https://e2e-guard-probe.example.invalid/operations.css'
      document.head.appendChild(link)
    })
    await expect.poll(() => blocked.length).toBe(1)
    expect(blocked[0]).toBe('https://e2e-guard-probe.example.invalid/operations.css')
    // The probe is the only expected block: it is cleared after asserting it,
    // so the fixture's teardown still fails on any other non-local request.
    blocked.length = 0
  })
})
