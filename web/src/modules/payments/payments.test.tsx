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
const INV = ID(1)
const SHA = 'a'.repeat(64)
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-1111', ...extra })
const blockedEnv = { outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'test-corr-1111' }
const blockedActions = [{ action: 'finance_approve_payment', blocked_by: ['D-41', 'OPD-08', 'OPD-09', 'D-43'] },
  { action: 'issue_next_payment_stage', blocked_by: ['D-32', 'BILLING_STAGE_LIFECYCLE_UNKNOWN'] }, { action: 'pay_online_gateway', blocked_by: ['D-25', 'D-26'] }]
const attempt = { attempt_id: ID(40), batch_id: ID(30), item_key: 'a', amount: 400, unit: 'IRT', state: 'PENDING_REVIEW', review_provenance: 'NOT_REVIEWED',
  evidence_state: 'PENDING_FINALISATION', submitted_at: '2026-11-03T09:00:00Z',
  receipts: [{ receipt_id: ID(50), upload_id: ID(60), state: 'QUARANTINED', media_type: 'application/pdf', size_bytes: 10, updated_at: '2026-11-03T09:00:00Z' }] }
const ctxView = (over: Record<string, unknown> = {}) => ({ invoice_id: INV, stage_no: 1, unit: 'IRT', due_amount: 1000, remaining_amount: 1000,
  pending_or_accepted_amount: 400, available_amount: 600, collection_version: 1, eligible: true, evaluation_status: 'SUCCESS', reasons: [], blocked_by: [],
  rollout_mode: 'CONTROLLED_WRITEBACK', attempts: [attempt], blocked_actions: blockedActions, ...over })
const features = (over: Record<string, boolean> = {}) => ({ hr_read: false, customers_identity: false, sales_read: false, billing_read: true,
  reporting_read: false, finance_shadow_read: false, biawin_fulfillment_read: false, contact_outcome_write: false, hr_request_write: false,
  writeback_read: false, payment_submission: true, finance_review_read: true, ...over })
const batchResult = (over: Record<string, unknown> = {}) => ({ rollout_mode: 'CONTROLLED_WRITEBACK', applied: true, batch_id: ID(31), invoice_id: INV, stage_no: 1,
  collection_version: 2, total_amount: 600, unit: 'IRT', evidence_state: 'PENDING_FINALISATION', writeback_state: 'AWAITING_EVIDENCE_FINALISATION', blocked_by: [],
  items: [{ item_key: 'x', outcome: 'SUCCESS', reasons: [], blocked_by: [], valid: true, committed: true, attempt_id: ID(41), amount: 600, unit: 'IRT',
    receipts: [{ receipt_id: ID(51), upload_id: ID(61), state: 'QUARANTINED' }] }], ...over })

type Route = (req: Request) => Response | Promise<Response>
function routes(over: Record<string, Route> = {}, feat: Record<string, boolean> = {}) {
  const r: Record<string, Route> = {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: features(feat) })),
    [`GET /api/v1/payments/invoices/${INV}/submission`]: () => json(env({ data: ctxView(), allowed_actions: { 'payments.batch.submit': { allowed: true } } })),
    'POST /api/v1/payments/uploads': () => json(env({ data: { rollout_mode: 'CONTROLLED_WRITEBACK', upload_id: ID(61), content_sha256: SHA, size_bytes: 10,
      media_type: 'application/pdf', state: 'UNBOUND', existing: false, blocked_by: [] } })),
    ...over,
  }
  for (const p of ['', '/items', '/stages', '/payments', '/receipts', '/totals']) {
    r[`GET /api/v1/invoices/${INV}${p}`] ??= () => json(blockedEnv)
  }
  return r
}

function renderAt(path: string, r: Record<string, Route>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls }
}

async function stageAndReview(amount: string) {
  const form = await screen.findByRole('group', { name: 'ثبت پرداخت' })
  const file = new File(['%PDF-1.4 synthetic'], 'receipt.pdf', { type: 'application/pdf' })
  fireEvent.change(within(form).getByLabelText(/^فیش‌ها \(/), { target: { files: [file] } })
  await within(form).findByRole('checkbox')
  fireEvent.change(within(form).getByLabelText(/مبلغ/), { target: { value: amount } })
  fireEvent.click(within(form).getByRole('checkbox'))
  fireEvent.click(within(form).getByRole('button', { name: 'بررسی پیش از ثبت' }))
  return screen.findByRole('dialog', { name: 'تایید ثبت پرداخت' })
}

describe('Slice 12 controlled financial writes', () => {
  it('feature off: the invoice page has no payment panel and makes no payments call', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({}, { payment_submission: false }))
    await waitFor(() => expect(calls.some(c => c.url.includes(`/invoices/${INV}`))).toBe(true))
    expect(screen.queryByText(/ثبت پرداخت کارت‌به‌کارت/)).toBeNull()
    expect(calls.some(c => c.url.includes('/payments/'))).toBe(false)
  })

  it('stages a receipt, confirms impact and submits once with a key and the collection version; success never claims review or legacy', async () => {
    let seen: { headers: Headers; body: Record<string, unknown> } | null = null
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({
      [`POST /api/v1/payments/invoices/${INV}/batches`]: async (req: Request) => {
        seen = { headers: req.headers, body: await req.clone().json() as Record<string, unknown> }
        return json(env({ data: batchResult() }))
      },
    }))
    expect(await screen.findByText('۶۰۰ تومان')).toBeTruthy()
    const dialog = await stageAndReview('600')
    expect(within(dialog).getByText(/تایید مالی، ثبت در دفتر، تغییر مرحله، کمیسیون و اشتراک انجام نمی‌شود/)).toBeTruthy()
    expect(within(dialog).getByText(/دسته کامل ثبت می‌شود یا هیچ/)).toBeTruthy()
    fireEvent.click(within(dialog).getByRole('button', { name: 'تایید و ثبت' }))
    expect(await screen.findByText(/پرداخت در سامانه جدید ثبت شد — در انتظار بررسی مالی؛ هنوز تایید یا ثبت مالی نشده است/)).toBeTruthy()
    expect(screen.getByText('پس از نهایی شدن شواهد به سامانه قدیمی ارسال می‌شود')).toBeTruthy()
    const s = seen as unknown as { headers: Headers; body: Record<string, unknown> }
    expect(s.headers.get('If-Match')).toBe('1')
    expect(s.headers.get('Idempotency-Key')).toMatch(/^[a-f0-9]{32}$/)
    const items = s.body.items as { amount: number; unit: string; receipts: { upload_id: string; content_sha256: string }[] }[]
    expect(s.body.entry_mode).toBe('RECEIPT')
    expect(items[0]).toMatchObject({ amount: 600, unit: 'IRT', receipts: [{ upload_id: ID(61), content_sha256: SHA }] })
    expect(calls.filter(c => c.url.endsWith('/payments/uploads')).length).toBe(1)
    await waitFor(() => expect(calls.filter(c => c.url.endsWith('/submission')).length).toBe(2))
  })

  it('a refused batch shows each item honestly and states that nothing was committed', async () => {
    renderAt(`/billing/invoices/${INV}`, routes({
      [`POST /api/v1/payments/invoices/${INV}/batches`]: () => json(env({ outcome: 'BLOCKED', reasons: ['ACTION_DENIED'], data: batchResult({
        applied: false, batch_id: null, writeback_state: 'NOT_REQUESTED', items: [
          { item_key: 'x', outcome: 'BLOCKED', reasons: ['ACTION_DENIED'], blocked_by: [], valid: false, committed: false, amount: 700, unit: 'IRT', receipts: [] }] }) })),
    }))
    const dialog = await stageAndReview('700')
    fireEvent.click(within(dialog).getByRole('button', { name: 'تایید و ثبت' }))
    expect(await screen.findByText('این دسته ثبت نشد (هیچ ردیفی ثبت نشد)')).toBeTruthy()
    expect(screen.getByText(/رد شد: کد: ACTION_DENIED/)).toBeTruthy()
    expect(screen.queryByText(/پرداخت در سامانه جدید ثبت شد/)).toBeNull()
  })

  it('UNKNOWN disables blind resubmission', async () => {
    renderAt(`/billing/invoices/${INV}`, routes({
      [`POST /api/v1/payments/invoices/${INV}/batches`]: () => json(env({ outcome: 'UNKNOWN', reasons: ['OUTCOME_UNKNOWN'], data: batchResult({ applied: false,
        writeback_state: 'NOT_REQUESTED', items: [] }) })),
    }))
    fireEvent.click(within(await stageAndReview('100')).getByRole('button', { name: 'تایید و ثبت' }))
    expect(await screen.findByText('نتیجه نامعلوم است و در حال تطبیق است')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'تایید و ثبت' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'بررسی پیش از ثبت' })).toBeNull()
  })

  it('CONFLICT refreshes the server state instead of overwriting', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({
      [`POST /api/v1/payments/invoices/${INV}/batches`]: () => json(env({ outcome: 'CONFLICT', reasons: ['STALE_STATE'], data: batchResult({ applied: false,
        batch_id: null, current_version: 2, writeback_state: 'NOT_REQUESTED', items: [] }) }), 409),
    }))
    fireEvent.click(within(await stageAndReview('100')).getByRole('button', { name: 'تایید و ثبت' }))
    expect(await screen.findByText(/تعارض: وضعیت مرحله تغییر کرده/)).toBeTruthy()
    await waitFor(() => expect(calls.filter(c => c.url.endsWith('/submission')).length).toBe(2))
  })

  it('not allowed by the server: no form; blocked financial actions are listed and never clickable', async () => {
    renderAt(`/billing/invoices/${INV}`, routes({
      [`GET /api/v1/payments/invoices/${INV}/submission`]: () => json(env({ data: ctxView({ eligible: false, evaluation_status: 'UNAUTHORIZED',
        reasons: ['ROW_SCOPE_DENIED'], attempts: [] }), allowed_actions: { 'payments.batch.submit': { allowed: false, reason_code: 'ROW_SCOPE_DENIED' } } })),
    }))
    expect(await screen.findByText(/ثبت پرداخت برای شما در دسترس نیست/)).toBeTruthy()
    expect(screen.queryByRole('group', { name: 'ثبت پرداخت' })).toBeNull()
    const list = screen.getByRole('list', { name: 'اقدام‌های مسدود پرداخت' })
    expect(within(list).getByText('تایید مالی پرداخت')).toBeTruthy()
    expect(within(list).getByText('صدور مرحله بعدی پرداخت')).toBeTruthy()
    expect(within(list).queryAllByRole('button')).toHaveLength(0)
    expect(screen.queryAllByRole('button').filter(b => /تایید مالی|رد مالی|مرحله بعد|استرداد|درگاه/.test(b.textContent ?? ''))).toHaveLength(0)
  })

  it('Finance review queue is read only: D-43 and maker / checker gates shown, no decision control', async () => {
    const review = (over: Record<string, unknown> = {}) => ({ attempt_id: ID(40), batch_id: ID(30), invoice_id: INV, stage_no: 1, amount: 400, unit: 'IRT',
      state: 'PENDING_REVIEW', review_provenance: 'NOT_REVIEWED', evidence_state: 'PENDING_FINALISATION', evidence_ready: false, reviewer_is_submitter: true,
      submitted_by_user_id: ID(6), submitted_at: '2026-11-03T09:00:00Z', receipts: attempt.receipts,
      gates: [{ action: 'finance.attempt.approve', outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED', 'POLICY_UNRESOLVED'], blocked_by: ['OPD-08', 'D-43', 'D-41'] },
        { action: 'finance.attempt.reject', outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], blocked_by: ['OPD-08', 'D-43'] }], ...over })
    const { calls } = renderAt('/finance/review', routes({
      'GET /api/v1/finance/review/attempts': () => json(env({ data: [review()], page: { has_more: false } })),
    }))
    expect(await screen.findByText('فیش آماده نیست؛ تصمیم مالی مسدود (D-43)')).toBeTruthy()
    expect(screen.getByText('شما ثبت‌کننده این پرداخت هستید؛ بررسی‌کننده مستقل نیستید')).toBeTruthy()
    expect(screen.getByText('تایید مالی: در دسترس نیست')).toBeTruthy()
    expect(screen.queryAllByRole('button').filter(b => /تایید|رد/.test(b.textContent ?? ''))).toHaveLength(0)
    expect(calls.every(c => c.method === 'GET')).toBe(true)
  })

  it('evidence diagnostics read only', async () => {
    renderAt('/payments/evidence', routes({
      'GET /api/v1/payments/evidence': () => json(env({ data: { receipts_by_state: { READY: 3, NOT_RECOVERABLE: 1 }, uploads_by_state: { BOUND: 4 },
        reconciliations: [{ reconciliation_id: ID(70), receipt_id: ID(50), attempt_id: ID(40), kind: 'OPENED', reason: 'NOT_RECOVERABLE', domain_owner: 'finance',
          next_safe_action: 'DOMAIN_OWNER_REVIEW', receipt_state: 'NOT_RECOVERABLE', opened_at: '2026-11-03T09:00:00Z' }] } })),
    }))
    expect(await screen.findByTestId('rcpt-READY')).toBeTruthy()
    expect(screen.getByText('بررسی توسط مالک دامنه (مالی)')).toBeTruthy()
    expect(screen.queryAllByRole('button').filter(b => /اصلاح|تلاش|retry/i.test(b.textContent ?? ''))).toHaveLength(0)
  })
})
