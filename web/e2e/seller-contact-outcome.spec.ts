import { csrfToken, expect, expectActionDenied, loginUI, recordOutcome, sessionGet, test, type Envelope } from './fixtures'

// R7-1: real browser → real SPA → real crm-api (R1 sessions, production
// subject resolver) → disposable PostgreSQL with SYNTHETIC data.
//
// What the shipped product does today: a seller authenticates, but holds no
// authority. The seller role carries no capability and no ACTIVE grant exists
// (D-42), so reading cases and recording a contact outcome are refused (403).
// The positive "record and see in history" path is therefore asserted as
// refused, not skipped; it becomes testable only after a verified grant
// decision (D-42 / R5-SEC-01, M2).

test.describe('R1 login and session', () => {
  test('wrong password gives the generic authentication error and no session', async ({ page, stack, blocked }) => {
    void blocked
    await loginUI(page, stack.sellerX.login, `${stack.sellerX.password}-wrong`)
    await expect(page.getByRole('alert')).toHaveText('نام کاربری یا گذرواژه درست نیست.')
    const s = await page.request.get('/api/v1/session')
    expect(((await s.json()) as { data: { authenticated: boolean } }).data.authenticated).toBe(false)
  })

  test('seller logs in, opens the held case, and logout invalidates the session', async ({ page, context, stack, browser, blocked }) => {
    void blocked
    await loginUI(page, stack.sellerX.login, stack.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as { data: { authenticated: boolean; session: { user_id: string } } }
    expect(s.data.authenticated).toBe(true)
    expect(s.data.session.user_id).toBe(stack.sellerX.userId)

    // Opens the existing case the seller holds: the UI shows the no-access
    // state and offers no contact-outcome action (D-42: no capability).
    await page.goto(`/sales/cases/${stack.caseHeldByX}`)
    await expect(page.getByText('دسترسی به این بخش ندارید').first()).toBeVisible()
    await expect(page.getByRole('form', { name: 'ثبت پیامد تماس' })).toHaveCount(0)
    expect((await sessionGet(page, `/api/v1/cases/${stack.caseHeldByX}`)).status()).toBe(403)

    // Logout ends the session server-side: the old cookie no longer works.
    const old = (await context.cookies()).find(c => c.name === '__Host-crm_session')
    expect(old, 'the real login set the session cookie').toBeTruthy()
    await page.getByRole('button', { name: 'خروج' }).click()
    await expect(page.getByRole('button', { name: 'ورود', exact: true })).toBeVisible()
    const replay = await browser.newContext({ baseURL: stack.baseURL })
    const r = await replay.request.get('/api/v1/session', { headers: { Cookie: `${old?.name}=${old?.value}` } })
    const after = (await r.json()) as { data: { authenticated: boolean } }
    expect(after.data.authenticated).toBe(false)
    await replay.close()
  })
})

test.describe('contact outcome command (write flag ON in the disposable database)', () => {
  test('a seller without the write capability gets 403 on the case it holds', async ({ page, stack, blocked }) => {
    void blocked
    await loginUI(page, stack.sellerX.login, stack.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    await expectActionDenied(await recordOutcome(page, stack, stack.caseHeldByX, `e2e-own-${Date.now()}`, csrf))
  })

  // Refused at the capability layer: with no grant the holder (row-scope)
  // check is never reached, so it cannot be exercised end to end yet (D-42).
  test("another seller's held case cannot receive a contact outcome", async ({ page, stack, blocked }) => {
    void blocked
    await loginUI(page, stack.sellerY.login, stack.sellerY.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    await expectActionDenied(await recordOutcome(page, stack, stack.caseHeldByX, `e2e-other-${Date.now()}`, csrf))
  })

  test('replaying the same idempotency key creates no duplicate', async ({ page, stack, blocked }) => {
    void blocked
    await loginUI(page, stack.sellerX.login, stack.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    const key = `e2e-replay-${Date.now()}`
    await expectActionDenied(await recordOutcome(page, stack, stack.caseHeldByX, key, csrf))
    await expectActionDenied(await recordOutcome(page, stack, stack.caseHeldByX, key, csrf))
    // The orchestrator asserts afterwards that no contact outcome, event or
    // command record exists in the database.
  })
})

test.describe('write flag OFF in the disposable database @flag-off', () => {
  test('the UI offers no action and the API refuses the command @flag-off', async ({ page, stack, blocked }) => {
    void blocked
    await loginUI(page, stack.sellerX.login, stack.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    await page.goto(`/sales/cases/${stack.caseHeldByX}`)
    await expect(page.getByRole('form', { name: 'ثبت پیامد تماس' })).toHaveCount(0)
    const csrf = await csrfToken(page)
    // The flag gate runs before authority and refuses through the result
    // envelope (contract: "off = BLOCKED / FEATURE_DISABLED"); nothing applies.
    const r = await recordOutcome(page, stack, stack.caseHeldByX, `e2e-off-${Date.now()}`, csrf)
    expect(r.status()).toBe(200)
    const body = (await r.json()) as Envelope
    expect(body.outcome).toBe('BLOCKED')
    expect(body.reasons).toEqual(['FEATURE_DISABLED'])
    expect(body.command?.applied).toBe(false)
    expect(body.command?.rollout_mode).toBe('TARGET_WRITE_DISABLED')
    expect(body.command?.writeback_state).toBe('NOT_REQUESTED')
  })
})

test.describe('isolation', () => {
  test('the network guard aborts and reports any non-local request', async ({ page, blocked }) => {
    await page.goto('/')
    // A deliberate off-machine request from inside the page.
    await page.evaluate(() => {
      const img = document.createElement('img')
      img.src = 'https://e2e-guard-probe.example.invalid/pixel.png'
      document.body.appendChild(img)
    })
    await expect.poll(() => blocked.length).toBe(1)
    expect(blocked[0]).toBe('https://e2e-guard-probe.example.invalid/pixel.png')
    // The probe proved the guard; clear it so the fixture's own check stays strict.
    blocked.length = 0
  })
})
