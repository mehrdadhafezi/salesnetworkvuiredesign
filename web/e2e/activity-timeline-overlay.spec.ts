import type { Locator, Page, Request } from '@playwright/test'
import { expect, guarded, required } from './fixtures'

// RM-P03 — unified, read-only activity timeline (Case and Customer pages).
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE: crm-api runs with
// CRM_ENV=test and the existing development auth stub (CRM_DEV_AUTH=1,
// refused in production). This browser context adds the stub header itself;
// the production SPA bundle carries no dev identity. Data is the R7-5
// integrated SYNTHETIC dataset. Every expected value comes from the
// orchestrator's SQL over the disposable database.

const DB = {
  kase: required('E2E_TL_CASE'),
  caseKey: required('E2E_TL_CASE_KEY'),
  history: Number(required('E2E_TL_CASE_HISTORY')),
  custody: Number(required('E2E_TL_CASE_CUSTODY')),
  attribution: Number(required('E2E_TL_CASE_ATTRIBUTION')),
  firstRaw: required('E2E_TL_CASE_FIRST_RAW'),
  customer: required('E2E_TL_CUSTOMER'),
  customerInvoices: Number(required('E2E_TL_CUSTOMER_INVOICES')),
  foreignInvoice: process.env.E2E_TL_FOREIGN_INVOICE ?? '',
}
const SHOTS = process.env.E2E_SCREENSHOT_DIR ?? ''
const HISTORY_PAGE = 100
const INVOICE_PAGE = 10

const HISTORY = 'سابقه وضعیت (فقط افزودنی)'
const CUSTODY = 'خط زمانی نگهداری (فقط افزودنی)'
const OUTCOMES = 'پیامدهای تماس (سامانه جدید)'
const ATTRIBUTION = 'انتساب تاریخی (اسنپ‌شات)'

function recordApi(page: Page): Request[] {
  const seen: Request[] = []
  page.on('request', r => { if (new URL(r.url()).pathname.startsWith('/api/')) seen.push(r) })
  return seen
}

function expectReadsOnly(reqs: Request[]) {
  expect(reqs.length).toBeGreaterThan(0)
  for (const r of reqs) expect(r.method(), r.url()).toBe('GET')
  for (const r of reqs) {
    const q = new URL(r.url()).searchParams
    for (const k of ['phone', 'customer_phone', 'mobile', 'name', 'source_key', 'legacy_key']) expect(q.has(k), `${k} in ${r.url()}`).toBe(false)
  }
}

const events = (lane: Locator) => lane.locator('li.tl-event')

async function shot(page: Page, region: Locator, name: string) {
  if (!SHOTS) return
  for (const [vp, size] of [['desktop', { width: 1366, height: 900 }], ['tablet', { width: 820, height: 1180 }], ['mobile', { width: 390, height: 844 }]] as const) {
    await page.setViewportSize(size)
    // No horizontal page scroll at any width.
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `${name} ${vp} horizontal overflow`).toBe(true)
    await region.screenshot({ path: `${SHOTS}/${name}-${vp}.png` })
  }
  await page.setViewportSize({ width: 1366, height: 900 })
}

const operator = guarded

operator.describe('dev-user-001 (synthetic read capabilities, test-only overlay): activity timeline', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('case: each source is its own lane with the database count; feature-off contact outcomes are never requested', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto(`/sales/cases/${DB.kase}`)
    await expect(page.getByRole('heading', { level: 1, name: `پرونده: ${DB.caseKey}` })).toBeVisible()
    const card = page.getByRole('region', { name: 'خط زمانی فعالیت پرونده' })

    const history = card.getByRole('region', { name: HISTORY })
    await expect(events(history)).toHaveCount(Math.min(DB.history, HISTORY_PAGE))
    await expect(events(history).first()).toContainText(DB.firstRaw)
    await expect(history.getByText(DB.history > HISTORY_PAGE ? 'فقط صفحه 1 بارگذاری شده است' : 'همه رویدادهای این منبع بارگذاری شده است')).toBeVisible()
    await expect(events(card.getByRole('region', { name: CUSTODY }))).toHaveCount(DB.custody)
    await expect(events(card.getByRole('region', { name: ATTRIBUTION }))).toHaveCount(DB.attribution)
    await expect(card.getByRole('region', { name: OUTCOMES }).getByText('این بخش هنوز فعال نشده است')).toBeVisible()

    // Combined view: legacy wall-clock times stay one group; nothing is claimed complete when it is not.
    await card.getByRole('button', { name: 'ترکیب زمانی' }).click()
    const combined = card.getByRole('region', { name: 'ترکیب زمانی' })
    await expect(combined.getByText(/این ترکیب کامل نیست/)).toContainText(OUTCOMES)
    const grouped = await combined.locator('li.tl-event').count()
    expect(grouped).toBe(Math.min(DB.history, HISTORY_PAGE) + DB.custody + DB.attribution)
    await expect(combined.getByRole('region', { name: 'زمان‌های سامانه جدید (لحظه رخداد)' })).toHaveCount(0)
    await shot(page, card, 'case-timeline-combined')
    await card.getByRole('button', { name: 'به تفکیک منبع' }).click()
    await shot(page, card, 'case-timeline')

    expect(reqs.some(r => new URL(r.url()).pathname.endsWith('/contact-outcomes'))).toBe(false)
    expectReadsOnly(reqs)
  })

  operator('customer: safe subset only; invoice issuance comes from RESOLVED links of this customer id', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto(`/customers/${DB.customer}`)
    const card = page.getByRole('region', { name: 'خط زمانی فعالیت مشتری' })
    await expect(card.getByText(/این خط زمانی کامل مشتری نیست/)).toBeVisible()
    await expect(events(card.getByRole('region', { name: 'رکورد مشتری' }))).toHaveCount(1)
    const invoices = card.getByRole('region', { name: 'صدور فاکتورهای با پیوند تأییدشده' })
    await expect(events(invoices)).toHaveCount(Math.min(DB.customerInvoices, INVOICE_PAGE))
    if (DB.foreignInvoice) await expect(invoices.getByText(DB.foreignInvoice, { exact: false })).toHaveCount(0)
    const call = reqs.find(r => { const u = new URL(r.url()); return u.pathname === '/api/v1/invoices' && u.searchParams.get('customer_id') === DB.customer })
    expect(call, 'invoice list filtered by the exact customer id').toBeTruthy()
    await shot(page, card, 'customer-timeline')
    expectReadsOnly(reqs)
  })
})

operator.describe('dev-user-002 (no capability, test-only overlay): every timeline source is denied', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  operator('case timeline: no event of any source reaches the DOM', async ({ page, blocked }) => {
    void blocked
    await page.goto(`/sales/cases/${DB.kase}`)
    const card = page.getByRole('region', { name: 'خط زمانی فعالیت پرونده' })
    for (const s of [HISTORY, ATTRIBUTION]) await expect(card.getByRole('region', { name: s }).getByText('دسترسی به این بخش ندارید')).toBeVisible()
    await expect(card.getByRole('region', { name: CUSTODY })).toBeVisible()
    await expect(card.locator('li.tl-event')).toHaveCount(0)
    expect(await page.content()).not.toContain(DB.firstRaw)
  })
})
