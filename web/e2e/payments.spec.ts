import { csrfToken, expectActionDenied, loginUI, sessionGet, type Envelope } from './fixtures'
import {
  collectionReads, expect, expectFeatureDisabled, invoiceReads, stageUpload, submitBatch, syntheticReceipt, test,
} from './payments-fixtures'

// R7-2: real browser → real SPA → real crm-api (R1 sessions, production
// subject resolver) → disposable PostgreSQL with the SYNTHETIC Slice 12
// payments scenario.
//
// What the shipped product does today: a seller authenticates but holds no
// authority. No seller capability and no ACTIVE grant exists (D-42; D-34 and
// the seller capability / read-scope decisions are open), so every invoice
// read, payment-attempt read, receipt upload and batch submission is refused
// (403 UNAUTHORIZED / ACTION_DENIED) even for the invoice's own seller. The
// positive "upload receipt → submit batch → see the attempt" workflow is
// asserted as REFUSED here, never skipped and never reported as executed.
// The orchestrator then proves no billing / payments / finance row changed
// and no receipt byte was stored.

const DENIED = 'دسترسی به این بخش ندارید'
const DISABLED = 'این بخش هنوز فعال نشده است'
const PANEL = 'ثبت پرداخت کارت‌به‌کارت با فیش (دستور کنترل‌شده)'

interface SessionBody {
  data: { authenticated: boolean; session?: { user_id: string } | null; features?: { payment_submission: boolean; billing_read: boolean; finance_review_read: boolean } }
}

test.describe('R1 login and logout on the payments stack', () => {
  test('the invoice seller logs in, sees the payment feature hint, and logout invalidates the session', async ({ page, context, browser, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.authenticated).toBe(true)
    expect(s.data.session?.user_id).toBe(pay.sellerX.userId)
    // Feature hints follow the flags only; they grant nothing.
    expect(s.data.features?.payment_submission).toBe(true)
    expect(s.data.features?.billing_read).toBe(true)
    expect(s.data.features?.finance_review_read).toBe(true)

    const old = (await context.cookies()).find(c => c.name === '__Host-crm_session')
    expect(old, 'the real login set the session cookie').toBeTruthy()
    await page.getByRole('button', { name: 'خروج' }).click()
    await expect(page.getByRole('button', { name: 'ورود', exact: true })).toBeVisible()

    // The old cookie no longer authenticates, and with no session the
    // payment surfaces deny the anonymous caller.
    const replay = await browser.newContext({ baseURL: pay.baseURL })
    const cookie = { Cookie: `${old?.name}=${old?.value}` }
    const after = (await (await replay.request.get('/api/v1/session', { headers: cookie })).json()) as SessionBody
    expect(after.data.authenticated).toBe(false)
    for (const path of ['/api/v1/invoices', `/api/v1/payments/invoices/${pay.invoiceOfX}/submission`]) {
      const r = await replay.request.get(path, { headers: cookie })
      expect(r.status(), path).toBe(403)
      expect(((await r.json()) as Envelope).outcome, path).toBe('UNAUTHORIZED')
    }
    await replay.close()
  })
})

test.describe('invoice and payment reads (flags ON in the disposable database)', () => {
  test('invoice list: the UI shows the no-access state and no row; the API refuses', async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    await page.goto('/billing/invoices')
    await expect(page.getByRole('heading', { name: 'فاکتورها', exact: true })).toBeVisible()
    await expect(page.getByText(DENIED).first()).toBeVisible()
    await expect(page.locator('a[href^="/billing/invoices/"]')).toHaveCount(0)
    await expectActionDenied(await sessionGet(page, '/api/v1/invoices'))
  })

  test("invoice detail of the seller's own invoice: every section refused, no submit control", async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    await page.goto(`/billing/invoices/${pay.invoiceOfX}`)
    await expect(page.getByText(DENIED).first()).toBeVisible()
    // Payment evidence stays locked (no payments.attempt.read allowed action).
    await expect(page.getByText('دسترسی به شواهد پرداخت ندارید').first()).toBeVisible()
    // The submission panel renders (flag ON hint) but its read is refused:
    // no file input, no review or submit control.
    const panel = page.getByRole('region', { name: PANEL, exact: true })
    await expect(panel).toBeVisible()
    await expect(panel.getByText(DENIED)).toBeVisible()
    await expect(page.getByRole('region', { name: 'فاکتور', exact: true }).getByText(DENIED)).toBeVisible()
    await expect(page.locator('input[type="file"]')).toHaveCount(0)
    await expect(page.getByRole('group', { name: 'ثبت پرداخت' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'تایید و ثبت' })).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'بررسی پیش از ثبت' })).toHaveCount(0)
    for (const path of invoiceReads(pay.invoiceOfX)) await expectActionDenied(await sessionGet(page, path))
  })

  test('payment attempts, batches, evidence and the finance review queue are refused in the UI and the API', async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    for (const [route, title] of [
      ['/billing/attempts', 'تلاش‌های پرداخت'],
      ['/payments/batches/00000000-0000-4000-8000-0000000000e3', 'دسته پرداخت'],
      ['/payments/evidence', 'شواهد پرداخت'],
      ['/finance/review', 'صف بررسی مالی'],
    ] as const) {
      await page.goto(route)
      await expect(page.getByRole('heading', { name: title, exact: true }), route).toBeVisible()
      await expect(page.getByText(DENIED).first(), route).toBeVisible()
    }
    for (const path of collectionReads) await expectActionDenied(await sessionGet(page, path))
  })
})

test.describe('receipt upload and batch submission (flags ON, controlled mode)', () => {
  test('receipt upload: refused for the invoice seller; refused without the CSRF token', async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    // Capability layer: no payments.upload.stage for any seller (D-42).
    await expectActionDenied(await stageUpload(page, pay, syntheticReceipt(1), csrf))
    // Cross-site guard: an authenticated unsafe request without the
    // session-bound token is rejected before any handler runs.
    const r = await stageUpload(page, pay, syntheticReceipt(2), null)
    expect(r.status()).toBe(403)
    expect(r.headers()['content-type']).toContain('application/problem+json')
    const p = (await r.json()) as { reason_code: string }
    expect(p.reason_code).toBe('ACTION_DENIED')
  })

  test("batch submission: refused for the invoice's own seller and for another seller", async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    await expectActionDenied(await submitBatch(page, pay, pay.invoiceOfX, `e2e-pay-own-${Date.now()}`, await csrfToken(page)))
    await page.getByRole('button', { name: 'خروج' }).click()
    await expect(page.getByRole('button', { name: 'ورود', exact: true })).toBeVisible()

    // Refused at the capability layer: with no grant the seller row-scope
    // check (OPD-10, ROW_SCOPE_DENIED) is never reached end to end.
    await loginUI(page, pay.sellerY.login, pay.sellerY.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    await expectActionDenied(await submitBatch(page, pay, pay.invoiceOfX, `e2e-pay-other-${Date.now()}`, await csrfToken(page)))
  })

  test('replaying the same idempotency key and a stale If-Match create nothing', async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)
    const key = `e2e-pay-replay-${Date.now()}`
    // Refused before the idempotency store: replay and version checks are
    // unreachable without a grant. Both calls answer the same refusal.
    await expectActionDenied(await submitBatch(page, pay, pay.invoiceOfX, key, csrf))
    await expectActionDenied(await submitBatch(page, pay, pay.invoiceOfX, key, csrf))
    await expectActionDenied(await submitBatch(page, pay, pay.invoiceOfX, key, csrf, '7'))
    // The orchestrator asserts afterwards that no billing, payments or
    // finance row changed and that the evidence store is empty.
  })
})

test.describe('payment surfaces with every payment flag OFF @flags-off', () => {
  test('the UI shows the disabled state and no submission panel @flags-off', async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const s = (await (await sessionGet(page, '/api/v1/session')).json()) as SessionBody
    expect(s.data.features?.payment_submission).toBe(false)
    expect(s.data.features?.billing_read).toBe(false)
    expect(s.data.features?.finance_review_read).toBe(false)

    await page.goto('/billing/invoices')
    await expect(page.getByText(DISABLED).first()).toBeVisible()
    await page.goto(`/billing/invoices/${pay.invoiceOfX}`)
    await expect(page.getByText(DISABLED).first()).toBeVisible()
    await expect(page.getByRole('region', { name: PANEL, exact: true })).toHaveCount(0)
    await expect(page.locator('input[type="file"]')).toHaveCount(0)
    await page.goto('/payments/evidence')
    await expect(page.getByText(DISABLED).first()).toBeVisible()
  })

  test('every read answers BLOCKED / FEATURE_DISABLED with no data @flags-off', async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    for (const path of [...invoiceReads(pay.invoiceOfX), ...collectionReads]) {
      const r = await sessionGet(page, path)
      await expectFeatureDisabled(r)
      const body = (await r.json()) as { data?: unknown; items?: unknown }
      expect(body.data, path).toBeUndefined()
      expect(body.items, path).toBeUndefined()
    }
  })

  test('upload and batch commands are refused through the result envelope; replay changes nothing @flags-off', async ({ page, pay, blocked }) => {
    void blocked
    await loginUI(page, pay.sellerX.login, pay.sellerX.password)
    await expect(page.getByRole('button', { name: 'خروج' })).toBeVisible()
    const csrf = await csrfToken(page)

    // The flag gate runs before authority (contract: off = BLOCKED / FEATURE_DISABLED).
    const up = await stageUpload(page, pay, syntheticReceipt(3), csrf)
    await expectFeatureDisabled(up)
    const ud = ((await up.json()) as { data: { rollout_mode: string; upload_id?: string } }).data
    expect(ud.rollout_mode).toBe('TARGET_WRITE_DISABLED')
    expect(ud.upload_id).toBeUndefined()

    const key = `e2e-pay-off-${Date.now()}`
    for (let i = 0; i < 2; i++) {
      const r = await submitBatch(page, pay, pay.invoiceOfX, key, csrf)
      await expectFeatureDisabled(r)
      const d = ((await r.json()) as {
        data: { applied: boolean; rollout_mode: string; writeback_state: string; items: unknown[]; batch_id?: string; replayed?: boolean }
      }).data
      expect(d.applied).toBe(false)
      expect(d.rollout_mode).toBe('TARGET_WRITE_DISABLED')
      expect(d.writeback_state).toBe('NOT_REQUESTED')
      expect(d.items).toEqual([])
      expect(d.batch_id).toBeUndefined()
      expect(d.replayed).toBeUndefined()
    }
  })
})

test.describe('isolation on the payments stack', () => {
  test('the network guard aborts and reports any non-local request from an invoice page', async ({ page, pay, blocked }) => {
    await page.goto(`/billing/invoices/${pay.invoiceOfX}`)
    await page.evaluate(() => {
      const link = document.createElement('link')
      link.rel = 'stylesheet'
      link.href = 'https://e2e-guard-probe.example.invalid/payments.css'
      document.head.appendChild(link)
    })
    await expect.poll(() => blocked.length).toBe(1)
    expect(blocked[0]).toBe('https://e2e-guard-probe.example.invalid/payments.css')
    blocked.length = 0
  })
})
