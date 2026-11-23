import type { Locator, Page, Request } from '@playwright/test'
import { expect, guarded, required } from './fixtures'

// RM-P05 — MIS operational workspace (landing counters, case inventory,
// custody evidence, exceptions), read only.
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE: crm-api runs with
// CRM_ENV=test and the existing development auth stub (CRM_DEV_AUTH=1,
// refused in production). This browser context adds the stub header itself;
// the production SPA bundle carries no dev identity. The MIS role holds no
// capability: nothing here is real-account evidence. Data is the R7-5
// integrated SYNTHETIC dataset; every expected value is the orchestrator's SQL.

const DB = {
  casesTotal: Number(required('E2E_MIS_CASES_TOTAL')),
  casesPresent: Number(required('E2E_MIS_CASES_PRESENT')),
  custodyEvents: Number(required('E2E_MIS_CUSTODY_EVENTS')),
  exc: required('E2E_MIS_EXC_CASE'),
  excParity: required('E2E_MIS_EXC_PARITY'),
  excParityCount: Number(required('E2E_MIS_EXC_PARITY_COUNT')),
  excKey: required('E2E_MIS_EXC_KEY'),
  excKeyCases: Number(required('E2E_MIS_EXC_KEY_CASES')),
  excEvents: Number(required('E2E_MIS_EXC_EVENTS')),
  excAttribution: Number(required('E2E_MIS_EXC_ATTRIBUTION')),
  unresolved: Number(required('E2E_MIS_UNRESOLVED')),
}
const SHOTS = process.env.E2E_SCREENSHOT_DIR ?? ''
const PAGE = 25
const fa = (n: number) => new Intl.NumberFormat('fa-IR').format(n)
const QUEUE: Record<string, string> = {
  CONFLICT: 'holder-conflict', MULTIPLE_CURRENT: 'holder-multiple', TARGET_UNKNOWN: 'holder-unknown', NOT_RECOVERABLE: 'holder-not-recoverable', HISTORY_GAP: 'history-gap',
}
const SECTIONS = ['امروز MIS', 'موجودی پرونده‌ها', 'نگهداری و انتساب', 'استثناها و تطبیق', 'کیفیت منبع', 'تطبیق و پوشش']
const OP_WORDS = /تخصیص|انتقال|برگشت|آزادسازی|بازتخصیص|استفاده مجدد|نهایی|assign|return|release|reuse|recycle/i

const misNav = (page: Page) => page.getByRole('navigation', { name: 'بخش‌های فضای کار MIS' })

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

async function noOperationControls(page: Page) {
  const names = await page.locator('main').getByRole('button').evaluateAll(els => els.map(e => (e.textContent ?? '').trim()))
  for (const n of names) expect(n, `unexpected control "${n}"`).not.toMatch(OP_WORDS)
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
    await region.screenshot({ path: `${SHOTS}/${name}-${vp}.png` })
  }
  await page.setViewportSize({ width: 1366, height: 900 })
}

const operator = guarded

operator.describe('dev-user-001 (synthetic read capabilities, test-only overlay): MIS workspace', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('landing: sections, counters equal the database, operations are statuses only', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/mis')
    await expect(page.getByRole('heading', { level: 1, name: 'امروز MIS', exact: true })).toBeVisible()
    await expect(misNav(page).getByRole('link')).toHaveText(SECTIONS)
    const inv = page.getByRole('group', { name: 'شمارنده‌های موجودی' })
    await expect(inv.getByText(fa(DB.casesTotal), { exact: true }).first()).toBeVisible()
    await expect(inv.getByText(fa(DB.casesTotal - DB.casesPresent), { exact: true }).first()).toBeVisible()
    await expect(page.getByRole('group', { name: 'شمارنده‌های نگهداری' }).getByText(fa(DB.custodyEvents), { exact: true }).first()).toBeVisible()
    const ops = page.getByRole('list', { name: 'وضعیت عملیات موجودی' })
    await expect(ops.getByText('در دسترس نیست', { exact: true })).toHaveCount(8)
    await expect(ops.getByRole('button')).toHaveCount(0)
    await noOperationControls(page)
    await shot(page, page.locator('main'), 'landing')
    expectReadsOnly(reqs)
  })

  operator('inventory: exact source key search, custody column, Case 360° and timeline links', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/mis/cases')
    await expect(page.getByRole('heading', { level: 1, name: 'موجودی پرونده‌ها', exact: true })).toBeVisible()
    const rows = page.locator('table.mis-table tbody tr')
    await expect(rows).toHaveCount(Math.min(PAGE, DB.casesTotal))
    await shot(page, page.locator('main'), 'inventory')
    const form = page.getByRole('form', { name: 'فیلتر موجودی پرونده‌ها' })
    await form.getByLabel('کلید منبع (دقیق)').fill(DB.excKey)
    await form.getByRole('button', { name: 'اعمال فیلتر' }).click()
    await expect(page).toHaveURL(new RegExp(`source_key=${DB.excKey}`))
    await expect(rows).toHaveCount(DB.excKeyCases)
    const row = rows.filter({ has: page.locator(`a[href="/sales/cases/${DB.exc}"]`) })
    await expect(row).toHaveCount(1)
    await expect(row.locator('.mis-custody-cell')).toBeVisible()
    await expect(row.getByRole('link', { name: 'خط زمانی' })).toHaveAttribute('href', `/sales/cases/${DB.exc}#activity`)
    const custodyReq = reqs.find(r => new URL(r.url()).pathname === '/api/v1/custody/assignments' && new URL(r.url()).searchParams.getAll('case_id').includes(DB.exc))
    expect(custodyReq, 'custody read for the visible page').toBeTruthy()
    await row.getByRole('link', { name: 'خط زمانی' }).click()
    await expect(page).toHaveURL(new RegExp(`/sales/cases/${DB.exc}#activity$`))
    await expect(page.getByRole('region', { name: 'خط زمانی فعالیت پرونده' })).toBeVisible()
    expectReadsOnly(reqs)
  })

  operator('custody evidence: current custody apart from transfer history and historical attribution', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto(`/mis/custody/${DB.exc}`)
    await expect(page.getByRole('heading', { level: 1 })).toContainText('شواهد نگهداری')
    await expect(page.getByRole('region', { name: 'استنتاج از رویدادها' })).toBeVisible()
    await expect(page.getByRole('region', { name: 'ثبت‌شده در منبع' })).toBeVisible()
    await expect(page.getByRole('table', { name: 'سابقه انتقال' }).locator('tbody tr')).toHaveCount(DB.excEvents)
    if (DB.excAttribution > 0) await expect(page.getByRole('table', { name: 'انتساب تاریخی' }).locator('tbody tr')).toHaveCount(DB.excAttribution)
    await noOperationControls(page)
    await shot(page, page.locator('main'), 'custody-evidence')
    await page.goto('/mis/custody?holder_link=UNRESOLVED')
    await expect(page.getByRole('heading', { level: 1, name: 'نگهداری و انتساب', exact: true })).toBeVisible()
    if (DB.unresolved > 0) await expect(page.locator('table.mis-table tbody tr')).toHaveCount(Math.min(PAGE, DB.unresolved))
    else await expect(page.getByText('نتیجه‌ای برای این فیلتر نیست')).toBeVisible()
    await shot(page, page.locator('main'), 'custody')
    expectReadsOnly(reqs)
  })

  operator('exceptions: the queue lists exactly the stored state with its counter', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    const queue = QUEUE[DB.excParity] ?? ''
    expect(queue, DB.excParity).not.toBe('')
    await page.goto(`/mis/exceptions?type=${queue}`)
    await expect(page.getByRole('heading', { level: 1, name: 'استثناها و تطبیق', exact: true })).toBeVisible()
    const link = page.getByRole('navigation', { name: 'صف‌های استثنا' }).locator(`a[href="/mis/exceptions?type=${queue}"]`)
    await expect(link.locator('.mis-count')).toHaveText(fa(DB.excParityCount))
    await expect(page.locator('table.mis-table tbody tr')).toHaveCount(Math.min(PAGE, DB.excParityCount))
    await expect(page.locator(`table.mis-table a[href="/mis/custody/${DB.exc}"]`)).toHaveCount(DB.excParityCount <= PAGE ? 1 : 0)
    await noOperationControls(page)
    await shot(page, page.locator('main'), 'exceptions')
    expectReadsOnly(reqs)
  })
})

operator.describe('dev-user-002 (no capability): every MIS section fails closed', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  operator('denied: no row, holder or counter value in the DOM', async ({ page, blocked }) => {
    void blocked
    for (const path of ['/mis/cases', '/mis/custody', `/mis/custody/${DB.exc}`, '/mis/exceptions?type=holder-conflict']) {
      await page.goto(path)
      await expect(page.getByText('دسترسی به این بخش ندارید').first(), path).toBeVisible()
      await expect(page.locator('table'), path).toHaveCount(0)
      await expect(page.getByText(/کاربر قدیمی #/), path).toHaveCount(0)
      await expect(misNav(page).getByRole('link'), path).toHaveCount(1)
    }
    await page.goto(`/mis/custody/${DB.exc}`)
    await shot(page, page.locator('main'), 'denied')
    await page.goto('/mis')
    await expect(page.getByRole('group', { name: 'شمارنده‌های موجودی' }).getByText(fa(DB.casesTotal), { exact: true })).toHaveCount(0)
  })
})
