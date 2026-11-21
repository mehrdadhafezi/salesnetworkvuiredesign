import type { Locator, Page, Request } from '@playwright/test'
import { expect, guarded, required } from './fixtures'

// RM-P04 — HR operational workspace (personnel, structure, assignments,
// positions, requests, exceptions, own access), read only.
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE: crm-api runs with
// CRM_ENV=test and the existing development auth stub (CRM_DEV_AUTH=1,
// refused in production). This browser context adds the stub header itself;
// the production SPA bundle carries no dev identity. dev-user-001 holds a
// synthetic in-memory HR role plus direct synthetic capabilities (no database
// grant); dev-user-002 holds nothing. Data is the R7-5 integrated SYNTHETIC
// dataset. Every expected count comes from the orchestrator's SQL. HR role
// runtime stays NOT LIVE VERIFIED: nothing here is real-account evidence.

const DB = {
  persons: Number(required('E2E_HR_PERSONS')),
  assignments: Number(required('E2E_HR_ASSIGNMENTS')),
  current: Number(required('E2E_HR_ASSIGNMENTS_CURRENT')),
  selfParent: required('E2E_HR_SELF_PARENT_PERSON'),
  selfParentAssignments: Number(required('E2E_HR_SELF_PARENT_ASSIGNMENTS')),
  positions: Number(required('E2E_HR_POSITIONS')),
  levels: Number(required('E2E_HR_LEVELS')),
  sharedUser: process.env.E2E_HR_SHARED_USER ?? '',
  sharedPersons: Number(process.env.E2E_HR_SHARED_USER_PERSONS ?? '0'),
}
const SHOTS = process.env.E2E_SCREENSHOT_DIR ?? ''
const ASSIGNMENT_PAGE = 50
const SECTIONS = ['نیروها', 'ساختار سازمانی', 'انتساب‌ها و گزارش‌دهی', 'سمت‌ها و سطوح', 'درخواست‌ها', 'استثناها', 'دسترسی من']
const HEADINGS: Record<string, string> = {
  'نیروها': 'نیروها', 'ساختار سازمانی': 'ساختار سازمانی', 'انتساب‌ها و گزارش‌دهی': 'انتساب‌ها و گزارش‌دهی', 'سمت‌ها و سطوح': 'سمت‌ها و سطوح',
  'درخواست‌ها': 'درخواست‌های جابجایی / قطع همکاری', 'استثناها': 'استثناهای سلسله‌مراتب', 'دسترسی من': 'دسترسی من',
}

const hrNav = (page: Page) => page.getByRole('navigation', { name: 'بخش‌های فضای کار منابع انسانی' })

function recordApi(page: Page): Request[] {
  const seen: Request[] = []
  page.on('request', r => { if (new URL(r.url()).pathname.startsWith('/api/')) seen.push(r) })
  return seen
}

function expectReadsOnly(reqs: Request[]) {
  expect(reqs.length).toBeGreaterThan(0)
  for (const r of reqs) expect(r.method(), r.url()).toBe('GET')
}

async function shot(page: Page, region: Locator, name: string) {
  if (!SHOTS) return
  for (const [vp, size] of [['desktop', { width: 1366, height: 900 }], ['tablet', { width: 820, height: 1180 }], ['mobile', { width: 390, height: 844 }]] as const) {
    await page.setViewportSize(size)
    // No horizontal page scroll at any width.
    const offenders = await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1 ? [] : Array.from(document.querySelectorAll('body *'))
      .filter(el => { const r = el.getBoundingClientRect(); return r.right > window.innerWidth + 1 || r.left < -1 })
      .filter(el => { for (let p = el.parentElement; p && p !== document.body; p = p.parentElement) if (getComputedStyle(p).overflowX !== 'visible') return false; return true })
      .slice(0, 8).map(el => `${el.tagName.toLowerCase()}.${(el.getAttribute('class') ?? '').replace(/ /g, '.')} ${Math.round(el.getBoundingClientRect().left)}..${Math.round(el.getBoundingClientRect().right)}`))
    expect(offenders, `${name} ${vp} horizontal overflow`).toEqual([])
    await region.screenshot({ path: `${SHOTS}/${name}-${vp}.png` })
  }
  await page.setViewportSize({ width: 1366, height: 900 })
}

const main = (page: Page) => page.locator('main#ws')

const operator = guarded

operator.describe('dev-user-001 (synthetic HR role, test-only overlay): HR operational workspace', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('section navigation reaches every workspace area; every API call is a read', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/hr')
    await expect(page.getByRole('heading', { level: 1, name: 'نیروها' })).toBeVisible()
    await expect(hrNav(page).getByRole('link')).toHaveText(SECTIONS)
    await expect(page.getByText('طراحی منابع انسانی: آماده بازبینی (منجمد نشده)')).toBeVisible()
    for (const label of SECTIONS) {
      await hrNav(page).getByRole('link', { name: label, exact: true }).click()
      await expect(page.getByRole('heading', { level: 1, name: HEADINGS[label] ?? label })).toBeVisible()
      await expect(hrNav(page).getByRole('link', { name: label, exact: true })).toHaveAttribute('aria-current', 'page')
      await expect(page.getByText('دسترسی به این بخش ندارید')).toHaveCount(0)
    }
    expectReadsOnly(reqs)
    await page.goto('/hr')
    await expect(page.locator('table.hr-table tbody tr')).toHaveCount(Math.min(DB.persons, 25))
    await shot(page, main(page), 'personnel')
  })

  operator('assignments: database counts, current-flag filter and the person scope', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/hr/assignments')
    const rows = page.getByRole('table', { name: 'انتساب‌های گزارش‌دهی' }).locator('tbody tr')
    await expect(rows).toHaveCount(Math.min(DB.assignments, ASSIGNMENT_PAGE))
    await page.getByLabel('جاری بودن (پرچم منبع)').selectOption('true')
    await page.getByRole('button', { name: 'اعمال فیلتر' }).click()
    await expect(page).toHaveURL(/is_current=true/)
    if (DB.current === 0) await expect(page.getByText('انتسابی برای این فیلتر نیست')).toBeVisible()
    else await expect(rows).toHaveCount(Math.min(DB.current, ASSIGNMENT_PAGE))
    await shot(page, main(page), 'assignments')

    // Person → assignments: the self-parent line is listed, with its finding, never resolved.
    await page.goto(`/hr/people/${DB.selfParent}`)
    await page.getByRole('link', { name: 'همه انتساب‌های این نیرو' }).click()
    await expect(page).toHaveURL(new RegExp(`person_id=${DB.selfParent}`))
    await expect(page.locator('.hr-scope').getByRole('link', { name: 'Synthetic Self Parent' })).toBeVisible()
    if (DB.selfParentAssignments === 0) await expect(page.getByText('انتسابی برای این فیلتر نیست')).toBeVisible()
    else {
      await expect(rows).toHaveCount(Math.min(DB.selfParentAssignments, ASSIGNMENT_PAGE))
      await expect(page.getByRole('table', { name: 'انتساب‌های گزارش‌دهی' }).getByText('گزارش به خود').first()).toBeVisible()
    }
    expectReadsOnly(reqs)
  })

  operator('positions and levels: database counts; a position links to the filtered directory; no role or access control', async ({ page, blocked }) => {
    void blocked
    await page.goto('/hr/positions')
    await expect(page.getByRole('table', { name: 'سمت‌ها' }).locator('tbody tr')).toHaveCount(Math.min(DB.positions, 200))
    await expect(page.getByRole('table', { name: 'سطوح' }).locator('tbody tr')).toHaveCount(Math.min(DB.levels, 200))
    await shot(page, main(page), 'positions')
    await expect(page.getByRole('button', { name: /افزودن|ویرایش|حذف|غیرفعال‌سازی|نگاشت|اعطا/ })).toHaveCount(0)
    await page.getByRole('table', { name: 'سمت‌ها' }).getByRole('link', { name: 'مشاهده نیروها' }).first().click()
    await expect(page).toHaveURL(/\/hr\?position_id=/)
    await expect(page.getByRole('heading', { level: 1, name: 'نیروها' })).toBeVisible()
    await expect(page.getByText('دسترسی به این بخش ندارید')).toHaveCount(0)
  })

  operator('person profile: account separate from the person; shared accounts stay a conflict', async ({ page, blocked }) => {
    void blocked
    await page.goto(`/hr/people/${DB.selfParent}`)
    const account = page.getByRole('region', { name: 'حساب کاربری (هویت ورود)' })
    await expect(account).toBeVisible()
    await expect(account.getByText(/حساب کاربری با شخص و با همکاری یکی نیست/)).toBeVisible()
    await expect(page.getByRole('button', { name: /رمز|بازنشانی|ساخت حساب|ورود به‌جای/ })).toHaveCount(0)
    // The stub identity is no organisation user: requests in its row scope are empty, never "all requests".
    await expect(page.getByRole('region', { name: 'درخواست‌های منابع انسانی این نیرو' }).getByText(/فقط درخواست‌هایی که شما/)).toBeVisible()
    await shot(page, main(page), 'person')
    if (DB.sharedUser) {
      await page.goto(`/hr?user_id=${DB.sharedUser}`)
      await expect(page.getByText('فقط پرونده‌های متصل به یک حساب کاربری')).toBeVisible()
      await expect(page.locator('table.hr-table tbody tr')).toHaveCount(DB.sharedPersons)
      await expect(page.getByText('تعارض: حساب مشترک')).toHaveCount(DB.sharedPersons)
    }
  })

  operator('requests: queues are server filters within the caller scope; nothing is written', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto('/hr/requests')
    await expect(page.getByText('درخواستی برای شما وجود ندارد')).toBeVisible()
    const filtered = page.waitForRequest(r => r.url().includes('/api/v1/hr/requests') && new URL(r.url()).searchParams.get('status') === 'pending_hr')
    await page.getByRole('button', { name: 'در انتظار منابع انسانی' }).click()
    await filtered
    await expect(page.getByText('درخواستی با این وضعیت در محدوده شما نیست')).toBeVisible()
    await shot(page, main(page), 'requests')
    expectReadsOnly(reqs)
  })

  operator('دسترسی من: credential, grant, role, position and capability are shown apart; no grant control', async ({ page, blocked }) => {
    void blocked
    await page.goto('/hr/access')
    const concepts = page.getByLabel('مفاهیم دسترسی')
    await expect(concepts.getByText('هویت آزمایشی توسعه (مصنوعی؛ فقط محیط توسعه/آزمون)')).toBeVisible()
    await expect(concepts.getByText('منابع انسانی', { exact: true })).toBeVisible()
    await expect(concepts.getByText(/واگذاری مستقیم بدون نقش/)).toBeVisible()
    await expect(concepts.getByRole('region', { name: 'سمت سازمانی' }).getByText('این روش ورود به حساب سازمانی متصل نیست.')).toBeVisible()
    const table = page.getByRole('table', { name: 'ارزیابی مجوزها' })
    await expect(table.getByRole('row', { name: /org\.person\.read/ })).toContainText('مجاز')
    await expect(page.getByRole('button', { name: /اعطا|لغو|افزودن نقش|ذخیره/ })).toHaveCount(0)
    await shot(page, main(page), 'access')
  })

  operator('structure: units link to their members in the directory', async ({ page, blocked }) => {
    void blocked
    await page.goto('/hr/structure')
    await expect(page.getByRole('heading', { level: 1, name: 'ساختار سازمانی' })).toBeVisible()
    await shot(page, main(page), 'structure')
  })
})

const viewer = guarded

viewer.describe('dev-user-002 (no grant): the workspace leaks nothing', () => {
  viewer.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  viewer('new sections show the refusal, no table and no names; the navigation offers only the current section and own access', async ({ page, blocked }) => {
    void blocked
    for (const [path, heading] of [['/hr/assignments', 'انتساب‌ها و گزارش‌دهی'], ['/hr/positions', 'سمت‌ها و سطوح']] as const) {
      await page.goto(path)
      await expect(page.getByRole('heading', { level: 1, name: heading }), path).toBeVisible()
      await expect(page.getByText('دسترسی به این بخش ندارید').first(), path).toBeVisible()
      await expect(page.getByRole('table'), path).toHaveCount(0)
      await expect(page.getByText(/Synthetic/), path).toHaveCount(0)
      await expect(hrNav(page).getByRole('link')).toHaveText([heading, 'دسترسی من'])
    }
    for (const path of ['/api/v1/organization/assignments', '/api/v1/organization/positions', `/api/v1/organization/persons?user_id=${DB.sharedUser || '00000000-0000-4000-8000-000000000000'}`,
      '/api/v1/hr/requests?status=pending_hr']) {
      const r = await page.request.get(path)
      expect(r.status(), path).toBe(403)
    }
    await page.goto('/hr/access')
    await expect(page.getByRole('heading', { level: 1, name: 'دسترسی من' })).toBeVisible()
    await expect(page.getByText('واگذاری فعالی که مجوزی بدهد ندارید.')).toBeVisible()
    await shot(page, main(page), 'access-denied')
  })
})
