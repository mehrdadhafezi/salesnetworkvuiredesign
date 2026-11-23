import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import type { EffectiveAccess } from '../../shared/api'
import { fakeApi, healthy, json, session, type Captured } from '../../test/fakeApi'
import { EXCEPTIONS, LIST_UNIT, UNIT_LABEL, exceptionById, listUnit } from './workspace/exceptions'
import { FINANCIAL_OPERATIONS } from './workspace/operations'
import { FINANCE_NAV_LABEL } from './workspace/sections'

// RM-P07 Finance operational workspace. SYNTHETIC fixtures only; a passing
// test is presentation evidence, never proof that production authorizes
// these reads (the Finance role is not live verified).

const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const INV = ID(1)
const INV2 = ID(2)
const ALL_CAPS = ['billing.invoice.read', 'payments.attempt.read', 'billing.reconciliation.read', 'finance.review.read', 'payments.evidence.read', 'finance.shadow.read']
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-p071', ...extra })
const denied = () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'test-corr-dn07' }, 403)
const blocked = () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'test-corr-bl07' })
const COMMAND_WORDS = /تأیید|تایید|رد مالی|^رد$|استرداد|تسویه|ثبت در دفتر|تلاش دوباره|اصلاح|تطبیق کن|approve|reject|refund|retry|post|settle/i

type Route = (req: Request) => Response | Promise<Response>

function evaluation(allow: string[]): EffectiveAccess {
  return {
    authenticated: true, hr_read_enabled: true,
    decisions: ALL_CAPS.map(c => allow.includes(c)
      ? { capability: c as never, allowed: true, outcome: 'SUCCESS' as const, scope: 'ORGANIZATION' as const, matched_roles: ['synthetic_dev_stub'] }
      : { capability: c as never, allowed: false, outcome: 'UNAUTHORIZED' as const, reason_code: 'ACTION_DENIED' as const, scope: 'NONE' as const, matched_roles: [] }),
  }
}

const features = { hr_read: false, customers_identity: false, sales_read: false, billing_read: true, reporting_read: false, finance_shadow_read: true,
  biawin_fulfillment_read: false, contact_outcome_write: false, hr_request_write: false, writeback_read: false, payment_submission: true,
  finance_review_read: true, commission_read: false, fulfillment_work: false }

const sessionOn = () => session({
  authenticated: true, auth_mode: 'session', session_state: 'active', login_enabled: true,
  session: { user_id: ID(201), display_name: 'کاربر ساختگی', csrf_token: 'a'.repeat(43), idle_expires_at: '2099-01-01T00:00:00Z', absolute_expires_at: '2099-01-01T00:00:00Z' },
  features,
})

const report = {
  billing: { invoices_present: 14, items_present: 20, stages_present: 18, invoices_without_items: 3, invoices_without_stages: 2,
    invoice_state_vocabulary_by_field: {}, unknown_raw_states: { MAIN: { synthetic_x: 2 } }, stage_state_mapping: {}, unknown_stage_raw_states: {},
    customer_link_states: { RESOLVED: 9, CONFLICT: 4, UNKNOWN: 1 }, case_link_states: { RESOLVED: 10, UNKNOWN: 4 }, unrepresentable_amounts: 1,
    invoices_with_raw_pre_invoice_value: 0 },
  payments: { attempts_present: 23, attempts_by_channel: { GATEWAY: 12, MANUAL_CARD: 10, UNKNOWN: 1 }, attempt_state_mapping: { MAPPED: 20, UNKNOWN: 2, EMPTY: 1 },
    attempt_states: { ACCEPTED: 15, REJECTED: 3, RECONCILIATION_REQUIRED: 2 }, unknown_raw_attempt_states: { verified: 2 }, approved_raw_count: 11,
    reconcile_error_raw_count: 2, approval_provenance: { GATEWAY_AUTO: 8, LEGACY_MANUAL_APPROVAL: 7, NOT_APPROVED: 6, UNKNOWN: 2 }, gateway_attempts: 12,
    gateway_states: {}, attempts_without_invoice: 5, unrepresentable_amounts: 0, attempt_state_regressions: 6, receipts_referenced: 31, receipts_unknown: 7,
    receipts_missing: 8 },
  f02: { code: 'F02', label: 'Open Pre-invoices', status: 'NOT_EVALUATED', count: null, definition: 'd', authority: 'a', coverage: 'c', reason: 'r' },
  notes: [],
}
const evidence = { receipts_by_state: { READY: 13, NOT_RECOVERABLE: 2, QUARANTINED: 1 }, uploads_by_state: { BOUND: 16 },
  reconciliations: [{ reconciliation_id: ID(70), receipt_id: ID(50), attempt_id: ID(41), kind: 'OPENED', reason: 'NOT_RECOVERABLE', domain_owner: 'finance',
    next_safe_action: 'DOMAIN_OWNER_REVIEW', receipt_state: 'NOT_RECOVERABLE', opened_at: '2026-11-03T09:00:00Z' }] }

const gates = [{ action: 'finance.attempt.approve', outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED', 'POLICY_UNRESOLVED'], blocked_by: ['OPD-08', 'D-43', 'D-41'] },
  { action: 'finance.attempt.reject', outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], blocked_by: ['OPD-08', 'D-43'] }]
function review(n: number, over: Record<string, unknown> = {}) {
  return { attempt_id: ID(40 + n), batch_id: ID(30), invoice_id: INV, stage_no: 1, amount: 400, unit: 'IRT', state: 'PENDING_REVIEW',
    review_provenance: 'NOT_REVIEWED', evidence_state: 'READY', evidence_ready: true, reviewer_is_submitter: false, submitted_by_user_id: ID(6),
    submitted_at: '2026-11-03T09:00:00Z', receipts: [{ receipt_id: ID(50 + n), upload_id: ID(60 + n), state: 'READY', media_type: 'application/pdf', size_bytes: 10,
      updated_at: '2026-11-03T09:00:00Z' }], gates, ...over }
}

function attempt(n: number, over: Record<string, unknown> = {}) {
  return { attempt_id: ID(400 + n), invoice_id: INV, invoice_link_state: 'RESOLVED', stage_no_raw: '1', channel: 'MANUAL_CARD', pay_method_raw: 'card',
    payment_source_raw: 'seller_manual', amount: { representation: 'EXACT', raw: '300', money: { amount: 300, unit: 'IRT' } }, state_raw: 'rejected',
    state_mapping: 'MAPPED', state: 'REJECTED', approval_provenance: 'NOT_APPROVED', gateway: null, created_at_raw: '2026-01-01', updated_at_raw: '2026-01-02',
    version: 1, source: { source_system: 'wp', source_spec: 'legacy.payment.v1', source_key: String(70 + n), first_batch_id: ID(900), last_batch_id: ID(901),
      presence: 'PRESENT', first_seen_at: '2026-11-01T09:00:00Z', last_seen_at: '2026-11-01T09:00:00Z' }, ...over }
}

function invoice(n: number, over: Record<string, unknown> = {}) {
  return { invoice_id: ID(n), code_raw: `C-${n}`, source: { source_system: 'wp', source_spec: 'legacy.invoice.v2', source_key: String(3000 + n), first_batch_id: ID(900),
    last_batch_id: ID(901), presence: 'PRESENT', first_seen_at: '2026-11-01T09:00:00Z', last_seen_at: '2026-11-01T09:00:00Z' }, issued_at_raw: '2026-01-01',
    lifecycle_state: 'UNKNOWN', unknown_state_count: 1, item_count: 0, stage_count: 1, items_source_complete: true, stages_source_complete: true,
    customer: { state: 'CONFLICT', id: null }, case: { state: 'UNKNOWN', id: null }, ...over }
}

function reviewRoute(req: Request): Response {
  const q = new URL(req.url).searchParams
  if (q.get('cursor') === 'rv-2') return json(env({ data: [review(3, { invoice_id: INV2, amount: 5000000, unit: 'IRR', evidence_state: 'PENDING_FINALISATION', evidence_ready: false })], page: { has_more: false } }))
  if (q.get('evidence_state') === 'NOT_RECOVERABLE') return json(env({ data: [], page: { has_more: false } }))
  if (q.get('evidence_state') === 'PENDING_FINALISATION') return json(env({ data: [review(3, { evidence_state: 'PENDING_FINALISATION', evidence_ready: false })], page: { has_more: false } }))
  return json(env({ data: [review(1), review(2, { reviewer_is_submitter: true })], page: { has_more: true, next_cursor: 'rv-2' } }))
}

function base(over: Record<string, Route> = {}, allow = ALL_CAPS): Record<string, Route> {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(sessionOn()),
    'GET /api/v1/authorization/effective-access': () => json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-ea07', data: evaluation(allow) }),
    'GET /api/v1/billing/reconciliation-report': () => json(env({ data: report })),
    'GET /api/v1/payments/evidence': () => json(env({ data: evidence })),
    'GET /api/v1/finance/review/attempts': reviewRoute,
    'GET /api/v1/payments/attempts': () => json(env({ items: [attempt(1)], page: { has_more: false } })),
    'GET /api/v1/invoices': () => json(env({ items: [invoice(1)], page: { has_more: false } })),
    'GET /api/v1/finance/shadow-runs': () => json(env({ notice: 'SHADOW', items: [], page: { has_more: false } })),
    ...over,
  }
}

function renderAt(path: string, routes: Record<string, Route>) {
  const { client, calls } = fakeApi(routes)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

const nav = () => screen.getByRole('navigation', { name: FINANCE_NAV_LABEL })
const reads = (calls: Captured[], url: string) => calls.filter(c => c.url === url)
function noCommands() {
  const buttons = screen.queryAllByRole('button').map(b => (b.textContent ?? '').trim())
  for (const b of buttons) expect(b, `unexpected control "${b}"`).not.toMatch(COMMAND_WORDS)
}
function onlyGets(calls: Captured[]) {
  expect(calls.length).toBeGreaterThan(0)
  for (const c of calls) expect(c.method, c.url).toBe('GET')
  for (const c of calls) for (const k of ['phone', 'mobile', 'q', 'name']) expect(c.search.has(k), `${k} in ${c.url}`).toBe(false)
}

describe('RM-P07 Finance workspace landing', () => {
  it('sections, counters with named source and scope, unknown is never zero, operations are statuses only', async () => {
    const { calls } = renderAt('/finance', base())
    await screen.findByRole('heading', { level: 1, name: 'میز کار مالی' })
    await waitFor(() => expect(within(nav()).getAllByRole('link').map(a => a.textContent)).toEqual(
      ['میز کار مالی', 'فاکتورها', 'تلاش‌های پرداخت (قدیمی)', 'صف بررسی و پرداخت‌های ثبت‌شده', 'شواهد رسید', 'استثناها و شواهد حل‌نشده']))
    const inv = await screen.findByRole('list', { name: 'شمارنده‌های فاکتور' })
    await waitFor(() => expect(within(inv).getByText('۱۴')).toBeTruthy())
    expect(within(inv).getByText(/billing\.invoices_present/)).toBeTruthy()
    const att = screen.getByRole('list', { name: 'شمارنده‌های تلاش قدیمی' })
    expect(within(att).getByText('۱۱')).toBeTruthy() // approved_raw_count, labelled as not a Finance decision
    expect(within(att).getByText('«approved» خام (بررسی مالی یا تسویه نیست)')).toBeTruthy()
    const sub = screen.getByRole('list', { name: 'شمارنده‌های پرداخت ثبت‌شده' })
    // The queue has no total in any contract: unavailable, never 0.
    expect(within(sub).getAllByText('در دسترس نیست').length).toBeGreaterThan(0)
    expect(within(sub).getByText(/صفحه‌بندی کلیدی دارد/)).toBeTruthy()
    expect(within(sub).getByText('۱۳')).toBeTruthy()
    const ops = screen.getByRole('list', { name: 'وضعیت عملیات مالی' })
    expect(within(ops).getAllByText('در دسترس نیست')).toHaveLength(FINANCIAL_OPERATIONS.length)
    expect(within(ops).queryAllByRole('button')).toHaveLength(0)
    expect(within(ops).queryAllByRole('link')).toHaveLength(0)
    expect(screen.getByRole('heading', { level: 2, name: 'دفتر سایه مالی' })).toBeTruthy()
    expect(await screen.findByText('هنوز هیچ اجرای سایه‌ای ثبت نشده است')).toBeTruthy()
    noCommands()
    onlyGets(calls)
    // One request per aggregate (counters, exception summary and tiles share it).
    expect(reads(calls, '/api/v1/billing/reconciliation-report')).toHaveLength(1)
    expect(reads(calls, '/api/v1/payments/evidence')).toHaveLength(1)
  })

  it('a denied or disabled aggregate leaves its counters unavailable, not zero; the other source still shows', async () => {
    renderAt('/finance', base({ 'GET /api/v1/billing/reconciliation-report': denied, 'GET /api/v1/payments/evidence': blocked }))
    const inv = await screen.findByRole('list', { name: 'شمارنده‌های فاکتور' })
    await waitFor(() => expect(within(inv).getAllByText('دسترسی به گزارش تطبیق صورت‌حساب ندارید').length).toBe(4))
    expect(within(inv).queryByText('۰')).toBeNull()
    const sub = screen.getByRole('list', { name: 'شمارنده‌های پرداخت ثبت‌شده' })
    expect(within(sub).getAllByText('شواهد پرداخت فعال نشده است').length).toBe(2)
    expect(within(sub).queryByText('۰')).toBeNull()
  })

  it('section navigation follows the shell resolver: review-only sees only the landing and the queue', async () => {
    renderAt('/finance/review', base({}, ['finance.review.read']))
    await screen.findByRole('heading', { level: 1, name: 'صف بررسی مالی' })
    await waitFor(() => expect(within(nav()).getAllByRole('link').map(a => a.textContent)).toEqual(['میز کار مالی', 'صف بررسی و پرداخت‌های ثبت‌شده', 'استثناها و شواهد حل‌نشده']))
    expect(within(nav()).queryByRole('link', { name: 'فاکتورها' })).toBeNull()
  })
})

describe('RM-P07 Finance review queue', () => {
  it('summary, submission source, links to the verified invoice and its evidence, maker / checker, no decision control', async () => {
    const { calls } = renderAt('/finance/review', base())
    const table = await screen.findByRole('table', { name: 'پرداخت‌های در انتظار بررسی مالی' })
    const rows = within(table).getAllByRole('row').slice(1)
    expect(rows).toHaveLength(2)
    const [first, second] = rows as [HTMLElement, HTMLElement]
    expect(within(first).getByRole('link', { name: `فاکتور ${INV.slice(0, 8)}…` }).getAttribute('href')).toBe(`/billing/invoices/${INV}`)
    expect(within(first).getByRole('link', { name: 'تلاش‌های قدیمی' }).getAttribute('href')).toBe(`/billing/invoices/${INV}#inv-attempts`)
    expect(within(first).getByRole('link', { name: 'ارجاع رسیدها' }).getAttribute('href')).toBe(`/billing/invoices/${INV}#inv-receipts`)
    expect(within(first).getByRole('link', { name: 'جزئیات مورد بررسی' }).getAttribute('href')).toBe(`/finance/review/${ID(41)}`)
    expect(within(first).getByText('ثبت در سامانه جدید (نه قدیمی)')).toBeTruthy()
    expect(within(first).getByText('ادعای ثبت‌شده؛ وصول وجه نیست')).toBeTruthy()
    expect(within(second).getByText('شما ثبت‌کننده این پرداخت هستید؛ بررسی‌کننده مستقل نیستید')).toBeTruthy()
    expect(within(first).queryByText('شما ثبت‌کننده این پرداخت هستید؛ بررسی‌کننده مستقل نیستید')).toBeNull()
    expect(screen.getAllByText('تایید مالی: در دسترس نیست')).toHaveLength(2)
    const summary = screen.getByRole('group', { name: 'خلاصه صفحه جاری' })
    expect(within(summary).getByText(/بررسی‌کننده مستقل نیستید \(ثبت‌کننده یا هویت نامعلوم\): ۱/)).toBeTruthy()
    expect(within(summary).getByText(/شمار کل صف در قرارداد وجود ندارد/)).toBeTruthy()
    noCommands()
    onlyGets(calls)
    expect(reads(calls, '/api/v1/finance/review/attempts')).toHaveLength(1)
  })

  it('server pagination keeps each unit apart and never sums amounts', async () => {
    const { calls } = renderAt('/finance/review', base())
    await screen.findByRole('table', { name: 'پرداخت‌های در انتظار بررسی مالی' })
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    await screen.findByText(/۵٬۰۰۰٬۰۰۰ ریال/)
    expect(screen.queryByText(/تومان/)).toBeNull() // the IRT rows of page 1 are gone, nothing was carried or summed
    const pages = reads(calls, '/api/v1/finance/review/attempts')
    expect(pages.map(c => c.search.get('cursor'))).toEqual([null, 'rv-2'])
    expect(screen.queryByText(/(جمع|مجموع)[^۰-۹0-9]{0,12}[۰-۹0-9]/)).toBeNull() // no total value anywhere
  })

  it('evidence_state and invoice_id are server filters; a malformed invoice id is never sent', async () => {
    const { calls, router } = renderAt('/finance/review', base())
    const form = await screen.findByRole('form', { name: 'فیلتر صف بررسی مالی' })
    fireEvent.change(within(form).getByLabelText('آمادگی فیش'), { target: { value: 'PENDING_FINALISATION' } })
    fireEvent.change(within(form).getByLabelText('شناسه فاکتور (دقیق)'), { target: { value: INV } })
    fireEvent.click(within(form).getByRole('button', { name: 'اعمال فیلتر' }))
    await waitFor(() => expect(router.state.location.search).toBe(`?invoice_id=${INV}&evidence_state=PENDING_FINALISATION`))
    await waitFor(() => expect(reads(calls, '/api/v1/finance/review/attempts').some(c => c.search.get('evidence_state') === 'PENDING_FINALISATION' && c.search.get('invoice_id') === INV)).toBe(true))
    const before = reads(calls, '/api/v1/finance/review/attempts').length
    const form2 = screen.getByRole('form', { name: 'فیلتر صف بررسی مالی' })
    fireEvent.change(within(form2).getByLabelText('شناسه فاکتور (دقیق)'), { target: { value: '09120000000' } })
    fireEvent.click(within(form2).getByRole('button', { name: 'اعمال فیلتر' }))
    expect(await screen.findByText(/شناسه بومی کامل \(UUID\)/)).toBeTruthy()
    expect(reads(calls, '/api/v1/finance/review/attempts')).toHaveLength(before)
    cleanup()
    const direct = renderAt('/finance/review?invoice_id=3001', base())
    expect(await screen.findByText(/درخواستی به سرور ارسال نشد/)).toBeTruthy()
    expect(reads(direct.calls, '/api/v1/finance/review/attempts')).toHaveLength(0)
  })

  it('denied and blocked states; review denial is independent of invoice access', async () => {
    renderAt('/finance/review', base({ 'GET /api/v1/finance/review/attempts': denied }, ['billing.invoice.read', 'payments.attempt.read']))
    expect(await screen.findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(screen.queryByRole('table')).toBeNull()
    cleanup()
    renderAt('/finance/review', base({ 'GET /api/v1/finance/review/attempts': blocked }))
    expect(await screen.findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    cleanup()
    // Invoice denied while the queue is readable: the queue still renders; the invoice link is only a link.
    renderAt('/finance/review', base({ [`GET /api/v1/invoices/${INV}`]: denied }, ['finance.review.read']))
    expect(await screen.findByRole('table', { name: 'پرداخت‌های در انتظار بررسی مالی' })).toBeTruthy()
  })

  it('review item: invoice evidence links and the submission is not a receipt of funds', async () => {
    renderAt(`/finance/review/${ID(41)}`, base({ [`GET /api/v1/finance/review/attempts/${ID(41)}`]: () => json(env({ data: review(1),
      allowed_actions: { 'finance.attempt.approve': { allowed: false, reason_code: 'POLICY_UNRESOLVED' }, 'finance.attempt.reject': { allowed: false, reason_code: 'POLICY_UNRESOLVED' } } })) }))
    expect(await screen.findByText(/ادعای ثبت‌شده؛ وصول وجه، پرداخت فاکتور یا تسویه نیست/)).toBeTruthy()
    expect(screen.getByRole('link', { name: 'سایر پرداخت‌های ثبت‌شده این فاکتور' }).getAttribute('href')).toBe(`/finance/review?invoice_id=${INV}`)
    expect(screen.getByRole('link', { name: 'مراحل پرداخت' }).getAttribute('href')).toBe(`/billing/invoices/${INV}#inv-stages`)
    noCommands()
  })
})

describe('RM-P07 Finance exceptions', () => {
  it('every queue lists with its server filter; regression, link and evidence queues; no-contract queues say so', async () => {
    const { calls } = renderAt('/finance/exceptions?type=attempt-regression', base())
    await screen.findByRole('heading', { level: 1, name: 'استثناها و شواهد حل‌نشده' })
    await waitFor(() => expect(reads(calls, '/api/v1/payments/attempts').some(c => c.search.get('has_state_regression') === 'true')).toBe(true))
    expect(await screen.findByText('شاهد قدیمی')).toBeTruthy()
    expect(screen.getByText(/تعداد ردیف سابقه است، نه تلاش/)).toBeTruthy()
    const queues = screen.getByRole('navigation', { name: 'صف‌های استثنای مالی' })
    expect(within(queues).getAllByRole('link')).toHaveLength(EXCEPTIONS.length)

    fireEvent.click(within(queues).getByRole('link', { name: /پیوند مشتری فاکتور: تعارض/ }))
    await waitFor(() => expect(reads(calls, '/api/v1/invoices').some(c => c.search.get('customer_link_state') === 'CONFLICT' && c.search.get('presence') === 'PRESENT')).toBe(true))
    expect(await screen.findByText('مشتری: تعارض')).toBeTruthy()

    fireEvent.click(within(queues).getByRole('link', { name: /پرداخت ثبت‌شده: فیش در انتظار نهایی‌سازی/ }))
    await waitFor(() => expect(reads(calls, '/api/v1/finance/review/attempts').some(c => c.search.get('evidence_state') === 'PENDING_FINALISATION')).toBe(true))

    fireEvent.click(within(queues).getByRole('link', { name: /تعارض یا ناهم‌خوانی مبالغ فاکتور/ }))
    expect(await screen.findByText('فهرست این صف در قرارداد موجود نیست')).toBeTruthy()
    noCommands()
    onlyGets(calls)
  })

  it('counts come from the named aggregate; absent / unknown-status counts are unavailable, never zero', async () => {
    renderAt('/finance/exceptions?type=invoice-absent', base())
    const queues = await screen.findByRole('navigation', { name: 'صف‌های استثنای مالی' })
    const link = (name: RegExp) => within(queues).getByRole('link', { name })
    await waitFor(() => expect(within(link(/تلاش پرداخت بدون فاکتور حل‌شده/)).getByText('۵')).toBeTruthy())
    expect(within(link(/^.*فاکتور غایب در آخرین واردسازی کامل/)).getByText('در دسترس نیست')).toBeTruthy()
    expect(within(link(/فاکتور با وضعیت خام ناشناخته/)).getByText('در دسترس نیست')).toBeTruthy()
    expect(screen.getAllByText(/شمار غایب‌ها در هیچ قراردادی نیست/).length).toBeGreaterThan(0)
  })

  it('a queue whose module is denied shows denied; other queues are unaffected', async () => {
    renderAt('/finance/exceptions?type=attempt-rejected', base({ 'GET /api/v1/payments/attempts': denied }))
    expect(await screen.findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(screen.queryByRole('table')).toBeNull()
  })
})

describe('RM-P07 receipt evidence', () => {
  it('legacy references and target receipts are separate counts; READY is not a receipt of funds; reconciliation links', async () => {
    const { calls } = renderAt('/finance/receipts', base())
    const legacy = await screen.findByRole('list', { name: 'شمارنده‌های رسید قدیمی' })
    await waitFor(() => expect(within(legacy).getByText('۳۱')).toBeTruthy())
    expect(within(legacy).getByText('۸')).toBeTruthy()
    const target = await screen.findByRole('list', { name: 'شمارنده‌های فیش سامانه جدید' })
    expect(within(target).getByText('۱۳')).toBeTruthy()
    expect(screen.getByText(/فیش آماده به معنای وصول وجه نیست/)).toBeTruthy()
    expect(screen.getByRole('link', { name: 'مورد بررسی' }).getAttribute('href')).toBe(`/finance/review/${ID(41)}`)
    expect(screen.getByRole('link', { name: /شواهد غیرقابل بازیابی/ }).getAttribute('href')).toBe('/finance/review?evidence_state=NOT_RECOVERABLE')
    noCommands()
    onlyGets(calls)
  })

  it('target evidence denied: legacy counts still render, target counts never become zero', async () => {
    renderAt('/finance/receipts', base({ 'GET /api/v1/payments/evidence': denied }))
    const legacy = await screen.findByRole('list', { name: 'شمارنده‌های رسید قدیمی' })
    await waitFor(() => expect(within(legacy).getByText('۳۱')).toBeTruthy())
    expect(screen.getAllByText('دسترسی به این بخش ندارید').length).toBeGreaterThan(0)
    expect(screen.queryByRole('list', { name: 'شمارنده‌های فیش سامانه جدید' })).toBeNull()
  })
})

describe('RM-P07 exception count / list semantics', () => {
  it('a counter and a list never claim the same set unless they count the same unit', () => {
    for (const x of EXCEPTIONS) {
      const lu = listUnit(x)
      if (x.count.kind === 'none' || lu === undefined || lu === x.countUnit) continue
      expect(x.cardinality, `${x.id}: ${x.countUnit} counted, ${lu} listed, without a cardinality note`).toMatch(/\S/)
    }
    // Receipt counts are only ever paired with no list (no receipt list contract), never with a payment list.
    for (const x of EXCEPTIONS.filter(e => e.countUnit === 'receipt' || e.countUnit === 'receipt-ref')) {
      expect(x.list.read, x.id).toBe('none')
      if (x.list.read === 'none') expect(x.list.why, x.id).toMatch(/قرارداد/)
    }
    // Every review-queue list is a list of submissions, and its count is never a receipt count.
    for (const x of EXCEPTIONS.filter(e => e.list.read === 'review')) {
      expect(LIST_UNIT.review).toBe('submission')
      expect(x.countUnit, x.id).not.toBe('receipt')
    }
  })

  it('target-receipt-nr: receipts counted as receipts, no payment list, the related submission queue named as a different set', () => {
    const x = exceptionById('target-receipt-nr')
    expect(x?.countUnit).toBe('receipt')
    expect(x?.list.read).toBe('none')
    expect(x?.label).toMatch(/شمار فیش/)
    if (x?.list.read === 'none') {
      expect(x.list.why).toMatch(/مجموعه دیگری/)
      expect(x.list.why).toMatch(/یک‌به‌یک ادعا نمی‌شود/)
    }
    const related = exceptionById('review-evidence-nr')
    expect(related?.list.read).toBe('review')
    expect(related?.countUnit).toBe('submission')
    expect(related?.count.kind).toBe('none') // no submission total exists; the receipt count is not reused for it
  })

  it('UI: the receipt queue shows a receipt count, no payment rows and no review request; the regression queue states the cardinality', async () => {
    const { calls } = renderAt('/finance/exceptions?type=target-receipt-nr', base())
    expect(await screen.findByText('فهرست این صف در قرارداد موجود نیست')).toBeTruthy()
    expect(screen.getByTestId('count-unit').textContent).toBe(UNIT_LABEL.receipt)
    expect(screen.getByTestId('list-unit').textContent).toBe('فهرست قراردادی ندارد')
    const queues = screen.getByRole('navigation', { name: 'صف‌های استثنای مالی' })
    await waitFor(() => expect(within(within(queues).getByRole('link', { name: /شمار فیش/ })).getByText(UNIT_LABEL.receipt)).toBeTruthy())
    expect(screen.queryByRole('table')).toBeNull()
    expect(reads(calls, '/api/v1/finance/review/attempts')).toHaveLength(0)
    cleanup()
    renderAt('/finance/exceptions?type=attempt-regression', base())
    expect((await screen.findByTestId('cardinality')).textContent).toMatch(/دو عدد برابر نیستند/)
    expect(screen.getByTestId('count-unit').textContent).toBe(UNIT_LABEL['history-row'])
    expect(screen.getByTestId('list-unit').textContent).toBe(UNIT_LABEL.attempt)
  })
})
