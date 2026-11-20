import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

// RM-P01 Customer ⇄ Case ⇄ Invoice read navigation. All data is SYNTHETIC.
// Navigation is offered only for RESOLVED links; every other state is shown
// without a link, and no identity is derived from phone or legacy keys.
const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const CUST = ID(1)
const CUST_OTHER = ID(2)
const CASE_A = ID(101)
const CASE_B = ID(102)
const CASE_X = ID(109) // a Case whose customer link is CONFLICT (candidate CUST)
const INV_A = ID(201)
const INV_U = ID(202)
const INV_R = ID(203)

const allowed = Object.fromEntries(['customers.customer.read', 'cases.case.read', 'billing.invoice.read'].map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-3600', allowed_actions: allowed, ...extra })
const prov = { source_system: 'wp', source_spec: 'legacy.invoice_issued_to.v1', source_key: '1001', batch_id: ID(900), imported_at: '2026-10-29T09:00:00Z',
  refreshed_at: '2026-10-29T09:00:00Z', presence: 'PRESENT' }
const caseSrc = (key: string) => ({ source_system: 'wp', source_spec: 'legacy.lead.v1', source_key: key, channel_raw: null, origin_ref_raw: null,
  first_batch_id: ID(900), last_batch_id: ID(900), presence: 'PRESENT', first_seen_at: '2026-10-31T09:00:00Z', last_seen_at: '2026-10-31T09:00:00Z' })
const invSrc = (key: string) => ({ source_system: 'wp', source_spec: 'legacy.invoice.v2', source_key: key, first_batch_id: ID(900), last_batch_id: ID(900),
  presence: 'PRESENT', first_seen_at: '2026-11-01T09:00:00Z', last_seen_at: '2026-11-01T09:00:00Z' })

const customer = { customer_id: CUST, subject_kind: 'INVOICE_ISSUED_TO', resolution_state: 'RESOLVED', resolution_reason: 'explicit account reference',
  name_masked: 'Sy********1', observed_raw: null, contact_count: 0, identifier_count: 0, evidence_count: 0, open_case_count: 0, conflict_case_count: 0, provenance: prov }

function caseRow(id: string, key: string, over: Record<string, unknown> = {}) {
  return { case_id: id, workflow_type: 'LEGACY_LEAD', state_namespace: 'lead.status', state_raw: 'new', state_mapping: 'KNOWN', customer_id: CUST,
    customer_link_state: 'RESOLVED', customer_link_reason: 'explicit customer reference', presence: 'PRESENT', source: caseSrc(key), alias_count: 0, history_count: 0,
    imported_at: '2026-10-31T09:00:00Z', refreshed_at: '2026-10-31T09:00:00Z', ...over }
}
const resolved = (id: string) => ({ id, state: 'RESOLVED', basis: 'explicit reference in the invoice row' })
const unlinked = (state: string, id: string | null = null) => ({ id, state, basis: `synthetic ${state} basis` })

function invoice(id: string, key: string, over: Record<string, unknown> = {}) {
  return { invoice_id: id, code_raw: `SYN-${key}`, lifecycle_state: 'UNKNOWN', customer: resolved(CUST), case: resolved(CASE_A), items_source_complete: true,
    stages_source_complete: true, item_count: 1, stage_count: 1, unknown_state_count: 0, issued_at_raw: null, source: invSrc(key), ...over }
}

const cases: Record<string, ReturnType<typeof caseRow>> = {
  [CASE_A]: caseRow(CASE_A, '7001'),
  [CASE_B]: caseRow(CASE_B, '7002'),
  [CASE_X]: caseRow(CASE_X, '7009', { customer_link_state: 'CONFLICT', customer_link_reason: 'two explicit references disagree' }),
}
const invoices: Record<string, ReturnType<typeof invoice>> = {
  [INV_A]: invoice(INV_A, '9001'),
  [INV_U]: invoice(INV_U, '9002', { case: unlinked('UNKNOWN') }),
  [INV_R]: invoice(INV_R, '9003', { customer: unlinked('RELATED', CUST_OTHER), case: unlinked('CONFLICT', CASE_B) }),
}

type Handler = (req: Request) => Response | Promise<Response>
const q = (req: Request) => new URL(req.url).searchParams

function routes(over: Record<string, Handler> = {}): Record<string, Handler> {
  const r: Record<string, Handler> = {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: true, sales_read: true, billing_read: true } })),
    [`GET /api/v1/customers/${CUST}`]: () => json(env({ data: { customer, identifiers: [], merge_cases: [] } })),
    [`GET /api/v1/customers/${CUST}/contact-points`]: () => json(env({ items: [], page: { has_more: false } })),
    [`GET /api/v1/customers/${CUST}/evidence`]: () => json(env({ items: [], page: { has_more: false } })),
    // Server contract: customer_id returns only RESOLVED links (two pages of one).
    'GET /api/v1/cases': req => {
      const s = q(req)
      if (s.get('customer_id') !== CUST) return json(env({ items: [], page: { has_more: false } }))
      return json(env(s.get('cursor') === 'cases-2' ? { items: [cases[CASE_B]], page: { has_more: false } }
        : { items: [cases[CASE_A]], page: { has_more: true, next_cursor: 'cases-2' } }))
    },
    'GET /api/v1/invoices': req => {
      const s = q(req)
      const all = Object.values(invoices).filter(v =>
        (!s.get('customer_id') || (v.customer.state === 'RESOLVED' && v.customer.id === s.get('customer_id'))) &&
        (!s.get('case_id') || (v.case.state === 'RESOLVED' && v.case.id === s.get('case_id'))))
      return json(env({ items: all, page: { has_more: false } }))
    },
    'GET /api/v1/custody/assignments': () => json(env({ items: [], page: { has_more: false } })),
  }
  for (const [id, c] of Object.entries(cases)) {
    r[`GET /api/v1/cases/${id}`] = () => json(env({ data: { case: c, aliases: [] } }))
    r[`GET /api/v1/cases/${id}/history`] = () => json(env({ items: [], page: { has_more: false } }))
    r[`GET /api/v1/cases/${id}/attribution`] = () => json(env({ items: [] }))
  }
  for (const [id, v] of Object.entries(invoices)) {
    r[`GET /api/v1/invoices/${id}`] = () => json(env({ data: { invoice: v, product_ref_raw: null, pay_method_raw: null, payment_plan_raw: null,
      current_stage_raw: null, updated_at_raw: null, states: [], declared: [] } }))
    r[`GET /api/v1/invoices/${id}/items`] = () => json(env({ items: [], page: { has_more: false } }))
    r[`GET /api/v1/invoices/${id}/stages`] = () => json(env({ items: [], page: { has_more: false } }))
  }
  return { ...r, ...over }
}

function renderAt(path: string, r: Record<string, Handler>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

const section = (name: string) => screen.getByRole('region', { name })
const hrefs = (el: HTMLElement) => within(el).queryAllByRole('link').map(a => a.getAttribute('href'))
const CASES_OF_CUSTOMER = 'پرونده‌های این مشتری (پیوند تأییدشده)'
const INVOICES_OF_CUSTOMER = 'فاکتورهای این مشتری (پیوند تأییدشده)'
const INVOICES_OF_CASE = 'فاکتورهای این پرونده (پیوند تأییدشده)'

describe('RM-P01 customer ⇄ case ⇄ invoice navigation', () => {
  it('customer page lists RESOLVED cases and invoices through exact-id filters, with cursor paging', async () => {
    const { calls } = renderAt(`/customers/${CUST}`, routes())
    const casesCard = await waitFor(() => section(CASES_OF_CUSTOMER))
    expect(await within(casesCard).findByRole('link', { name: 'legacy.lead.v1 #7001' })).toBeTruthy()
    expect(hrefs(casesCard)).toEqual([`/sales/cases/${CASE_A}`])
    const caseCall = calls.find(c => c.url === '/api/v1/cases')
    expect(caseCall?.search.get('customer_id')).toBe(CUST)
    // Only the native id travels: no phone, name or legacy-key identity parameter.
    expect([...(caseCall?.search.keys() ?? [])].sort()).toEqual(['customer_id', 'limit'])

    fireEvent.click(within(casesCard).getByRole('button', { name: 'صفحه بعد' }))
    expect(await within(casesCard).findByRole('link', { name: 'legacy.lead.v1 #7002' })).toBeTruthy()
    const second = calls.filter(c => c.url === '/api/v1/cases').at(-1)
    expect(second?.search.get('cursor')).toBe('cases-2')
    expect(second?.search.get('customer_id')).toBe(CUST)
    expect(within(casesCard).getByText('صفحه 2')).toBeTruthy()
    fireEvent.click(within(casesCard).getByRole('button', { name: 'صفحه قبل' }))
    expect(await within(casesCard).findByRole('link', { name: 'legacy.lead.v1 #7001' })).toBeTruthy()

    const invCard = section(INVOICES_OF_CUSTOMER)
    await within(invCard).findByRole('link', { name: 'legacy.invoice.v2 #9001' })
    // INV_R (RELATED customer) is not returned for this customer.
    expect(within(invCard).queryByText(/#9003/)).toBeNull()
    // INV_U's case is UNKNOWN: invoice is listed and linked, its case is not.
    expect(hrefs(invCard).sort()).toEqual([`/billing/invoices/${INV_A}`, `/billing/invoices/${INV_U}`, `/sales/cases/${CASE_A}`].sort())
    expect(within(invCard).getAllByText('نامعلوم').length).toBe(1)
    expect(calls.find(c => c.url === '/api/v1/invoices')?.search.get('customer_id')).toBe(CUST)
    for (const c of calls) expect(c.method).toBe('GET')
  })

  it('navigates Customer → Case → Invoice and back (Invoice → Case, Invoice → Customer)', async () => {
    const { router, calls } = renderAt(`/customers/${CUST}`, routes())
    fireEvent.click(await within(await waitFor(() => section(CASES_OF_CUSTOMER))).findByRole('link', { name: 'legacy.lead.v1 #7001' }))
    await screen.findByRole('heading', { level: 1, name: 'پرونده: legacy.lead.v1 #7001' })
    expect(router.state.location.pathname).toBe(`/sales/cases/${CASE_A}`)
    // Case → customer link (RESOLVED) and the case's RESOLVED invoices.
    expect(screen.getByRole('link', { name: 'مشاهده مشتری' }).getAttribute('href')).toBe(`/customers/${CUST}`)
    const invCard = await waitFor(() => section(INVOICES_OF_CASE))
    await within(invCard).findByRole('link', { name: 'legacy.invoice.v2 #9001' })
    expect(calls.filter(c => c.url === '/api/v1/invoices').at(-1)?.search.get('case_id')).toBe(CASE_A)
    expect(within(invCard).queryByText(/#9002|#9003/)).toBeNull()

    fireEvent.click(within(invCard).getByRole('link', { name: 'legacy.invoice.v2 #9001' }))
    await screen.findByRole('heading', { level: 1, name: 'فاکتور: legacy.invoice.v2 #9001' })
    expect(screen.getByRole('link', { name: 'مشاهده مشتری' }).getAttribute('href')).toBe(`/customers/${CUST}`)
    fireEvent.click(screen.getByRole('link', { name: 'مشاهده پرونده' }))
    await screen.findByRole('heading', { level: 1, name: 'پرونده: legacy.lead.v1 #7001' })
    expect(router.state.location.pathname).toBe(`/sales/cases/${CASE_A}`)

    await router.navigate(`/billing/invoices/${INV_A}`)
    await screen.findByRole('heading', { level: 1, name: 'فاکتور: legacy.invoice.v2 #9001' })
    fireEvent.click(await screen.findByRole('link', { name: 'مشاهده مشتری' }))
    await screen.findByRole('heading', { level: 1, name: 'مشتری: Sy********1' })
    expect(router.state.location.pathname).toBe(`/customers/${CUST}`)
  })

  it('never links an unconfirmed relationship, even when it carries a candidate id', async () => {
    renderAt(`/billing/invoices/${INV_R}`, routes())
    await screen.findByRole('heading', { level: 1, name: 'فاکتور: legacy.invoice.v2 #9003' })
    const invoiceCard = section('فاکتور')
    expect(within(invoiceCard).getByText('مرتبط (تأیید نشده)')).toBeTruthy()
    expect(within(invoiceCard).getByText('تعارض')).toBeTruthy()
    expect(within(invoiceCard).queryByRole('link', { name: /مشاهده/ })).toBeNull()
    expect(document.querySelector(`a[href="/customers/${CUST_OTHER}"], a[href="/sales/cases/${CASE_B}"]`)).toBeNull()
    expect(within(invoiceCard).getAllByText(/پیوند تأیید نشده است/).length).toBe(2)
  })

  it('a CONFLICT case shows its state and reason without a customer link', async () => {
    renderAt(`/sales/cases/${CASE_X}`, routes())
    await screen.findByRole('heading', { level: 1, name: 'پرونده: legacy.lead.v1 #7009' })
    const caseCard = section('پرونده')
    expect(within(caseCard).getByText('تعارض')).toBeTruthy()
    expect(within(caseCard).getByText(/two explicit references disagree/)).toBeTruthy()
    expect(within(caseCard).queryByRole('link', { name: 'مشاهده مشتری' })).toBeNull()
    expect(document.querySelector(`a[href="/customers/${CUST}"]`)).toBeNull()
    // No invoice is RESOLVED to this case: an honest empty state, not an error.
    expect(await within(section(INVOICES_OF_CASE)).findByText('فاکتوری با پیوند تأییدشده به این پرونده ثبت نشده است')).toBeTruthy()
  })

  it('shows every relationship state with a distinct label', async () => {
    const pairs: [string, string][] = [['RESOLVED', 'پیوند تأییدشده'], ['RELATED', 'مرتبط (تأیید نشده)'], ['UNKNOWN', 'نامعلوم'],
      ['CONFLICT', 'تعارض'], ['NOT_RECOVERABLE', 'قابل بازیابی نیست']]
    const labels = pairs.map(([, l]) => l)
    for (const [st, label] of pairs) {
      const v = invoice(INV_A, '9001', { customer: st === 'RESOLVED' ? resolved(CUST) : unlinked(st, CUST), case: unlinked('UNKNOWN') })
      const { router } = renderAt(`/billing/invoices/${INV_A}`, routes({ [`GET /api/v1/invoices/${INV_A}`]: () => json(env({ data: { invoice: v,
        product_ref_raw: null, pay_method_raw: null, payment_plan_raw: null, current_stage_raw: null, updated_at_raw: null, states: [], declared: [] } })) }))
      const card = await waitFor(() => section('فاکتور'))
      expect((await within(card).findAllByText(label)).length).toBeGreaterThan(0)
      expect(within(card).queryAllByRole('link', { name: 'مشاهده مشتری' }).length).toBe(st === 'RESOLVED' ? 1 : 0)
      router.dispose()
      document.body.innerHTML = ''
    }
    expect(new Set(labels).size).toBe(labels.length)
  })

  it('renders UNAUTHORIZED sections as locked with no data in the DOM', async () => {
    const denied = () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'test-corr-1' }, 403)
    renderAt(`/customers/${CUST}`, routes({ 'GET /api/v1/cases': denied, 'GET /api/v1/invoices': denied }))
    const casesCard = await waitFor(() => section(CASES_OF_CUSTOMER))
    expect(await within(casesCard).findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(await within(section(INVOICES_OF_CUSTOMER)).findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(document.body.textContent).not.toMatch(/#7001|#9001/)
    // Denied is never rendered as an empty list.
    expect(within(casesCard).queryByText(/با پیوند تأییدشده/)).toBeNull()
    expect(within(section(INVOICES_OF_CUSTOMER)).queryByText(/با پیوند تأییدشده/)).toBeNull()
    // The customer section itself still renders.
    expect(await screen.findByRole('heading', { level: 1, name: 'مشتری: Sy********1' })).toBeTruthy()
  })

  it('renders BLOCKED (flag off) and server errors honestly, never as empty', async () => {
    const blocked = () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'test-corr-2' })
    const problem = () => json({ type: '/problems/contract-invalid', title: 'Invalid request', status: 400, reason_code: 'CONTRACT_INVALID',
      correlation_id: 'corr-400' }, 400, 'application/problem+json')
    renderAt(`/customers/${CUST}`, routes({ 'GET /api/v1/cases': blocked, 'GET /api/v1/invoices': problem }))
    const casesCard = await waitFor(() => section(CASES_OF_CUSTOMER))
    expect(await within(casesCard).findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    expect(within(casesCard).getByText(/sales\.read_shadow/)).toBeTruthy()
    const invCard = section(INVOICES_OF_CUSTOMER)
    expect(await within(invCard).findByText('Invalid request')).toBeTruthy()
    expect(within(invCard).getByText(/corr-400/)).toBeTruthy()
    expect(document.body.textContent).not.toContain('ثبت نشده است؛')
    expect(within(invCard).queryByText(/فاکتوری با پیوند تأییدشده/)).toBeNull()
  })

  it('shows an honest empty state when no RESOLVED case exists', async () => {
    renderAt(`/customers/${CUST}`, routes({ 'GET /api/v1/cases': () => json(env({ items: [], page: { has_more: false } })) }))
    const casesCard = await waitFor(() => section(CASES_OF_CUSTOMER))
    expect(await within(casesCard).findByText('پرونده‌ای با پیوند تأییدشده به این مشتری ثبت نشده است')).toBeTruthy()
    expect(within(casesCard).getByText(/تلفن و کلید قدیمی هرگز پیوند نمی‌سازند/)).toBeTruthy()
    expect(within(casesCard).queryByRole('navigation')).toBeNull()
  })

  it('does not link a row the server returned without a matching RESOLVED link (defensive)', async () => {
    const odd = caseRow(CASE_X, '7009', { customer_link_state: 'CONFLICT' })
    renderAt(`/customers/${CUST}`, routes({ 'GET /api/v1/cases': () => json(env({ items: [odd], page: { has_more: false } })) }))
    const casesCard = await waitFor(() => section(CASES_OF_CUSTOMER))
    expect(await within(casesCard).findByText('legacy.lead.v1 #7009')).toBeTruthy()
    expect(hrefs(casesCard)).toEqual([])
  })
})
