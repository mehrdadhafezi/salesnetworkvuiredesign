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
const CASE = ID(1)
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-1111', ...extra })
const blocked = [{ action: 'return_case', blocked_by: ['ADR-01', 'F01', 'OPD-03'] }, { action: 'issue_invoice', blocked_by: ['ADR-01'] },
  { action: 'record_duplicate_outcome', blocked_by: ['D-29', 'D-31'] }]
const view = (over: Record<string, unknown> = {}) => ({ case_id: CASE, case_version: 3, workflow_type: 'LEGACY_LEAD', rollout_mode: 'CONTROLLED_WRITEBACK',
  history: [{ event_id: ID(50), command_id: ID(51), outcome: 'NO_ANSWER', before_version: 2, after_version: 3, actor_user_id: ID(6), correlation_id: 'c',
    occurred_at: '2026-11-02T09:00:00Z' }], blocked_actions: blocked, current: { outcome: 'NO_ANSWER', last_command_id: ID(51), recorded_at: '2026-11-02T09:00:00Z' },
  ...over })
const caseRow = { case_id: CASE, workflow_type: 'LEGACY_LEAD', state_namespace: 'lead.status', state_raw: 'assigned', state_mapping: 'KNOWN', customer_id: null,
  customer_link_state: 'UNKNOWN', customer_link_reason: 'synthetic', presence: 'PRESENT', alias_count: 0, history_count: 0, imported_at: '2026-10-31T09:00:00Z',
  refreshed_at: '2026-10-31T09:00:00Z', source: { source_system: 'wp', source_spec: 'legacy.lead.v1', source_key: '101', channel_raw: null, origin_ref_raw: null,
    first_batch_id: ID(900), last_batch_id: ID(900), presence: 'PRESENT', first_seen_at: '2026-10-31T09:00:00Z', last_seen_at: '2026-10-31T09:00:00Z' } }
const features = (over: Record<string, boolean> = {}) => ({ hr_read: false, customers_identity: false, sales_read: true, billing_read: false, reporting_read: false,
  finance_shadow_read: false, biawin_fulfillment_read: false, contact_outcome_write: true, hr_request_write: true, writeback_read: true, ...over })

type Route = (req: Request) => Response | Promise<Response>
function routes(over: Record<string, Route> = {}, feat: Record<string, boolean> = {}) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: features(feat) })),
    [`GET /api/v1/cases/${CASE}`]: () => json(env({ allowed_actions: { 'cases.case.read': { allowed: true } }, data: { case: caseRow, aliases: [] } })),
    [`GET /api/v1/cases/${CASE}/history`]: () => json(env({ items: [], page: { has_more: false } })),
    [`GET /api/v1/cases/${CASE}/attribution`]: () => json(env({ items: [] })),
    [`GET /api/v1/cases/${CASE}/contact-outcomes`]: () => json(env({ data: view(), allowed_actions: { 'cases.contact_outcome.record': { allowed: true } } })),
    ...over,
  } as Record<string, Route>
}

function renderAt(path: string, r: Record<string, Route>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls }
}

const posted = (r: Request) => r.clone().json() as Promise<Record<string, unknown>>

describe('Slice 11 controlled writes', () => {
  it('feature off: the case page stays read-only (no outcome panel, no command call)', async () => {
    const { calls } = renderAt(`/sales/cases/${CASE}`, routes({}, { contact_outcome_write: false }))
    await screen.findAllByText(/legacy.lead.v1/)
    expect(screen.queryByText('پیامد تماس (دستور کنترل‌شده)')).toBeNull()
    expect(calls.some(c => c.url.endsWith('/contact-outcomes'))).toBe(false)
  })

  it('records an outcome: confirmation shows subject, current → new, impact; POST carries a key and the expected version', async () => {
    let seen: { headers: Headers; body: Record<string, unknown> } | null = null
    const { calls } = renderAt(`/sales/cases/${CASE}`, routes({
      [`POST /api/v1/cases/${CASE}/contact-outcomes`]: async (req: Request) => {
        seen = { headers: req.headers, body: await posted(req) }
        return json(env({ command: { command_id: ID(60), rollout_mode: 'CONTROLLED_WRITEBACK', applied: true, writeback_state: 'LEGACY_WRITEBACK_PENDING',
          blocked_by: [], version: 4 } }))
      },
    }))
    const form = await screen.findByRole('form', { name: 'ثبت پیامد تماس' })
    fireEvent.change(within(form).getByLabelText('پیامد'), { target: { value: 'NOT_PURCHASED' } })
    fireEvent.change(within(form).getByLabelText(/دلیل/), { target: { value: 'synthetic reason' } })
    fireEvent.submit(form)
    const dialog = await screen.findByRole('dialog', { name: 'تایید ثبت پیامد تماس' })
    expect(within(dialog).getByText(CASE)).toBeTruthy()
    expect(within(dialog).getByText(/مالکیت پرونده، فاکتور، پرداخت، مالی و اشتراک تغییر نمی‌کنند/)).toBeTruthy()
    fireEvent.click(within(dialog).getByRole('button', { name: 'تایید و ثبت' }))
    expect(await screen.findByText('در انتظار ثبت در سامانه قدیمی')).toBeTruthy()
    expect(screen.getByText('در سامانه جدید ثبت شد')).toBeTruthy()
    const s = seen as unknown as { headers: Headers; body: Record<string, unknown> }
    expect(s.headers.get('If-Match')).toBe('3')
    expect(s.headers.get('Idempotency-Key')).toMatch(/^[a-f0-9]{32}$/)
    expect(s.body).toEqual({ outcome: 'NOT_PURCHASED', reason: 'synthetic reason' })
    await waitFor(() => expect(calls.filter(c => c.url.endsWith('/contact-outcomes') && c.method === 'GET').length).toBe(2))
  })

  it('UNKNOWN is shown as reconciling and blind resubmission is disabled', async () => {
    renderAt(`/sales/cases/${CASE}`, routes({
      [`POST /api/v1/cases/${CASE}/contact-outcomes`]: () => json(env({ outcome: 'UNKNOWN', reasons: ['OUTCOME_UNKNOWN'],
        command: { rollout_mode: 'CONTROLLED_WRITEBACK', applied: false, writeback_state: 'NOT_REQUESTED', blocked_by: [] } })),
    }))
    fireEvent.submit(await screen.findByRole('form', { name: 'ثبت پیامد تماس' }))
    fireEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'تایید و ثبت' }))
    expect(await screen.findByText('نتیجه نامعلوم است و در حال تطبیق است')).toBeTruthy()
    expect(screen.queryByRole('form', { name: 'ثبت پیامد تماس' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'تایید و ثبت' })).toBeNull()
  })

  it('CONFLICT refreshes server state instead of overwriting', async () => {
    const { calls } = renderAt(`/sales/cases/${CASE}`, routes({
      [`POST /api/v1/cases/${CASE}/contact-outcomes`]: () => json(env({ outcome: 'CONFLICT', reasons: ['STALE_STATE'],
        command: { rollout_mode: 'CONTROLLED_WRITEBACK', applied: false, writeback_state: 'NOT_REQUESTED', blocked_by: [], current_version: 5 } }), 409),
    }))
    fireEvent.submit(await screen.findByRole('form', { name: 'ثبت پیامد تماس' }))
    fireEvent.click(within(await screen.findByRole('dialog')).getByRole('button', { name: 'تایید و ثبت' }))
    expect(await screen.findByText(/تعارض: وضعیت تغییر کرده/)).toBeTruthy()
    await waitFor(() => expect(calls.filter(c => c.url.endsWith('/contact-outcomes') && c.method === 'GET').length).toBe(2))
  })

  it('not allowed by the server: no form; blocked legacy actions are listed, never clickable', async () => {
    renderAt(`/sales/cases/${CASE}`, routes({
      [`GET /api/v1/cases/${CASE}/contact-outcomes`]: () => json(env({ data: view(), allowed_actions: { 'cases.contact_outcome.record': { allowed: false, reason_code: 'ROW_SCOPE_DENIED' } } })),
    }))
    expect(await screen.findByText(/ثبت پیامد تماس برای شما در دسترس نیست/)).toBeTruthy()
    expect(screen.queryByRole('form', { name: 'ثبت پیامد تماس' })).toBeNull()
    const list = screen.getByRole('list', { name: 'اقدام‌های مسدود' })
    expect(within(list).getByText('برگشت پرونده')).toBeTruthy()
    expect(within(list).getByText('صدور فاکتور')).toBeTruthy()
    expect(within(list).queryAllByRole('button')).toHaveLength(0)
    expect(screen.queryAllByRole('button').filter(b => /برگشت|صدور فاکتور|انتقال|تخصیص|آماده/.test(b.textContent ?? ''))).toHaveLength(0)
  })

  it('HR requests: create with confirmation; decisions only for the current reviewer; final apply never a control', async () => {
    const req = (status: string, decide: boolean) => ({ request_id: ID(70), request_type: 'terminate_seller', status, subject_user_id: ID(7),
      requested_by_user_id: ID(4), current_reviewer_user_id: status === 'pending_review' ? ID(3) : null, reason: 'synthetic', version: 2,
      created_at: '2026-11-02T09:00:00Z', updated_at: '2026-11-02T09:00:00Z', blocked_actions: [{ action: 'apply_final', blocked_by: ['HR_APPLY_IMPLEMENTATION_BLOCKED', 'OPD-05', 'SD-04'] }],
      allowed_actions: { approve_step: { allowed: decide }, reject: { allowed: decide } } })
    let created: Record<string, unknown> | null = null
    renderAt('/hr/requests', routes({
      'GET /api/v1/hr/requests': () => json(env({ rollout_mode: 'CONTROLLED_WRITEBACK', items: [req('pending_review', true)], page: { has_more: false },
        allowed_actions: { 'org.hr_request.create': { allowed: true } } })),
      'POST /api/v1/hr/requests': async (r: Request) => {
        created = await posted(r)
        return json(env({ command: { rollout_mode: 'CONTROLLED_WRITEBACK', applied: true, writeback_state: 'LEGACY_WRITEBACK_PENDING', blocked_by: [],
          request_id: ID(71), request_status: 'pending_review' } }))
      },
    }))
    expect(await screen.findByText('نیازمند تصمیم شما')).toBeTruthy()
    const form = screen.getByRole('form', { name: 'ثبت درخواست منابع انسانی' })
    fireEvent.change(within(form).getByLabelText('شناسه کاربر فروشنده'), { target: { value: ID(7) } })
    fireEvent.change(within(form).getByLabelText('دلیل'), { target: { value: 'synthetic reason' } })
    fireEvent.submit(form)
    const dialog = await screen.findByRole('dialog', { name: 'تایید ثبت درخواست' })
    expect(within(dialog).getByText(/سلسله‌مراتب، استخدام، دسترسی و پرونده‌ها تغییر نمی‌کنند/)).toBeTruthy()
    fireEvent.click(within(dialog).getByRole('button', { name: 'تایید و ثبت' }))
    expect(await screen.findByText('در انتظار ثبت در سامانه قدیمی')).toBeTruthy()
    expect(created).toEqual({ request_type: 'terminate_seller', subject_user_id: ID(7), reason: 'synthetic reason' })
    document.body.innerHTML = ''
    renderAt(`/hr/requests/${ID(70)}`, routes({
      [`GET /api/v1/hr/requests/${ID(70)}`]: () => json(env({ rollout_mode: 'CONTROLLED_WRITEBACK', request: req('pending_hr', false),
        history: [{ event_id: ID(72), command_id: ID(73), action: 'created', to_status: 'pending_review', after_version: 1, actor_user_id: ID(4),
          correlation_id: 'c', occurred_at: '2026-11-02T09:00:00Z' }] })),
    }))
    expect(await screen.findByText('در انتظار منابع انسانی (اعمال نشده)')).toBeTruthy()
    expect(screen.getByText('اعمال نهایی منابع انسانی')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'تایید مرحله' })).toBeNull()
    expect(screen.queryAllByRole('button').filter(b => /اعمال/.test(b.textContent ?? ''))).toHaveLength(0)
  })

  it('write-back diagnostics: counts and states, no fix or retry control', async () => {
    renderAt('/writeback', routes({
      'GET /api/v1/writeback/summary': () => json(env({ data: { states: [{ key: 'APPLIED', count: 3 }, { key: 'UNKNOWN', count: 1 }],
        parity: [{ key: 'MATCH', count: 3 }], reconciliations: [{ key: 'WRITEBACK_UNKNOWN', count: 1 }] } })),
      'GET /api/v1/writeback/commands': () => json(env({ items: [{ command_id: ID(80), source_module: 'cases', command_type: 'cases.contact_outcome',
        subject_kind: 'case', subject_id: CASE, rollout_mode: 'CONTROLLED_WRITEBACK', state: 'UNKNOWN', attempts: 1, received_at: '2026-11-02T09:00:00Z',
        updated_at: '2026-11-02T09:00:00Z' }], page: { has_more: false } })),
    }))
    expect(await screen.findByTestId('wb-UNKNOWN')).toBeTruthy()
    expect(screen.getByTestId('wb-APPLIED').textContent).toBe('3')
    const table = await screen.findByRole('table', { name: 'فرمان‌های بازنویسی' })
    expect(within(table).getByText('نتیجه نامعلوم؛ در حال تطبیق (تکرار نکنید)')).toBeTruthy()
    expect(screen.queryAllByRole('button').filter(b => /تکرار|اصلاح|رفع|retry|fix/i.test(b.textContent ?? ''))).toHaveLength(0)
  })
})
