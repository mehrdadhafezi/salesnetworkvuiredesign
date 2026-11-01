import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

// All data is SYNTHETIC.
const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const RUN = ID(900)
const allowed = Object.fromEntries(['reporting.metrics.read', 'reporting.mis.read'].map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-8888', allowed_actions: allowed, ...extra })
const run = { run_id: RUN, projection_version: 1, source_cutoff: '2026-11-01T09:00:00Z', generated_at: '2026-11-01T09:00:00Z', outcome: 'SUCCESS',
  sources: [{ source: 'billing', status: 'AVAILABLE', rows_read: 4, reason: 'published' }] }
const meta = (report: string, filters: Record<string, string>, extra: Record<string, unknown> = {}) => ({
  query: { report, scope: 'ORGANIZATION', time_basis: 'CURRENT_SNAPSHOT', run_id: RUN, filters }, available: true, run, freshness: 'FRESH',
  coverage: 'AVAILABLE', source_coverage: { billing: 'AVAILABLE' }, ...extra })
const contract = (id: string, name: string, status = 'FINAL', reason?: string) => ({ metric_id: id, version: 1, name, grain: 'Invoice', source: 'billing FactSource',
  cohort: 'present', time_bases: ['CURRENT_SNAPSHOT'], scope: 'ORGANIZATION', formula: 'count(...)', unit: 'count', freshness_sla_seconds: 86400,
  coverage: 'staged', status, ...(reason ? { status_reason: reason } : {}) })
const invoice = (n: number) => ({ invoice_id: ID(n), source_spec: 'synthetic.invoice.v2', source_key: String(n), presence: 'PRESENT', customer_link_state: 'UNKNOWN',
  case_link_state: 'UNKNOWN', main: { raw: 'synthetic_unfamiliar', mapping: 'UNKNOWN' }, invoice: { raw: null, mapping: 'EMPTY' }, payment: { raw: null, mapping: 'EMPTY' },
  workflow: { raw: null, mapping: 'EMPTY' }, return: { raw: null, mapping: 'EMPTY' }, unknown_state_count: 1, item_count: 1, stage_count: 1, items_complete: true,
  stages_complete: true, first_seen_at: '2026-11-01T09:00:00Z', units: [{ unit: 'IRT', item_state: 'COMPLETE', stage_state: 'NONE', payment_state: 'NONE', class: 'UNKNOWN' }] })

function routes(over: Record<string, (req: Request) => Response> = {}, enabled = true) {
  const unknownFilters = { presence: 'PRESENT', has_unknown_status: 'true' }
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: false, sales_read: false, billing_read: false, reporting_read: enabled } })),
    'GET /api/v1/mis/overview': () => json(env({ data: { run, freshness: 'FRESH', coverage: 'PARTIAL', metrics: [
      { contract: contract('invoices.unknown_status', 'Invoices with an unknown raw state value'), value_state: 'VALUE', meta: meta('invoices', unknownFilters),
        summary: { total: 3, counts: {} } },
      { contract: contract('cases.unknown_state', 'Cases with an unmapped source state'), value_state: 'VALUE', meta: meta('cases', { presence: 'PRESENT' }),
        summary: { total: 0, counts: {} } },
      { contract: contract('custody.by_parity', 'Current custody parity'), value_state: 'UNKNOWN', meta: meta('custody', {}, { available: false, coverage: 'MISSING' }) },
    ], deferred: [contract('cases.completion_rate', 'Completion Rate', 'NOT_FINAL', 'NOT_FINAL: OPD-07')] } })),
    'GET /api/v1/reports/invoices': (req: Request) => {
      const p = new URL(req.url).searchParams
      const filters = Object.fromEntries([...p.entries()].filter(([k]) => ['presence', 'has_unknown_status', 'unit'].includes(k)))
      const page = p.get('cursor') ? { items: [invoice(3)], page: { has_more: false } } : { items: [invoice(1), invoice(2)], page: { has_more: true, next_cursor: 'p-2' } }
      return json(env({ meta: meta('invoices', filters), summary: { total: 3, counts: { main_mapping: { UNKNOWN: 3 } } }, ...page }))
    },
    [`GET /api/v1/reports/invoices/${ID(2)}`]: () => json(env({ meta: meta('invoices', unknownFilters), member: true, data: invoice(2) })),
    'GET /api/v1/reports/cases': () => json(env({ meta: meta('cases', {}, { available: false, run: undefined, freshness: 'UNKNOWN', coverage: 'UNKNOWN' }) })),
    'GET /api/v1/mis/source-quality': () => json(env({ data: { run, freshness: 'STALE', rows: [
      { source: 'customers', grain: 'Customer', coverage: 'AVAILABLE', valid: 1, duplicate_candidates: 1, invalid: 0, unknown: 2, conflict: null,
        note: 'duplicate candidates share only a phone: they are NEVER merged' },
      { source: 'custody', grain: 'Case custody projection', coverage: 'MISSING', valid: null, note: 'missing' },
    ] } })),
    ...over,
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

const exportWords = /export|csv|xlsx|download|خروجی|دانلود/i
const noExport = () => {
  expect(screen.queryAllByRole('button').filter(b => exportWords.test(b.textContent ?? ''))).toHaveLength(0)
  expect(screen.queryAllByRole('link').filter(a => exportWords.test(a.textContent ?? '') || exportWords.test(a.getAttribute('href') ?? ''))).toHaveLength(0)
}

describe('MIS / reporting read (Slice 8)', () => {
  it('shows reporting navigation only when the server reports the feature, and never an Invoice Review page', async () => {
    renderAt('/', routes({}, false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    await waitFor(() => expect(screen.getByText('ورود هنوز فعال نیست')).toBeTruthy())
    expect(screen.queryByRole('link', { name: 'امروز MIS' })).toBeNull()
    expect(screen.queryByRole('link', { name: /Invoice Review|بازبینی فاکتور/ })).toBeNull()
  })

  it('has no Invoice Review route (D-24)', async () => {
    renderAt('/invoice-review', routes())
    expect(await screen.findByRole('heading', { level: 1, name: 'صفحه پیدا نشد' })).toBeTruthy()
  })

  it('shows distinct value states: a real zero, a value, unknown (never zero) and NOT FINAL', async () => {
    renderAt('/mis', routes())
    const unknownCard = await screen.findByRole('region', { name: 'Invoices with an unknown raw state value' })
    expect(within(unknownCard).getByText('3')).toBeTruthy()
    const zeroCard = screen.getByRole('region', { name: 'Cases with an unmapped source state' })
    expect(within(zeroCard).getByText('0')).toBeTruthy()
    const missing = screen.getByRole('region', { name: 'Current custody parity' })
    expect(within(missing).getByText('نامعلوم (بدون مقدار؛ صفر نیست)')).toBeTruthy()
    expect(within(missing).queryByText('0')).toBeNull()
    expect(within(screen.getByRole('list', { name: 'شاخص‌های معوق' })).getByText('Completion Rate')).toBeTruthy()
    // Every card makes its definition, scope, time basis, freshness and coverage discoverable.
    expect(within(unknownCard).getByText(/تعریف شاخص/)).toBeTruthy()
    expect(within(unknownCard).getByText(/دامنه: ORGANIZATION/)).toBeTruthy()
    expect(within(unknownCard).getByText('تازه')).toBeTruthy()
    noExport()
  })

  it('card → table → detail keep the SAME query contract; paging never changes the total', async () => {
    const { calls } = renderAt('/mis', routes())
    const card = await screen.findByRole('region', { name: 'Invoices with an unknown raw state value' })
    const link = within(card).getByRole('link', { name: /جدول همین شاخص/ })
    const href = new URL(link.getAttribute('href') ?? '', 'http://x')
    expect(href.pathname).toBe('/reports/invoices')
    expect(href.searchParams.get('run_id')).toBe(RUN)
    expect(href.searchParams.get('has_unknown_status')).toBe('true')
    fireEvent.click(link)
    expect(await screen.findByTestId('summary-total')).toBeTruthy()
    const reportCalls = () => calls.filter(c => c.url === '/api/v1/reports/invoices')
    await waitFor(() => expect(reportCalls().length).toBeGreaterThan(0))
    const first = reportCalls().at(-1)
    expect(first?.search.get('run_id')).toBe(RUN)
    expect(first?.search.get('has_unknown_status')).toBe('true')
    expect(first?.search.get('presence')).toBe('PRESENT')
    expect(screen.getByTestId('summary-total').textContent).toBe('3')
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    await waitFor(() => expect(reportCalls().at(-1)?.search.get('cursor')).toBe('p-2'))
    expect(reportCalls().at(-1)?.search.get('run_id')).toBe(RUN)
    expect((await screen.findByTestId('summary-total')).textContent).toBe('3')
    fireEvent.click(screen.getByRole('button', { name: 'صفحه قبل' }))
    const rows = await screen.findAllByRole('button', { name: 'جزئیات' })
    fireEvent.click(rows[1] as HTMLElement)
    expect(await screen.findByText('عضو همین پرس‌وجو است (همان شرط کارت و جدول)')).toBeTruthy()
    const detail = calls.filter(c => c.url === `/api/v1/reports/invoices/${ID(2)}`).at(-1)
    expect(detail?.search.get('run_id')).toBe(RUN)
    expect(detail?.search.get('has_unknown_status')).toBe('true')
    expect(detail?.search.get('presence')).toBe('PRESENT')
    for (const c of calls) expect(c.method).toBe('GET')
    noExport()
  })

  it('changing a filter changes the one query contract (and evaluates at the latest cutoff)', async () => {
    const { calls } = renderAt(`/reports/invoices?presence=PRESENT&run_id=${RUN}`, routes())
    await screen.findByTestId('summary-total')
    const form = screen.getByRole('form', { name: 'فیلتر گزارش' })
    fireEvent.change(within(form).getByLabelText('unit'), { target: { value: 'IRR' } })
    fireEvent.submit(form)
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/reports/invoices').at(-1)?.search.get('unit')).toBe('IRR'))
    const last = calls.filter(c => c.url === '/api/v1/reports/invoices').at(-1)
    expect(last?.search.get('presence')).toBe('PRESENT')
    expect(last?.search.get('run_id')).toBeNull()
    expect(screen.queryByLabelText(/phone|تلفن/i)).toBeNull()
  })

  it('a query no run can answer is UNKNOWN, never zero', async () => {
    renderAt('/reports/cases', routes())
    expect(await screen.findByText('هیچ اجرای پروجکشنی به این پرس‌وجو پاسخ نمی‌دهد')).toBeTruthy()
    expect(screen.getByText('نامعلوم است، نه صفر.')).toBeTruthy()
    expect(screen.queryByTestId('summary-total')).toBeNull()
  })

  it('source quality separates not-applicable, unknown (missing source) and real counts; duplicates are never merges', async () => {
    renderAt('/mis/source-quality', routes())
    const table = await screen.findByRole('table', { name: 'کیفیت منبع' })
    expect(within(table).getByText('قابل اعمال نیست')).toBeTruthy()
    expect(within(table).getAllByText('نامعلوم').length).toBeGreaterThan(0)
    expect(within(table).getByText(/NEVER merged/)).toBeTruthy()
    noExport()
  })

  it('renders blocked and denied honestly', async () => {
    renderAt('/mis', routes({ 'GET /api/v1/mis/overview': () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'c-1' }) }))
    expect(await screen.findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    expect(screen.getByText(/reporting\.read_shadow/)).toBeTruthy()
  })
})
