import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import type { EffectiveAccess } from '../../shared/api'
import { fakeApi, healthy, json, session, type Captured } from '../../test/fakeApi'
import { OPERATIONS } from './operations'
import { MIS_NAV_LABEL } from './workspace/SectionNav'

// RM-P05 MIS operational workspace. Synthetic fixtures only; a passing test is
// presentation evidence, never proof that production authorizes these reads
// (the MIS role carries no capability today).

const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const MIS_CAPS = ['cases.case.read', 'custody.custody.read', 'sales.report.read', 'reporting.mis.read']
const allowedActions = (caps: string[]) => Object.fromEntries(caps.map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>, caps = MIS_CAPS) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-9999', allowed_actions: allowedActions(caps), ...extra })
const denied = () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'test-corr-dn01' }, 403)
const blocked = () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'test-corr-bl01' })

type Route = (req: Request) => Response | Promise<Response>

function evaluation(allow: string[]): EffectiveAccess {
  const caps = [...new Set([...MIS_CAPS, 'billing.invoice.read', 'org.person.read'])]
  return {
    authenticated: true, hr_read_enabled: true,
    decisions: caps.map(c => allow.includes(c)
      ? { capability: c as never, allowed: true, outcome: 'SUCCESS' as const, scope: 'ORGANIZATION' as const, matched_roles: ['synthetic_dev_stub'] }
      : { capability: c as never, allowed: false, outcome: 'UNAUTHORIZED' as const, reason_code: 'ACTION_DENIED' as const, scope: 'NONE' as const, matched_roles: [] }),
  }
}

const features = { hr_read: false, customers_identity: false, sales_read: true, billing_read: false, reporting_read: true, finance_shadow_read: false,
  biawin_fulfillment_read: false, contact_outcome_write: false, hr_request_write: false, writeback_read: false, payment_submission: false,
  finance_review_read: false, commission_read: false, fulfillment_work: false }

const sessionOn = (over: Record<string, unknown> = {}) => session({
  authenticated: true, auth_mode: 'session', session_state: 'active', login_enabled: true,
  session: { user_id: ID(201), display_name: 'کاربر ساختگی', csrf_token: 'a'.repeat(43), idle_expires_at: '2099-01-01T00:00:00Z', absolute_expires_at: '2099-01-01T00:00:00Z' },
  features, ...over,
})

const T0 = '2026-11-01T09:00:00Z'
function kase(n: number, over: Record<string, unknown> = {}) {
  return {
    case_id: ID(n), workflow_type: 'DISTRIBUTION_ITEM', state_namespace: 'legacy.dist_item.status', state_raw: 'assigned', state_mapping: 'KNOWN',
    customer_id: null, customer_link_state: 'UNKNOWN', customer_link_reason: 'no explicit reference', presence: 'PRESENT',
    source: { source_system: 'wp', source_spec: 'legacy.dist_item.v1', source_key: String(500 + n), channel_raw: null, origin_ref_raw: null,
      first_batch_id: ID(900), last_batch_id: ID(901), presence: 'PRESENT', first_seen_at: T0, last_seen_at: T0 },
    alias_count: 0, history_count: 0, imported_at: T0, refreshed_at: T0, ...over,
  }
}
function custody(n: number, over: Record<string, unknown> = {}) {
  return {
    case_id: ID(n), derivation: 'DERIVED', derived_holder: { legacy_key: '41', user_id: ID(341) }, derived_position_raw: 'seller',
    legacy_holder: { legacy_key: '41', user_id: ID(341) }, legacy_position_raw: 'seller', legacy_source_spec: 'legacy.dist_item.v1',
    parity: 'MATCH', event_count: 2, finding_count: 0, presence: 'PRESENT', refreshed_at: T0, ...over,
  }
}
const coverage = {
  cases_total: 12, cases_present: 9, cases_by_source_spec: {}, cases_by_workflow_type: {}, cases_with_unknown_state_mapping: 2, sourceless_cases: 0,
  aliases_by_resolution: { RELATED: 1 }, status_history_rows: 4, cases_with_status_history: 2, attribution_by_facet_and_resolution: {},
  customer_link_states: { UNKNOWN: 8, CONFLICT: 1 },
  legacy_leads_only: { code: 'F03', label: 'Legacy Leads Only (not Total Cases)', status: 'EVALUATED', count: 3, definition: 'd', authority: 'a', coverage: 'c', reason: 'r' },
  total_cases: { code: 'F03', label: 'Total Cases', status: 'NOT_EVALUATED', count: null, definition: 'd', authority: 'a', coverage: 'c', reason: 'r' },
  f02: { code: 'F02', label: 'Open Pre-invoices', status: 'NOT_EVALUATED', count: null, definition: 'd', authority: 'a', coverage: 'c', reason: 'r' },
  source_coverage: {},
}
const parity = {
  assignments_present: 9, current_holder_parity: { MATCH: 5, CONFLICT: 2, TARGET_UNKNOWN: 2 }, derivation_states: { DERIVED: 7, UNKNOWN: 2 },
  findings_by_kind: { SOURCE_CURRENT_MISMATCH: 2, MISSING_EVENT_HISTORY: 3 }, custody_events: 17, custody_events_by_kind: {}, cases_with_custody_events: 7,
  unknown_legacy_actions: {}, full_parity: false, notes: [],
}

const CASE_ROWS = () => [kase(1), kase(2, { state_mapping: 'UNKNOWN', state_raw: 'synthetic_unfamiliar', customer_link_state: 'CONFLICT' }), kase(3),
  kase(7, { presence: 'ABSENT_IN_LATEST', source: { ...kase(7).source, presence: 'ABSENT_IN_LATEST', last_batch_id: ID(900) } })]

function casesRoute(req: Request): Response {
  const q = new URL(req.url).searchParams
  // RM-P06 exact case_id batch read: only the named Cases, never a widening.
  const ids = q.getAll('case_id')
  if (ids.length) return json(env({ items: CASE_ROWS().filter(c => ids.includes(c.case_id)), page: { has_more: false } }))
  if (q.get('cursor') === 'c-2') return json(env({ items: [kase(3)], page: { has_more: false } }))
  if (q.get('presence') === 'ABSENT_IN_LATEST') return json(env({ items: [kase(7, { presence: 'ABSENT_IN_LATEST' })], page: { has_more: false } }))
  return json(env({ items: [kase(1), kase(2, { state_mapping: 'UNKNOWN', state_raw: 'synthetic_unfamiliar', customer_link_state: 'CONFLICT' })], page: { has_more: true, next_cursor: 'c-2' } }))
}

function custodyRoute(req: Request): Response {
  const q = new URL(req.url).searchParams
  const ids = q.getAll('case_id')
  const all = [custody(1), custody(2, { parity: 'CONFLICT', legacy_holder: { legacy_key: '77', user_id: null }, finding_count: 1 }),
    custody(7, { presence: 'ABSENT_IN_LATEST' })]
  if (ids.length) return json(env({ items: all.filter(a => ids.includes(a.case_id)), page: { has_more: false } }))
  if (q.get('parity') === 'CONFLICT') return json(env({ items: [all[1]], page: { has_more: false } }))
  return json(env({ items: all, page: { has_more: false } }))
}

function base(over: Record<string, Route> = {}, ev = evaluation(MIS_CAPS), sess = sessionOn()): Record<string, Route> {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(sess),
    'GET /api/v1/authorization/effective-access': () => json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-ea01', data: ev }),
    'GET /api/v1/cases': casesRoute,
    'GET /api/v1/custody/assignments': custodyRoute,
    'GET /api/v1/sales/coverage-report': () => json(env({ data: coverage })),
    'GET /api/v1/sales/custody-parity': () => json(env({ data: parity })),
    'GET /api/v1/mis/overview': () => json(env({ data: { freshness: 'FRESH', coverage: 'AVAILABLE', run: null, metrics: [], deferred: [] } })),
    ...over,
  }
}

function renderAt(path: string, routes: Record<string, Route>) {
  const { client, calls } = fakeApi(routes)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

const misNav = () => screen.getByRole('navigation', { name: MIS_NAV_LABEL })
const navLabels = () => within(misNav()).getAllByRole('link').map(a => a.textContent)
const onlyGets = (calls: Captured[]) => calls.every(c => c.method === 'GET')
const OP_WORDS = /تخصیص|انتقال|برگشت|آزادسازی|بازتخصیص|استفاده مجدد|نهایی|assign|return|release|reuse|recycle/i
const noOperationControls = () => {
  for (const b of screen.queryAllByRole('button')) expect(b.textContent ?? '').not.toMatch(OP_WORDS)
}

describe('RM-P05 MIS workspace navigation', () => {
  it('lists the sections the server allows (through the existing destinations) and marks the current one', async () => {
    renderAt('/mis', base())
    expect(await screen.findByRole('heading', { level: 1, name: 'امروز MIS' })).toBeTruthy()
    await waitFor(() => expect(navLabels()).toEqual(['امروز MIS', 'موجودی پرونده‌ها', 'نگهداری و انتساب', 'استثناها و تطبیق', 'کیفیت منبع', 'تطبیق و پوشش']))
    expect(within(misNav()).getByRole('link', { current: 'page' }).textContent).toBe('امروز MIS')
    expect(screen.getByText('طراحی MIS: آماده بازبینی (منجمد نشده)')).toBeTruthy()
    fireEvent.click(within(misNav()).getByRole('link', { name: 'موجودی پرونده‌ها' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'موجودی پرونده‌ها' })).toBeTruthy()
    expect(within(misNav()).getByRole('link', { current: 'page' }).textContent).toBe('موجودی پرونده‌ها')
  })

  it('fails closed: without ALLOWED decisions only the current section is offered and no row is rendered', async () => {
    renderAt('/mis/custody', base({ 'GET /api/v1/custody/assignments': denied }, evaluation([])))
    expect(await screen.findByRole('heading', { level: 1, name: 'نگهداری و انتساب' })).toBeTruthy()
    await waitFor(() => expect(navLabels()).toEqual(['نگهداری و انتساب']))
    expect(await screen.findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(screen.queryByRole('table')).toBeNull()
    expect(screen.queryByText(/کاربر قدیمی #/)).toBeNull()
  })

  it('sales sections are hidden when sales_read is off, even with the capability', async () => {
    renderAt('/mis', base({}, evaluation(MIS_CAPS), sessionOn({ features: { ...features, sales_read: false } })))
    await waitFor(() => expect(navLabels()).toEqual(['امروز MIS', 'کیفیت منبع', 'تطبیق و پوشش']))
  })

  it('the global navigation offers no MIS link to a principal that only reads cases', async () => {
    renderAt('/', base({}, evaluation(['cases.case.read'])))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    await waitFor(() => expect(screen.getAllByRole('link', { name: 'پرونده‌های فروش' }).length).toBeGreaterThan(0))
    expect(screen.queryByRole('link', { name: 'موجودی پرونده‌ها' })).toBeNull()
    expect(screen.queryByRole('link', { name: 'امروز MIS' })).toBeNull()
  })
})

describe('RM-P05 landing counters', () => {
  it('every counter shows the value of its named aggregate; the absent count is cases_total − cases_present', async () => {
    const { calls } = renderAt('/mis', base())
    const inv = await screen.findByRole('group', { name: 'شمارنده‌های موجودی' })
    await waitFor(() => expect(within(inv).getByText('۱۲')).toBeTruthy())
    expect(within(inv).getByText('۹')).toBeTruthy()
    expect(within(inv).getByText('۳')).toBeTruthy()
    expect(within(inv).getByText('coverage-report · cases_total')).toBeTruthy()
    const cus = screen.getByRole('group', { name: 'شمارنده‌های نگهداری' })
    expect(within(cus).getByText('۱۷')).toBeTruthy()
    expect(screen.getByText('تطابق کامل ادعا نمی‌شود')).toBeTruthy()
    const exc = screen.getByRole('list', { name: 'خلاصه استثناها' })
    const conflict = within(exc).getByRole('link', { name: 'تعارض نگهدار' }).closest('li') as HTMLElement
    expect(within(conflict).getByText('۲')).toBeTruthy()
    expect(within(conflict).getByText('GET /api/v1/sales/custody-parity · current_holder_parity.CONFLICT')).toBeTruthy()
    // No aggregate exists for unresolved holders: unavailable, never zero.
    const unresolved = within(exc).getByRole('link', { name: 'هویت نگهدار حل‌نشده' }).closest('li') as HTMLElement
    expect(within(unresolved).getByText('در دسترس نیست')).toBeTruthy()
    expect(within(unresolved).queryByText('۰')).toBeNull()
    expect(onlyGets(calls)).toBe(true)
  })

  it('a denied or disabled source never becomes zero', async () => {
    renderAt('/mis', base({ 'GET /api/v1/sales/coverage-report': denied, 'GET /api/v1/sales/custody-parity': blocked }))
    const inv = await screen.findByRole('group', { name: 'شمارنده‌های موجودی' })
    await waitFor(() => expect(screen.getByText('دسترسی به این بخش ندارید')).toBeTruthy())
    expect(screen.getAllByText('این بخش هنوز فعال نشده است').length).toBeGreaterThan(0)
    expect(within(inv).queryByText('۰')).toBeNull()
    expect(within(inv).getAllByText('—').length).toBe(4)
    // A denied read makes the shell re-check the session, which re-evaluates navigation access once; the
    // access-gated summary may unmount and remount meanwhile, so assert on the settled render.
    await waitFor(() => {
      const conflict = within(screen.getByRole('list', { name: 'خلاصه استثناها' })).getByRole('link', { name: 'تعارض نگهدار' }).closest('li') as HTMLElement
      expect(within(conflict).queryByText('۰')).toBeNull()
    })
  })

  it('lists every inventory operation as unavailable with its reason and renders no control for any', async () => {
    renderAt('/mis', base())
    const ops = await screen.findByRole('list', { name: 'وضعیت عملیات موجودی' })
    expect(within(ops).getAllByRole('listitem')).toHaveLength(OPERATIONS.length)
    expect(within(ops).getAllByText('در دسترس نیست')).toHaveLength(OPERATIONS.length)
    expect(within(ops).queryAllByRole('button')).toHaveLength(0)
    expect(within(ops).queryAllByRole('link')).toHaveLength(0)
    expect(within(ops).getByText(/ABSENT_IN_LATEST\) هرگز پرونده را قابل استفاده مجدد نمی‌کند/)).toBeTruthy()
    noOperationControls()
  })
})

describe('RM-P05 case inventory', () => {
  it('sends only server filters, keeps keyset paging, and reads custody for exactly the visible page', async () => {
    const { calls } = renderAt('/mis/cases', base())
    expect(await screen.findByRole('link', { name: 'legacy.dist_item.v1 #501' })).toBeTruthy()
    const custodyCall = () => calls.filter(c => c.url === '/api/v1/custody/assignments').at(-1)
    await waitFor(() => expect(custodyCall()?.search.getAll('case_id')).toEqual([ID(1), ID(2)]))
    const rows = screen.getAllByRole('row').slice(1)
    expect(within(rows[0] as HTMLElement).getByText('نگهدار تأییدشده (منبع و رویدادها یکسان)')).toBeTruthy()
    expect(within(rows[1] as HTMLElement).getByText('تعارض: منبع و رویدادها متفاوت‌اند')).toBeTruthy()
    expect(within(rows[1] as HTMLElement).getByText('(نگاشت‌نشده)')).toBeTruthy()
    expect(within(rows[1] as HTMLElement).getByText('تعارض')).toBeTruthy()

    const form = screen.getByRole('form', { name: 'فیلتر موجودی پرونده‌ها' })
    fireEvent.change(within(form).getByLabelText('کلید منبع (دقیق)'), { target: { value: '501' } })
    fireEvent.change(within(form).getByLabelText('حضور در آخرین واردسازی'), { target: { value: 'ABSENT_IN_LATEST' } })
    fireEvent.change(within(form).getByLabelText('پیوند مشتری'), { target: { value: 'CONFLICT' } })
    fireEvent.submit(form)
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/cases').at(-1)?.search.get('presence')).toBe('ABSENT_IN_LATEST'))
    const last = calls.filter(c => c.url === '/api/v1/cases').at(-1)
    expect(last?.search.get('source_key')).toBe('501')
    expect(last?.search.get('customer_link_state')).toBe('CONFLICT')
    expect([...(last?.search.keys() ?? [])].sort()).toEqual(['customer_link_state', 'limit', 'presence', 'source_key'])
    expect(await screen.findByText('غایب در آخرین واردسازی', { selector: '.pill *, .pill' })).toBeTruthy()
    expect(screen.queryByLabelText(/تلفن|phone|نام مشتری/i)).toBeNull()
    expect(onlyGets(calls)).toBe(true)
  })

  it('pages with the server cursor', async () => {
    const { calls } = renderAt('/mis/cases', base())
    await screen.findByRole('link', { name: 'legacy.dist_item.v1 #501' })
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    expect(await screen.findByRole('link', { name: 'legacy.dist_item.v1 #503' })).toBeTruthy()
    expect(calls.filter(c => c.url === '/api/v1/cases').at(-1)?.search.get('cursor')).toBe('c-2')
    fireEvent.click(screen.getByRole('button', { name: 'صفحه قبل' }))
    expect(await screen.findByRole('link', { name: 'legacy.dist_item.v1 #501' })).toBeTruthy()
  })

  it('an invalid source key is refused locally and never sent', async () => {
    const { calls } = renderAt('/mis/cases?source_key=09x', base())
    expect(await screen.findByText('کلید منبع نامعتبر است')).toBeTruthy()
    expect(calls.some(c => c.url === '/api/v1/cases')).toBe(false)
  })

  it('without the custody hint the custody column is not requested and says so', async () => {
    const { calls } = renderAt('/mis/cases', base({ 'GET /api/v1/cases': req => {
      const r = casesRoute(req)
      return r.json().then(b => json({ ...b, allowed_actions: allowedActions(['cases.case.read']) }))
    } }))
    await screen.findByRole('link', { name: 'legacy.dist_item.v1 #501' })
    expect(screen.getAllByText('بدون دسترسی نگهداری').length).toBe(2)
    expect(calls.some(c => c.url === '/api/v1/custody/assignments')).toBe(false)
  })

  it('links every row to Case 360° and its timeline, and opens a case by full id only', async () => {
    const { router } = renderAt('/mis/cases', base())
    const link = await screen.findByRole('link', { name: 'legacy.dist_item.v1 #501' })
    expect(link.getAttribute('href')).toBe(`/sales/cases/${ID(1)}`)
    const row = link.closest('tr') as HTMLElement
    expect(within(row).getByRole('link', { name: 'خط زمانی' }).getAttribute('href')).toBe(`/sales/cases/${ID(1)}#activity`)
    expect(within(row).getByRole('link', { name: 'نگهداری و انتساب' }).getAttribute('href')).toBe(`/mis/custody/${ID(1)}`)
    const jump = screen.getByRole('form', { name: 'رفتن به پرونده' })
    fireEvent.change(within(jump).getByLabelText('شناسه پرونده (UUID)'), { target: { value: '0912' } })
    fireEvent.submit(jump)
    expect(await screen.findByRole('alert')).toBeTruthy()
    expect(router.state.location.pathname).toBe('/mis/cases')
    fireEvent.change(within(jump).getByLabelText('شناسه پرونده (UUID)'), { target: { value: ID(2).toUpperCase() } })
    fireEvent.submit(jump)
    await waitFor(() => expect(router.state.location.pathname).toBe(`/sales/cases/${ID(2)}`))
  })
})

const caseDetail = (n: number, over: Record<string, unknown> = {}) => json(env({ data: { case: kase(n, over), aliases: [] } }))
const events = [
  { event_id: ID(801), source_spec: 'legacy.dist_item_log.v1', source_key: '9001', legacy_action: 'distribute', kind: 'HOLDER_ASSIGNED', context: 'DISTRIBUTION',
    actor: { legacy_key: '9', user_id: ID(309) }, from: { legacy_key: '12', user_id: ID(312) }, to: { legacy_key: '41', user_id: ID(341) },
    from_position_raw: 'manager', to_position_raw: 'seller', occurred_raw: '2025-01-02 10:00:00', recorded_at: T0 },
  { event_id: ID(802), source_spec: 'legacy.dist_item_log.v1', source_key: '9002', legacy_action: 'synthetic_unverified_action', kind: 'UNKNOWN', context: 'UNKNOWN',
    from_position_raw: null, to_position_raw: null, occurred_raw: '2025-01-03 10:00:00', recorded_at: T0 },
]
const custodyDetail = (over: Record<string, unknown> = {}) => json(env({ data: {
  assignment: custody(2, { parity: 'CONFLICT', legacy_holder: { legacy_key: '77', user_id: null }, finding_count: 1, ...over }), events,
  findings: [{ finding_id: ID(851), kind: 'SOURCE_CURRENT_MISMATCH', detail: 'legacy holder 77 differs from derived 41', event_id: null, presence: 'PRESENT', recorded_at: T0 }],
} }))
const attribution = () => json(env({ items: [
  { snapshot_id: ID(861), facet: 'ORIGINAL_OWNER', resolution: 'RESOLVED', subject: { legacy_key: '12', user_id: ID(312) }, position_raw: 'manager', basis: 'first distribution', source_time_raw: '2025-01-02', evidence_spec: 'legacy.dist_item_log.v1', evidence_key: '9001', captured_batch_id: ID(900), captured_at: T0 },
  { snapshot_id: ID(862), facet: 'CREDIT_OWNER', resolution: 'UNKNOWN', position_raw: null, basis: 'not evaluated', source_time_raw: null, evidence_spec: null, evidence_key: null, captured_batch_id: ID(900), captured_at: T0 },
] }))

describe('RM-P05 case custody evidence', () => {
  const routes = (over: Record<string, Route> = {}) => base({
    [`GET /api/v1/cases/${ID(2)}`]: () => caseDetail(2),
    [`GET /api/v1/custody/cases/${ID(2)}`]: () => custodyDetail(),
    [`GET /api/v1/cases/${ID(2)}/attribution`]: attribution,
    ...over,
  })

  it('keeps current custody, transfer history, findings and historical attribution apart', async () => {
    const { calls } = renderAt(`/mis/custody/${ID(2)}`, routes())
    expect(await screen.findByRole('heading', { level: 1, name: 'شواهد نگهداری: legacy.dist_item.v1 #502' })).toBeTruthy()
    const derived = await screen.findByRole('region', { name: 'استنتاج از رویدادها' })
    expect(within(derived).getByText('کاربر قدیمی #41')).toBeTruthy()
    const legacy = screen.getByRole('region', { name: 'ثبت‌شده در منبع' })
    expect(within(legacy).getByText('کاربر قدیمی #77')).toBeTruthy()
    expect(within(legacy).getByText('هویت حل‌نشده')).toBeTruthy()
    const history = screen.getByRole('table', { name: 'سابقه انتقال' })
    const rows = within(history).getAllByRole('row').slice(1)
    expect(rows).toHaveLength(2)
    expect(within(rows[0] as HTMLElement).getByText('تغییر نگهدار')).toBeTruthy()
    expect(within(rows[1] as HTMLElement).getByText('اقدام ناشناخته (مقدار خام حفظ شد)')).toBeTruthy()
    expect(screen.getByText('نگهدار منبع با رویدادها نمی‌خواند')).toBeTruthy()
    const attr = screen.getByRole('table', { name: 'انتساب تاریخی' })
    expect(within(attr).getByText('مالک اولیه')).toBeTruthy()
    expect(within(attr).getByText('کاربر قدیمی #12')).toBeTruthy()
    // The credit owner is not derived from the current holder.
    expect(within(attr).queryByText('کاربر قدیمی #41')).toBeNull()
    expect(screen.getByRole('link', { name: 'نمای ۳۶۰ پرونده' }).getAttribute('href')).toBe(`/sales/cases/${ID(2)}`)
    expect(screen.getByRole('link', { name: 'خط زمانی فعالیت پرونده' }).getAttribute('href')).toBe(`/sales/cases/${ID(2)}#activity`)
    noOperationControls()
    expect(onlyGets(calls)).toBe(true)
  })

  it('custody denied: no holder, event or finding reaches the DOM; the case read still renders', async () => {
    renderAt(`/mis/custody/${ID(2)}`, routes({ [`GET /api/v1/custody/cases/${ID(2)}`]: denied }))
    expect(await screen.findByRole('heading', { level: 1, name: 'شواهد نگهداری: legacy.dist_item.v1 #502' })).toBeTruthy()
    await waitFor(() => expect(screen.getAllByText('دسترسی به این بخش ندارید').length).toBe(3))
    expect(screen.queryByText('کاربر قدیمی #41')).toBeNull()
    expect(screen.queryByText('کاربر قدیمی #77')).toBeNull()
    expect(screen.queryByRole('table', { name: 'سابقه انتقال' })).toBeNull()
    expect(screen.queryByText(/synthetic_unverified_action/)).toBeNull()
  })

  it('an absent case keeps its holder and history and is never offered for reuse', async () => {
    renderAt(`/mis/custody/${ID(2)}`, routes({
      [`GET /api/v1/cases/${ID(2)}`]: () => caseDetail(2, { presence: 'ABSENT_IN_LATEST' }),
      [`GET /api/v1/custody/cases/${ID(2)}`]: () => custodyDetail({ presence: 'ABSENT_IN_LATEST' }),
    }))
    expect(await screen.findByText(/غیبت هرگز پرونده را قابل استفاده مجدد یا آزاد نمی‌کند/)).toBeTruthy()
    expect(screen.getByText(/نگهدار پاک یا آزاد نشده است/)).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'استنتاج از رویدادها' })).getByText('کاربر قدیمی #41')).toBeTruthy()
    expect(screen.queryByText(/قابل استفاده مجدد است|eligible/i)).toBeNull()
    noOperationControls()
  })

  it('an invalid id is refused without a request', async () => {
    const { calls } = renderAt('/mis/custody/0912', routes())
    expect(await screen.findByText('شناسه پرونده نامعتبر است')).toBeTruthy()
    expect(calls.some(c => c.url.startsWith('/api/v1/cases/') || c.url.startsWith('/api/v1/custody/cases/'))).toBe(false)
  })
})

describe('RM-P05 exceptions queue', () => {
  it('opens a queue with its existing server filter and shows its count source', async () => {
    const { calls } = renderAt('/mis/exceptions', base())
    const queues = await screen.findByRole('navigation', { name: 'صف‌های استثنا' })
    fireEvent.click(within(queues).getByRole('link', { name: /تعارض نگهدار/ }))
    expect(await screen.findByText('نگهدار ثبت‌شده در منبع با نگهدار استنتاجی از رویدادها متفاوت است.')).toBeTruthy()
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/custody/assignments').at(-1)?.search.get('parity')).toBe('CONFLICT'))
    expect(calls.filter(c => c.url === '/api/v1/custody/assignments').at(-1)?.search.get('presence')).toBe('PRESENT')
    expect(await screen.findByRole('table', { name: 'پروجکشن‌های نگهداری' })).toBeTruthy()
    expect(screen.getAllByText('GET /api/v1/sales/custody-parity · current_holder_parity.CONFLICT').length).toBeGreaterThan(0)
    expect(within(queues).getByRole('link', { current: true }).textContent).toContain('تعارض نگهدار')
    expect(onlyGets(calls)).toBe(true)
  })

  it('the unresolved-identity queue uses holder_link and has no fabricated count', async () => {
    const { calls } = renderAt('/mis/exceptions?type=unresolved-holder', base())
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/custody/assignments').at(-1)?.search.get('holder_link')).toBe('UNRESOLVED'))
    expect((await screen.findAllByText(/هیچ قرارداد تجمیعی برای این حالت وجود ندارد/)).length).toBeGreaterThan(0)
  })

  it('cases missing from the latest import are listed by presence, with their custody evidence intact', async () => {
    const { calls } = renderAt('/mis/exceptions?type=absent-latest', base())
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/cases').at(-1)?.search.get('presence')).toBe('ABSENT_IN_LATEST'))
    const link = await screen.findByRole('link', { name: 'legacy.dist_item.v1 #507' })
    const row = link.closest('tr') as HTMLElement
    expect(within(row).getByText('نگهدار تأییدشده (منبع و رویدادها یکسان)')).toBeTruthy()
    expect(within(row).getByText('غایب در آخرین واردسازی')).toBeTruthy()
    noOperationControls()
  })

  it('a queue without a list contract says so and sends no list request', async () => {
    const { calls } = renderAt('/mis/exceptions?type=sourceless', base())
    expect(await screen.findByText('فهرست این صف در قرارداد موجود نیست')).toBeTruthy()
    expect(calls.some(c => c.url === '/api/v1/cases' || c.url === '/api/v1/custody/assignments')).toBe(false)
  })
})

describe('RM-P05 Case 360° timeline anchor', () => {
  it('the case page exposes the activity timeline section the MIS links target', async () => {
    renderAt(`/sales/cases/${ID(2)}#activity`, base({
      [`GET /api/v1/cases/${ID(2)}`]: () => caseDetail(2),
      [`GET /api/v1/cases/${ID(2)}/history`]: () => json(env({ items: [], page: { has_more: false } })),
      [`GET /api/v1/cases/${ID(2)}/attribution`]: attribution,
      [`GET /api/v1/custody/cases/${ID(2)}`]: () => custodyDetail(),
    }))
    const section = await screen.findByRole('region', { name: 'خط زمانی فعالیت پرونده' })
    expect(section.closest('#activity')).toBeTruthy()
  })
})

describe('RM-P06 custody-to-case evidence', () => {
  const custodyPage = () => calls0.filter(c => c.url === '/api/v1/cases')
  let calls0: Captured[] = []

  it('each custody row shows its case source from ONE batched cases read, with native id and three links', async () => {
    const { calls } = renderAt('/mis/custody', base())
    calls0 = calls
    const table = await screen.findByRole('table', { name: 'پروجکشن‌های نگهداری' })
    await waitFor(() => expect(within(table).getAllByTestId('case-source')).toHaveLength(3))
    const caseCalls = custodyPage()
    // One request for the whole page (no N+1), exact ids, nothing else.
    expect(caseCalls).toHaveLength(1)
    expect(caseCalls[0]?.search.getAll('case_id')).toEqual([ID(1), ID(2), ID(7)])
    expect([...(caseCalls[0]?.search.keys() ?? [])].filter(k => k !== 'case_id').sort()).toEqual(['limit'])
    const rows = within(table).getAllByRole('row').slice(1)
    const r1 = rows[0] as HTMLElement
    expect(within(r1).getByText('wp · legacy.dist_item.v1 · #501')).toBeTruthy()
    expect(within(r1).getByText('منبع: حاضر')).toBeTruthy()
    expect(within(r1).getByText(ID(1))).toBeTruthy()
    expect(within(r1).getByRole('link', { name: 'نمای ۳۶۰' }).getAttribute('href')).toBe(`/sales/cases/${ID(1)}`)
    expect(within(r1).getByRole('link', { name: 'شواهد نگهداری' }).getAttribute('href')).toBe(`/mis/custody/${ID(1)}`)
    expect(within(r1).getByRole('link', { name: 'خط زمانی' }).getAttribute('href')).toBe(`/sales/cases/${ID(1)}#activity`)
    // ABSENT_IN_LATEST stays absent, never "available".
    const r3 = rows[2] as HTMLElement
    expect(within(r3).getByText('منبع: غایب در آخرین واردسازی')).toBeTruthy()
    expect(within(r3).queryByText(/حاضر|قابل استفاده|eligible/i)).toBeNull()
    noOperationControls()
    expect(onlyGets(calls)).toBe(true)
  })

  it('without the cases hint the case source is not requested; custody evidence still renders', async () => {
    const { calls } = renderAt('/mis/custody', base({ 'GET /api/v1/custody/assignments': req => {
      const r = custodyRoute(req)
      return r.json().then(b => json({ ...b, allowed_actions: allowedActions(['custody.custody.read']) }))
    } }))
    const table = await screen.findByRole('table', { name: 'پروجکشن‌های نگهداری' })
    expect(within(table).getAllByText('منبع پرونده: بدون دسترسی خواندن پرونده')).toHaveLength(3)
    expect(within(table).getAllByText('کاربر قدیمی #41').length).toBeGreaterThan(0)
    expect(calls.some(c => c.url === '/api/v1/cases')).toBe(false)
    expect(document.body.textContent).not.toMatch(/legacy\.dist_item\.v1 · #|#501/)
  })

  it('cases denied by the server: no source system, spec, key or batch reaches the DOM; custody stays', async () => {
    renderAt('/mis/custody', base({ 'GET /api/v1/cases': denied }))
    const table = await screen.findByRole('table', { name: 'پروجکشن‌های نگهداری' })
    await waitFor(() => expect(within(table).getAllByText('منبع پرونده: بدون دسترسی خواندن پرونده')).toHaveLength(3))
    expect(within(table).getAllByText('کاربر قدیمی #41').length).toBeGreaterThan(0)
    const text = document.body.textContent ?? ''
    expect(text).not.toMatch(/#50[127]|legacy\.dist_item\.v1 · |wp · /)
    expect(text).not.toContain(ID(900))
    expect(screen.queryAllByTestId('case-source')).toHaveLength(0)
  })

  it('loading, disabled, failed and missing case sources are distinct and never shown as confirmed', async () => {
    let release: (() => void) | undefined
    const { calls } = renderAt('/mis/custody', base({ 'GET /api/v1/cases': () => new Promise<Response>(res => { release = () => res(blocked()) }) }))
    const table = await screen.findByRole('table', { name: 'پروجکشن‌های نگهداری' })
    expect(within(table).getAllByText('منبع پرونده: در حال خواندن…')).toHaveLength(3)
    await waitFor(() => expect(release).toBeDefined())
    release?.()
    await waitFor(() => expect(within(table).getAllByText('منبع پرونده: غیرفعال')).toHaveLength(3))
    expect(calls.filter(c => c.url === '/api/v1/cases')).toHaveLength(1)
  })

  it('a read failure is UNKNOWN, and a custody row whose case is not returned is UNKNOWN (no guessed relationship)', async () => {
    renderAt('/mis/custody', base({ 'GET /api/v1/cases': () => json({ type: 'about:blank', title: 'x', status: 500 }, 500) }))
    expect((await screen.findAllByText('منبع پرونده نامعلوم (خطای خواندن)')).length).toBe(3)
    cleanup()
    renderAt('/mis/custody', base({ 'GET /api/v1/cases': () => json(env({ items: [kase(1)], page: { has_more: false } })) }))
    expect((await screen.findAllByText('منبع پرونده نامعلوم (پرونده در ماژول پرونده یافت نشد)')).length).toBe(2)
    expect(screen.getAllByTestId('case-source')).toHaveLength(1)
  })

  it('exception queues backed by custody are enriched with the same single batched read', async () => {
    const { calls } = renderAt('/mis/exceptions?type=holder-conflict', base())
    const table = await screen.findByRole('table', { name: 'پروجکشن‌های نگهداری' })
    expect(await within(table).findByText('wp · legacy.dist_item.v1 · #502')).toBeTruthy()
    const caseCalls = calls.filter(c => c.url === '/api/v1/cases')
    expect(caseCalls).toHaveLength(1)
    expect(caseCalls[0]?.search.getAll('case_id')).toEqual([ID(2)])
    noOperationControls()
  })
})

describe('RM-P06 recorded provenance on the case custody evidence page', () => {
  const routes = (over: Record<string, Route> = {}) => base({
    [`GET /api/v1/cases/${ID(2)}`]: () => caseDetail(2),
    [`GET /api/v1/custody/cases/${ID(2)}`]: () => custodyDetail(),
    [`GET /api/v1/cases/${ID(2)}/attribution`]: attribution,
    ...over,
  })

  it('shows first and latest observed batch, the native id, and import-run status as unavailable', async () => {
    renderAt(`/mis/custody/${ID(2)}`, routes())
    const list = await screen.findByLabelText('خاستگاه ثبت‌شده')
    expect(within(list).getByText(ID(900))).toBeTruthy()
    expect(within(list).getByText(ID(901))).toBeTruthy()
    expect(within(list).getByText('در دسترس نیست')).toBeTruthy()
    expect(within(list).getByText(/قرارداد خواندن اجرای واردسازی، فایل و دسته در API وجود ندارد/)).toBeTruthy()
    expect(within(list).queryByText(/موفق$|کامل است|SUCCESS/)).toBeNull()
    expect(screen.getByText(ID(2))).toBeTruthy()
  })

  it('cases denied: no provenance reaches the DOM; custody evidence still renders', async () => {
    renderAt(`/mis/custody/${ID(2)}`, routes({ [`GET /api/v1/cases/${ID(2)}`]: denied }))
    expect(await screen.findByRole('region', { name: 'استنتاج از رویدادها' })).toBeTruthy()
    expect(screen.queryByLabelText('خاستگاه ثبت‌شده')).toBeNull()
    expect(document.body.textContent).not.toContain(ID(900))
  })
})
