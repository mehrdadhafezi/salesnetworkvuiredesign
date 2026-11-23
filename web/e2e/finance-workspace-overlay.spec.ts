import type { Locator, Page, Request } from '@playwright/test'
import { expect, guarded, required } from './fixtures'

// RM-P07 — Finance operational workspace (landing, review queue, exceptions,
// receipt evidence), read only.
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE: crm-api runs with
// CRM_ENV=test and the existing development auth stub (CRM_DEV_AUTH=1,
// refused in production). This browser context adds the stub header itself;
// the production SPA bundle carries no dev identity. The Finance role is not
// live verified: nothing here is real-account evidence. Data is SYNTHETIC;
// every expected value is the orchestrator's SQL.

const DB = {
  invoices: Number(required('E2E_FW_INVOICES')),
  attempts: Number(required('E2E_FW_ATTEMPTS')),
  reviewAll: Number(required('E2E_FW_REVIEW_ALL')),
  reviewReady: Number(required('E2E_FW_REVIEW_READY')),
  reviewPending: Number(required('E2E_FW_REVIEW_PENDING')),
  reviewInvoice: required('E2E_FW_REVIEW_INVOICE'),
  reviewInvN: Number(required('E2E_FW_REVIEW_INV_N')),
  reviewKey: required('E2E_FW_REVIEW_KEY'),
  regression: Number(required('E2E_FW_REGRESSION')),
  receiptsRef: Number(required('E2E_FW_RECEIPTS_REF')),
  unknownInvoice: Number(required('E2E_FW_UNKNOWN_INVOICE')),
}
// Dedicated variable: a regression run that sets the generic E2E_SCREENSHOT_DIR never writes RM-P07
// evidence, and these files are always prefixed "finance-" (docs/governance pins the directory content).
const SHOTS = process.env.E2E_FINANCE_SCREENSHOT_DIR ?? ''
const fa = (n: number) => new Intl.NumberFormat('fa-IR').format(n)
const SECTIONS = ['میز کار مالی', 'فاکتورها', 'تلاش‌های پرداخت (قدیمی)', 'صف بررسی و پرداخت‌های ثبت‌شده', 'شواهد رسید', 'استثناها و شواهد حل‌نشده']
const COMMAND_WORDS = /تأیید|تایید|رد مالی|^رد$|استرداد|تسویه|ثبت در دفتر|تلاش دوباره|اصلاح|approve|reject|refund|retry|settle|post/i

const fwNav = (page: Page) => page.getByRole('navigation', { name: 'بخش‌های میز کار مالی' })
const reviewTable = (page: Page) => page.getByRole('table', { name: 'پرداخت‌های در انتظار بررسی مالی' })

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
    for (const k of ['phone', 'customer_phone', 'mobile', 'name', 'q']) expect(q.has(k), `${k} in ${r.url()}`).toBe(false)
  }
}

async function noCommandControls(page: Page) {
  const names = await page.locator('main').getByRole('button').evaluateAll(els => els.map(e => (e.textContent ?? '').trim()))
  for (const n of names) expect(n, `unexpected control "${n}"`).not.toMatch(COMMAND_WORDS)
}

async function shot(page: Page, region: Locator, name: string) {
  if (!SHOTS) return
  for (const [vp, size] of [['desktop', { width: 1366, height: 900 }], ['tablet', { width: 820, height: 1180 }], ['mobile', { width: 390, height: 844 }]] as const) {
    await page.setViewportSize(size)
    const offenders = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1 ? [] : Array.from(document.querySelectorAll('body *'))
      .filter(el => { const r = el.getBoundingClientRect(); return r.right > window.innerWidth + 1 || r.left < -1 })
      .filter(el => { for (let p = el.parentElement; p && p !== document.body; p = p.parentElement) if (getComputedStyle(p).overflowX !== 'visible') return false; return true })
      .slice(0, 8).map(el => `${el.tagName.toLowerCase()}.${(el.getAttribute('class') ?? '').replace(/ /g, '.')} ${Math.round(el.getBoundingClientRect().left)}..${Math.round(el.getBoundingClientRect().right)}`))
    expect(offenders, `${name} ${vp} horizontal overflow`).toEqual([])
    await region.screenshot({ path: `${SHOTS}/finance-${name}-${vp}.png` })
  }
  await page.setViewportSize({ width: 1366, height: 900 })
}

async function noOverflow(page: Page, name: string) {
  for (const size of [{ width: 1366, height: 900 }, { width: 820, height: 1180 }, { width: 390, height: 844 }]) {
    await page.setViewportSize(size)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `${name} ${size.width}px overflow`).toBe(true)
  }
  await page.setViewportSize({ width: 1366, height: 900 })
}

const operator = guarded

operator.describe('dev-user-001 (synthetic read capabilities, test-only overlay): Finance workspace', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('landing: sections, counters equal the database, queue total unavailable, operations are statuses only', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/finance')
    await expect(page.getByRole('heading', { level: 1, name: 'میز کار مالی', exact: true })).toBeVisible()
    await expect(fwNav(page).getByRole('link')).toHaveText(SECTIONS)
    const inv = page.getByRole('list', { name: 'شمارنده‌های فاکتور' })
    await expect(inv.getByText(fa(DB.invoices), { exact: true }).first()).toBeVisible()
    const att = page.getByRole('list', { name: 'شمارنده‌های تلاش قدیمی' })
    await expect(att.getByText(fa(DB.attempts), { exact: true }).first()).toBeVisible()
    const sub = page.getByRole('list', { name: 'شمارنده‌های پرداخت ثبت‌شده' })
    await expect(sub.getByText('در دسترس نیست').first()).toBeVisible() // no queue total in any contract; never 0
    const ops = page.getByRole('list', { name: 'وضعیت عملیات مالی' })
    await expect(ops.getByText('در دسترس نیست', { exact: true })).toHaveCount(9)
    await expect(ops.getByRole('button')).toHaveCount(0)
    await expect(ops.getByRole('link')).toHaveCount(0)
    await noCommandControls(page)
    await shot(page, page.locator('main'), 'landing')
    expectReadsOnly(reqs)
    expect(reqs.filter(r => new URL(r.url()).pathname === '/api/v1/billing/reconciliation-report')).toHaveLength(1)
  })

  operator('review queue: server filters, submission source, maker / checker gate, link to the RM-P02 invoice detail', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/finance/review')
    await expect(page.getByRole('heading', { level: 1, name: 'صف بررسی مالی', exact: true })).toBeVisible()
    await expect(reviewTable(page).locator('tbody tr')).toHaveCount(DB.reviewAll)
    await expect(reviewTable(page).getByText('ثبت در سامانه جدید (نه قدیمی)').first()).toBeVisible()
    await expect(reviewTable(page).getByText('تایید مالی: در دسترس نیست').first()).toBeVisible()
    await shot(page, page.locator('main'), 'queue')

    const form = page.getByRole('form', { name: 'فیلتر صف بررسی مالی' })
    await form.getByLabel('آمادگی فیش').selectOption('READY')
    await form.getByRole('button', { name: 'اعمال فیلتر' }).click()
    await expect(page).toHaveURL(/evidence_state=READY/)
    await expect(reviewTable(page).locator('tbody tr')).toHaveCount(DB.reviewReady)
    expect(reqs.some(r => new URL(r.url()).searchParams.get('evidence_state') === 'READY')).toBe(true)

    await page.goto(`/finance/review?evidence_state=PENDING_FINALISATION`)
    await expect(reviewTable(page).locator('tbody tr')).toHaveCount(DB.reviewPending)
    await page.goto(`/finance/review?invoice_id=${DB.reviewInvoice}`)
    await expect(reviewTable(page).locator('tbody tr')).toHaveCount(DB.reviewInvN)

    // A non-UUID invoice reference (legacy key / phone) is never sent.
    const before = reqs.filter(r => new URL(r.url()).pathname === '/api/v1/finance/review/attempts').length
    await page.getByRole('form', { name: 'فیلتر صف بررسی مالی' }).getByLabel('شناسه فاکتور (دقیق)').fill('09120000000')
    await page.getByRole('form', { name: 'فیلتر صف بررسی مالی' }).getByRole('button', { name: 'اعمال فیلتر' }).click()
    await expect(page.getByText(/شناسه بومی کامل \(UUID\)/)).toBeVisible()
    expect(reqs.filter(r => new URL(r.url()).pathname === '/api/v1/finance/review/attempts').length).toBe(before)

    // Evidence navigation: invoice detail (RM-P02) lands on its legacy attempts section.
    await reviewTable(page).getByRole('link', { name: 'تلاش‌های قدیمی' }).first().click()
    await expect(page).toHaveURL(new RegExp(`/billing/invoices/${DB.reviewInvoice}#inv-attempts$`))
    await expect(page.getByRole('heading', { level: 1, name: `فاکتور: ${DB.reviewKey}` })).toBeVisible()
    await noCommandControls(page)
    expectReadsOnly(reqs)
  })

  operator('exceptions: regression and unresolved-invoice queues read with their server filters; no-contract queue says so', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/finance/exceptions?type=attempt-regression')
    await expect(page.getByRole('heading', { level: 1, name: 'استثناها و شواهد حل‌نشده', exact: true })).toBeVisible()
    const rows = page.locator('main table.fw-table tbody tr')
    await expect(rows).toHaveCount(DB.regression)
    expect(reqs.some(r => new URL(r.url()).pathname === '/api/v1/payments/attempts' && new URL(r.url()).searchParams.get('has_state_regression') === 'true')).toBe(true)
    await shot(page, page.locator('main'), 'exceptions')

    const queues = page.getByRole('navigation', { name: 'صف‌های استثنای مالی' })
    await queues.getByRole('link', { name: /تلاش پرداخت بدون فاکتور حل‌شده/ }).click()
    await expect(page).toHaveURL(/type=attempt-no-invoice/)
    if (DB.unknownInvoice === 0) await expect(page.getByText('موردی برای این صف ثبت نشده است')).toBeVisible()
    else await expect(rows).toHaveCount(Math.min(DB.unknownInvoice, 25))
    await queues.getByRole('link', { name: /تعارض یا ناهم‌خوانی مبالغ فاکتور/ }).click()
    await expect(page.getByText('فهرست این صف در قرارداد موجود نیست')).toBeVisible()
    await noCommandControls(page)
    expectReadsOnly(reqs)
  })

  operator('receipts: legacy references counted from the report; target diagnostics disabled independently', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/finance/receipts')
    await expect(page.getByRole('heading', { level: 1, name: 'شواهد رسید', exact: true })).toBeVisible()
    const legacy = page.getByRole('list', { name: 'شمارنده‌های رسید قدیمی' })
    await expect(legacy.getByText(fa(DB.receiptsRef), { exact: true }).first()).toBeVisible()
    // payments.manual_write is OFF: the target evidence read is BLOCKED and shows no number.
    await expect(page.getByText('این بخش هنوز فعال نشده است').first()).toBeVisible()
    await expect(page.getByRole('list', { name: 'شمارنده‌های فیش سامانه جدید' })).toHaveCount(0)
    await shot(page, page.locator('main'), 'receipts')
    await noCommandControls(page)
    expectReadsOnly(reqs)
  })
})

operator.describe('dev-user-002 (no capability, test-only overlay): the Finance workspace stays locked', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  operator('queue, landing and exceptions denied: no review, invoice or payment data reaches the DOM', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/finance/review')
    await expect(page.getByRole('heading', { level: 1, name: 'صف بررسی مالی', exact: true })).toBeVisible()
    await expect(page.getByText('دسترسی به این بخش ندارید').first()).toBeVisible()
    await expect(page.locator('main table')).toHaveCount(0)
    await shot(page, page.locator('main'), 'denied')
    await page.goto('/finance')
    await expect(page.getByRole('heading', { level: 1, name: 'میز کار مالی', exact: true })).toBeVisible()
    await expect(page.getByRole('list', { name: 'شمارنده‌های فاکتور' }).getByText('دسترسی به گزارش تطبیق صورت‌حساب ندارید').first()).toBeVisible()
    await expect(page.getByRole('list', { name: 'شمارنده‌های فاکتور' }).getByText(fa(DB.invoices), { exact: true })).toHaveCount(0)
    await page.goto('/finance/exceptions?type=attempt-regression')
    await expect(page.getByText('دسترسی به این بخش ندارید').first()).toBeVisible()
    await expect(page.locator('main table')).toHaveCount(0)
    await noOverflow(page, 'denied-exceptions')
    expectReadsOnly(reqs)
  })
})
