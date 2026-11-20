import type { Page, Request } from '@playwright/test'
import { expect, guarded, required } from './fixtures'

// RM-P01 — Customer ⇄ Case ⇄ Invoice read navigation.
// TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE: crm-api runs with
// CRM_ENV=test and the existing development auth stub (CRM_DEV_AUTH=1,
// refused in production). This browser context adds the stub header itself;
// the production SPA bundle carries no dev identity. Data is the R7-5
// integrated SYNTHETIC dataset plus one synthetic Case RESOLVED to the
// chosen customer (see the Go orchestrator). Every expected value comes
// from the orchestrator's SQL over the disposable database.

const DB = {
  customer: required('E2E_C360_CUSTOMER'),
  synthCase: required('E2E_C360_SYNTH_CASE'),
  kase: required('E2E_C360_CASE'),
  caseKey: required('E2E_C360_CASE_KEY'),
  caseLink: required('E2E_C360_CASE_LINK'),
  invoice: required('E2E_C360_INVOICE'),
  invoiceKey: required('E2E_C360_INVOICE_KEY'),
  customerInvoices: Number(required('E2E_C360_CUSTOMER_INVOICES')),
  caseInvoices: Number(required('E2E_C360_CASE_INVOICES')),
  unlinkedInvoice: required('E2E_C360_UNLINKED_INVOICE'),
  unlinkedCustomerLink: required('E2E_C360_UNLINKED_CUSTOMER_LINK'),
}
const SHOTS = process.env.E2E_SCREENSHOT_DIR ?? ''

const CASES_OF_CUSTOMER = 'پرونده‌های این مشتری (پیوند تأییدشده)'
const INVOICES_OF_CUSTOMER = 'فاکتورهای این مشتری (پیوند تأییدشده)'
const INVOICES_OF_CASE = 'فاکتورهای این پرونده (پیوند تأییدشده)'
const relationLabel: Record<string, string> = {
  RESOLVED: 'پیوند تأییدشده', RELATED: 'مرتبط (تأیید نشده)', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض', NOT_RECOVERABLE: 'قابل بازیابی نیست',
}
const PAGE_SIZE = 10

/** Records every API request of the page; the suite asserts they are reads. */
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

const listCall = (reqs: Request[], path: string, key: string, value: string) =>
  reqs.find(r => { const u = new URL(r.url()); return u.pathname === path && u.searchParams.get(key) === value })

async function shot(page: Page, name: string) {
  if (!SHOTS) return
  for (const [vp, size] of [['desktop', { width: 1366, height: 900 }], ['tablet', { width: 820, height: 1180 }], ['mobile', { width: 390, height: 844 }]] as const) {
    await page.setViewportSize(size)
    // No horizontal page scroll at any width.
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1), `${name} ${vp} horizontal overflow`).toBe(true)
    await page.screenshot({ path: `${SHOTS}/${name}-${vp}.png`, fullPage: true })
  }
  await page.setViewportSize({ width: 1366, height: 900 })
}

const operator = guarded

operator.describe('dev-user-001 (synthetic read capabilities, test-only overlay): Customer ⇄ Case ⇄ Invoice', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('Customer → Case → Customer, and Customer → Invoice → Case → Invoice → Customer', async ({ page, blocked }) => {
    void blocked
    const reqs = recordApi(page)
    await page.goto(`/customers/${DB.customer}`)
    await expect(page.getByRole('heading', { level: 1, name: /^مشتری: / })).toBeVisible()

    // Customer → its RESOLVED Cases (the synthetic fixture) through customer_id.
    const cases = page.getByRole('region', { name: CASES_OF_CUSTOMER })
    const synthLink = cases.getByRole('link', { name: 'synthetic.rm_p01.v1 #1' })
    await expect(synthLink).toBeVisible()
    await expect(cases.getByRole('link')).toHaveCount(1)
    expect(listCall(reqs, '/api/v1/cases', 'customer_id', DB.customer)).toBeTruthy()
    // Imported Cases are never attributed to this customer (no RESOLVED case link in the import).
    await expect(cases.getByText(DB.caseKey)).toHaveCount(0)

    // Customer → its RESOLVED invoices (count == database).
    const invs = page.getByRole('region', { name: INVOICES_OF_CUSTOMER })
    await expect(invs.locator('tbody tr')).toHaveCount(Math.min(DB.customerInvoices, PAGE_SIZE))
    expect(listCall(reqs, '/api/v1/invoices', 'customer_id', DB.customer)).toBeTruthy()
    await shot(page, 'customer')

    // Customer → synthetic Case → back to the customer (RESOLVED link).
    await synthLink.click()
    await expect(page).toHaveURL(new RegExp(`/sales/cases/${DB.synthCase}$`))
    await expect(page.getByRole('heading', { level: 1, name: 'پرونده: synthetic.rm_p01.v1 #1' })).toBeVisible()
    const back = page.getByRole('region', { name: 'پرونده' }).getByRole('link', { name: 'مشاهده مشتری' })
    await expect(back).toHaveAttribute('href', `/customers/${DB.customer}`)
    await back.click()
    await expect(page).toHaveURL(new RegExp(`/customers/${DB.customer}$`))

    // Customer → Invoice.
    await page.getByRole('region', { name: INVOICES_OF_CUSTOMER }).getByRole('link', { name: DB.invoiceKey }).click()
    await expect(page).toHaveURL(new RegExp(`/billing/invoices/${DB.invoice}$`))
    await expect(page.getByRole('heading', { level: 1, name: `فاکتور: ${DB.invoiceKey}` })).toBeVisible()
    const card = page.getByRole('region', { name: 'فاکتور' })
    await expect(card.getByRole('link', { name: 'مشاهده مشتری' })).toHaveAttribute('href', `/customers/${DB.customer}`)
    await expect(card.getByRole('link', { name: 'مشاهده پرونده' })).toHaveAttribute('href', `/sales/cases/${DB.kase}`)
    // The existing invoice evidence sections are preserved.
    for (const s of ['شواهد وضعیت (هر فیلد جدا)', 'اقلام (اسنپ‌شات صدور)', 'مراحل پرداخت']) await expect(page.getByRole('region', { name: s })).toBeVisible()
    await shot(page, 'invoice')

    // Invoice → Case (imported): its own customer link is shown as-is, unlinked.
    await card.getByRole('link', { name: 'مشاهده پرونده' }).click()
    await expect(page).toHaveURL(new RegExp(`/sales/cases/${DB.kase}$`))
    await expect(page.getByRole('heading', { level: 1, name: `پرونده: ${DB.caseKey}` })).toBeVisible()
    const caseCard = page.getByRole('region', { name: 'پرونده' })
    await expect(caseCard.locator('.rel-link').getByText(relationLabel[DB.caseLink] ?? DB.caseLink)).toBeVisible()
    await expect(caseCard.getByRole('link', { name: 'مشاهده مشتری' })).toHaveCount(DB.caseLink === 'RESOLVED' ? 1 : 0)
    // Existing case sections are preserved.
    for (const s of ['چهار مالک (بدون ادغام)', 'منبع و پیوندها', 'سابقه وضعیت (فقط افزودنی)', 'خط زمانی نگهداری (فقط افزودنی)']) {
      await expect(page.getByRole('region', { name: s })).toBeVisible()
    }
    // Case → its RESOLVED invoices through case_id (count == database).
    const caseInvs = page.getByRole('region', { name: INVOICES_OF_CASE })
    await expect(caseInvs.locator('tbody tr')).toHaveCount(Math.min(DB.caseInvoices, PAGE_SIZE))
    expect(listCall(reqs, '/api/v1/invoices', 'case_id', DB.kase)).toBeTruthy()
    await shot(page, 'case')

    // Case → Invoice → Customer.
    await caseInvs.getByRole('link', { name: DB.invoiceKey }).click()
    await expect(page).toHaveURL(new RegExp(`/billing/invoices/${DB.invoice}$`))
    await page.getByRole('region', { name: 'فاکتور' }).getByRole('link', { name: 'مشاهده مشتری' }).click()
    await expect(page).toHaveURL(new RegExp(`/customers/${DB.customer}$`))
    await expect(page.getByRole('heading', { level: 1, name: /^مشتری: / })).toBeVisible()

    expectReadsOnly(reqs)
  })

  operator('an invoice whose customer link is not RESOLVED shows its state and offers no customer link', async ({ page, blocked }) => {
    void blocked
    await page.goto(`/billing/invoices/${DB.unlinkedInvoice}`)
    const card = page.getByRole('region', { name: 'فاکتور' })
    await expect(card.getByText(relationLabel[DB.unlinkedCustomerLink] ?? DB.unlinkedCustomerLink).first()).toBeVisible()
    await expect(card.getByRole('link', { name: 'مشاهده مشتری' })).toHaveCount(0)
    await expect(card.getByText('پیوند تأیید نشده است؛ پیمایش ارائه نمی‌شود.').first()).toBeVisible()
  })

  operator('an unknown customer id is an honest empty page, and a malformed id is a contract error (never data)', async ({ page, blocked }) => {
    void blocked
    const r = await page.request.get('/api/v1/invoices?customer_id=00000000-0000-4000-8000-0000000000ff')
    expect(r.status()).toBe(200)
    expect(((await r.json()) as { items: unknown[] }).items).toEqual([])
    for (const p of ['/api/v1/cases?customer_id=09120000000', '/api/v1/invoices?case_id=7001', '/api/v1/invoices?phone=09120000000']) {
      const bad = await page.request.get(p)
      expect(bad.status(), p).toBe(400)
      expect(bad.headers()['content-type'], p).toContain('application/problem+json')
    }
  })
})

operator.describe('dev-user-002 (no capability, test-only overlay): relationship sections stay locked', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  operator('customer page: every section is denied; no case or invoice data reaches the DOM', async ({ page, blocked }) => {
    void blocked
    await page.goto(`/customers/${DB.customer}`)
    for (const s of [CASES_OF_CUSTOMER, INVOICES_OF_CUSTOMER]) {
      await expect(page.getByRole('region', { name: s }).getByText('دسترسی به این بخش ندارید')).toBeVisible()
      await expect(page.getByRole('region', { name: s }).locator('table')).toHaveCount(0)
    }
    const html = await page.content()
    for (const leak of [DB.invoiceKey, 'synthetic.rm_p01.v1', DB.invoice, DB.synthCase]) expect(html).not.toContain(leak)
  })
})
