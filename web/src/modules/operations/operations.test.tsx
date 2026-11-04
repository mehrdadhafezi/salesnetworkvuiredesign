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
const CASE = ID(1401)
const EXEC = ID(1402)
const allowed = Object.fromEntries(['fulfillment.work.read', 'fulfillment.work.assign', 'fulfillment.work.execute', 'biawin.activation.read',
  'integration.provider_operation.read', 'integration.provider_operation.reconcile'].map(c => [c, { allowed: true }]))
const ok = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-1414', allowed_actions: allowed, ...extra })
const workCase = (over: Record<string, unknown> = {}) => ({
  case_id: CASE, item_id: ID(1403), subscription_id: ID(1404), customer_id: ID(1405), service_key: 'prod-syn.svc-01', service_version: 1,
  service_title: 'خدمت آزمایشی', workflow_key: 'operational_task_v1', origin_mode: 'SHADOW_COMPARE', state: 'QUEUED', assignee_user_id: null, version: 1,
  current_supervisors: [ID(1406)], financial_behaviour: 'NONE', integration_provider: 'NONE', evidence_requirements: [{ type: 'service-form', required: true }],
  completion_outcomes: ['EXECUTED', 'NOT_EXECUTED'], scope: { supervisor: true, executor: false }, created_at: '2026-11-04T09:00:00Z', updated_at: '2026-11-04T09:00:00Z', ...over,
})
const blocked = [{ command: 'biawin.wallet_charge', blocked_by: ['D-38'] }, { command: 'biawin.activate_production', blocked_by: ['D-35'] },
  { command: 'fulfillment.cancel_case', blocked_by: ['OPD-04'] }]
const detail = (over: Record<string, unknown> = {}) => ok({ case: workCase(over), evidence: [], blocked_commands: blocked,
  history: [{ event_id: ID(1410), kind: 'CREATED', detail: {}, supervisors: [ID(1406)], after_version: 1, correlation_id: 'c', occurred_at: '2026-11-04T09:00:00Z' }] })

function routes(enabled = true) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: false, sales_read: false, billing_read: false, reporting_read: false,
      finance_shadow_read: false, biawin_fulfillment_read: false, commission_read: false, fulfillment_work: enabled } })),
    'GET /api/v1/fulfillment/work-cases': () => json(ok({ items: [workCase()], page: { has_more: false } })),
    [`GET /api/v1/fulfillment/work-cases/${CASE}`]: () => json(detail()),
    'GET /api/v1/biawin/activation-decisions': () => json(ok({ items: [
      { activation_id: ID(1420), business_key: 'invoice:x|product:prod-syn#1', invoice_id: ID(1421), invoice_ref: 'legacy-invoice-2001', customer_link_state: 'RESOLVED',
        customer_id: ID(1405), existing_customer: true, route: 'ADDITIONAL_SERVICES', product_key: 'prod-syn', product_version: 1, decision: 'SHADOW_ACTIVATED', reasons: [],
        blocked_by: [], subscription_id: ID(1404), item_count: 3, rollout_mode: 'SHADOW_COMPARE', correlation_id: 'c', message_id: ID(1422), decided_at: '2026-11-04T09:00:00Z' },
      { activation_id: ID(1423), business_key: 'invoice:y|product:prod-cmp#1', invoice_id: ID(1424), customer_link_state: 'RESOLVED', existing_customer: true,
        route: 'AFTER_SALES_COMPLAINT', product_key: 'prod-cmp', product_version: 1, decision: 'NOT_APPLICABLE', reasons: [], blocked_by: [], item_count: 0,
        rollout_mode: 'SHADOW_COMPARE', correlation_id: 'c', message_id: ID(1425), decided_at: '2026-11-04T09:00:00Z' },
    ] })),
    'GET /api/v1/integration/provider-operations': () => json(ok({ connector: 'NOT_CONFIGURED', items: [
      { operation_id: ID(1430), provider: 'BIAWIN', op_type: 'SERVICE_ORDER', subject_case_id: CASE, reference: 'syn-ref-0001', state: 'UNKNOWN', attempts: 1,
        reconciliation_owner: 'operations', version: 3, correlation_id: 'c', updated_at: '2026-11-04T09:00:00Z' },
      { operation_id: ID(1431), provider: 'BIAWIN', op_type: 'WALLET_CHARGE', reference: 'syn-ref-0002', state: 'BLOCKED', blocked_by: 'D-38', attempts: 0,
        reconciliation_owner: 'operations', version: 1, correlation_id: 'c', updated_at: '2026-11-04T09:00:00Z' },
    ] })),
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

// No control may imply activating, charging, dispatching, cancelling or reopening.
const forbiddenWords = /فعال‌سازی کن|شارژ کن|ارسال به بیاوین|تلاش مجدد|لغو کن|بازگشایی کن|activate|charge|dispatch|retry|cancel|reopen/i
const noForbidden = () => expect(screen.queryAllByRole('button').filter(b => forbiddenWords.test(b.textContent ?? ''))).toHaveLength(0)

describe('Controlled operations (Slice 14)', () => {
  it('shows navigation only when the server reports the feature', async () => {
    renderAt('/', routes(false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    expect(screen.queryByRole('link', { name: 'صف کار اجرای خدمت' })).toBeNull()
  })

  it('queue: scoped cases, GET only, nothing created by viewing', async () => {
    const { calls } = renderAt('/operations/work', routes())
    const table = await screen.findByRole('table', { name: 'صف کار اجرای خدمت' })
    expect(within(table).getByText('خدمت آزمایشی')).toBeTruthy()
    expect(within(table).getByText('ناظر خدمت')).toBeTruthy()
    noForbidden()
    for (const c of calls) expect(c.method).toBe('GET')
  })

  it('flag off: BLOCKED is a locked state, never an empty queue', async () => {
    const r = routes()
    r['GET /api/v1/fulfillment/work-cases'] = () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'c' })
    renderAt('/operations/work', r)
    await waitFor(() => expect(screen.queryByRole('table')).toBeNull())
    expect(await screen.findByText(/fulfillment.work_read/)).toBeTruthy()
  })

  it('case detail: decision-blocked commands listed as locked; assign needs confirmation, key and expected version', async () => {
    const posts: Request[] = []
    const r = routes()
    r[`POST /api/v1/fulfillment/work-cases/${CASE}/assignment`] = async (req: Request) => {
      posts.push(req.clone())
      return json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'c', command: { rollout_mode: 'SHADOW_COMPARE', applied: false, blocked_by: [], version: 1, replayed: false } })
    }
    const { calls } = renderAt(`/operations/work/${CASE}`, r)
    const blockedList = await screen.findByRole('list', { name: 'دستورهای مسدود' })
    expect(within(blockedList).getByText('شارژ کیف پول بیاوین')).toBeTruthy()
    expect(within(blockedList).getByText('D-38')).toBeTruthy()
    expect(within(blockedList).getByText('D-35')).toBeTruthy()
    noForbidden()
    expect(calls.filter(c => c.method !== 'GET')).toHaveLength(0)
    // No executor control for a supervisor-only scope.
    expect(screen.queryByRole('form', { name: 'ثبت نتیجه' })).toBeNull()
    fireEvent.change(screen.getByLabelText(/شناسه کاربر مجری/), { target: { value: EXEC } })
    fireEvent.click(screen.getByRole('button', { name: 'بررسی تخصیص' }))
    const dialog = await screen.findByRole('dialog', { name: 'تایید دستور' })
    expect(within(dialog).getByText(`تخصیص به مجری ${EXEC}`)).toBeTruthy()
    expect(posts).toHaveLength(0)
    fireEvent.click(within(dialog).getByRole('button', { name: 'تایید و ارسال' }))
    expect(await screen.findByText('ارزیابی و ثبت قصد شد (اعمال نشد)')).toBeTruthy()
    expect(posts).toHaveLength(1)
    const [post] = posts
    if (!post) throw new Error('no POST')
    expect(post.headers.get('If-Match')).toBe('1')
    expect((post.headers.get('Idempotency-Key') ?? '').length).toBeGreaterThanOrEqual(16)
    expect(await post.json()).toEqual({ assignee_user_id: EXEC })
  })

  it('executor: completion UNKNOWN is never shown as success and blocks resubmission', async () => {
    const r = routes()
    r[`GET /api/v1/fulfillment/work-cases/${CASE}`] = () => json(detail({ state: 'IN_PROGRESS', assignee_user_id: EXEC, version: 4, scope: { supervisor: false, executor: true } }))
    r[`POST /api/v1/fulfillment/work-cases/${CASE}/outcome`] = () => json({ outcome: 'UNKNOWN', reasons: ['OUTCOME_UNKNOWN'], correlation_id: 'c' })
    renderAt(`/operations/work/${CASE}`, r)
    const form = await screen.findByRole('form', { name: 'ثبت نتیجه' })
    expect(screen.queryByRole('form', { name: 'تخصیص پرونده' })).toBeNull()
    expect(within(form).getByText(/service-form/)).toBeTruthy()
    fireEvent.click(within(form).getByRole('button', { name: 'بررسی نتیجه' }))
    fireEvent.click(within(await screen.findByRole('dialog', { name: 'تایید دستور' })).getByRole('button', { name: 'تایید و ارسال' }))
    expect(await screen.findByText('نتیجه نامعلوم است و باید تطبیق شود')).toBeTruthy()
    expect(screen.queryByText(/اعمال شد/)).toBeNull()
    expect(screen.queryByRole('form', { name: 'ثبت نتیجه' })).toBeNull()
  })

  it('not executed requires a reason before any request', async () => {
    const r = routes()
    r[`GET /api/v1/fulfillment/work-cases/${CASE}`] = () => json(detail({ state: 'IN_PROGRESS', assignee_user_id: EXEC, scope: { supervisor: false, executor: true } }))
    const { calls } = renderAt(`/operations/work/${CASE}`, r)
    const form = await screen.findByRole('form', { name: 'ثبت نتیجه' })
    fireEvent.change(within(form).getByLabelText('نتیجه'), { target: { value: 'NOT_EXECUTED' } })
    fireEvent.click(within(form).getByRole('button', { name: 'بررسی نتیجه' }))
    expect(await screen.findByText('برای «انجام نشد» دلیل الزامی است.')).toBeTruthy()
    expect(calls.filter(c => c.method !== 'GET')).toHaveLength(0)
  })

  it('activation decisions: complaint never enters Biawin; read only', async () => {
    const { calls } = renderAt('/operations/activations', routes())
    const table = await screen.findByRole('table', { name: 'تصمیم‌های فعال‌سازی' })
    expect(within(table).getByText('شکایت پس از فروش (به بیاوین نمی‌رود)')).toBeTruthy()
    expect(within(table).getByText('فعال‌سازی سایه (بدون اثر واقعی)')).toBeTruthy()
    noForbidden()
    for (const c of calls) expect(c.method).toBe('GET')
  })

  it('provider operations: connector not configured, UNKNOWN not delivered, wallet charge blocked, no dispatch control', async () => {
    const { calls } = renderAt('/operations/provider-operations', routes())
    const table = await screen.findByRole('table', { name: 'عملیات ارائه‌دهنده' })
    expect(screen.getByText('اتصال بیاوین: پیکربندی نشده (NOT_CONFIGURED)')).toBeTruthy()
    expect(within(table).getByText('نامعلوم — تحویل تأیید نشده؛ ارسال مجدد ممنوع')).toBeTruthy()
    expect(within(table).getByText('شارژ کیف پول (مسدود — D-38)')).toBeTruthy()
    expect(within(table).queryByText(/تحویل شد/)).toBeNull()
    noForbidden()
    // Only the UNKNOWN row offers human reconciliation.
    expect(within(table).getAllByRole('button')).toHaveLength(1)
    for (const c of calls) expect(c.method).toBe('GET')
  })
})
