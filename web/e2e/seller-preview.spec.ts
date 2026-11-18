import { expect, test, type Page, type TestInfo } from '@playwright/test'

// RM-E03 — real-browser checks of the DEV/TEST-only Seller V1 preview (/dev/seller/*).
// SYNTHETIC fixtures only. Every /api and /health request is intercepted and recorded: the preview
// must not call any business endpoint and must never send POST/PUT/PATCH/DELETE.
// Run: npx playwright test -c playwright.seller.config.ts
// Evidence: RM_E03_EVIDENCE_DIR=../docs/evidence/rm-e03 npx playwright test -c playwright.seller.config.ts

const OUT = process.env.RM_E03_EVIDENCE_DIR
const SECTIONS = [
  ['leads', 'شماره‌های من'],
  ['conversions', 'آماده‌های تبدیل'],
  ['repeat', 'نیاز به اقدام مجدد'],
  ['invoices', 'فاکتورها'],
  ['behavior', 'رفتار مشتریان'],
  ['request', 'درخواست اضافه کردن شماره'],
  ['wallet', 'کیف پول من'],
] as const

type Kind = 'desktop' | 'tablet' | 'mobile'
const kindOf = (w: number): Kind => (w <= 760 ? 'mobile' : w <= 1100 ? 'tablet' : 'desktop')

interface Req { method: string; path: string }

/** Answer every API/health call with an unauthenticated synthetic session and record it. */
async function guard(page: Page): Promise<Req[]> {
  const log: Req[] = []
  // Path-prefix match only: Vite serves app modules such as /src/shared/api/client.ts, which must load.
  await page.route(url => url.pathname.startsWith('/api/') || url.pathname === '/health' || url.pathname.startsWith('/health/'), async route => {
    const req = route.request()
    const path = new URL(req.url()).pathname
    log.push({ method: req.method(), path })
    if (path === '/api/v1/session') {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ outcome: 'SUCCESS', reasons: [], correlation_id: 'syn-corr-e03', data: { authenticated: false, auth_mode: 'session', session_state: 'invalid', login_enabled: true, server_time: '2026-11-16T09:00:00Z', display_zone: 'Asia/Tehran', features: {} } }) })
      return
    }
    await route.fulfill({ status: 404, contentType: 'application/json', body: '{}' })
  })
  return log
}

/** Only the app-level session probe (made by SessionProvider for every route) is tolerated; no business endpoint, no write. */
function expectNoBusinessTraffic(log: Req[]) {
  expect(log.filter(r => r.method !== 'GET')).toEqual([])
  expect(log.filter(r => r.path !== '/api/v1/session')).toEqual([])
}

async function open(page: Page, path: string) {
  await page.goto(`/dev/seller/${path}`)
  await expect(page.locator('.sl-root')).toBeVisible()
  await expect(page.locator('h1')).toHaveCount(1)
}

async function noPageHorizontalScroll(page: Page) {
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)
  expect(overflow).toBeLessThanOrEqual(0)
}

async function settled(page: Page, name: string | RegExp) {
  const d = page.getByRole('dialog', { name })
  await expect(d).toBeVisible()
  await d.evaluate(el => Promise.all(el.getAnimations().map(a => a.finished)))
  return d
}

async function snap(page: Page, info: TestInfo, name: string) {
  if (!OUT) return
  await page.evaluate(() => document.fonts.ready)
  await page.screenshot({ path: `${OUT}/${info.project.name.replace('seller-', '')}-${name}.png`, fullPage: true })
}

test.describe('all viewports', () => {
  test('all seven screens render in RTL with one h1, no page-level horizontal scroll and no business traffic', async ({ page }, info) => {
    const log = await guard(page)
    for (const [path, title] of SECTIONS) {
      await open(page, path)
      await expect(page.locator('.sl-root')).toHaveAttribute('dir', 'rtl')
      await expect(page.locator('h1')).toHaveText(title)
      await expect(page.locator('.ds-demo-tag, .ds-m-title').first()).toBeAttached()
      await noPageHorizontalScroll(page)
      await snap(page, info, path)
    }
    expectNoBusinessTraffic(log)
  })

  test('unknown /dev/seller section falls back to leads', async ({ page }) => {
    await guard(page)
    await page.goto('/dev/seller/not-a-section')
    await expect(page).toHaveURL(/\/dev\/seller\/leads$/)
  })

  test('honest system states: loading, empty, error and unknown are distinct and never zero', async ({ page }) => {
    await guard(page)
    await open(page, 'leads?sim=loading')
    await expect(page.locator('[data-kind="loading"]')).toBeVisible()
    await open(page, 'leads?sim=empty')
    await expect(page.getByText('در این صف موردی وجود ندارد')).toBeVisible()
    await open(page, 'invoices?sim=error')
    await expect(page.getByRole('alert').filter({ hasText: 'امکان دریافت لیست وجود ندارد' })).toBeVisible()
    await page.getByRole('button', { name: 'تلاش مجدد' }).click()
    await expect(page.getByRole('alert')).toHaveCount(0)
    await open(page, 'wallet')
    await expect(page.getByText(/در هیچ‌یک از جمع‌ها لحاظ نشده است/)).toBeVisible()
    await expect(page.locator('.sl-wcard').nth(2)).toContainText('—')
    await expect(page.locator('.sl-wcard').nth(2)).not.toContainText('۰')
    await open(page, 'leads?sim=offline')
    await expect(page.locator('[data-kind="offline"]')).toBeVisible()
  })

  test('extra-number request: required, format and length validation; submit creates nothing', async ({ page }, info) => {
    const log = await guard(page)
    await open(page, 'request')
    const phone = page.getByLabel(/شماره موبایل مشتری/)
    await page.getByRole('button', { name: 'بررسی فرم (نمایشی)' }).click()
    await expect(page.getByText('شماره موبایل مشتری الزامی است.')).toBeVisible()
    await expect(phone).toHaveAttribute('aria-invalid', 'true')
    await expect(phone).toBeFocused()
    await phone.fill('12345')
    await page.getByRole('button', { name: 'بررسی فرم (نمایشی)' }).click()
    await expect(page.getByText('شماره باید ۱۱ رقم و با ۰۹ شروع شود.')).toBeVisible()
    await phone.fill('۰۹۰۰۰۰۰۱۲۳۴')
    await page.getByRole('button', { name: 'بررسی فرم (نمایشی)' }).click()
    await expect(phone).not.toHaveAttribute('aria-invalid', 'true')
    await expect(page.locator('.sl-toast').filter({ hasText: 'OPD-10' })).toBeVisible()
    await expect(page.locator('.sl-mini-list > li')).toHaveCount(3) // the synthetic list never grows
    await snap(page, info, 'request-validated')
    expectNoBusinessTraffic(log)
  })
})

test.describe('desktop and wide', () => {
  test.skip(({ viewport }) => kindOf(viewport?.width ?? 0) !== 'desktop', 'desktop only')

  test('desktop navigation: aria-current follows the route; bottom nav hidden', async ({ page }) => {
    await guard(page)
    await open(page, 'leads')
    const nav = page.getByRole('navigation', { name: 'بخش‌های پنل فروشنده' })
    await expect(nav).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'ناوبری موبایل' })).toBeHidden()
    await expect(nav.locator('[aria-current="page"]')).toContainText('شماره‌های من')
    await nav.getByRole('link', { name: /فاکتورها/ }).click()
    await expect(page).toHaveURL(/\/dev\/seller\/invoices$/)
    await expect(page.locator('h1')).toHaveText('فاکتورها')
    await expect(nav.locator('[aria-current="page"]')).toContainText('فاکتورها')
    await nav.getByRole('link', { name: /آماده‌های تبدیل/ }).click()
    await expect(page.locator('h1')).toHaveText('آماده‌های تبدیل')
  })

  test('leads: queue filtering, local search, filters, sort and pagination', async ({ page }) => {
    const log = await guard(page)
    await open(page, 'leads')
    const rows = page.locator('.sl-tbl tbody tr')
    await expect(page.getByRole('button', { name: /بدون وضعیت/ }).first()).toHaveAttribute('aria-pressed', 'true')
    await expect(rows).toHaveCount(9)
    await page.locator('[data-queue="no_answer"]').click()
    await expect(rows).toHaveCount(5)
    for (const pill of await rows.locator('td:first-child').allTextContents()) expect(pill).toContain('جواب نداده')
    await page.locator('[data-queue="all"]').click()
    await expect(rows).toHaveCount(10)
    await expect(page.locator('.sl-tfoot')).toContainText('۲۵')
    await page.getByRole('button', { name: 'صفحه بعدی' }).click()
    await expect(page.getByRole('button', { name: 'صفحه ۲' })).toHaveAttribute('aria-current', 'page')
    await expect(rows).toHaveCount(10)
    await page.getByRole('button', { name: 'صفحه ۳' }).click()
    await expect(rows).toHaveCount(5)
    await expect(page.getByRole('button', { name: 'صفحه بعدی' })).toBeDisabled()
    // Search: Persian digits match ASCII fixture phones; resets to page 1.
    await page.getByRole('searchbox', { name: 'جستجو در این صف' }).fill('۰۹۰۰۰۰۰۰۱۰۴')
    await expect(rows).toHaveCount(1)
    await expect(rows.first()).toContainText('مشتری نمونه ت')
    await page.getByRole('searchbox', { name: 'جستجو در این صف' }).fill('تبریز')
    await expect(rows).toHaveCount(1)
    await page.getByRole('searchbox', { name: 'جستجو در این صف' }).fill('هیچ‌چیز')
    await expect(page.getByText('نتیجه‌ای پیدا نشد')).toBeVisible()
    await page.getByRole('button', { name: 'پاک کردن جستجو و فیلترها' }).click()
    await expect(rows).toHaveCount(10)
    // Source filter menu.
    await page.getByRole('button', { name: 'منبع' }).click()
    await page.getByRole('menuitem', { name: 'درخواستی' }).click()
    await expect(page.getByRole('button', { name: 'منبع: درخواستی' })).toBeVisible()
    for (const t of await rows.allTextContents()) expect(t).toContain('درخواستی')
    expectNoBusinessTraffic(log)
  })

  test('leads drawer: opens from a row, outcome + conditional validation, J steps, Esc closes and restores focus', async ({ page }, info) => {
    const log = await guard(page)
    await open(page, 'leads?queue=all')
    const row = page.locator('.sl-tbl tbody tr').filter({ hasText: 'مشتری نمونه ت' }).first()
    await row.focus()
    await page.keyboard.press('Enter')
    const d = await settled(page, 'مشتری نمونه ت')
    await expect(page).toHaveURL(/lead=900104/)
    await expect(d.getByText('آخرین فعالیت')).toBeVisible()
    // Submit disabled until an outcome is chosen (never a silent no-op).
    await expect(d.getByRole('button', { name: 'یک نتیجه انتخاب کنید' })).toBeDisabled()
    await d.getByRole('button', { name: /^عدم خرید/ }).click()
    await d.getByRole('button', { name: 'ثبت عدم خرید' }).click()
    await expect(d.getByText('نوشتن دلیل برای این نتیجه الزامی است.')).toBeVisible()
    await expect(d.getByLabel(/دلیل عدم خرید/)).toBeFocused()
    await d.getByLabel(/دلیل عدم خرید/).fill('قیمت (نمونه)')
    await d.getByRole('button', { name: 'ثبت عدم خرید' }).click()
    await expect(page.locator('.sl-toast').filter({ hasText: 'نمایشی — چیزی ثبت یا ارسال نشد' })).toBeVisible()
    await d.getByRole('button', { name: /^تماس مجدد/ }).click()
    await d.getByLabel('تاریخ شمسی', { exact: false }).fill('')
    await d.getByRole('button', { name: /^ثبت تماس مجدد/ }).click()
    await expect(d.getByText('این فیلد الزامی است.')).toBeVisible()
    await snap(page, info, 'leads-drawer-validation')
    // Keyboard: focus the drawer frame, J moves to the next lead, number keys pick outcomes.
    await d.focus()
    await page.keyboard.press('j')
    await settled(page, 'مشتری نمونه ذ') // next row of the current list (newest first)
    await page.keyboard.press('2')
    await expect(page.getByRole('dialog').getByRole('button', { name: /^تماس مجدد/ })).toHaveAttribute('aria-pressed', 'true')
    await page.keyboard.press('Escape')
    await expect(page.getByRole('dialog', { name: 'مشتری نمونه ذ' })).toBeHidden()
    await expect(page).not.toHaveURL(/lead=/)
    await expect(row).toBeFocused() // focus returns to the row that opened the drawer
    expectNoBusinessTraffic(log)
  })

  test('keyboard: digit keys switch queues, F focuses search, Ctrl+K palette finds synthetic records', async ({ page }) => {
    await guard(page)
    await open(page, 'leads')
    await page.locator('h1').click()
    await page.keyboard.press('2')
    await expect(page.locator('[data-queue="no_answer"]')).toHaveAttribute('aria-pressed', 'true')
    await page.keyboard.press('0')
    await expect(page.locator('[data-queue="all"]')).toHaveAttribute('aria-pressed', 'true')
    await page.locator('h1').click()
    await page.keyboard.press('f')
    await expect(page.getByRole('searchbox', { name: 'جستجو در این صف' })).toBeFocused()
    await page.keyboard.press('Escape')
    await page.locator('h1').click()
    await page.keyboard.press('Control+k')
    const pal = page.getByRole('dialog', { name: 'جستجو و فرمان‌ها' })
    await expect(pal).toBeVisible()
    await page.keyboard.type('TST-1007')
    await page.keyboard.press('Enter')
    await expect(page).toHaveURL(/\/invoices\?inv=TST-1007/)
    await settled(page, 'فاکتور TST-1007')
  })

  test('invoices: queues, submitted ≠ confirmed, Outcome Unknown banner, row menu with explained disabled items', async ({ page }, info) => {
    const log = await guard(page)
    await open(page, 'invoices')
    const rows = page.locator('.sl-tbl tbody tr')
    await expect(rows).toHaveCount(9) // RM-E04C / A7 added TST-1015 (IRR) to exercise the unit split
    await page.locator('[data-queue="rejected"]').click()
    await expect(rows).toHaveCount(1)
    await expect(rows.first()).toContainText('فیش خوانا نیست')
    await page.locator('[data-queue="all"]').click()
    await rows.filter({ hasText: 'TST-1004' }).click()
    let d = await settled(page, 'فاکتور TST-1004')
    await expect(d.getByText(/تا تایید مالی، پرداخت‌شده یا کسرشده از مانده حساب نمی‌شود/)).toBeVisible()
    await expect(d.locator('.sl-sum').nth(1)).toContainText('۰')
    await page.keyboard.press('Escape')
    await rows.filter({ hasText: 'TST-1007' }).click()
    d = await settled(page, 'فاکتور TST-1007')
    await expect(d.locator('[data-kind="unknown"]')).toContainText('پیش از هر ارسال مجدد باید تطبیق شود')
    await snap(page, info, 'invoices-drawer-unknown')
    await page.keyboard.press('Escape')
    await page.getByRole('button', { name: 'اقدام‌های بیشتر TST-1001' }).click()
    const menu = page.getByRole('menu')
    await expect(menu.getByRole('menuitem', { name: /ارسال مجدد لینک پرداخت/ })).toHaveAttribute('aria-disabled', 'true')
    await expect(menu).toContainText('پیش‌نمایش پیامکی ارسال نمی‌کند.')
    await page.keyboard.press('Escape')
    expectNoBusinessTraffic(log)
  })

  test('conversions + repeat: attribution facts keep the original seller; credit owner stays unknown', async ({ page }, info) => {
    await guard(page)
    await open(page, 'conversions?case=104')
    const d = await settled(page, 'مشتری نمونه ز')
    await expect(d.getByText('مالک اعتبار')).toBeVisible()
    await expect(d.locator('[data-state="unknown"]').first()).toBeVisible()
    await snap(page, info, 'conversions-drawer')
    await open(page, 'repeat?item=301')
    const r = await settled(page, 'مشتری نمونه ت')
    await expect(r.getByText('فروشنده نمونه دیگر · SYN-0002')).toBeVisible()
    await expect(r.getByText(/این انتساب تاریخی را بازنویسی نمی‌کند/)).toBeVisible()
    await page.keyboard.press('Escape')
    await page.locator('[data-queue="done"]').click()
    await expect(page.locator('.sl-tbl tbody tr')).toHaveCount(1)
  })

  test('behavior: local search and «فقط فعال»; opening the invoice from the drawer', async ({ page }) => {
    const log = await guard(page)
    await open(page, 'behavior')
    const rows = page.locator('.sl-tbl tbody tr')
    await expect(rows).toHaveCount(6)
    await page.getByRole('button', { name: 'فقط فعال' }).click()
    await expect(rows).toHaveCount(2)
    await page.getByRole('button', { name: 'همه', exact: true }).click()
    await page.getByRole('searchbox').fill('TST-1003')
    await expect(rows).toHaveCount(1)
    await rows.first().click()
    const d = await settled(page, 'مشتری نمونه ت')
    await d.getByRole('button', { name: 'مشاهده فاکتور' }).click()
    await expect(page).toHaveURL(/\/invoices\?inv=TST-1003/)
    await settled(page, 'فاکتور TST-1003')
    expectNoBusinessTraffic(log)
  })
})

test.describe('tablet', () => {
  test.skip(({ viewport }) => kindOf(viewport?.width ?? 0) !== 'tablet', 'tablet only')

  test('tablet: optional columns collapse, the table scrolls inside its panel, nav stays in the header', async ({ page }) => {
    await guard(page)
    await open(page, 'leads?queue=all')
    await expect(page.getByRole('navigation', { name: 'بخش‌های پنل فروشنده' })).toBeVisible()
    await expect(page.getByRole('navigation', { name: 'ناوبری موبایل' })).toBeHidden()
    await expect(page.locator('.sl-tbl th.sl-col-opt').first()).toBeHidden()
    await expect(page.locator('.sl-tbl')).toBeVisible()
    await noPageHorizontalScroll(page)
    await page.locator('.sl-tbl tbody tr').first().click()
    const d = await settled(page, /مشتری نمونه/)
    const box = await d.boundingBox()
    expect(box?.width ?? 0).toBeLessThanOrEqual(560)
  })
})

test.describe('mobile', () => {
  test.skip(({ viewport }) => kindOf(viewport?.width ?? 0) !== 'mobile', 'mobile only')

  test('mobile: cards replace the table, bottom navigation and «بیشتر» reach every section', async ({ page }) => {
    const log = await guard(page)
    await open(page, 'leads')
    await expect(page.getByRole('navigation', { name: 'بخش‌های پنل فروشنده' })).toBeHidden()
    const bottom = page.getByRole('navigation', { name: 'ناوبری موبایل' })
    await expect(bottom).toBeVisible()
    await expect(page.locator('.sl-tbl')).toBeHidden()
    await expect(page.locator('.sl-lcard').first()).toBeVisible()
    await bottom.getByRole('link', { name: /فاکتورها/ }).click()
    await expect(page.locator('h1')).toHaveText('فاکتورها')
    await expect(page.locator('.sl-lcard').first()).toBeVisible()
    await bottom.getByRole('button', { name: 'بیشتر' }).click()
    await page.getByRole('menuitem', { name: /رفتار مشتریان/ }).click()
    await expect(page.locator('h1')).toHaveText('رفتار مشتریان')
    await expect(bottom.getByRole('button', { name: 'بیشتر' })).toHaveAttribute('aria-current', 'page')
    // Stackable table rows carry their column label on phones.
    await expect(page.locator('.sl-tbl.sl-stackable td[data-l="فاکتور"]').first()).toBeVisible()
    await noPageHorizontalScroll(page)
    expectNoBusinessTraffic(log)
  })

  test('mobile: lead card opens a bottom sheet; Esc closes it', async ({ page }, info) => {
    await guard(page)
    await open(page, 'leads')
    await page.locator('.sl-lcard').first().getByRole('button', { name: 'ثبت نتیجه' }).click()
    const d = await settled(page, /مشتری نمونه/)
    const box = await d.boundingBox()
    expect(Math.round(box?.width ?? 0)).toBe(390)
    await expect(d.locator('.sl-outcome').first()).toBeVisible()
    await snap(page, info, 'leads-sheet')
    await page.keyboard.press('Escape')
    await expect(d).toBeHidden()
    await noPageHorizontalScroll(page)
  })
})

// ---------------------------------------------------------------------------------------------------------
// RM-E04C / A7 — Seller UI states in a real browser (all viewports). Evidence (optional):
// RM_E04C_A7_EVIDENCE_DIR=../docs/evidence/rm-e04c-a7 npx playwright test -c playwright.seller.config.ts -g "A7"

const A7_OUT = process.env.RM_E04C_A7_EVIDENCE_DIR

async function a7snap(page: Page, info: TestInfo, name: string) {
  if (!A7_OUT) return
  await page.evaluate(() => document.fonts.ready)
  await page.screenshot({ path: `${A7_OUT}/${info.project.name.replace('seller-', '')}-${name}.png`, fullPage: true })
}

test.describe('A7 states', () => {
  test('A7: blocking answers show a generic panel and no data on every screen; no traffic', async ({ page }, info) => {
    const log = await guard(page)
    for (const sim of ['action_denied', 'row_scope_denied', 'feature_disabled', 'session_expired'] as const) {
      for (const [path] of SECTIONS) {
        await open(page, `${path}?sim=${sim}&inv=TST-1003&lead=900104`)
        await expect(page.locator('[data-ui-state] [data-kind="locked"]')).toBeVisible()
        await expect(page.locator('.sl-tbl, .sl-lcard, [role="dialog"]')).toHaveCount(0)
        await expect(page.locator('body')).not.toContainText(/مشتری نمونه|TST-\d{4}|0900000|۰۹۰۰۰۰۰/)
        await expect(page.getByRole('button', { name: 'تلاش مجدد' })).toHaveCount(0)
        await noPageHorizontalScroll(page)
      }
      await a7snap(page, info, `a7-${sim}`)
    }
    expectNoBusinessTraffic(log)
  })

  test('A7: failure shows the correlation id; retry re-reads only', async ({ page }, info) => {
    const log = await guard(page)
    await open(page, 'invoices?sim=error')
    const alert = page.getByRole('alert').filter({ hasText: 'امکان دریافت لیست وجود ندارد' })
    await expect(alert).toContainText('syn-corr-a7-500')
    await a7snap(page, info, 'a7-failure')
    await alert.getByRole('button', { name: 'تلاش مجدد' }).click()
    await expect(page.getByRole('alert')).toHaveCount(0)
    await expect(page.locator('.sl-tbl tbody tr:visible, .sl-lcard:visible').first()).toBeVisible()
    expectNoBusinessTraffic(log)
  })

  test('A7: exact empty shows ۰; absent count shows «—»', async ({ page }, info) => {
    await guard(page)
    await open(page, 'leads?sim=empty&queue=all')
    await expect(page.locator('[data-queue="all"]')).toContainText('۰')
    await expect(page.getByText('در این صف موردی وجود ندارد')).toBeVisible()
    await open(page, 'leads?sim=count_absent&queue=all')
    await expect(page.locator('[data-queue="all"] .ds-count-unknown')).toBeAttached()
    await expect(page.locator('[data-queue="all"]')).not.toContainText('۰')
    await a7snap(page, info, 'a7-count-absent')
  })

  test('A7: PARTIAL withholds dependent amounts; FIELD_DENIED shows «محدود» only; UNKNOWN maps to نامعلوم', async ({ page }, info) => {
    await guard(page)
    await open(page, 'invoices?sim=partial')
    await expect(page.locator('[data-kind="incomplete"]').first()).toContainText('COVERAGE_INCOMPLETE')
    await expect(page.locator('[data-withheld="true"]').first()).toBeAttached()
    await a7snap(page, info, 'a7-partial-invoices')
    await open(page, 'leads?sim=field_denied&queue=all')
    await expect(page.locator('[data-restricted="true"]:visible').first()).toBeVisible()
    await expect(page.locator('main')).not.toContainText(/مشتری نمونه|0900000|۰۹۰۰۰۰۰/)
    await a7snap(page, info, 'a7-field-denied')
    await open(page, 'invoices?sim=unknown&inv=TST-1003')
    const d = await settled(page, 'فاکتور TST-1003')
    await expect(d.locator('[data-state="unknown"]').first()).toBeVisible()
    await a7snap(page, info, 'a7-unknown')
  })

  test('A7: FIELD_DENIED leaves no phone or name in attributes or accessible names (S6 timelines included)', async ({ page }) => {
    await guard(page)
    await open(page, 'request?sim=field_denied')
    for (const b of await page.getByRole('button', { name: /^روند/ }).all()) await b.click()
    const attrs = await page.evaluate(() => Array.from(document.querySelectorAll('*')).flatMap(el => Array.from(el.attributes).map(a => a.value)).join('\n'))
    expect(attrs).not.toMatch(/0900000\d{4}|۰۹۰۰۰۰۰[۰-۹]{4}|مشتری نمونه/)
    await expect(page.locator('body')).not.toContainText(/0900000|۰۹۰۰۰۰۰|مشتری نمونه/)
    await expect(page.getByRole('list', { name: /روند درخواست ثبت‌شده در/ }).first()).toBeAttached()
  })

  test('A7: missing and out-of-scope detail render the same «یافت نشد»', async ({ page }, info) => {
    await guard(page)
    await open(page, 'invoices?inv=TST-9999')
    const a = await (await settled(page, 'یافت نشد')).innerText()
    await open(page, 'invoices?inv=TST-1001&sim=not_found')
    const b = await (await settled(page, 'یافت نشد')).innerText()
    expect(b).toBe(a)
    expect(a).not.toMatch(/TST-\d{4}/)
    await a7snap(page, info, 'a7-not-found')
  })

  test('A7: wallet keeps units apart, settlement undefined, history is not cash', async ({ page }, info) => {
    await guard(page)
    await open(page, 'wallet')
    await expect(page.getByTestId('wallet-settlement')).toHaveText('— تعریف نشده')
    await expect(page.locator('.sl-hero b[data-unit="IRT"]')).toContainText('تومان')
    await expect(page.locator('.sl-hero b[data-unit="IRR"]')).toContainText('ریال')
    await noPageHorizontalScroll(page)
    await a7snap(page, info, 'a7-wallet')
  })
})

