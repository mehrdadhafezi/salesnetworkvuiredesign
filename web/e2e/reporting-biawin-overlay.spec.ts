import type { Page } from '@playwright/test'
import { allReads, expect, expectDenied, expectNoData, NAV, PAGES, readOnly, REPORTS, required, T, unknownId } from './reporting-biawin-fixtures'

// R7-7 class B — TEST-ONLY OVERLAY, BACKEND-ONLY AUTHORIZATION EVIDENCE.
// crm-api runs with CRM_ENV=test and the existing development auth stub
// (CRM_DEV_AUTH=1, refused in production, never combined with sessions).
// This browser context adds the stub header itself; the production SPA
// bundle carries no dev identity. dev-user-001 holds synthetic in-memory
// reporting / Biawin read capabilities (no database grant); dev-user-002
// holds nothing. The data is the R7-5 integrated SYNTHETIC dataset. Every
// expected value below comes from the orchestrator's own SQL over the
// disposable database, so each check is rendered UI == API answer ==
// database. Nothing here shows that production authorizes these reads.

const num = (k: string) => Number(required(k))
const DB = {
  rpRun: required('E2E_RP_RUN_ID'),
  cases: num('E2E_RP_CASES'),
  casesUnknown: num('E2E_RP_CASES_UNKNOWN'),
  legacyLeads: num('E2E_RP_LEGACY_LEADS'),
  custody: num('E2E_RP_CUSTODY'),
  custodyConflict: num('E2E_RP_CUSTODY_CONFLICT'),
  invoices: num('E2E_RP_INVOICES'),
  payments: num('E2E_RP_PAYMENTS'),
  paymentsUnmapped: num('E2E_RP_PAYMENTS_UNMAPPED'),
  paymentsAccepted: num('E2E_RP_PAYMENTS_ACCEPTED'),
  irtSum: required('E2E_RP_IRT_SUM'),
  acceptedIrtSum: required('E2E_RP_ACCEPTED_IRT_SUM'),
  notFinal: num('E2E_RP_NOT_FINAL'),
  contracts: num('E2E_RP_CONTRACTS'),
  bwRun: required('E2E_BW_RUN_ID'),
  subscriptions: num('E2E_BW_SUBSCRIPTIONS'),
  states: Object.fromEntries(required('E2E_BW_STATE_COUNTS').split(',').map(kv => kv.split('=')).map(([k, v]) => [k, Number(v)])) as Record<string, number>,
  unresolvedSub: required('E2E_BW_UNRESOLVED_SUB'),
  unresolved: required('E2E_BW_UNRESOLVED_KEY').split(':') as [string, string],
  unresolvedN: num('E2E_BW_UNRESOLVED_N'),
  unmapped: required('E2E_BW_UNMAPPED_KEY').split(':') as [string, string],
  multiSub: required('E2E_BW_MULTI_SUB'),
  multiKey: required('E2E_BW_MULTI_KEY'),
  multiItems: num('E2E_BW_MULTI_ITEMS'),
  parity: required('E2E_BW_PARITY').split(','),
  parityRows: num('E2E_BW_PARITY_ROWS'),
}
const reportTotal: Record<string, number> = { cases: DB.cases, custody: DB.custody, invoices: DB.invoices, payments: DB.payments }

interface Meta { available: boolean; freshness: string; coverage: string; query: { run_id?: string; report: string }; run?: { run_id: string } | null }
interface Metric { contract: { metric_id: string; name: string; status: string }; value_state: string; summary?: { total: number; amounts?: { unit: string; money?: { amount: string } }[] }; meta: Meta }
interface Envelope<D> { outcome: string; reasons: string[]; data: D }
interface ReportAnswer { outcome: string; reasons: string[]; meta: Meta; summary?: { total: number }; items: Record<string, unknown>[]; page: { has_more: boolean; next_cursor?: string | null } }
interface SubscriptionList { outcome: string; reasons: string[]; meta: { query: { run_id: string } }; summary: { key: string; count: number }[]; items: unknown[] }

async function getJSON<R>(page: Page, path: string): Promise<R> {
  const r = await page.request.get(path)
  expect(r.status(), path).toBe(200)
  return (await r.json()) as R
}

/** Walks a cursor-paged list with a small page size and returns every id once. */
async function walk(page: Page, path: string, idKey: string, limit: number, outcome = 'SUCCESS'): Promise<string[]> {
  const seen: string[] = []
  let cursor: string | null | undefined
  for (let i = 0; i < 50; i++) {
    const sep = path.includes('?') ? '&' : '?'
    const r = await getJSON<ReportAnswer>(page, `${path}${sep}limit=${limit}${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ''}`)
    expect(r.outcome, path).toBe(outcome)
    expect(r.items.length, path).toBeLessThanOrEqual(limit)
    for (const it of r.items) seen.push(String(it[idKey]))
    if (!r.page.has_more) return seen
    cursor = r.page.next_cursor
    expect(cursor, `${path}: has_more needs a cursor`).toBeTruthy()
  }
  throw new Error(`${path}: pagination did not terminate`)
}

/** Only read controls (filters, pager, row detail) are offered in the workspace: no action of any kind. */
async function expectOnlyReadControls(page: Page) {
  const allowed = new Set([T.applyFilter, T.nextPage, T.prevPage, T.details])
  const names = await page.locator('main').getByRole('button').evaluateAll(els => els.map(e => (e.textContent ?? '').trim()))
  for (const n of names) expect(allowed.has(n), `unexpected control "${n}"`).toBe(true)
  await expect(page.getByRole('dialog')).toHaveCount(0)
}

/** Biawin reads of a run older than 24h answer PARTIAL / STALE_STATE (the R7-5 cutoff is fixed in the past): stale is never shown as fresh. */
const STALE = { outcome: 'PARTIAL', reasons: ['STALE_STATE'] }

/** An allowlisted filter of the page. */
const select = (page: Page, name: string) => page.locator(`main select[name="${name}"]`)

const operator = readOnly

operator.describe('dev-user-001 (synthetic read capabilities, test-only overlay): pages render the seeded projections and shadow import', () => {
  operator.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-001' } })

  operator('the session reports the dev stub; navigation follows the flags', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    const s = await getJSON<{ data: { auth_mode: string; authenticated: boolean; features: Record<string, boolean> } }>(page, '/api/v1/session')
    expect(s.data.auth_mode).toBe('dev_stub')
    expect(s.data.authenticated).toBe(true)
    expect(s.data.features.reporting_read).toBe(true)
    expect(s.data.features.biawin_fulfillment_read).toBe(true)
    await page.goto('/')
    for (const name of NAV) await expect(page.getByRole('link', { name, exact: true }), name).toBeVisible()
  })

  operator('MIS overview: health, final metrics with their real values, deferred contracts without a value', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    const api = await getJSON<Envelope<{ freshness: string; coverage: string; run: { run_id: string } | null; metrics: Metric[]; deferred: { metric_id: string; status: string }[] }>>(page, '/api/v1/mis/overview')
    expect(api.outcome).toBe('SUCCESS')
    expect(api.data.run?.run_id).toBe(DB.rpRun)
    await page.goto('/')
    await page.getByRole('link', { name: T.mis, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.mis, exact: true })).toBeVisible()
    await expect(page.getByText(T.readOnly).first()).toBeVisible()
    const freshness: Record<string, string> = { FRESH: 'تازه', STALE: 'کهنه (قدیمی‌تر از قرارداد)', UNKNOWN: 'تازگی نامعلوم' }
    await expect(page.getByText(freshness[api.data.freshness] ?? api.data.freshness, { exact: true }).first()).toBeVisible()

    // Every FINAL metric renders exactly the API value; the API value equals the database.
    const fromDB: Record<string, number> = {
      'cases.staged_total': DB.cases, 'cases.legacy_leads_only': DB.legacyLeads, 'cases.unknown_state': DB.casesUnknown,
      'invoices.staged_total': DB.invoices, 'payments.by_state': DB.payments, 'payments.accepted_attempt_amount': DB.paymentsAccepted,
    }
    await expect(page.locator('section.rp-card')).toHaveCount(api.data.metrics.length)
    for (const m of api.data.metrics) {
      const card = page.getByRole('region', { name: m.contract.name, exact: true })
      expect(m.value_state, m.contract.metric_id).toBe('VALUE')
      await expect(card.locator('.rp-value'), m.contract.metric_id).toHaveText(String(m.summary?.total))
      if (m.contract.metric_id in fromDB) expect(m.summary?.total, m.contract.metric_id).toBe(fromDB[m.contract.metric_id])
    }
    const amount = (id: string) => page.getByRole('region', { name: api.data.metrics.find(m => m.contract.metric_id === id)?.contract.name ?? id, exact: true })
    await expect(amount('payments.by_state').getByText(`${DB.irtSum} IRT`, { exact: true })).toBeVisible()
    await expect(amount('payments.accepted_attempt_amount').getByText(`${DB.acceptedIrtSum} IRT`, { exact: true })).toBeVisible()

    // Deferred (NOT FINAL) contracts are listed without a value.
    expect(api.data.deferred.every(d => d.status === 'NOT_FINAL')).toBe(true)
    const deferred = page.getByRole('list', { name: 'شاخص‌های معوق' })
    await expect(deferred.locator('li')).toHaveCount(api.data.deferred.length)
    await expect(deferred.locator('.num, .rp-value')).toHaveCount(0)
    expect(api.data.deferred.length, 'deferred contracts are NOT_FINAL contracts of the database').toBeGreaterThan(0)
    expect(api.data.deferred.length).toBeLessThanOrEqual(DB.notFinal)
    await expectOnlyReadControls(page)
  })

  operator('MIS drill-down keeps the query contract: the card value equals the report total', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    const api = await getJSON<Envelope<{ metrics: Metric[] }>>(page, '/api/v1/mis/overview')
    const m = api.data.metrics.find(x => x.contract.metric_id === 'cases.unknown_state')
    expect(m?.summary?.total, 'cases.unknown_state = database').toBe(DB.casesUnknown)
    await page.goto('/mis')
    const card = page.getByRole('region', { name: m?.contract.name ?? '', exact: true })
    await expect(card.locator('.rp-value')).toHaveText(String(DB.casesUnknown))
    await card.getByRole('link', { name: /جدول همین شاخص/ }).click()
    await expect(page).toHaveURL(/\/reports\/cases\?/)
    await expect(page).toHaveURL(new RegExp(`run_id=${DB.rpRun}`))
    await expect(page.getByTestId('summary-total')).toHaveText(String(DB.casesUnknown))
    await expect(page.locator('table.rp-table tbody tr')).toHaveCount(DB.casesUnknown)
  })

  operator('source quality: counts per source equal the API; a not-applicable count is never shown as zero', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    type Row = { source: string; coverage: string; valid?: number | null; duplicate_candidates?: number | null; invalid?: number | null; unknown?: number | null; conflict?: number | null }
    const api = await getJSON<Envelope<{ rows: Row[] }>>(page, '/api/v1/mis/source-quality')
    await page.goto('/mis/source-quality')
    await expect(page.getByRole('heading', { name: T.sourceQuality, exact: true })).toBeVisible()
    const rows = page.locator('table.rp-table tbody tr')
    await expect(rows).toHaveCount(api.data.rows.length)
    let notApplicable = 0
    for (const [i, r] of api.data.rows.entries()) {
      const cells = rows.nth(i).locator('td')
      for (const [j, v] of [r.valid, r.duplicate_candidates, r.invalid, r.unknown, r.conflict].entries()) {
        const want = r.coverage !== 'AVAILABLE' && r.coverage !== 'PARTIAL' ? 'نامعلوم' : v === null || v === undefined ? 'قابل اعمال نیست' : String(v)
        if (want === 'قابل اعمال نیست') notApplicable++
        await expect(cells.nth(3 + j), `${r.source} column ${j}`).toHaveText(want)
      }
    }
    expect(notApplicable, 'the dataset has not-applicable counts (null is not zero)').toBeGreaterThan(0)
    const cases = api.data.rows.find(r => r.source === 'cases')
    expect(cases?.unknown, 'cases unknown = unmapped raw state (database)').toBe(DB.casesUnknown)
    const custody = api.data.rows.find(r => r.source === 'custody')
    expect(custody?.conflict, 'custody conflict (database)').toBe(DB.custodyConflict)
  })

  operator('reconciliation: F02 is NOT EVALUATED (no number), F03 values equal the database, runs and contracts are listed', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    const api = await getJSON<Envelope<{ f02: Metric; f03_total_cases: Metric; f03_legacy_leads_only: Metric; runs: { run_id: string }[]; contracts: unknown[] }>>(page, '/api/v1/mis/reconciliation')
    await page.goto('/mis/reconciliation')
    await expect(page.getByRole('heading', { name: T.reconciliation, exact: true })).toBeVisible()
    const f02 = page.getByRole('region', { name: api.data.f02.contract.name, exact: true })
    expect(api.data.f02.value_state).toBe('NOT_EVALUATED')
    expect(api.data.f02.summary).toBeUndefined()
    await expect(f02.getByText('ارزیابی نشده (NOT FINAL)')).toBeVisible()
    await expect(f02.locator('.rp-value')).toHaveCount(0)
    await expect(f02.getByRole('link', { name: /جدول همین شاخص/ })).toHaveCount(0)
    await expect(page.getByRole('region', { name: api.data.f03_total_cases.contract.name, exact: true }).locator('.rp-value')).toHaveText(String(DB.cases))
    await expect(page.getByRole('region', { name: api.data.f03_legacy_leads_only.contract.name, exact: true }).locator('.rp-value')).toHaveText(String(DB.legacyLeads))
    const runs = page.locator('table.rp-table tbody tr')
    await expect(runs).toHaveCount(api.data.runs.length)
    await expect(runs.first().locator('td').first()).toHaveText(DB.rpRun.slice(0, 8))
    expect(api.data.contracts.length, 'contracts = database').toBe(DB.contracts)
    await expect(page.getByRole('list', { name: 'قراردادهای شاخص' }).locator('li')).toHaveCount(api.data.contracts.length)
    await expect(page.getByRole('button', { name: T.nextPage })).toBeDisabled()
  })

  for (const [report, heading] of REPORTS) {
    operator(`report ${report}: summary, table and API equal the database; cursor paging visits every row once`, async ({ page, blocked, writes }) => {
      void blocked
      void writes
      const api = await getJSON<ReportAnswer>(page, `/api/v1/reports/${report}?limit=25`)
      expect(api.outcome).toBe('SUCCESS')
      expect(api.meta.run?.run_id).toBe(DB.rpRun)
      expect(api.summary?.total).toBe(reportTotal[report])
      await page.goto(`/reports/${report}`)
      await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible()
      await expect(page.getByTestId('summary-total')).toHaveText(String(reportTotal[report]))
      await expect(page.locator('table.rp-table tbody tr')).toHaveCount(api.items.length)
      await expect(page.getByText(`run ${DB.rpRun.slice(0, 8)}`, { exact: false }).first()).toBeVisible()
      const id = report === 'payments' ? 'attempt_id' : report === 'invoices' ? 'invoice_id' : 'case_id'
      const ids = await walk(page, `/api/v1/reports/${report}?run_id=${DB.rpRun}`, id, 2)
      expect(ids.length).toBe(reportTotal[report])
      expect(new Set(ids).size).toBe(reportTotal[report])
      await expectOnlyReadControls(page)
    })
  }

  operator('payments report: amounts per explicit unit, an unmapped state stays UNMAPPED, filters and a real zero', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    await page.goto('/reports/payments')
    await expect(page.getByTestId('summary-total')).toHaveText(String(DB.payments))
    await expect(page.getByText(`${DB.irtSum} IRT (${DB.payments})`, { exact: true })).toBeVisible()
    await expect(page.locator('table.rp-table tbody td').filter({ hasText: /→ UNMAPPED$/ })).toHaveCount(DB.paymentsUnmapped)
    // Filter: the card, the table and the API share one query contract.
    await select(page, 'state').selectOption('ACCEPTED')
    await page.getByRole('button', { name: T.applyFilter }).click()
    await expect(page).toHaveURL(/state=ACCEPTED/)
    await expect(page.getByTestId('summary-total')).toHaveText(String(DB.paymentsAccepted))
    await expect(page.getByText(`${DB.acceptedIrtSum} IRT (${DB.paymentsAccepted})`, { exact: true })).toBeVisible()
    await expect(page.locator('table.rp-table tbody tr')).toHaveCount(DB.paymentsAccepted)
    const accepted = await getJSON<ReportAnswer>(page, '/api/v1/reports/payments?state=ACCEPTED')
    expect(accepted.summary?.total).toBe(DB.paymentsAccepted)
    // A filter with no member: a real, evaluated zero (the projection answered).
    await select(page, 'state').selectOption('CANCELLED')
    await page.getByRole('button', { name: T.applyFilter }).click()
    await expect(page).toHaveURL(/state=CANCELLED/)
    const cancelled = await getJSON<ReportAnswer>(page, '/api/v1/reports/payments?state=CANCELLED')
    expect(cancelled.meta.available).toBe(true)
    expect(cancelled.summary?.total).toBe(0)
    await expect(page.getByTestId('summary-total')).toHaveText('0')
    await expect(page.getByText('صفر ردیف (نتیجه واقعی این پرس‌وجو)')).toBeVisible()
  })

  operator('invoice and custody reports: UNKNOWN and CONFLICT evidence is shown as such', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    const inv = await getJSON<ReportAnswer>(page, '/api/v1/reports/invoices?limit=25')
    const unknownMain = inv.items.filter(i => (i.main as { mapping: string }).mapping === 'UNKNOWN')
    expect(unknownMain.length, 'the dataset has an invoice with an unknown main state').toBeGreaterThan(0)
    await page.goto('/reports/invoices')
    for (const i of unknownMain) {
      const raw = (i.main as { raw: string }).raw
      await expect(page.locator('table.rp-table tbody td').filter({ hasText: `${raw} (UNKNOWN)` })).toHaveCount(1)
    }
    const classes = inv.items.flatMap(i => (i.units as { unit: string; class: string }[]).map(u => `${u.unit}: ${u.class}`))
    for (const c of ['UNKNOWN', 'INCOMPLETE']) expect(classes.some(x => x.endsWith(c)), `an invoice total class ${c}`).toBe(true)
    for (const c of new Set(classes)) await expect(page.locator('table.rp-table tbody td').filter({ hasText: c }).first(), c).toBeVisible()
    await page.goto('/reports/custody')
    await expect(page.locator('table.rp-table tbody td').filter({ hasText: /^CONFLICT$/ })).toHaveCount(DB.custodyConflict)
  })

  operator('row detail is pinned to the run that answered; an unknown run is UNKNOWN, never zero', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    await page.goto('/reports/cases')
    await page.getByRole('button', { name: T.details }).first().click()
    await expect(page).toHaveURL(new RegExp(`run_id=${DB.rpRun}`))
    await expect(page).toHaveURL(/row=/)
    await expect(page.getByText('عضو همین پرس‌وجو است (همان شرط کارت و جدول)')).toBeVisible()
    await expect(page.getByRole('list', { name: 'مقادیر ردیف' }).locator('li')).toHaveCount(6)
    // A pinned run that does not exist answers no value and the page shows none (never 0).
    // DEF-RP-01 (fixed in #62): every report list answers an explicit run_id that names no
    // projection run with 404 problem+json, like the row reads, and carries no value of any kind.
    for (const report of ['cases', 'custody', 'invoices', 'payments']) {
      const path = `/api/v1/reports/${report}?run_id=${unknownId}`
      const rp = await page.request.get(path)
      expect(rp.status(), `${path} (DEF-RP-01)`).toBe(404)
      expect(rp.headers()['content-type'], path).toContain('application/problem+json')
      const raw = await rp.text()
      const rpBody = JSON.parse(raw) as { status?: number; summary?: unknown; items?: unknown; meta?: unknown; data?: unknown; page?: unknown }
      expect(rpBody.status, path).toBe(404)
      for (const k of ['summary', 'items', 'meta', 'data', 'page'] as const) expect(rpBody[k], `${path} ${k}`).toBeUndefined()
      for (const leak of ['"total"', '"amount"', '"money"', '"counts"', 'IRT', 'IRR']) expect(raw, `${path} carries ${leak}`).not.toContain(leak)
    }
    // Biawin answers an unknown run as "no run": UNKNOWN, with no meta, counts or rows.
    const bw = await getJSON<{ outcome: string; meta?: unknown; summary?: unknown; items?: unknown }>(page, `/api/v1/biawin/subscriptions?run_id=${unknownId}`)
    expect(bw.outcome).toBe('UNKNOWN')
    expect(bw.meta).toBeUndefined()
    expect(bw.summary).toBeUndefined()
    expect(bw.items).toBeUndefined()
    expect((await page.request.get(`/api/v1/reports/cases/${unknownId}?run_id=${DB.rpRun}`)).status(), 'unknown row').toBe(404)
    for (const path of [`/reports/cases?run_id=${unknownId}`, `/biawin/subscriptions?run_id=${unknownId}`]) {
      await page.goto(path)
      await expect(page.getByTestId('summary-total'), path).toHaveCount(0)
      await expect(page.locator('[data-testid^="sub-count-"]'), path).toHaveCount(0)
      await expect(page.getByRole('table'), path).toHaveCount(0)
    }
  })

  operator('Biawin subscriptions: state counts, rows and links equal the shadow import; unresolved and unmapped evidence stays visible', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    const api = await getJSON<SubscriptionList>(page, '/api/v1/biawin/subscriptions?limit=50')
    expect({ outcome: api.outcome, reasons: api.reasons }).toEqual(STALE)
    expect(api.meta.query.run_id).toBe(DB.bwRun)
    expect(api.items.length).toBe(DB.subscriptions)
    for (const [k, n] of Object.entries(DB.states)) expect(api.summary.find(c => c.key === k)?.count ?? 0, k).toBe(n)
    await page.goto('/')
    await page.getByRole('link', { name: T.subscriptions, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.subscriptions, exact: true })).toBeVisible()
    await expect(page.getByRole('note', { name: 'هشدار نمای سایه' })).toBeVisible()
    await expect(page.getByText(`run ${DB.bwRun.slice(0, 8)}`, { exact: false })).toBeVisible()
    await expect(page.getByText('کهنه (قدیمی‌تر از ۲۴ ساعت)', { exact: true })).toBeVisible()
    for (const [k, n] of Object.entries(DB.states)) await expect(page.getByTestId(`sub-count-${k}`), k).toHaveText(String(n))
    const rows = page.locator('table.bw-table tbody tr')
    await expect(rows).toHaveCount(DB.subscriptions)
    const [uKey, uInvoice] = DB.unresolved
    const unresolved = rows.filter({ has: page.getByRole('link', { name: `#${uKey}`, exact: true }) })
    await expect(unresolved.locator('td').nth(5)).toContainText('نامعلوم')
    await expect(unresolved.locator('td').nth(5)).toContainText(`#${uInvoice}`)
    const [mKey, mRaw] = DB.unmapped
    const unmapped = rows.filter({ has: page.getByRole('link', { name: `#${mKey}`, exact: true }) })
    await expect(unmapped.locator('td').nth(2)).toContainText('نامعلوم (مقدار خام حفظ شد)')
    await expect(unmapped.locator('td').nth(2)).toContainText(`raw: ${mRaw}`)
    const walked = await walk(page, `/api/v1/biawin/subscriptions?run_id=${DB.bwRun}`, 'subscription_id', 2, STALE.outcome)
    expect(new Set(walked).size).toBe(DB.subscriptions)
    expect(walked.length).toBe(DB.subscriptions)
    await expectOnlyReadControls(page)
  })

  operator('Biawin filters: unresolved invoices and an empty state (a real zero in this run)', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    await page.goto('/biawin/subscriptions')
    await select(page, 'invoice_link_state').selectOption('UNKNOWN')
    await page.getByRole('button', { name: T.applyFilter }).click()
    await expect(page).toHaveURL(/invoice_link_state=UNKNOWN/)
    await expect(page.locator('table.bw-table tbody tr')).toHaveCount(DB.unresolvedN)
    await expect(page.getByRole('link', { name: `#${DB.unresolved[0]}`, exact: true })).toBeVisible()
    expect(DB.states.EMPTY, 'the dataset has no EMPTY subscription').toBe(0)
    await page.goto('/biawin/subscriptions?state=EMPTY')
    await expect(page.getByText('هیچ اشتراکی با این فیلتر در این اجرا نیست')).toBeVisible()
    await expect(page.getByTestId('sub-count-EMPTY')).toHaveText('0')
    const api = await getJSON<{ outcome: string; items: unknown[] }>(page, '/api/v1/biawin/subscriptions?state=EMPTY')
    expect(api.outcome).toBe(STALE.outcome)
    expect(api.items).toEqual([])
  })

  operator('Biawin subscription detail: every item of the pinned run, unresolved links stay UNKNOWN', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    await page.goto('/biawin/subscriptions')
    await page.getByRole('link', { name: `#${DB.multiKey}`, exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`/biawin/subscriptions/${DB.multiSub}\\?run_id=${DB.bwRun}`))
    await expect(page.getByRole('heading', { name: T.subscription, exact: true })).toBeVisible()
    await expect(page.getByText(`آیتم‌ها (${DB.multiItems})`)).toBeVisible()
    await expect(page.locator('table.bw-table tbody tr')).toHaveCount(DB.multiItems)
    const items = await getJSON<{ item_count: number; items: unknown[] }>(page, `/api/v1/biawin/subscriptions/${DB.multiSub}/items?run_id=${DB.bwRun}&limit=50`)
    expect(items.item_count).toBe(DB.multiItems)
    const walked = await walk(page, `/api/v1/biawin/subscriptions/${DB.multiSub}/items?run_id=${DB.bwRun}`, 'item_id', 5, STALE.outcome)
    expect(walked.length).toBe(DB.multiItems)
    expect(new Set(walked).size).toBe(DB.multiItems)
    await page.goto(`/biawin/subscriptions/${DB.unresolvedSub}`)
    await expect(page.getByRole('heading', { name: `اشتراک #${DB.unresolved[0]}` })).toBeVisible()
    const invoice = page.locator('dt', { hasText: /^فاکتور منبع$/ }).locator('xpath=following-sibling::dd[1]')
    await expect(invoice).toContainText('نامعلوم')
    await expect(invoice).toContainText(`#${DB.unresolved[1]}`)
    await expectOnlyReadControls(page)
    // An id that names no subscription is a 404, never an empty subscription.
    expect((await page.request.get(`/api/v1/biawin/subscriptions/${unknownId}`)).status()).toBe(404)
  })

  operator('Biawin parity: the recorded check, its counts and an UNKNOWN outcome (never MATCH, never zero)', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    const [legacy, target, matched, missing, extra, unknown, conflict, notrec, outcome] = DB.parity
    const api = await getJSON<{ outcome: string; reasons: string[]; check: { outcome: string; reasons: string[]; rows: { source_key: string; result: string }[] } }>(page, '/api/v1/biawin/parity?limit=50')
    expect({ outcome: api.outcome, reasons: api.reasons }).toEqual(STALE)
    expect(api.check.outcome).toBe(outcome)
    expect(api.check.rows.length).toBe(DB.parityRows)
    await page.goto('/')
    await page.getByRole('link', { name: T.parity, exact: true }).click()
    await expect(page.getByRole('heading', { name: T.parity, exact: true })).toBeVisible()
    const label: Record<string, string> = { MATCH: 'منطبق', MISMATCH: 'مغایر', UNKNOWN: 'نامعلوم (صفر نیست)', INCOMPLETE: 'ناقص' }
    await expect(page.getByText(label[outcome ?? ''] ?? '', { exact: true })).toBeVisible()
    for (const r of api.check.reasons) await expect(page.getByText(r, { exact: true })).toBeVisible()
    for (const [id, v] of [['par-legacy', legacy], ['par-target', target], ['par-matched', matched], ['par-missing', missing], ['par-extra', extra],
      ['par-unknown', unknown], ['par-conflict', conflict], ['par-notrec', notrec]] as [string, string][]) {
      await expect(page.getByTestId(id), id).toHaveText(v)
    }
    await expect(page.locator('table.bw-table tbody tr')).toHaveCount(DB.parityRows)
    const unknownRows = api.check.rows.filter(r => r.result === 'UNKNOWN')
    expect(unknownRows.length, 'the dataset has an UNKNOWN parity row').toBeGreaterThan(0)
    await select(page, 'result').selectOption('UNKNOWN')
    await page.getByRole('button', { name: T.applyFilter }).click()
    await expect(page).toHaveURL(/result=UNKNOWN/)
    await expect(page.locator('table.bw-table tbody tr')).toHaveCount(unknownRows.length)
    for (const r of unknownRows) await expect(page.getByText(`#${r.source_key}`, { exact: true })).toBeVisible()
    await expectOnlyReadControls(page)
  })
})

const viewer = readOnly

viewer.describe('dev-user-002 (no grant): every reporting / MIS / Biawin read is refused', () => {
  viewer.use({ extraHTTPHeaders: { 'X-Dev-User': 'dev-user-002' } })

  viewer('the pages show the no-access state; the API answers 403 ACTION_DENIED', async ({ page, blocked, writes }) => {
    void blocked
    void writes
    for (const [path, heading] of [...PAGES, [`/biawin/subscriptions/${DB.multiSub}`, T.subscription] as [string, string]]) {
      await page.goto(path)
      await expect(page.getByRole('heading', { name: heading, exact: true }), path).toBeVisible()
      await expectNoData(page, T.denied, path)
    }
    const reads = allReads({ caseId: unknownId, invoiceId: unknownId, attemptId: unknownId, subscription: DB.multiSub, reportingRun: DB.rpRun, biawinRun: DB.bwRun })
    for (const path of reads.filter(p => !p.startsWith('/api/v1/invoice-review'))) await expectDenied(await page.request.get(path), path)
  })
})
