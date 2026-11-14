import { expect, guarded, required } from './fixtures'
import { T } from './hr-fixtures'

// R7-6 class B — TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE.
// crm-api runs with CRM_ENV=test and the existing development auth stub
// (CRM_DEV_AUTH=1, refused in production, never combined with sessions).
// This browser context adds the stub header itself; the production SPA
// bundle carries no dev identity. dev-user-001 holds a synthetic in-memory
// HR role (no database grant); dev-user-002 holds nothing. The data is the
// R7-5 integrated SYNTHETIC dataset: imported SHADOW organisation evidence,
// not live HR data. Nothing here shows that production authorizes these
// reads.

const persons = Number(required('E2E_HR_PERSONS'))
const cycles = Number(required('E2E_HR_CYCLE_FINDINGS'))
const selfParents = Number(required('E2E_HR_SELF_PARENT'))
const sellerUserId = required('E2E_HR_SELLER_USER_ID')
const selfParentPerson = required('E2E_HR_SELF_PARENT_PERSON')

interface List { outcome: string; reasons: string[]; items?: unknown[]; page?: { has_more: boolean; next_cursor?: string | null } }

const operator = guarded

operator.describe('dev-user-001 (synthetic HR role, test-only overlay): the HR read pages render the shadow evidence', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('the session reports the dev stub, never a real session', async ({ page, blocked }) => {
    void blocked
    const r = await page.request.get('/api/v1/session')
    expect(r.status()).toBe(200)
    const s = (await r.json()) as { data: { auth_mode: string; authenticated: boolean; features?: { hr_read: boolean } } }
    expect(s.data.auth_mode).toBe('dev_stub')
    expect(s.data.authenticated).toBe(true)
    await page.goto('/')
    await expect(page.getByRole('button', { name: T.login, exact: true })).toHaveCount(0)
  })

  operator('workforce: every imported person, read-only, with unresolved evidence shown as such', async ({ page, blocked }) => {
    void blocked
    await page.goto('/')
    await page.getByRole('link', { name: T.workforce, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.workforce, exact: true })).toBeVisible()
    await expect(page.getByText('فقط خواندنی').first()).toBeVisible()
    const rows = page.locator('table.hr-table tbody tr')
    await expect(rows).toHaveCount(Math.min(persons, 25))
    // The self-parent person links twice: as a person and as their own (conflicting) manager line.
    await expect(page.getByRole('link', { name: 'Synthetic Self Parent', exact: true }).first()).toBeVisible()
    // Multiple current parents are a conflict, never a chosen manager.
    await expect(page.getByText('تعارض: چند مدیر جاری').first()).toBeVisible()
    // No create / edit / move control on a read-only page.
    await expect(page.getByRole('button', { name: /افزودن|ویرایش|حذف|انتقال/ })).toHaveCount(0)
    const api = (await (await page.request.get('/api/v1/organization/persons?limit=25')).json()) as List
    expect(api.outcome).toBe('SUCCESS')
    expect(api.items?.length).toBe(Math.min(persons, 25))
    if (persons <= 25) await expect(page.getByRole('button', { name: 'صفحه بعد' })).toBeDisabled()
  })

  operator('filters and server-side pagination', async ({ page, blocked }) => {
    void blocked
    await page.goto('/hr')
    await page.getByLabel('جست‌وجو (نام یا کد)').fill('Synthetic Cycle')
    await page.getByRole('button', { name: 'اعمال فیلتر' }).click()
    await expect(page).toHaveURL(/q=Synthetic\+Cycle/)
    await expect(page.locator('table.hr-table tbody tr')).toHaveCount(2)
    await page.goto('/hr?q=no-such-synthetic-person')
    await expect(page.getByText('نتیجه‌ای برای این فیلتر نیست')).toBeVisible()
    // Cursor pagination: limit 5 walks every person exactly once.
    const seen = new Set<string>()
    let cursor: string | null | undefined
    for (let i = 0; i < 10; i++) {
      const q = cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''
      const r = (await (await page.request.get(`/api/v1/organization/persons?limit=5${q}`)).json()) as List
      expect(r.outcome).toBe('SUCCESS')
      for (const p of (r.items ?? []) as { person_id: string }[]) seen.add(p.person_id)
      if (!r.page?.has_more) break
      cursor = r.page.next_cursor
    }
    expect(seen.size).toBe(persons)
  })

  operator('person: a self-parent line is reported as a finding, not resolved', async ({ page, blocked }) => {
    void blocked
    await page.goto(`/hr/people/${selfParentPerson}`)
    await expect(page.getByRole('heading', { name: 'Synthetic Self Parent', exact: true })).toBeVisible()
    await expect(page.getByText('گزارش به خود').first()).toBeVisible()
  })

  operator('structure and hierarchy exceptions (cycles and self-parent lines are listed, never auto-fixed)', async ({ page, blocked }) => {
    void blocked
    await page.goto('/')
    await page.getByRole('link', { name: T.structure, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.structure, exact: true })).toBeVisible()
    await expect(page.getByText(T.denied)).toHaveCount(0)
    await page.getByRole('link', { name: T.exceptions, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.exceptions, exact: true })).toBeVisible()
    const findings = page.locator('ul.hr-list')
    await expect(findings.getByText('چرخه در سلسله‌مراتب').first()).toBeVisible()
    for (const [kind, n] of [['CYCLE', cycles], ['SELF_PARENT', selfParents]] as [string, number][]) {
      const r = (await (await page.request.get(`/api/v1/organization/hierarchy-findings?kind=${kind}&limit=100`)).json()) as List
      expect(r.outcome, kind).toBe('SUCCESS')
      expect(r.items?.length, kind).toBe(n)
    }
    await page.getByLabel('نوع یافته').selectOption('SELF_PARENT')
    await page.getByRole('button', { name: 'اعمال فیلتر' }).click()
    await expect(findings.getByText('گزارش به خود').first()).toBeVisible()
    await expect(findings.locator('li')).toHaveCount(selfParents)
    await expect(findings.getByText('چرخه در سلسله‌مراتب')).toHaveCount(0)
  })

  operator('HR requests: the stub identity is no organisation user, so creating is refused and nothing is written', async ({ page, blocked }) => {
    void blocked
    const r = await page.request.post('/api/v1/hr/requests', {
      headers: { 'Idempotency-Key': `e2e-hr-overlay-${Date.now()}` },
      data: { request_type: 'terminate_seller', subject_user_id: sellerUserId, reason: 'synthetic e2e reason' },
    })
    const body = (await r.json()) as { outcome: string; reasons: string[]; command?: { applied: boolean; command_id?: string } }
    expect(body.outcome).toBe('UNAUTHORIZED')
    expect(body.reasons).toEqual(['ROW_SCOPE_DENIED'])
    expect(body.command?.applied ?? false).toBe(false)
    expect(body.command?.command_id).toBeUndefined()
    // The orchestrator asserts organization.hr_request is still empty.
  })
})

const viewer = guarded

viewer.describe('dev-user-002 (no grant): every HR read is refused', () => {
  viewer.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  viewer('the pages show the no-access state; the API answers 403', async ({ page, blocked }) => {
    void blocked
    for (const [path, heading] of [['/hr', T.workforce], ['/hr/structure', T.structure], ['/hr/exceptions', T.exceptions]] as [string, string][]) {
      await page.goto(path)
      await expect(page.getByRole('heading', { name: heading, exact: true }), path).toBeVisible()
      await expect(page.getByText(T.denied).first(), path).toBeVisible()
      await expect(page.getByRole('table'), path).toHaveCount(0)
    }
    for (const path of ['/api/v1/organization/persons', '/api/v1/organization/hierarchy-findings', '/api/v1/hr/requests']) {
      const r = await page.request.get(path)
      expect(r.status(), path).toBe(403)
      const b = (await r.json()) as { outcome: string; reasons: string[] }
      expect(b.outcome, path).toBe('UNAUTHORIZED')
    }
  })
})
