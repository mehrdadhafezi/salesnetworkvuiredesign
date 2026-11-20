import type { Page, Request } from '@playwright/test'
import { expect, guarded, required } from './fixtures'

// RM-P02 — invoice 360° workspace.
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE: crm-api runs with
// CRM_ENV=test and the existing development auth stub (CRM_DEV_AUTH=1,
// refused in production); this browser context adds the stub header itself.
// Data: the R7-2 synthetic payments scenario plus three target payments
// submitted through the real command before the browser starts. Every
// expected value comes from the orchestrator's SQL or from the API answer
// itself (rendered UI == API answer == database). The browser never writes.

const DB = {
  reviewInvoice: required('E2E_WS_REVIEW_INVOICE'),
  reviewKey: required('E2E_WS_REVIEW_KEY'),
  reviewN: Number(required('E2E_WS_REVIEW_N')),
  reviewAll: Number(required('E2E_WS_REVIEW_ALL')),
  reviewCustomer: required('E2E_WS_REVIEW_CUSTOMER'),
  reviewCase: required('E2E_WS_REVIEW_CASE'),
  attInvoice: required('E2E_WS_ATT_INVOICE'),
  attKey: required('E2E_WS_ATT_KEY'),
  attN: Number(required('E2E_WS_ATT_N')),
  attHistory: process.env.E2E_WS_ATT_HISTORY ?? '',
  attStages: Number(required('E2E_WS_ATT_STAGES')),
  attItems: Number(required('E2E_WS_ATT_ITEMS')),
}
const SHOTS = process.env.E2E_SCREENSHOT_DIR ?? ''
const PAGE = 25

const S = {
  overview: 'نمای کلی و پیوندها', facts: 'فاکتور', states: 'شواهد وضعیت (هر فیلد جدا)', amounts: 'شواهد جمع (به تفکیک واحد؛ بدون برنده)',
  items: 'اقلام (اسنپ‌شات صدور)', stages: 'مراحل پرداخت', attempts: 'تلاش‌های پرداخت', receipts: 'رسیدها (فقط فراداده ارجاع)', review: 'بررسی مالی', provenance: 'منشأ',
}
const relationLabel: Record<string, string> = {
  RESOLVED: 'پیوند تأییدشده', RELATED: 'مرتبط (تأیید نشده)', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض', NOT_RECOVERABLE: 'قابل بازیابی نیست',
}
const region = (page: Page, name: string) => page.getByRole('region', { name, exact: true })

function recordApi(page: Page): Request[] {
  const seen: Request[] = []
  page.on('request', r => { if (new URL(r.url()).pathname.startsWith('/api/')) seen.push(r) })
  return seen
}

async function noOverflow(page: Page, what: string) {
  // On failure, name the widest offenders (tag.class and width) so the cause is visible in the report.
  const offenders = await page.evaluate(() => {
    if (document.documentElement.scrollWidth <= window.innerWidth + 1) return []
    return Array.from(document.querySelectorAll<HTMLElement>('main *'))
      .filter(el => el.getBoundingClientRect().width > window.innerWidth + 1)
      .slice(-8).map(el => `${el.tagName.toLowerCase()}.${el.className} ${Math.round(el.getBoundingClientRect().width)}px`)
  })
  expect(offenders, `${what}: horizontal overflow`).toEqual([])
}

async function shots(page: Page, name: string) {
  for (const [vp, size] of [['desktop', { width: 1366, height: 900 }], ['tablet', { width: 820, height: 1180 }], ['mobile', { width: 390, height: 844 }]] as const) {
    await page.setViewportSize(size)
    await noOverflow(page, `${name} ${vp}`)
    if (SHOTS) await page.screenshot({ path: `${SHOTS}/${name}-${vp}.png`, fullPage: true })
  }
  await page.setViewportSize({ width: 1366, height: 900 })
}

const operator = guarded

operator.describe('dev-user-001 (synthetic read capabilities, test-only overlay): invoice 360° workspace', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('payments evidence: sections render from the real API, attempts open their own history, amounts stay per unit', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto(`/billing/invoices/${DB.attInvoice}`)
    await expect(page.getByRole('heading', { level: 1, name: `فاکتور: ${DB.attKey}` })).toBeVisible()
    for (const name of Object.values(S)) await expect(region(page, name)).toBeVisible()

    // Counts == database.
    await expect(region(page, S.attempts).locator('tbody tr')).toHaveCount(Math.min(DB.attN, PAGE))
    await expect(region(page, S.stages).getByRole('list', { name: 'مراحل پرداخت' }).locator('li')).toHaveCount(Math.min(DB.attStages, PAGE))
    if (DB.attItems > 0) await expect(region(page, S.items).getByRole('list', { name: 'اقلام' }).locator('li')).toHaveCount(Math.min(DB.attItems, PAGE))
    await expect(region(page, S.attempts).getByText('پذیرش درگاه یا تأیید قدیمی، تأیید نهایی مالی یا ثبت حسابداری نیست.')).toBeVisible()

    // Amount evidence: one card per unit of the API answer, never a cross-unit total.
    const totals = await (await page.request.get(`/api/v1/invoices/${DB.attInvoice}/totals`)).json() as { data: { units: { unit: string; class: string }[] } }
    const cards = region(page, S.amounts).getByRole('region', { name: /^واحد / })
    await expect(cards).toHaveCount(totals.data.units.length)
    await expect(region(page, S.amounts).getByText(/جمع قطعی فاکتور نیست/)).toBeVisible()
    await expect(region(page, S.overview).getByText(/جمع قطعی فاکتور تعیین نشده است/)).toBeVisible()

    // Per-attempt history (append-only), opened on demand.
    const attempts = region(page, S.attempts)
    const toggle = attempts.getByRole('button', { name: 'نمایش سابقه' }).first()
    await toggle.click()
    await expect(attempts.getByRole('button', { name: 'بستن سابقه' }).first()).toHaveAttribute('aria-expanded', 'true')
    const hist = attempts.getByLabel('سابقه وضعیت تلاش').first()
    await expect(hist.locator('li').or(hist.getByText('تغییر وضعیتی ثبت نشده است')).first()).toBeVisible()
    expect(reqs.some(r => /\/api\/v1\/payments\/attempts\/[0-9a-f-]{36}\/history/.test(new URL(r.url()).pathname))).toBe(true)

    for (const r of reqs) expect(r.method(), r.url()).toBe('GET')
    await shots(page, 'invoice-workspace')
  })

  operator('finance review: only this invoice’s submitted attempts, gate shown, link opens the existing review page', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto(`/billing/invoices/${DB.reviewInvoice}`)
    await expect(page.getByRole('heading', { level: 1, name: `فاکتور: ${DB.reviewKey}` })).toBeVisible()
    const rv = region(page, S.review)
    await expect(rv.locator('tbody tr')).toHaveCount(DB.reviewN)
    expect(DB.reviewAll).toBeGreaterThan(DB.reviewN) // another invoice's attempt exists and is not shown
    expect(reqs.find(r => { const u = new URL(r.url()); return u.pathname === '/api/v1/finance/review/attempts' && u.searchParams.get('invoice_id') === DB.reviewInvoice })).toBeTruthy()
    await expect(rv.getByText('تایید مالی: در دسترس نیست').first()).toBeVisible()
    await expect(rv.getByRole('button')).toHaveCount(0) // no decision control
    await shots(page, 'invoice-review')

    // Relationships of this invoice: shown with their state, linked only when RESOLVED.
    const ov = region(page, S.overview)
    await expect(ov.getByText(relationLabel[DB.reviewCustomer] ?? DB.reviewCustomer).first()).toBeVisible()
    await expect(ov.getByRole('link', { name: 'مشاهده مشتری' })).toHaveCount(DB.reviewCustomer === 'RESOLVED' ? 1 : 0)
    await expect(ov.getByRole('link', { name: 'مشاهده پرونده' })).toHaveCount(DB.reviewCase === 'RESOLVED' ? 1 : 0)

    // In-page navigation works.
    await page.getByRole('navigation', { name: 'بخش‌های فاکتور' }).getByRole('link', { name: 'بررسی مالی' }).click()
    await expect(page).toHaveURL(/#inv-review$/)
    await expect(rv).toBeInViewport()

    // Review link → the existing Finance review page for that attempt.
    await rv.getByRole('link').first().click()
    await expect(page).toHaveURL(/\/finance\/review\/[0-9a-f-]{36}$/)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    for (const r of reqs) expect(r.method(), r.url()).toBe('GET')
  })
})

operator.describe('dev-user-002 (no capability, test-only overlay): the workspace stays locked', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  operator('invoice detail denied: dependent reads are never requested and no invoice, payment or review data reaches the DOM', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto(`/billing/invoices/${DB.reviewInvoice}`)
    for (const name of [S.overview, S.facts]) await expect(region(page, name).getByText('دسترسی به این بخش ندارید')).toBeVisible()
    // Invoice-dependent sections wait for a successful invoice detail; they never mount or fetch here.
    for (const name of [S.items, S.stages, S.review]) {
      await expect(region(page, name).getByText('این بخش فقط پس از دریافت موفق فاکتور نمایش داده می‌شود')).toBeVisible()
    }
    for (const name of [S.attempts, S.receipts, S.amounts]) await expect(region(page, name).getByText('دسترسی به شواهد پرداخت ندارید')).toBeVisible()
    const dependent = reqs.map(r => new URL(r.url()).pathname).filter(p => p.startsWith('/api/v1/finance/') || p.startsWith('/api/v1/payments/') ||
      (p.startsWith(`/api/v1/invoices/${DB.reviewInvoice}/`)))
    expect(dependent).toEqual([])
    await expect(page.locator('table')).toHaveCount(0)
    const html = await page.content()
    for (const leak of [DB.reviewKey, 'zibal', 'PENDING_REVIEW', 'تایید مالی: در دسترس نیست']) expect(html).not.toContain(leak)
    await noOverflow(page, 'denied')
  })
})
