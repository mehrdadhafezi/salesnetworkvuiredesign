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
const RUN = ID(990)
const FRUN = ID(991)
const NOTICE = 'SHADOW: imported legacy subscription and fulfillment evidence for architecture validation and parity only. Not live provider state; no activation, provider, shipping or workflow authority.'
const allowed = Object.fromEntries(['biawin.subscription.read', 'fulfillment.case.read', 'biawin.parity.read'].map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-1010', notice: NOTICE, allowed_actions: allowed, ...extra })
const run = (id: string, extra: Record<string, unknown> = {}) => ({
  record_kind: 'SHADOW', run_id: id, run_seq: 1, normalization_version: 1, snapshot_id: 'syn-bw-1', source_cutoff: '2026-11-01T09:00:00Z',
  started_at: '2026-11-01T09:05:00Z', completed_at: '2026-11-01T09:05:00Z', outcome: 'SUCCESS', reasons: [], counts: { subscription: 3 }, freshness: 'FRESH',
  sources: [{ source: 'project_membership', status: 'AVAILABLE', rows_read: 3 }], ...extra,
})
const meta = (id = RUN, filters: Record<string, string> = {}, r = run(id)) => ({ record_kind: 'SHADOW', notice: NOTICE, query: { run_id: id, filters }, run: r })
const link = (state: string, extra: Record<string, unknown> = {}) => ({ state, ...extra })
const sub = (n: number, state: string, raw: string | null, extra: Record<string, unknown> = {}) => ({
  subscription_id: ID(n), source_spec: 'legacy.project_membership.v1', source_key: String(n), provider: 'biawin', product_version_state: 'UNKNOWN',
  product_title_snapshot: `Synthetic Bundle ${n}`, product_key: '77', customer: link('RESOLVED', { id: ID(1) }), invoice: link('RESOLVED', { id: ID(2), legacy_key: '2001' }),
  first_payment_invoice: link('UNKNOWN'), case: link('UNKNOWN'), state_namespace: 'legacy.project_membership.status', state_raw: raw, state_mapped: state,
  activation_evidence: 'NOT_EVIDENCED', item_count: 2, ...extra,
})
const item = (n: number, title: string) => ({ item_id: ID(n), subscription_id: ID(501), source_key: String(n), state: 'NOT_EVIDENCED', service_title_snapshot: title, invoice_links: [] })
const fcase = (n: number, kind: string, status: string, raw: string | null, extra: Record<string, unknown> = {}) => ({
  case_id: ID(n), kind, source_spec: 'legacy.operations_execution_case.v1', source_key: String(n), invoice_link_state: 'UNKNOWN',
  status_namespace: 'legacy.execution.status', status_raw: raw, status_mapped: status, assignments: [], ...extra,
})

function routes(over: Record<string, (req: Request) => Response> = {}, enabled = true) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: false, sales_read: false, billing_read: false, reporting_read: false,
      finance_shadow_read: false, biawin_fulfillment_read: enabled } })),
    'GET /api/v1/biawin/subscriptions': (req: Request) => {
      const p = new URL(req.url).searchParams
      const page = p.get('cursor')
        ? { items: [sub(505, 'ACTIVE', 'active')], page: { has_more: false } }
        : { items: [sub(501, 'ACTIVE', 'active'), sub(502, 'SUSPENDED', 'suspended'), sub(503, 'UNKNOWN', 'synthetic_paused', { customer: link('CONFLICT'), case: link('UNKNOWN', { legacy_key: '77' }) })],
          page: { has_more: true, next_cursor: 's-2' } }
      return json(env({ meta: meta(), summary: [{ key: 'ACTIVE', count: 2 }, { key: 'SUSPENDED', count: 1 }, { key: 'UNKNOWN', count: 1 }], ...page }))
    },
    [`GET /api/v1/biawin/subscriptions/${ID(501)}`]: () => json(env({ meta: meta(), subscription: sub(501, 'ACTIVE', 'active'),
      history: [{ source_key: '901', event_class: 'PROVIDER_EVENT_EVIDENCE', event_key_raw: 'project_remote_failed', occurred_at_raw: '2025-01-02 10:00:00' }] })),
    [`GET /api/v1/biawin/subscriptions/${ID(501)}/items`]: () => json(env({ meta: meta(), item_count: 2,
      items: [item(601, 'Synthetic Service A'), item(602, 'Synthetic Service A')], page: { has_more: false } })),
    'GET /api/v1/biawin/parity': () => json(env({ meta: meta(), check: { check_id: ID(77), run_id: RUN, evidence_snapshot_id: 'syn-bw-2', evidence_cutoff: '2026-11-02T09:00:00Z',
      evidence_status: 'AVAILABLE', checked_at: '2026-11-02T09:10:00Z', outcome: 'MISMATCH', reasons: ['LINKAGE_CONFLICT'], legacy_count: 3, target_count: 2, matched: 1,
      missing_target: 1, extra_target: 0, unknown: 1, conflict: 1, not_recoverable: 0, cutoff_mismatch: true,
      rows: [{ source_key: '501', result: 'CONFLICT', legacy_class: 'KNOWN_NOT_ACTIVE', target_state: 'ACTIVE' }, { source_key: '506', result: 'MISSING_TARGET', legacy_class: 'ACTIVE' }] },
    page: { has_more: false } })),
    'GET /api/v1/fulfillment/cases': () => json(env({ meta: meta(FRUN), summary: [{ kind: 'EXECUTION', status: 'COMPLETED', count: 1 }, { kind: 'SHIPPING', status: 'UNKNOWN', count: 1 }],
      items: [fcase(1004, 'EXECUTION', 'COMPLETED', 'completed', { subscription_id: ID(502), assignments: [{ role: 'EXPERT', user_key: '7' }] }),
        fcase(1302, 'SHIPPING', 'UNKNOWN', 'synthetic_lost', { tracking_ref_sha256: 'ab'.repeat(32) })], page: { has_more: false } })),
    [`GET /api/v1/fulfillment/cases/${ID(1004)}`]: () => json(env({ meta: meta(FRUN), case: fcase(1004, 'EXECUTION', 'COMPLETED', 'completed', { subscription_id: ID(502) }),
      evidence: [{ source_key: '1201', outcome_mapped: 'EXECUTED', outcome_raw: 'executed' }] })),
    [`GET /api/v1/fulfillment/cases/${ID(1004)}/history`]: () => json(env({ meta: meta(FRUN), items: [{ source_spec: 'legacy.operations_execution_event.v1', source_key: '1101',
      event_key_raw: 'status_changed', old_status_raw: 'assigned', new_status_raw: 'completed' }], page: { has_more: false } })),
    ...over,
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

// No control may imply activation, a provider call or a workflow action.
const mutationWords = /فعال کن|فعال‌سازی کن|تعلیق کن|خاتمه|سفارش|شارژ|ارسال کن|تخصیص|تکمیل کن|بارگذاری|ارتقا بده|activate|suspend|terminate|order|charge|ship|assign|complete|upload|upgrade|portal|edit|delete|export|csv/i
const noMutation = () => {
  expect(screen.queryAllByRole('button').filter(b => mutationWords.test(b.textContent ?? ''))).toHaveLength(0)
  expect(screen.queryAllByRole('link').filter(a => /\/(biawin|fulfillment)\/.*(activate|suspend|terminate|order|charge|ship|assign|complete|upload|upgrade)/i
    .test(a.getAttribute('href') ?? ''))).toHaveLength(0)
  expect(screen.queryAllByRole('textbox')).toHaveLength(0)
}

describe('Biawin / fulfillment READ shadow (Slice 10)', () => {
  it('shows navigation only when the server reports the feature', async () => {
    renderAt('/', routes({}, false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    await waitFor(() => expect(screen.getByText('ورود هنوز فعال نیست')).toBeTruthy())
    expect(screen.queryByRole('link', { name: 'اشتراک‌های بیاوین' })).toBeNull()
  })

  it('subscriptions: SHADOW banner, raw states kept, UNKNOWN and CONFLICT explicit, counts independent of the page, paging pins the run', async () => {
    const { calls } = renderAt('/biawin/subscriptions', routes())
    expect(await screen.findByText('SHADOW · سایه')).toBeTruthy()
    const table = await screen.findByRole('table', { name: 'اشتراک‌های سایه' })
    expect(within(table).getByText('raw: synthetic_paused')).toBeTruthy()
    expect(within(table).getByText('نامعلوم (مقدار خام حفظ شد)')).toBeTruthy()
    expect(within(table).getByText('تعارض')).toBeTruthy()
    expect(within(table).getAllByText('نسخه محصول: نامعلوم').length).toBe(3)
    expect(screen.getByTestId('sub-count-ACTIVE').textContent).toBe('2')
    // A real zero is a zero (EMPTY state absent), distinct from unknown.
    expect(screen.getByTestId('sub-count-EMPTY').textContent).toBe('0')
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/biawin/subscriptions').at(-1)?.search.get('cursor')).toBe('s-2'))
    expect(calls.filter(c => c.url === '/api/v1/biawin/subscriptions').at(-1)?.search.get('run_id')).toBe(RUN)
    expect(screen.getByTestId('sub-count-ACTIVE').textContent).toBe('2')
    noMutation()
    for (const c of calls) expect(c.method).toBe('GET')
  })

  it('subscription detail: items from the same run, identical titles never merged, provider event is evidence only', async () => {
    const { calls } = renderAt(`/biawin/subscriptions/${ID(501)}?run_id=${RUN}`, routes())
    const table = await screen.findByRole('table', { name: 'آیتم‌های اشتراک' })
    expect(within(table).getAllByText('Synthetic Service A').length).toBe(2)
    expect(screen.getByText('شاهد رویداد تأمین‌کننده (فراخوانی نیست)', { exact: false })).toBeTruthy()
    expect(screen.getByText('زمان فعال‌سازی ثبت نشده (از زمان ورود استنتاج نمی‌شود)', { exact: false })).toBeTruthy()
    expect(calls.find(c => c.url === `/api/v1/biawin/subscriptions/${ID(501)}/items`)?.search.get('run_id')).toBe(RUN)
    noMutation()
  })

  it('subscription detail without a pinned run pins the run the detail answered for its items', async () => {
    const { calls } = renderAt(`/biawin/subscriptions/${ID(501)}`, routes())
    expect(await screen.findByRole('table', { name: 'آیتم‌های اشتراک' })).toBeTruthy()
    expect(calls.find(c => c.url === `/api/v1/biawin/subscriptions/${ID(501)}`)?.search.get('run_id')).toBeNull()
    expect(calls.find(c => c.url === `/api/v1/biawin/subscriptions/${ID(501)}/items`)?.search.get('run_id')).toBe(RUN)
  })

  it('parity: counts, cutoff mismatch, conflict and missing target are distinct; never a rate', async () => {
    renderAt('/biawin/parity', routes())
    expect(await screen.findByText('مغایر')).toBeTruthy()
    expect(screen.getByTestId('par-missing').textContent).toBe('1')
    expect(screen.getByTestId('par-extra').textContent).toBe('0')
    expect(screen.getByTestId('par-conflict').textContent).toBe('1')
    expect(screen.getByText('برش شواهد با برش اجرای هدف یکی نیست')).toBeTruthy()
    const table = screen.getByRole('table', { name: 'سطرهای تطبیق' })
    expect(within(table).getByText('در هدف نیست')).toBeTruthy()
    expect(screen.queryByText(/%|نرخ تطبیق|امتیاز/)).toBeNull()
    noMutation()
  })

  it('parity: a run without a recorded check is UNKNOWN, never MATCH', async () => {
    renderAt('/biawin/parity', routes({ 'GET /api/v1/biawin/parity': () => json(env({ outcome: 'UNKNOWN', reasons: ['OUTCOME_UNKNOWN'], meta: meta() })) }))
    expect(await screen.findByText('برای این اجرا تطبیقی ثبت نشده است')).toBeTruthy()
    expect(screen.queryByText('منطبق')).toBeNull()
  })

  it('cases: kinds explicit; COMPLETED qualified; hashed tracking only; no assignment or shipping control', async () => {
    renderAt('/fulfillment/cases', routes())
    const table = await screen.findByRole('table', { name: 'پرونده‌های اجرای خدمت' })
    expect(within(table).getByText('تکمیل‌شده (فقط همین پرونده)')).toBeTruthy()
    expect(within(table).getByText('raw: synthetic_lost')).toBeTruthy()
    expect(screen.getByTestId('case-count-EXECUTION-COMPLETED').textContent).toBe('1')
    expect(screen.queryByText(/SYN-TRK/)).toBeNull()
    noMutation()
  })

  it('case detail: completion implies nothing about invoice, subscription or finance; history from the same run', async () => {
    const { calls } = renderAt(`/fulfillment/cases/${ID(1004)}?run_id=${FRUN}`, routes())
    expect(await screen.findByText(/فقط همین پرونده تکمیل شده است/)).toBeTruthy()
    expect(await screen.findByRole('list', { name: 'تاریخچه پرونده' })).toBeTruthy()
    expect(calls.find(c => c.url === `/api/v1/fulfillment/cases/${ID(1004)}/history`)?.search.get('run_id')).toBe(FRUN)
    noMutation()
  })

  it('a filter change re-evaluates at the latest run; only allowlisted filters, never a phone', async () => {
    const { calls } = renderAt(`/biawin/subscriptions?run_id=${RUN}`, routes())
    await screen.findByRole('table', { name: 'اشتراک‌های سایه' })
    const form = screen.getByRole('form', { name: 'فیلتر اشتراک‌ها' })
    fireEvent.change(within(form).getByLabelText('state'), { target: { value: 'UNKNOWN' } })
    fireEvent.submit(form)
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/biawin/subscriptions').at(-1)?.search.get('state')).toBe('UNKNOWN'))
    const last = calls.filter(c => c.url === '/api/v1/biawin/subscriptions').at(-1)
    expect(last?.search.get('run_id')).toBeNull()
    for (const k of last?.search.keys() ?? []) {
      expect(['state', 'customer_link_state', 'invoice_link_state', 'activation_evidence', 'limit', 'cursor', 'run_id']).toContain(k)
      expect(k).not.toMatch(/phone|mobile/i)
    }
  })

  it('no import run yet is UNKNOWN on every list page — never zero, never a transport failure', async () => {
    const unknown = () => json({ outcome: 'UNKNOWN', reasons: ['SOURCE_UNAVAILABLE'], correlation_id: 'c-1', notice: NOTICE })
    for (const [path, url] of [['/biawin/subscriptions', 'GET /api/v1/biawin/subscriptions'], ['/fulfillment/cases', 'GET /api/v1/fulfillment/cases'],
      ['/biawin/parity', 'GET /api/v1/biawin/parity']] as const) {
      document.body.innerHTML = ''
      renderAt(path, routes({ [url]: unknown }))
      expect(await screen.findByText('هنوز هیچ اجرای ورودی ثبت نشده است')).toBeTruthy()
      expect(screen.queryByText('ارتباط با سرویس برقرار نشد')).toBeNull()
      expect(screen.queryByRole('table')).toBeNull()
    }
  })

  it('a stale or partial run is labelled, not hidden', async () => {
    renderAt('/biawin/subscriptions', routes({ 'GET /api/v1/biawin/subscriptions': () => json(env({ outcome: 'PARTIAL', reasons: ['STALE_STATE'],
      meta: meta(RUN, {}, run(RUN, { freshness: 'STALE', outcome: 'PARTIAL', sources: [{ source: 'shipping_order', status: 'MISSING', rows_read: 0 }] })),
      summary: [], items: [], page: { has_more: false } })) }))
    expect(await screen.findByText('کهنه (قدیمی‌تر از ۲۴ ساعت)')).toBeTruthy()
    expect(screen.getByText('ورود فنی ناقص')).toBeTruthy()
    expect(screen.getByText('shipping_order: منبع موجود نیست')).toBeTruthy()
    expect(screen.getByText('هیچ اشتراکی با این فیلتر در این اجرا نیست')).toBeTruthy()
  })

  it('blocked, unauthorized and not-found are distinct states', async () => {
    renderAt('/biawin/subscriptions', routes({ 'GET /api/v1/biawin/subscriptions': () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'c-1', notice: NOTICE }) }))
    expect(await screen.findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    expect(screen.getByText(/biawin_fulfillment\.read_shadow/)).toBeTruthy()
    document.body.innerHTML = ''
    renderAt('/fulfillment/cases', routes({ 'GET /api/v1/fulfillment/cases': () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-1' }, 403) }))
    expect(await screen.findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    document.body.innerHTML = ''
    renderAt(`/biawin/subscriptions/${ID(999)}`, routes({ [`GET /api/v1/biawin/subscriptions/${ID(999)}`]: () => json({ type: '/problems/not-found', title: 'یافت نشد',
      status: 404, reason_code: 'NOT_FOUND', correlation_id: 'c-404' }, 404, 'application/problem+json') }))
    expect(await screen.findByText('یافت نشد')).toBeTruthy()
  })
})
