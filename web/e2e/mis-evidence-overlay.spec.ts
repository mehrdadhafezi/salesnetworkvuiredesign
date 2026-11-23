import type { Locator, Page, Request } from '@playwright/test'
import { expect, guarded, required } from './fixtures'

// RM-P06 — MIS case-to-custody evidence continuity, read only.
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE: crm-api runs with
// CRM_ENV=test and the existing development auth stub (CRM_DEV_AUTH=1,
// refused in production). The MIS role holds no capability: nothing here is
// real-account evidence. Data is the R7-5 integrated SYNTHETIC dataset; every
// expected value is the orchestrator's SQL. "Cases denied while custody is
// allowed" is produced by answering GET /api/v1/cases with the server's own
// UNAUTHORIZED envelope in this browser context only.

const DB = {
  exc: required('E2E_MIS_EXC_CASE'),
  excParity: required('E2E_MIS_EXC_PARITY'),
  system: required('E2E_MIS_EXC_SYSTEM'),
  spec: required('E2E_MIS_EXC_SPEC'),
  key: required('E2E_MIS_EXC_KEY'),
  srcPresence: required('E2E_MIS_EXC_SRC_PRESENCE'),
  firstBatch: required('E2E_MIS_EXC_FIRST_BATCH'),
  lastBatch: required('E2E_MIS_EXC_LAST_BATCH'),
  page: required('E2E_MIS_CUSTODY_PAGE').split(','),
  pageWithCase: Number(required('E2E_MIS_CUSTODY_PAGE_CASE')),
}
const SHOTS = process.env.E2E_SCREENSHOT_DIR ?? ''
const QUEUE: Record<string, string> = {
  CONFLICT: 'holder-conflict', MULTIPLE_CURRENT: 'holder-multiple', TARGET_UNKNOWN: 'holder-unknown', NOT_RECOVERABLE: 'holder-not-recoverable', HISTORY_GAP: 'history-gap',
}
const OP_WORDS = /تخصیص|انتقال|برگشت|آزادسازی|بازتخصیص|استفاده مجدد|نهایی|assign|return|release|reuse|recycle/i
const sourceRef = `${DB.system} · ${DB.spec} · #${DB.key}`

function recordApi(page: Page): Request[] {
  const seen: Request[] = []
  page.on('request', r => { if (new URL(r.url()).pathname.startsWith('/api/')) seen.push(r) })
  return seen
}
const casesReads = (reqs: Request[]) => reqs.filter(r => new URL(r.url()).pathname === '/api/v1/cases')

function expectReadsOnly(reqs: Request[]) {
  expect(reqs.length).toBeGreaterThan(0)
  for (const r of reqs) expect(r.method(), r.url()).toBe('GET')
  for (const r of reqs) {
    const q = new URL(r.url()).searchParams
    for (const k of ['phone', 'customer_phone', 'mobile', 'name', 'q', 'source_key']) expect(q.has(k), `${k} in ${r.url()}`).toBe(false)
  }
}

async function noOperationControls(page: Page) {
  const names = await page.locator('main').getByRole('button').evaluateAll(els => els.map(e => (e.textContent ?? '').trim()))
  for (const n of names) expect(n, `unexpected control "${n}"`).not.toMatch(OP_WORDS)
}

async function shot(page: Page, region: Locator, name: string) {
  for (const [vp, size] of [['desktop', { width: 1366, height: 900 }], ['tablet', { width: 820, height: 1180 }], ['mobile', { width: 390, height: 844 }]] as const) {
    await page.setViewportSize(size)
    const offenders = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1 ? [] : Array.from(document.querySelectorAll('body *'))
      .filter(el => { const r = el.getBoundingClientRect(); return r.right > window.innerWidth + 1 || r.left < -1 })
      .filter(el => { for (let p = el.parentElement; p && p !== document.body; p = p.parentElement) if (getComputedStyle(p).overflowX !== 'visible') return false; return true })
      .slice(0, 8).map(el => `${el.tagName.toLowerCase()}.${(el.getAttribute('class') ?? '').replace(/ /g, '.')} ${Math.round(el.getBoundingClientRect().left)}..${Math.round(el.getBoundingClientRect().right)}`))
    expect(offenders, `${name} ${vp} horizontal overflow`).toEqual([])
    if (SHOTS) await region.screenshot({ path: `${SHOTS}/${name}-${vp}.png` })
  }
  await page.setViewportSize({ width: 1366, height: 900 })
}

const operator = guarded

operator.describe('dev-user-001 (synthetic read capabilities, test-only overlay): custody rows carry their case source', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('custody page: one exact case_id batch read; source, native id and three links per row', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/mis/custody')
    await expect(page.getByRole('heading', { level: 1, name: 'نگهداری و انتساب', exact: true })).toBeVisible()
    const rows = page.locator('table.mis-table tbody tr')
    await expect(rows).toHaveCount(DB.page.length)
    await expect(page.getByTestId('case-source')).toHaveCount(DB.pageWithCase)
    const reads = casesReads(reqs)
    expect(reads, 'exactly one cases read for the page (no N+1)').toHaveLength(1)
    const q = new URL(reads[0]?.url() ?? 'http://x/').searchParams
    expect(q.getAll('case_id')).toEqual(DB.page)
    const keys: string[] = []
    q.forEach((_, k) => { if (!keys.includes(k)) keys.push(k) })
    expect(keys.sort()).toEqual(['case_id', 'limit'])
    if (DB.page.includes(DB.exc)) {
      const row = rows.filter({ hasText: DB.exc })
      await expect(row.getByText(sourceRef, { exact: true })).toBeVisible()
      await expect(row.getByRole('link', { name: 'نمای ۳۶۰' })).toHaveAttribute('href', `/sales/cases/${DB.exc}`)
      await expect(row.getByRole('link', { name: 'شواهد نگهداری' })).toHaveAttribute('href', `/mis/custody/${DB.exc}`)
      await expect(row.getByRole('link', { name: 'خط زمانی' })).toHaveAttribute('href', `/sales/cases/${DB.exc}#activity`)
    }
    await noOperationControls(page)
    await shot(page, page.locator('main'), 'custody-source')
    expectReadsOnly(reqs)
  })

  operator('exceptions: the queue rows are enriched by one batch read and navigate to the evidence', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    const queue = QUEUE[DB.excParity] ?? ''
    expect(queue, DB.excParity).not.toBe('')
    await page.goto(`/mis/exceptions?type=${queue}`)
    const table = page.getByRole('table', { name: 'پروجکشن‌های نگهداری' })
    const row = table.locator('tbody tr').filter({ hasText: DB.exc })
    await expect(row.getByText(sourceRef, { exact: true })).toBeVisible()
    await expect(row.getByText(DB.srcPresence === 'PRESENT' ? 'منبع: حاضر' : 'منبع: غایب در آخرین واردسازی', { exact: true })).toBeVisible()
    expect(casesReads(reqs)).toHaveLength(1)
    await noOperationControls(page)
    await shot(page, page.locator('main'), 'exceptions-source')
    await row.getByRole('link', { name: 'شواهد نگهداری' }).click()
    await expect(page).toHaveURL(new RegExp(`/mis/custody/${DB.exc}$`))
    const prov = page.getByLabel('خاستگاه ثبت‌شده')
    if (DB.firstBatch === DB.lastBatch) {
      await expect(prov.getByText(DB.firstBatch, { exact: true })).toHaveCount(2)
      await expect(prov.getByText('(همان دسته اول)')).toBeVisible()
    } else {
      await expect(prov.getByText(DB.firstBatch, { exact: true })).toHaveCount(1)
      await expect(prov.getByText(DB.lastBatch, { exact: true })).toHaveCount(1)
    }
    await expect(prov.getByText('در دسترس نیست', { exact: true })).toBeVisible()
    await shot(page, page.locator('main'), 'provenance')
    await page.getByRole('link', { name: 'خط زمانی فعالیت پرونده' }).click()
    await expect(page).toHaveURL(new RegExp(`/sales/cases/${DB.exc}#activity$`))
    await expect(page.getByRole('region', { name: 'خط زمانی فعالیت پرونده' })).toBeVisible()
    expectReadsOnly(reqs)
  })

  operator('cases denied, custody allowed: custody evidence stays, no case source reaches the DOM', async ({ page, blocked }) => {
    void blocked
    await page.route('**/api/v1/cases**', route => route.fulfill({ status: 403, contentType: 'application/json',
      body: JSON.stringify({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'e2e-denied-0001' }) }))
    const reqs = recordApi(page)
    await page.goto('/mis/custody')
    const rows = page.locator('table.mis-table tbody tr')
    await expect(rows).toHaveCount(DB.page.length)
    await expect(page.getByText('منبع پرونده: بدون دسترسی خواندن پرونده', { exact: true })).toHaveCount(DB.page.length)
    await expect(page.getByTestId('case-source')).toHaveCount(0)
    const text = await page.locator('main').innerText()
    expect(text).not.toContain(sourceRef)
    expect(text).not.toContain(DB.firstBatch)
    await page.goto(`/mis/custody/${DB.exc}`)
    await expect(page.getByRole('region', { name: 'استنتاج از رویدادها' })).toBeVisible()
    await expect(page.getByLabel('خاستگاه ثبت‌شده')).toHaveCount(0)
    const detail = await page.locator('main').innerText()
    expect(detail).not.toContain(DB.firstBatch)
    expect(detail).not.toContain(`${DB.system} · ${DB.spec} · ${DB.key}`)
    await shot(page, page.locator('main'), 'cases-denied')
    expectReadsOnly(reqs)
  })
})

operator.describe('dev-user-002 (no capability): fails closed', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  operator('denied: no custody row and no case source', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/mis/custody')
    await expect(page.getByText('دسترسی به این بخش ندارید').first()).toBeVisible()
    await expect(page.locator('table')).toHaveCount(0)
    await expect(page.getByTestId('case-source')).toHaveCount(0)
    expect(casesReads(reqs), 'no cases read without the server hint').toHaveLength(0)
  })
})
