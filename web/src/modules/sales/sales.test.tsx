import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

// All data is SYNTHETIC. No phone or name exists in the contract.
const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const allowed = Object.fromEntries(['cases.case.read', 'custody.custody.read', 'sales.report.read'].map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-9999', allowed_actions: allowed, ...extra })
const src = (key: string, spec = 'legacy.distribution_item.v1') => ({ source_system: 'wp', source_spec: spec, source_key: key, channel_raw: 'mis_pool', origin_ref_raw: null,
  first_batch_id: ID(900), last_batch_id: ID(900), presence: 'PRESENT', first_seen_at: '2026-10-31T09:00:00Z', last_seen_at: '2026-10-31T09:00:00Z' })

function caseRow(n: number, over: Record<string, unknown> = {}) {
  return { case_id: ID(n), workflow_type: 'DISTRIBUTION_ITEM', state_namespace: 'distribution_item.status', state_raw: 'delivered_to_seller', state_mapping: 'KNOWN',
    customer_id: null, customer_link_state: 'UNKNOWN', customer_link_reason: 'no explicit customer reference in the staged source', presence: 'PRESENT',
    source: src(String(500 + n)), alias_count: 0, history_count: 0, imported_at: '2026-10-31T09:00:00Z', refreshed_at: '2026-10-31T09:00:00Z', ...over }
}

const assignment = (n: number, parity: string, over: Record<string, unknown> = {}) => ({ case_id: ID(n), derivation: 'DERIVED', derived_holder: { legacy_key: '5', user_id: ID(705) },
  derived_position_raw: 'seller', legacy_holder: { legacy_key: '5', user_id: ID(705) }, legacy_position_raw: 'seller', legacy_source_spec: 'legacy.distribution_item.v1',
  parity, event_count: 2, finding_count: 0, presence: 'PRESENT', refreshed_at: '2026-10-31T09:00:00Z', ...over })

const metric = (code: string, label: string, status: string, count: number | null) => ({ code, label, status, count, definition: 'synthetic definition',
  authority: 'GATE-0-PRODUCT-INVARIANTS.md', coverage: 'synthetic coverage', reason: 'synthetic reason' })

function routes(over: Record<string, (req: Request) => Response> = {}, enabled = true) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: false, sales_read: enabled } })),
    'GET /api/v1/cases': (req: Request) => {
      const cursor = new URL(req.url).searchParams.get('cursor')
      return json(env(cursor ? { items: [caseRow(3)], page: { has_more: false } }
        : { items: [caseRow(1), caseRow(2, { state_raw: 'synthetic_unfamiliar_status', state_mapping: 'UNKNOWN' })], page: { has_more: true, next_cursor: 'c-2' } }))
    },
    [`GET /api/v1/cases/${ID(1)}`]: () => json(env({ data: { case: caseRow(1), aliases: [{ alias_id: ID(60), case_id: ID(1), related_case_id: ID(2),
      related_source_spec: 'legacy.lead.v1', related_source_key: '101', relation: 'CONVERSION_LINK', resolution: 'RELATED',
      basis: 'explicit link; continuation vs related-distinct unresolved (OPD-01)', presence: 'PRESENT', recorded_at: '2026-10-31T09:00:00Z' }] } })),
    [`GET /api/v1/cases/${ID(1)}/history`]: () => json(env({ items: [], page: { has_more: false } })),
    [`GET /api/v1/cases/${ID(1)}/attribution`]: () => json(env({ items: [
      { snapshot_id: ID(80), facet: 'ORIGINAL_OWNER', resolution: 'RESOLVED', subject: { legacy_key: '4', user_id: ID(704) }, position_raw: 'supervisor',
        basis: 'first holder-assigning custody log row', source_time_raw: '2025-02-01 08:00:00', evidence_spec: 'legacy.distribution_item_log.v1', evidence_key: '1',
        captured_batch_id: ID(900), captured_at: '2026-10-31T09:00:00Z' },
      { snapshot_id: ID(81), facet: 'CREDIT_OWNER', resolution: 'UNKNOWN', position_raw: null, basis: 'not evaluated', source_time_raw: null, evidence_spec: null,
        evidence_key: null, captured_batch_id: ID(900), captured_at: '2026-10-31T09:00:00Z' },
    ] })),
    [`GET /api/v1/custody/cases/${ID(1)}`]: () => json(env({ data: { assignment: assignment(1, 'CONFLICT', { legacy_holder: { legacy_key: '7', user_id: null } }),
      events: [{ event_id: ID(70), source_spec: 'legacy.distribution_item_log.v1', source_key: '1', legacy_action: 'item_created_from_mis_pool', kind: 'HOLDER_ASSIGNED',
        context: 'MIS', to: { legacy_key: '5', user_id: ID(705) }, from_position_raw: null, to_position_raw: 'seller', occurred_raw: '2025-02-01 08:00:00', recorded_at: '2026-10-31T09:00:00Z' }],
      findings: [{ finding_id: ID(71), kind: 'SOURCE_CURRENT_MISMATCH', detail: 'legacy holder differs', event_id: null, presence: 'PRESENT', recorded_at: '2026-10-31T09:00:00Z' }] } })),
    'GET /api/v1/custody/assignments': () => json(env({ items: [assignment(1, 'MATCH'), assignment(2, 'CONFLICT', { legacy_holder: { legacy_key: '7', user_id: null } }),
      assignment(3, 'NOT_RECOVERABLE', { derivation: 'NONE', derived_holder: undefined, legacy_holder: undefined })], page: { has_more: false } })),
    'GET /api/v1/sales/coverage-report': () => json(env({ data: { cases_total: 6, cases_present: 6, cases_by_source_spec: { 'legacy.lead.v1': 2 },
      cases_by_workflow_type: {}, cases_with_unknown_state_mapping: 2, sourceless_cases: 0, aliases_by_resolution: { RELATED: 1 }, status_history_rows: 2,
      cases_with_status_history: 1, attribution_by_facet_and_resolution: {}, customer_link_states: { UNKNOWN: 6 },
      legacy_leads_only: metric('F03', 'Legacy Leads Only (not Total Cases)', 'EVALUATED', 2),
      total_cases: metric('F03', 'Total Cases (staged sources only; not a proven cross-source total)', 'PARTIAL', 6),
      f02: metric('F02', 'Open Pre-invoices', 'NOT_EVALUATED', null), source_coverage: { legacy_seller_flow_states: 'UNKNOWN (not staged in Slice 6)' } } })),
    'GET /api/v1/sales/custody-parity': () => json(env({ data: { assignments_present: 3, current_holder_parity: { MATCH: 1, CONFLICT: 1, NOT_RECOVERABLE: 1 },
      derivation_states: {}, findings_by_kind: {}, custody_events: 2, custody_events_by_kind: {}, cases_with_custody_events: 2, unknown_legacy_actions: {},
      full_parity: false, notes: ['synthetic note'] } })),
    ...over,
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls }
}

// No assign/transfer/return/release/ready/contact-outcome/invoice control.
const actionWords = /تخصیص|انتقال|برگشت|آزادسازی|آماده|ثبت نتیجه|صدور فاکتور|ویرایش|حذف|assign|transfer|return|release|ready|invoice/i
const noActions = () => expect(screen.queryAllByRole('button').filter(b => actionWords.test(b.textContent ?? ''))).toHaveLength(0)

describe('Sales read (Slice 6)', () => {
  it('shows sales navigation only when the server reports the feature', async () => {
    renderAt('/', routes({}, false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    await waitFor(() => expect(screen.getByText('ورود هنوز فعال نیست')).toBeTruthy())
    expect(screen.queryByRole('link', { name: 'پرونده‌های فروش' })).toBeNull()
  })

  it('lists cases with server filters and cursor pagination, raw unknown states visible, no actions', async () => {
    const { calls } = renderAt('/sales/cases', routes())
    expect(await screen.findByRole('link', { name: 'legacy.distribution_item.v1 #501' })).toBeTruthy()
    expect(screen.getByText('synthetic_unfamiliar_status')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    expect(await screen.findByRole('link', { name: 'legacy.distribution_item.v1 #503' })).toBeTruthy()
    expect(calls.filter(c => c.url === '/api/v1/cases').at(-1)?.search.get('cursor')).toBe('c-2')
    const form = screen.getByRole('form', { name: 'فیلتر پرونده‌ها' })
    fireEvent.change(within(form).getByLabelText('نگاشت وضعیت'), { target: { value: 'UNKNOWN' } })
    fireEvent.submit(form)
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/cases').at(-1)?.search.get('state_mapping')).toBe('UNKNOWN'))
    expect(screen.queryByLabelText(/تلفن/)).toBeNull()
    for (const c of calls) expect(c.method).toBe('GET')
    noActions()
  })

  it('renders blocked, denied and error states honestly', async () => {
    renderAt('/sales/cases', routes({ 'GET /api/v1/cases': () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'c-1' }) }))
    expect(await screen.findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    expect(screen.getByText(/sales\.read_shadow/)).toBeTruthy()
  })

  it('renders UNAUTHORIZED as locked, never as empty', async () => {
    renderAt('/sales/cases', routes({ 'GET /api/v1/cases': () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-1' }, 403) }))
    expect(await screen.findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(screen.queryByText('پرونده‌ای وارد نشده است')).toBeNull()
  })

  it('shows the four owners separately, attribution apart from the current org, and the custody timeline', async () => {
    renderAt(`/sales/cases/${ID(1)}`, routes())
    const owners = await screen.findByRole('region', { name: 'نگهدار فعلی' })
    expect(await within(owners).findByText('تعارض: منبع و رویدادها متفاوت‌اند')).toBeTruthy()
    const original = screen.getByRole('region', { name: 'مالک و فروشنده اولیه' })
    expect(await within(original).findByText(/کاربر قدیمی #4/)).toBeTruthy()
    expect(within(original).getByText(/سرپرست امروز سرپرست دیروز نیست/)).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'اقدام‌کننده بعدی' })).getByText('در این برش مدل نشده است')).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'صاحب اعتبار' })).getByText('نامعلوم')).toBeTruthy()
    expect(await screen.findByText('item_created_from_mis_pool')).toBeTruthy()
    expect(screen.getByText('نگهدار منبع با رویدادها نمی‌خواند')).toBeTruthy()
    expect(screen.getByText(/OPD-01/)).toBeTruthy()
    noActions()
  })

  it('distinguishes proven, conflicting and not-recoverable holders in the custody list', async () => {
    renderAt('/sales/custody', routes())
    expect(await screen.findByText('نگهدار تأییدشده (منبع و رویدادها یکسان)')).toBeTruthy()
    expect(screen.getByText('تعارض: منبع و رویدادها متفاوت‌اند')).toBeTruthy()
    expect(screen.getByText('قابل بازیابی نیست')).toBeTruthy()
    noActions()
  })

  it('shows F02 as not evaluated (null is not zero) and F03 labels exactly', async () => {
    renderAt('/sales/report', routes())
    expect(await screen.findByRole('region', { name: 'Open Pre-invoices' })).toBeTruthy()
    expect(screen.getByText('بدون عدد (صفر نیست)')).toBeTruthy()
    expect(screen.getByRole('region', { name: 'Legacy Leads Only (not Total Cases)' })).toBeTruthy()
    expect(screen.getByRole('region', { name: 'Total Cases (staged sources only; not a proven cross-source total)' })).toBeTruthy()
    expect(await screen.findByText('تطابق کامل ادعا نمی‌شود')).toBeTruthy()
  })
})
