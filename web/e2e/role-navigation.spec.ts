import { expect, test, type Page, type Route } from '@playwright/test'

// RM-E02 — real-browser regression of the role-oriented, fail-closed navigation on the
// PRODUCTION bundle (playwright.nav.config.ts). The API is answered by synthetic
// fixtures through page.route: this is presentation evidence, never business
// authorization (no grant, role, flag or account exists anywhere).
// Evidence screenshots: RM_E02_EVIDENCE_DIR=../docs/evidence/rm-e02 npx playwright test -c playwright.nav.config.ts

const OUT = process.env.RM_E02_EVIDENCE_DIR
const NAV = 'ناوبری اصلی'
const SAFE = ['خانه', 'وضعیت سامانه']
const NO_GRANT = 'دسترسی فعالی برای شما ثبت نشده است'

const CAPS = [
  'identity.user.read', 'org.person.read', 'org.employment.read', 'org.position.read', 'org.unit.read', 'org.assignment.read',
  'customers.customer.read', 'customers.merge_case.read', 'customers.identity_report.read', 'cases.case.read', 'custody.custody.read',
  'sales.report.read', 'billing.invoice.read', 'payments.attempt.read', 'billing.reconciliation.read', 'reporting.metrics.read',
  'reporting.mis.read', 'reporting.invoice_review.read', 'finance.shadow.read', 'finance.ledger.read', 'finance.reconciliation.read',
  'biawin.subscription.read', 'fulfillment.case.read', 'biawin.parity.read', 'cases.contact_outcome.record', 'org.hr_request.create',
  'org.hr_request.review', 'org.hr_request.read', 'integration.writeback.read',
]
const FEATURES = ['hr_read', 'customers_identity', 'sales_read', 'billing_read', 'reporting_read', 'finance_shadow_read', 'biawin_fulfillment_read',
  'contact_outcome_write', 'hr_request_write', 'writeback_read', 'payment_submission', 'finance_review_read', 'commission_read', 'fulfillment_work']

interface Fixture {
  /** Capability → matched roles (ALLOWED); everything else UNAUTHORIZED. */
  allow?: Record<string, string[]>
  authenticated?: boolean
  /** Delay the effective-access answer until `release()` (ms timeout otherwise). */
  holdEvaluation?: boolean
}

const env = (data: unknown) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'syn-corr-0001', data })
function sessionBody(authenticated: boolean) {
  return env({
    authenticated, auth_mode: 'session', session_state: authenticated ? 'active' : 'invalid', login_enabled: true,
    server_time: '2026-11-16T09:00:00Z', display_zone: 'Asia/Tehran',
    features: Object.fromEntries(FEATURES.map(f => [f, true])),
    ...(authenticated ? { session: { user_id: '00000000-0000-4000-8000-0000000000a1', display_name: 'کاربر ساختگی', csrf_token: 'a'.repeat(43), idle_expires_at: '2099-01-01T00:00:00Z', absolute_expires_at: '2099-01-01T00:00:00Z' } } : {}),
  })
}
function evaluation(allow: Record<string, string[]>) {
  return env({
    authenticated: true, hr_read_enabled: true,
    decisions: CAPS.map(c => allow[c]
      ? { capability: c, allowed: true, outcome: 'SUCCESS', scope: 'ORGANIZATION', matched_roles: allow[c] }
      : { capability: c, allowed: false, outcome: 'UNAUTHORIZED', reason_code: 'ACTION_DENIED', scope: 'NONE', matched_roles: [] }),
  })
}

/** Routes every API call to synthetic fixtures; records methods; returns a release hook and the log. */
async function serve(page: Page, f: Fixture) {
  const log: { method: string; path: string; devHeader: boolean }[] = []
  let session = f.authenticated ?? true
  let release: () => void = () => {}
  const held = new Promise<void>(r => { release = r })
  await page.route(/\/(api|health)\//, async (route: Route) => {
    const req = route.request()
    const path = new URL(req.url()).pathname
    log.push({ method: req.method(), path, devHeader: 'x-dev-user' in req.headers() })
    const ok = (body: unknown, status = 200) => route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) })
    if (path === '/health/live') return ok({ status: 'live' })
    if (path === '/health/ready') return ok({ status: 'ready', checks: [] })
    if (path === '/api/v1/session') return ok(sessionBody(session))
    if (path === '/api/v1/authorization/effective-access') {
      if (f.holdEvaluation) await held
      return ok(evaluation(f.allow ?? {}))
    }
    if (path === '/api/v1/organization/persons' && f.allow?.['org.person.read']?.includes('expire-me')) {
      session = false
      return ok({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'syn-corr-0401' }, 401)
    }
    // Every page read is refused: navigation must never depend on page data.
    return ok({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'syn-corr-0403' }, 403)
  })
  return { log, release: () => release() }
}

const ALL_ROLES = Object.fromEntries(CAPS.map((c, i) => [c, [i % 2 ? 'finance' : 'hr']]))
const SELLER = { 'cases.case.read': ['seller'], 'billing.invoice.read': ['seller'], 'customers.customer.read': ['seller'] }

async function noPageHorizontalScroll(page: Page) {
  expect(await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(0)
}
async function shot(page: Page, name: string) {
  if (OUT) await page.screenshot({ path: `${OUT}/${name}-${page.viewportSize()?.width ?? 0}.png` })
}

test.describe('desktop', () => {
  test.skip(({ viewport }) => (viewport?.width ?? 0) < 761, 'desktop only')

  test('no ACTIVE grant: safe links only, explicit no-access state, no business call', async ({ page }) => {
    const { log } = await serve(page, {})
    await page.goto('/')
    await expect(page.getByRole('heading', { name: NO_GRANT })).toBeVisible()
    await expect(page.getByRole('navigation', { name: NAV }).getByRole('link')).toHaveText(SAFE)
    await expect(page.getByRole('navigation', { name: 'ناوبری موبایل' })).toBeHidden()
    expect(log.map(l => l.path).sort()).toEqual(['/api/v1/authorization/effective-access', '/api/v1/session'])
    await noPageHorizontalScroll(page)
    await shot(page, 'no-access')
  })

  test('single verified role: role crumb, role-mapped destinations first, deep link current', async ({ page }) => {
    await serve(page, { allow: SELLER })
    await page.goto('/billing/invoices/00000000-0000-4000-8000-0000000000b1')
    const nav = page.getByRole('navigation', { name: NAV })
    await expect(nav.getByRole('link')).toHaveText(['خانه', 'پرونده‌های فروش', 'فاکتورها', 'مشتریان', 'وضعیت سامانه'])
    await expect(nav.getByRole('link', { name: 'فاکتورها' })).toHaveAttribute('aria-current', 'page')
    await expect(page.getByText('نقش: فروشنده')).toBeVisible()
    await shot(page, 'deep-link-refused')
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'بخش‌های نقش شما' })).toBeVisible()
    await shot(page, 'single-role')
  })

  test('many destinations: priority+ overflow keeps the current item; «بیشتر» is keyboard operable and routes without a reload', async ({ page }) => {
    await serve(page, { allow: ALL_ROLES })
    await page.goto('/finance/reconciliation')
    const nav = page.getByRole('navigation', { name: NAV })
    const current = nav.locator('[aria-current="page"]')
    await expect(current).toHaveText('تطبیق سایه')
    await expect(current).toBeVisible()
    const more = nav.getByRole('button', { name: /بیشتر/ })
    await expect(more).toBeVisible()
    expect(await nav.evaluate(n => n.scrollWidth - n.clientWidth)).toBeLessThanOrEqual(1)
    await noPageHorizontalScroll(page)
    await page.evaluate(() => { (window as unknown as { __noReload: boolean }).__noReload = true })
    await more.focus()
    await page.keyboard.press('Enter')
    const menu = page.getByRole('menu', { name: 'بیشتر' })
    await expect(menu).toBeVisible()
    await expect(menu.getByRole('menuitem').first()).toBeFocused()
    await page.keyboard.press('Escape')
    await expect(menu).toBeHidden()
    await expect(more).toBeFocused()
    await page.keyboard.press('Enter')
    await page.keyboard.press('End')
    const last = menu.getByRole('menuitem').last()
    const label = (await last.textContent())?.trim() ?? ''
    await page.keyboard.press('Enter')
    await expect(page.getByRole('heading', { level: 1 })).toHaveText(label)
    expect(await page.evaluate(() => (window as unknown as { __noReload?: boolean }).__noReload)).toBe(true)
    await expect(nav.locator('[aria-current="page"]')).toHaveText(label)
    await page.goto('/')
    await expect(page.getByText(/سیاست انتخاب نقش هنوز تعریف نشده است/)).toBeVisible()
    await shot(page, 'multi-role-overflow')
  })

  test('keyboard: skip link reaches main; nav links are reachable in order and show a focus ring', async ({ page }) => {
    await serve(page, { allow: SELLER })
    await page.goto('/')
    await expect(page.getByRole('navigation', { name: NAV }).getByRole('link')).toHaveCount(5)
    await page.keyboard.press('Tab')
    await expect(page.getByRole('link', { name: 'پرش به محتوای اصلی' })).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(page.locator('main#ws')).toBeFocused()
    const first = page.getByRole('navigation', { name: NAV }).getByRole('link', { name: 'خانه' })
    await first.focus()
    await page.keyboard.press('Tab')
    const next = page.getByRole('navigation', { name: NAV }).getByRole('link', { name: 'پرونده‌های فروش' })
    await expect(next).toBeFocused()
    expect(await next.evaluate(el => getComputedStyle(el).outlineStyle)).toBe('solid')
  })

  test('evaluation pending: nothing protected shown or requested; then the allowed set appears', async ({ page }) => {
    const { release, log } = await serve(page, { allow: SELLER, holdEvaluation: true })
    await page.goto('/')
    await expect(page.getByRole('status').filter({ hasText: 'در حال بررسی دسترسی‌ها…' })).toHaveAttribute('aria-busy', 'true')
    await expect(page.getByRole('navigation', { name: NAV }).getByRole('link')).toHaveText(SAFE)
    expect(log.filter(l => !['/api/v1/session', '/api/v1/authorization/effective-access'].includes(l.path))).toEqual([])
    release()
    await expect(page.getByRole('navigation', { name: NAV }).getByRole('link', { name: 'فاکتورها' })).toBeVisible()
  })

  test('navigation issues no write; production bundle has no DEV route, link or identity header', async ({ page }) => {
    const { log } = await serve(page, { allow: ALL_ROLES })
    await page.goto('/')
    const nav = page.getByRole('navigation', { name: NAV })
    for (const name of ['نیروها', 'ساختار سازمانی', 'خانه']) {
      await nav.getByRole('link', { name, exact: true }).click()
      await expect(nav.getByRole('link', { name, exact: true })).toHaveAttribute('aria-current', 'page')
    }
    // «وضعیت سامانه» sits in the overflow menu at this width.
    await nav.getByRole('button', { name: /بیشتر/ }).click()
    await page.getByRole('menu', { name: 'بیشتر' }).getByRole('menuitem', { name: 'وضعیت سامانه' }).click()
    await expect(page.getByRole('heading', { level: 1, name: 'وضعیت سامانه' })).toBeVisible()
    await expect(nav).not.toContainText('DEV')
    await page.goto('/dev/design-system')
    await expect(page.getByRole('heading', { level: 1, name: 'صفحه پیدا نشد' })).toBeVisible()
    expect(log.filter(l => l.method !== 'GET')).toEqual([])
    expect(log.some(l => l.devHeader)).toBe(false)
  })

  test('session expires mid-visit: the login page with its notice replaces the page; business links disappear', async ({ page }) => {
    await serve(page, { allow: { 'org.person.read': ['expire-me'] } })
    await page.goto('/hr')
    await expect(page.getByText('نشست شما پایان یافته است')).toBeVisible()
    await expect(page.getByRole('navigation', { name: NAV }).getByRole('link')).toHaveText(SAFE)
  })
})

test.describe('mobile', () => {
  test.skip(({ viewport }) => (viewport?.width ?? 9999) > 760, 'mobile only')

  test('no-access: the bottom bar carries only the safe destinations; the page never scrolls sideways', async ({ page }) => {
    await serve(page, {})
    await page.goto('/')
    await expect(page.getByRole('heading', { name: NO_GRANT })).toBeVisible()
    await expect(page.getByRole('navigation', { name: NAV })).toBeHidden()
    const bottom = page.getByRole('navigation', { name: 'ناوبری موبایل' })
    await expect(bottom.getByRole('link')).toHaveText(['خانه', 'سامانه'])
    const box = await bottom.boundingBox()
    expect(Math.round((box?.y ?? 0) + (box?.height ?? 0))).toBe(844)
    await noPageHorizontalScroll(page)
    await shot(page, 'no-access')
  })

  test('authorised set: four primary destinations + «بیشتر»; the menu routes and marks «بیشتر» current', async ({ page }) => {
    await serve(page, { allow: ALL_ROLES })
    await page.goto('/')
    const bottom = page.getByRole('navigation', { name: 'ناوبری موبایل' })
    await expect(bottom.getByRole('link')).toHaveText(['خانه', 'نیروها', 'ساختار', 'استثناها'])
    const more = bottom.getByRole('button', { name: 'بیشتر' })
    await shot(page, 'multi-role')
    await more.click()
    const menu = page.getByRole('menu', { name: 'بیشتر' })
    await expect(menu).toBeVisible()
    await menu.getByRole('menuitem', { name: /فاکتورها/ }).first().click()
    await expect(page.getByRole('heading', { level: 1, name: 'فاکتورها' })).toBeVisible()
    await expect(more).toHaveAttribute('aria-current', 'page')
    await expect(page.locator('.ds-m-title')).toHaveText('فاکتورها')
    const heights = await bottom.locator('a, button').evaluateAll(els => els.map(e => e.getBoundingClientRect().height))
    for (const h of heights) expect(h).toBeGreaterThanOrEqual(40)
    await noPageHorizontalScroll(page)
  })
})
