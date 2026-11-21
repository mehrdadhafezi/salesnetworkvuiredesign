import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

// RM-P02 invoice 360° workspace. All data is SYNTHETIC. The workspace only
// composes existing read endpoints; it never sums across units, never shows
// a canonical total, never treats gateway / legacy acceptance as a Finance
// decision and never links an unconfirmed relationship.
const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const INV = ID(1)
const CUST = ID(500)
const CASE = ID(600)
const caps = (...c: string[]) => Object.fromEntries(c.map(x => [x, { allowed: true }]))
const FULL = caps('billing.invoice.read', 'payments.attempt.read')
const env = (extra: Record<string, unknown>, allowed = FULL) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-0202', allowed_actions: allowed, ...extra })
const src = (key: string, spec = 'legacy.invoice.v2') => ({ source_system: 'wp', source_spec: spec, source_key: key, first_batch_id: ID(900), last_batch_id: ID(901),
  presence: 'PRESENT', first_seen_at: '2026-11-01T09:00:00Z', last_seen_at: '2026-11-01T09:00:00Z' })
const irt = (n: string, raw = n) => ({ raw, representation: 'REPRESENTABLE', money: { amount: n, unit: 'IRT' } })
const page = (items: unknown[], next?: string) => ({ items, page: next ? { has_more: true, next_cursor: next } : { has_more: false } })

const invoice = (over: Record<string, unknown> = {}) => ({ invoice_id: INV, code_raw: 'SYN-0202', lifecycle_state: 'UNKNOWN',
  customer: { id: CUST, state: 'RESOLVED', basis: 'explicit issued-to reference' }, case: { id: CASE, state: 'RESOLVED', basis: 'explicit lead reference' },
  items_source_complete: true, stages_source_complete: false, item_count: 2, stage_count: 1, unknown_state_count: 1, issued_at_raw: '2025-03-01 10:00:00',
  source: src('2001'), ...over })

const detail = (inv = invoice()) => ({ invoice: inv, product_ref_raw: '301', pay_method_raw: 'card', payment_plan_raw: 'installment', current_stage_raw: '1',
  updated_at_raw: null, states: [{ field: 'MAIN', namespace: 'legacy_invoice.status', raw: 'synthetic_unfamiliar_status', mapping: 'UNKNOWN' }],
  declared: [{ kind: 'FINAL_TOTAL', amount: irt('1000', '1000.00') }, { kind: 'REMAINING_AMOUNT', amount: { raw: null, representation: 'ABSENT' } },
    { kind: 'DISCOUNT_TOTAL', amount: { raw: '0.50', representation: 'UNREPRESENTABLE' } }] })

const attempt = (n: number, over: Record<string, unknown> = {}) => ({ attempt_id: ID(300 + n), invoice_id: INV, invoice_link_state: 'RESOLVED', stage_no_raw: '1',
  channel: 'GATEWAY', pay_method_raw: 'gateway', payment_source_raw: 'zibal', amount: irt('600'), state_namespace: 'legacy_payment.status', state_raw: 'approved',
  state_mapping: 'MAPPED', state: 'ACCEPTED', approval_provenance: 'GATEWAY_AUTO', uploaded_by_type_raw: 'seller', created_at_raw: null, updated_at_raw: null, version: 1,
  source: src(String(70 + n), 'legacy.payment.v1'),
  gateway: { provider_raw: 'zibal', has_authority_reference: true, has_gateway_reference: false, state_raw: 'verified', state_mapping: 'MAPPED', state: 'VERIFIED' }, ...over })

const sum = (state: string, amount?: string, unit = 'IRT') => ({ state, rows: amount ? 1 : 0, ...(amount ? { money: { amount, unit } } : {}) })
const totals = (over: Record<string, unknown> = {}) => ({ invoice_id: INV, compared_declared: ['FINAL_TOTAL'], payment_evidence_complete: false, unclassified_attempts: 1,
  units: [
    { unit: 'IRT', declared: [{ kind: 'FINAL_TOTAL', money: { amount: '1000', unit: 'IRT' } }], item_sum: sum('COMPLETE', '1000'), stage_sum: sum('INCOMPLETE', '400'),
      payment_sum: sum('COMPLETE', '600'), class: 'MISMATCH' },
    { unit: 'IRR', declared: [], item_sum: sum('COMPLETE', '5000', 'IRR'), stage_sum: sum('NONE'), payment_sum: sum('NONE'), class: 'UNKNOWN' },
  ], ...over })

const review = (n: number, over: Record<string, unknown> = {}) => ({ attempt_id: ID(800 + n), batch_id: ID(810), invoice_id: INV, stage_no: 1, amount: 250, unit: 'IRT',
  state: 'PENDING_REVIEW', review_provenance: 'NOT_REVIEWED', evidence_state: 'PENDING_FINALISATION', evidence_ready: false, reviewer_is_submitter: false,
  submitted_by_user_id: ID(706), submitted_at: '2026-11-03T08:00:00Z', receipts: [],
  gates: [{ action: 'finance.attempt.approve', outcome: 'BLOCKED', reasons: ['POLICY_UNRESOLVED'], blocked_by: ['D-41', 'D-43'] },
    { action: 'finance.attempt.reject', outcome: 'BLOCKED', reasons: ['POLICY_UNRESOLVED'], blocked_by: ['D-41', 'OPD-08'] }], ...over })

type Handler = (req: Request) => Response | Promise<Response>
const q = (req: Request) => new URL(req.url).searchParams

function routes(over: Record<string, Handler> = {}, allowed = FULL): Record<string, Handler> {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: true, sales_read: true, billing_read: true } })),
    [`GET /api/v1/invoices/${INV}`]: () => json(env({ data: detail() }, allowed)),
    [`GET /api/v1/invoices/${INV}/items`]: () => json(env(page([
      { item_id: ID(11), product_ref_raw: '301', title_snapshot: 'Synthetic product A', kind_raw: 'main', qty_raw: '1', unit_price: irt('600'), line_total: irt('600'),
        is_free_raw: '0', created_at_raw: null, source: src('31', 'legacy.invoice_item.v1') }]), allowed)),
    [`GET /api/v1/invoices/${INV}/stages`]: () => json(env(page([{ stage_id: ID(41), stage_no_raw: '1', stage_type_raw: 'installment', requested: irt('400'),
      state_namespace: 'legacy_invoice_payment_stage.status', state_raw: 'synthetic_stage_status', state_mapping: 'UNKNOWN', state: null, pay_method_raw: null,
      payment_source_raw: null, approved_at_raw: null, paid_at_raw: null, created_at_raw: null, source: src('41', 'legacy.invoice_payment_stage.v1') }]), allowed)),
    [`GET /api/v1/invoices/${INV}/payments`]: req => json(env(q(req).get('cursor') === 'p-2'
      ? page([attempt(2, { channel: 'MANUAL_CARD', gateway: undefined, approval_provenance: 'NOT_APPROVED', state_raw: 'reconcile_error', state: 'RECONCILIATION_REQUIRED' })])
      : page([attempt(1)], 'p-2'), allowed)),
    [`GET /api/v1/invoices/${INV}/receipts`]: () => json(env(page([{ receipt_id: ID(51), invoice_id: INV, owner_kind: 'STAGE', stage_no_raw: '1', ref_index: 0,
      has_reference: true, host: 'files.example.test', extension: 'jpg', state: 'REFERENCE_ONLY', source: src('41', 'legacy.invoice_payment_stage.v1') }]), allowed)),
    [`GET /api/v1/invoices/${INV}/totals`]: () => json(env({ data: totals() }, allowed)),
    [`GET /api/v1/payments/attempts/${ID(301)}/history`]: req => json(env(q(req).get('cursor') === 'h-2'
      ? page([{ history_id: ID(92), old_state_raw: 'approved', new_state_raw: 'reconcile_error', regression: true, batch_id: ID(901), recorded_at: '2026-11-02T09:00:00Z' }])
      : page([{ history_id: ID(91), old_state_raw: 'pending', new_state_raw: 'approved', regression: false, batch_id: ID(900), recorded_at: '2026-11-01T09:00:00Z' }], 'h-2'))),
    'GET /api/v1/finance/review/attempts': req => json(q(req).get('invoice_id') === INV
      ? { outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-r', data: [review(1)], page: { has_more: false } }
      : { outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-r', data: [], page: { has_more: false } }),
    ...over,
  }
}

function renderAt(path: string, r: Record<string, Handler>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

const region = (name: string) => screen.getByRole('region', { name })
const open = async () => {
  await screen.findByRole('heading', { level: 1, name: 'فاکتور: legacy.invoice.v2 #2001' })
  return screen
}
const writeWords = /تأیید کن|تایید کن|رد کن|استرداد|صدور|ثبت دفتر|approve|reject|refund|post/i

describe('RM-P02 invoice 360° workspace', () => {
  it('shows every section with working in-page navigation and no decision control', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes())
    await open()
    const nav = screen.getByRole('navigation', { name: 'بخش‌های فاکتور' })
    const anchors = within(nav).getAllByRole('link')
    expect(anchors.map(a => a.getAttribute('href'))).toEqual(['#inv-overview', '#inv-facts', '#inv-states', '#inv-amounts', '#inv-items', '#inv-stages',
      '#inv-attempts', '#inv-receipts', '#inv-review', '#inv-provenance'])
    for (const a of anchors) expect(document.getElementById((a.getAttribute('href') ?? '').slice(1))).not.toBeNull()
    for (const t of ['نمای کلی و پیوندها', 'فاکتور', 'شواهد وضعیت (هر فیلد جدا)', 'شواهد جمع (به تفکیک واحد؛ بدون برنده)', 'اقلام (اسنپ‌شات صدور)', 'مراحل پرداخت',
      'تلاش‌های پرداخت', 'رسیدها (فقط فراداده ارجاع)', 'بررسی مالی', 'منشأ']) expect(region(t)).toBeTruthy()
    await within(region('بررسی مالی')).findByRole('table')
    expect(screen.queryAllByRole('button').filter(b => writeWords.test(b.textContent ?? ''))).toHaveLength(0)
    for (const c of calls) expect(c.method).toBe('GET')
  })

  it('overview: RESOLVED relationships navigate; lifecycle and total are never claimed', async () => {
    renderAt(`/billing/invoices/${INV}`, routes())
    await open()
    const ov = region('نمای کلی و پیوندها')
    expect(within(ov).getByRole('link', { name: 'مشاهده مشتری' }).getAttribute('href')).toBe(`/customers/${CUST}`)
    expect(within(ov).getByRole('link', { name: 'مشاهده پرونده' }).getAttribute('href')).toBe(`/sales/cases/${CASE}`)
    expect(within(ov).getByText('تعیین نشده (TAD #20)')).toBeTruthy()
    expect(within(ov).getByText(/جمع قطعی فاکتور تعیین نشده است/)).toBeTruthy()
    expect(within(ov).getByText('منبع ناقص')).toBeTruthy()
    expect(within(ov).getByText('1 مقدار خام نگاشت‌نشده')).toBeTruthy()
    // Per-unit classes and incomplete payment evidence, from the totals read.
    expect(await within(ov).findByText('ناهم‌خوان')).toBeTruthy()
    expect(within(ov).getByText('نامعلوم (شاهد کافی نیست)')).toBeTruthy()
    expect(within(ov).getByText('شواهد پرداخت ناقص')).toBeTruthy()
  })

  it('missing and unresolved relationships are shown, never linked', async () => {
    const inv = invoice({ customer: { id: null, state: 'UNKNOWN', basis: 'no customer for the issued-to subject' },
      case: { id: CASE, state: 'CONFLICT', basis: 'two lead references disagree' } })
    renderAt(`/billing/invoices/${INV}`, routes({ [`GET /api/v1/invoices/${INV}`]: () => json(env({ data: detail(inv) })) }))
    await open()
    const ov = region('نمای کلی و پیوندها')
    expect(within(ov).getByText('نامعلوم')).toBeTruthy()
    expect(within(ov).getByText('تعارض')).toBeTruthy()
    expect(within(ov).queryByRole('link', { name: /مشاهده/ })).toBeNull()
    expect(document.querySelector(`a[href="/sales/cases/${CASE}"]`)).toBeNull()
  })

  it('amount evidence stays per unit; incomplete, absent and unrepresentable are never zero; no cross-unit total', async () => {
    renderAt(`/billing/invoices/${INV}`, routes())
    await open()
    const am = region('شواهد جمع (به تفکیک واحد؛ بدون برنده)')
    const irtCard = await within(am).findByRole('region', { name: 'واحد IRT (تومان)' })
    const irrCard = within(am).getByRole('region', { name: 'واحد IRR (ریال)' })
    expect(within(irtCard).getByText('ناهم‌خوان')).toBeTruthy()
    expect(within(irtCard).getByText('(ناقص)')).toBeTruthy() // stage sum incomplete
    expect(within(irrCard).getAllByText('(بدون سهم (صفر نیست))').length).toBe(2)
    expect(within(irrCard).getByText('در این واحد اعلام نشده (صفر نیست)')).toBeTruthy()
    expect(within(am).getByText(/شواهد پرداخت ناقص است \(تلاش‌های طبقه‌بندی‌نشده: 1\)/)).toBeTruthy()
    // Declared evidence: absent and unrepresentable stay distinct from zero.
    expect(within(am).getByText('ثبت نشده (صفر نیست)')).toBeTruthy()
    expect(within(am).getByText(/گرد نشد/)).toBeTruthy()
    // Nothing adds IRT and IRR (1000 + 5000) or the item / payment sums together.
    expect(am.textContent).not.toMatch(/6000|6,000|۶۰۰۰|1600/)
  })

  it('conflicting declared totals are shown as a conflict, not resolved', async () => {
    const conflict = totals({ payment_evidence_complete: true, unclassified_attempts: 0, units: [{ unit: 'IRT',
      declared: [{ kind: 'FINAL_TOTAL', money: { amount: '1000', unit: 'IRT' } }, { kind: 'PAYMENT_TOTAL_AMOUNT', money: { amount: '900', unit: 'IRT' } }],
      item_sum: sum('COMPLETE', '1000'), stage_sum: sum('COMPLETE', '1000'), payment_sum: sum('COMPLETE', '1000'), class: 'CONFLICT' }] })
    renderAt(`/billing/invoices/${INV}`, routes({ [`GET /api/v1/invoices/${INV}/totals`]: () => json(env({ data: conflict })) }))
    await open()
    const am = region('شواهد جمع (به تفکیک واحد؛ بدون برنده)')
    expect(await within(am).findByText('تعارض در مبالغ اعلامی')).toBeTruthy()
    const unit = within(am).getByRole('region', { name: 'واحد IRT (تومان)' })
    expect(within(unit).getByText('جمع نهایی اعلامی')).toBeTruthy()
    expect(within(unit).getByText('جمع پرداخت اعلامی')).toBeTruthy()
    expect(within(region('نمای کلی و پیوندها')).queryByText('شواهد پرداخت ناقص')).toBeNull()
  })

  it('linked payments: attempts page with the server cursor and each attempt opens its own history', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes())
    await open()
    const at = region('تلاش‌های پرداخت')
    await within(at).findByText('تلاش: پذیرفته‌شده (نه بررسی مالی، نه تسویه)')
    expect(within(at).getByText(/درگاه · پذیرش خودکار درگاه \(بررسی مالی نیست\)/)).toBeTruthy()
    expect(within(at).getByText('درگاه: تأییدشده توسط درگاه')).toBeTruthy()
    expect(within(at).getByText(/تأیید نهایی مالی یا ثبت حسابداری نیست/)).toBeTruthy()

    const toggle = within(at).getByRole('button', { name: 'نمایش سابقه' })
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    fireEvent.click(toggle)
    expect(within(at).getByRole('button', { name: 'بستن سابقه' }).getAttribute('aria-expanded')).toBe('true')
    const hist = await within(at).findByLabelText('سابقه وضعیت تلاش')
    expect(await within(hist).findByText('pending → approved')).toBeTruthy()
    fireEvent.click(within(hist).getByRole('button', { name: 'صفحه بعد' }))
    expect(await within(hist).findByText('approved → reconcile_error')).toBeTruthy()
    expect(within(hist).getByText('بازگشت از پذیرفته‌شده')).toBeTruthy()
    expect(calls.filter(c => c.url === `/api/v1/payments/attempts/${ID(301)}/history`).at(-1)?.search.get('cursor')).toBe('h-2')

    fireEvent.click(within(within(at).getByRole('navigation', { name: 'صفحه‌بندی تلاش‌ها' })).getByRole('button', { name: 'صفحه بعد' }))
    expect(await within(at).findByText('تلاش: نیازمند تطبیق')).toBeTruthy()
    expect(within(at).getByText(/کارت به کارت \(دستی\) · تأیید نشده/)).toBeTruthy()
    expect(calls.filter(c => c.url === `/api/v1/invoices/${INV}/payments`).at(-1)?.search.get('cursor')).toBe('p-2')
  })

  it('finance review: this invoice only, with the D-43 / maker-checker gate and a link to the existing review page', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes())
    await open()
    const rv = region('بررسی مالی')
    const link = await within(rv).findByRole('link', { name: /۲۵۰/ })
    expect(link.getAttribute('href')).toBe(`/finance/review/${ID(801)}`)
    expect(within(rv).getByText('فیش آماده نیست؛ تصمیم مالی مسدود (D-43)')).toBeTruthy()
    expect(within(rv).getByText('تایید مالی: در دسترس نیست')).toBeTruthy()
    expect(within(rv).getByText(/تصمیم مالی \(تأیید یا رد\) در این نسخه وجود ندارد/)).toBeTruthy()
    const call = calls.find(c => c.url === '/api/v1/finance/review/attempts')
    expect(call?.search.get('invoice_id')).toBe(INV)
    expect([...(call?.search.keys() ?? [])].sort()).toEqual(['invoice_id', 'limit'])
  })

  it('finance review unauthorized / blocked / empty is local to its section', async () => {
    const cases: [Handler, RegExp][] = [
      [() => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-1' }, 403), /دسترسی به این بخش ندارید/],
      [() => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'c-2' }), /این بخش هنوز فعال نشده است/],
      [() => json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'c-3', data: [], page: { has_more: false } }), /در انتظار بررسی مالی نیست/],
    ]
    for (const [handler, want] of cases) {
      const { router } = renderAt(`/billing/invoices/${INV}`, routes({ 'GET /api/v1/finance/review/attempts': handler }))
      await open()
      const rv = region('بررسی مالی')
      expect(await within(rv).findByText(want)).toBeTruthy()
      expect(within(rv).queryByRole('table')).toBeNull()
      expect(document.body.textContent).not.toContain(ID(801))
      // Other sections are unaffected.
      expect(await within(region('تلاش‌های پرداخت')).findByText('تلاش: پذیرفته‌شده (نه بررسی مالی، نه تسویه)')).toBeTruthy()
      router.dispose()
      document.body.innerHTML = ''
    }
    expect(true).toBe(true)
  })

  it('a failing section does not take down the others', async () => {
    const boom = () => json({ type: '/problems/internal', title: 'Internal error', status: 500, reason_code: 'INTERNAL', correlation_id: 'corr-500' }, 500,
      'application/problem+json')
    renderAt(`/billing/invoices/${INV}`, routes({ [`GET /api/v1/invoices/${INV}/items`]: boom, [`GET /api/v1/invoices/${INV}/totals`]: boom }))
    await open()
    expect(await within(region('اقلام (اسنپ‌شات صدور)')).findByText('Internal error')).toBeTruthy()
    expect(await within(region('شواهد جمع (به تفکیک واحد؛ بدون برنده)')).findByText('Internal error')).toBeTruthy()
    expect(within(region('نمای کلی و پیوندها')).getByText('در دسترس نیست')).toBeTruthy()
    expect(await within(region('مراحل پرداخت')).findByText(/synthetic_stage_status/)).toBeTruthy()
    expect(await within(region('رسیدها (فقط فراداده ارجاع)')).findByText('فقط ارجاع (فایل بررسی نشده)')).toBeTruthy()
  })

  it('without payments.attempt.read: payment sections are locked and payment endpoints are never called', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({}, caps('billing.invoice.read')))
    await open()
    for (const t of ['تلاش‌های پرداخت', 'رسیدها (فقط فراداده ارجاع)', 'شواهد جمع (به تفکیک واحد؛ بدون برنده)']) {
      expect(within(region(t)).getByText('دسترسی به شواهد پرداخت ندارید')).toBeTruthy()
    }
    expect(within(region('نمای کلی و پیوندها')).getByText('دسترسی به شواهد پرداخت ندارید')).toBeTruthy()
    await waitFor(() => expect(calls.some(c => c.url.endsWith('/stages'))).toBe(true))
    for (const p of ['/payments', '/receipts', '/totals']) expect(calls.some(c => c.url === `/api/v1/invoices/${INV}${p}`)).toBe(false)
    expect(document.body.textContent).not.toMatch(/zibal|files\.example\.test|reconcile_error/)
  })

  it('invoice detail denied: every invoice section is locked and no invoice data reaches the DOM', async () => {
    const denied = () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-9' }, 403)
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({ [`GET /api/v1/invoices/${INV}`]: denied }))
    await screen.findByRole('heading', { level: 1, name: 'فاکتور: …' })
    await waitFor(() => expect(within(region('نمای کلی و پیوندها')).getByText('دسترسی به این بخش ندارید')).toBeTruthy())
    expect(document.body.textContent).not.toMatch(/SYN-0202|Synthetic product|zibal|2001/)
    expect(calls.map(c => c.url).filter(u => u !== '/api/v1/session' && !u.startsWith('/health'))).toEqual([`/api/v1/invoices/${INV}`])
  })
})

// RM-P02 security correction: every invoice-specific dependent read (the
// invoice-filtered Finance review queue, items, stages, payments, receipts,
// totals) is mounted and requested only after the invoice detail read
// returned SUCCESS. The review endpoint keeps its own finance.review.read
// check; nothing here grants or infers a permission.
describe('RM-P02 dependent reads wait for a successful invoice detail', () => {
  const DEPENDENT = ['/api/v1/finance/review/attempts', `/api/v1/invoices/${INV}/items`, `/api/v1/invoices/${INV}/stages`,
    `/api/v1/invoices/${INV}/payments`, `/api/v1/invoices/${INV}/receipts`, `/api/v1/invoices/${INV}/totals`]
  const dependentCalls = (calls: { url: string }[]) => calls.filter(c => DEPENDENT.includes(c.url) || c.url.includes('/payments/attempts/'))
  const settleSome = () => new Promise(r => setTimeout(r, 50))

  it('1. invoice detail UNAUTHORIZED → no Finance review (or other dependent) request', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({
      [`GET /api/v1/invoices/${INV}`]: () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-11' }, 403) }))
    await waitFor(() => expect(within(region('نمای کلی و پیوندها')).getByText('دسترسی به این بخش ندارید')).toBeTruthy())
    await settleSome()
    expect(dependentCalls(calls)).toEqual([])
    const rv = region('بررسی مالی')
    expect(within(rv).getByText('این بخش فقط پس از دریافت موفق فاکتور نمایش داده می‌شود')).toBeTruthy()
    expect(within(rv).queryByRole('table')).toBeNull()
    expect(document.body.textContent).not.toContain(ID(801))
  })

  it('2. invoice detail BLOCKED → no Finance review request', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({
      [`GET /api/v1/invoices/${INV}`]: () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'c-12' }) }))
    await waitFor(() => expect(within(region('نمای کلی و پیوندها')).getByText('این بخش هنوز فعال نشده است')).toBeTruthy())
    await settleSome()
    expect(dependentCalls(calls)).toEqual([])
    expect(within(region('بررسی مالی')).getByText('این بخش فقط پس از دریافت موفق فاکتور نمایش داده می‌شود')).toBeTruthy()
  })

  it('not found / failed invoice detail → no Finance review request', async () => {
    for (const status of [404, 500]) {
      const { calls, router } = renderAt(`/billing/invoices/${INV}`, routes({ [`GET /api/v1/invoices/${INV}`]: () => json({ type: '/problems/x',
        title: status === 404 ? 'Resource not found' : 'Internal error', status, reason_code: status === 404 ? 'NOT_FOUND' : 'INTERNAL', correlation_id: `c-${status}` },
      status, 'application/problem+json') }))
      await waitFor(() => expect(within(region('نمای کلی و پیوندها')).getByText(status === 404 ? 'Resource not found' : 'Internal error')).toBeTruthy())
      await settleSome()
      expect(dependentCalls(calls)).toEqual([])
      router.dispose()
      document.body.innerHTML = ''
    }
    expect(true).toBe(true)
  })

  it('3 + 4. while the invoice detail is loading nothing dependent is requested; after SUCCESS the review read runs', async () => {
    let release: () => void = () => undefined
    const gate = new Promise<void>(r => { release = r })
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({
      [`GET /api/v1/invoices/${INV}`]: async () => { await gate; return json(env({ data: detail() })) } }))
    await waitFor(() => expect(calls.some(c => c.url === `/api/v1/invoices/${INV}`)).toBe(true))
    await settleSome()
    expect(dependentCalls(calls)).toEqual([])
    expect(within(region('بررسی مالی')).getByText('در انتظار دریافت فاکتور…')).toBeTruthy()
    expect(within(region('اقلام (اسنپ‌شات صدور)')).getByText('در انتظار دریافت فاکتور…')).toBeTruthy()

    release()
    await open()
    const rv = region('بررسی مالی')
    expect(await within(rv).findByRole('table')).toBeTruthy()
    const order = calls.map(c => c.url)
    const reviewAt = order.indexOf('/api/v1/finance/review/attempts')
    expect(reviewAt).toBeGreaterThan(order.indexOf(`/api/v1/invoices/${INV}`))
    expect(calls[reviewAt]?.search.get('invoice_id')).toBe(INV)
    expect(await within(region('اقلام (اسنپ‌شات صدور)')).findByText(/Synthetic product A/)).toBeTruthy()
  })

  it('5. Finance review UNAUTHORIZED after a successful detail → only its own section locks', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, routes({
      'GET /api/v1/finance/review/attempts': () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-15' }, 403) }))
    await open()
    expect(await within(region('بررسی مالی')).findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(calls.filter(c => c.url === '/api/v1/finance/review/attempts')).toHaveLength(1) // asked once, the server decided
    expect(within(region('نمای کلی و پیوندها')).getByRole('link', { name: 'مشاهده مشتری' })).toBeTruthy()
    expect(await within(region('تلاش‌های پرداخت')).findByText('تلاش: پذیرفته‌شده (نه بررسی مالی، نه تسویه)')).toBeTruthy()
    expect(await within(region('اقلام (اسنپ‌شات صدور)')).findByText(/Synthetic product A/)).toBeTruthy()
    expect(document.body.textContent).not.toContain(ID(801))
  })
})
