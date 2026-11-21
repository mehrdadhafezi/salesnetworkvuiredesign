import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session, type Captured } from '../../test/fakeApi'

// RM-P03 activity timeline on the Case, Invoice and Customer pages. All data
// is SYNTHETIC. Every source is authorized on its own; a denied, blocked or
// gated source contributes nothing and leaves nothing in the DOM; every
// request is a GET; no join goes through phone, name or legacy key.
const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
type Handler = (req: Request) => Response | Promise<Response>
const caps = (...c: string[]) => Object.fromEntries(c.map(x => [x, { allowed: true }]))
const env = (extra: Record<string, unknown>, allowed: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-0303', allowed_actions: allowed, ...extra })
const denied = () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'test-corr-0303' }, 403)
const q = (req: Request) => new URL(req.url).searchParams
const lane = (name: string) => screen.getByRole('region', { name })

function renderAt(path: string, r: Record<string, Handler>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls }
}

function readOnlyNoIdentityJoin(calls: Captured[]) {
  expect(calls.length).toBeGreaterThan(0)
  for (const c of calls) {
    expect(c.method).toBe('GET')
    for (const k of ['phone', 'customer_phone', 'mobile', 'name', 'legacy_key', 'source_key']) expect(c.search.has(k), `${k} in ${c.url}`).toBe(false)
  }
}

/* ------------------------------- Case ---------------------------------- */

const CASE = ID(1)
const CASE_CAPS = caps('cases.case.read', 'custody.custody.read')
const caseSrc = { source_system: 'wp', source_spec: 'legacy.lead.v1', source_key: '501', channel_raw: null, origin_ref_raw: null, first_batch_id: ID(900),
  last_batch_id: ID(900), presence: 'PRESENT', first_seen_at: '2026-10-31T09:00:00Z', last_seen_at: '2026-10-31T09:00:00Z' }
const caseRow = { case_id: CASE, workflow_type: 'LEGACY_LEAD', state_namespace: 'legacy_lead.status', state_raw: 'callback', state_mapping: 'KNOWN', customer_id: null,
  customer_link_state: 'UNKNOWN', customer_link_reason: 'no explicit customer reference', presence: 'PRESENT', source: caseSrc, alias_count: 0, history_count: 3,
  imported_at: '2026-10-31T09:00:00Z', refreshed_at: '2026-10-31T09:00:00Z' }
const hist = (n: number, from: string, to: string, occurred: string | null) => ({ history_id: ID(90 + n), source_spec: 'legacy.lead_status_history.v1', source_key: String(9000 + n),
  old_state_raw: from, new_state_raw: to, actor: { legacy_key: '12', user_id: null }, occurred_raw: occurred, recorded_at: '2026-10-31T09:00:00Z' })
const outcome = (n: number, o: string, at: string, prev: string | null) => ({ event_id: ID(60 + n), command_id: ID(65 + n), outcome: o, reason: o === 'NOT_PURCHASED' ? 'synthetic reason' : null,
  callback_at: null, previous_outcome: prev, before_version: n, after_version: n + 1, actor_user_id: ID(705), correlation_id: `corr-${n}`, occurred_at: at })

function caseRoutes(over: Record<string, Handler> = {}, opts: { allowed?: Record<string, unknown>; outcomeFeature?: boolean } = {}): Record<string, Handler> {
  const allowed = opts.allowed ?? CASE_CAPS
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { sales_read: true, contact_outcome_write: opts.outcomeFeature ?? false } })),
    [`GET /api/v1/cases/${CASE}`]: () => json(env({ data: { case: caseRow, aliases: [] } }, allowed)),
    [`GET /api/v1/cases/${CASE}/history`]: req => json(env(q(req).get('cursor') === 'h-2'
      ? { items: [hist(3, 'assigned', 'callback', 'garbled-time')], page: { has_more: false } }
      : { items: [hist(1, 'new', 'assigned', '2025-01-02 10:00:00'), hist(2, 'assigned', 'assigned', '2025-01-02 10:00:00')], page: { has_more: true, next_cursor: 'h-2' } }, allowed)),
    [`GET /api/v1/cases/${CASE}/attribution`]: () => json(env({ items: [{ snapshot_id: ID(80), facet: 'ORIGINAL_SELLER', resolution: 'RESOLVED', subject: { legacy_key: '4', user_id: ID(704) },
      position_raw: 'seller', basis: 'first holder-assigning custody log row', source_time_raw: '2025-01-01', evidence_spec: 'legacy.distribution_item_log.v1', evidence_key: '1',
      captured_batch_id: ID(900), captured_at: '2026-10-31T09:00:00Z' }] }, allowed)),
    [`GET /api/v1/custody/cases/${CASE}`]: () => json(env({ data: {
      assignment: { case_id: CASE, derivation: 'DERIVED', derived_holder: { legacy_key: '5', user_id: ID(705) }, derived_position_raw: 'seller', legacy_holder: { legacy_key: '5', user_id: ID(705) },
        legacy_position_raw: 'seller', legacy_source_spec: 'legacy.lead.v1', parity: 'MATCH', event_count: 1, finding_count: 1, presence: 'PRESENT', refreshed_at: '2026-10-31T09:00:00Z' },
      events: [{ event_id: ID(70), source_spec: 'legacy.distribution_item_log.v1', source_key: '1', legacy_action: 'synthetic_secret_custody_action', kind: 'HOLDER_ASSIGNED', context: 'MIS',
        to: { legacy_key: '5', user_id: ID(705) }, from_position_raw: null, to_position_raw: 'seller', occurred_raw: '2025-01-01 09:00:00', recorded_at: '2026-10-31T09:00:00Z' }],
      findings: [{ finding_id: ID(71), kind: 'HISTORY_GAP', detail: 'synthetic gap', event_id: null, presence: 'PRESENT', recorded_at: '2026-10-31T09:00:00Z' }] } }, allowed)),
    [`GET /api/v1/cases/${CASE}/contact-outcomes`]: () => json(env({ data: { case_id: CASE, case_version: 3, workflow_type: 'LEGACY_LEAD', rollout_mode: 'SHADOW_COMPARE',
      history: [outcome(2, 'NOT_PURCHASED', '2026-11-02T08:00:00Z', 'NO_ANSWER'), outcome(1, 'NO_ANSWER', '2026-11-01T08:00:00Z', null)], blocked_actions: [] } }, { 'cases.contact_outcome.record': { allowed: false, reason_code: 'ACTION_DENIED' } })),
    ...over,
  }
}

describe('RM-P03 case activity timeline', () => {
  it('integrates state history, custody, contact outcomes and attribution as distinguishable sources', async () => {
    const { calls } = renderAt(`/sales/cases/${CASE}`, caseRoutes({}, { outcomeFeature: true }))
    const history = await screen.findByRole('region', { name: 'سابقه وضعیت (فقط افزودنی)' })
    expect(await within(history).findByText('new → assigned')).toBeTruthy()
    expect(within(history).getAllByText('تغییر وضعیت پرونده')).toHaveLength(2)
    expect(within(history).getByText('فقط صفحه 1 بارگذاری شده است')).toBeTruthy()
    // Server pagination is kept; an unrecognised time stays raw and marked.
    fireEvent.click(within(history).getByRole('button', { name: 'صفحه بعد' }))
    expect(await within(history).findByText('garbled-time')).toBeTruthy()
    expect(within(history).getByText('قالب زمان شناخته نشد')).toBeTruthy()
    expect(calls.filter(c => c.url.endsWith('/history')).at(-1)?.search.get('cursor')).toBe('h-2')

    const custody = lane('خط زمانی نگهداری (فقط افزودنی)')
    expect(await within(custody).findByText('synthetic_secret_custody_action')).toBeTruthy()
    expect(within(custody).getByText(/یافته نگهداری ثبت شده است/)).toBeTruthy()
    const outcomes = lane('پیامدهای تماس (سامانه جدید)')
    expect(await within(outcomes).findByText('ثبت پیامد تماس: عدم خرید')).toBeTruthy()
    expect(within(outcomes).getByText('NO_ANSWER → NOT_PURCHASED')).toBeTruthy()
    expect(within(outcomes).getByText('جدیدترین ابتدا (ترتیب سرور)')).toBeTruthy()
    expect(within(lane('انتساب تاریخی (اسنپ‌شات)')).getByText('انتساب تاریخی: فروشنده اولیه')).toBeTruthy()
    // One contact-outcome request, shared by the panel and the timeline.
    expect(calls.filter(c => c.url.endsWith('/contact-outcomes'))).toHaveLength(1)
    readOnlyNoIdentityJoin(calls)
  })

  it('combined view never interleaves legacy wall-clock times with target instants and flags equal timestamps', async () => {
    renderAt(`/sales/cases/${CASE}`, caseRoutes({}, { outcomeFeature: true }))
    const t = await screen.findByRole('group', { name: 'خط زمانی فعالیت پرونده' })
    await within(lane('پیامدهای تماس (سامانه جدید)')).findByText('ثبت پیامد تماس: عدم خرید')
    await within(lane('خط زمانی نگهداری (فقط افزودنی)')).findByText('synthetic_secret_custody_action')
    fireEvent.click(within(t).getByRole('button', { name: 'ترکیب زمانی' }))
    const legacy = within(t).getByRole('region', { name: 'زمان‌های منبع قدیمی' })
    const labels = [...legacy.querySelectorAll('.tl-label')].map(n => n.textContent)
    // attribution (day 2025-01-01) → custody 09:00 → two history rows at the same second.
    expect(labels).toEqual(['انتساب تاریخی: فروشنده اولیه', 'تغییر نگهدار', 'تغییر وضعیت پرونده', 'تغییر وضعیت پرونده'])
    expect(within(legacy).getAllByText('هم‌زمان با رویداد دیگر؛ ترتیب بر اساس منبع')).toHaveLength(2)
    expect(within(legacy).getAllByText('دقت زمانی متفاوت؛ ترتیب نسبی نامطمئن').length).toBeGreaterThan(0)
    const target = within(t).getByRole('region', { name: 'زمان‌های سامانه جدید (لحظه رخداد)' })
    expect([...target.querySelectorAll('.tl-label')].map(n => n.textContent)).toEqual(['ثبت پیامد تماس: جواب نداده', 'ثبت پیامد تماس: عدم خرید'])
    expect(within(t).getByText(/این ترکیب کامل نیست/).textContent).toMatch(/سابقه وضعیت/)
  })

  it('feature off: contact outcomes are not requested and the source says so', async () => {
    const { calls } = renderAt(`/sales/cases/${CASE}`, caseRoutes())
    const outcomes = await screen.findByRole('region', { name: 'پیامدهای تماس (سامانه جدید)' })
    expect(await within(outcomes).findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    await screen.findByText('new → assigned')
    expect(calls.some(c => c.url.endsWith('/contact-outcomes'))).toBe(false)
  })

  it('custody not allowed: never requested, locked, and nothing of it reaches the DOM or the combined view', async () => {
    const { calls } = renderAt(`/sales/cases/${CASE}`, caseRoutes({}, { allowed: caps('cases.case.read'), outcomeFeature: true }))
    const custody = await screen.findByRole('region', { name: 'خط زمانی نگهداری (فقط افزودنی)' })
    expect(await within(custody).findByText('دسترسی به نگهداری ندارید')).toBeTruthy()
    await screen.findByText('new → assigned')
    fireEvent.click(screen.getByRole('button', { name: 'ترکیب زمانی' }))
    expect(screen.getByText(/این ترکیب کامل نیست/).textContent).toMatch(/خط زمانی نگهداری/)
    expect(calls.some(c => c.url.startsWith('/api/v1/custody'))).toBe(false)
    expect(document.body.textContent).not.toMatch(/synthetic_secret_custody_action|synthetic gap/)
  })

  it('a denied contact-outcome read is shown as such; its facts never appear', async () => {
    renderAt(`/sales/cases/${CASE}`, caseRoutes({ [`GET /api/v1/cases/${CASE}/contact-outcomes`]: denied }, { outcomeFeature: true }))
    const outcomes = await screen.findByRole('region', { name: 'پیامدهای تماس (سامانه جدید)' })
    expect(await within(outcomes).findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(document.body.textContent).not.toMatch(/synthetic reason|عدم خرید/)
  })
})

/* ------------------------------ Invoice -------------------------------- */

const INV = ID(2)
const CUST = ID(500)
const INV_CAPS = caps('billing.invoice.read', 'payments.attempt.read')
const isrc = (key: string, spec = 'legacy.invoice.v2') => ({ source_system: 'wp', source_spec: spec, source_key: key, first_batch_id: ID(900), last_batch_id: ID(901),
  presence: 'PRESENT', first_seen_at: '2026-11-01T09:00:00Z', last_seen_at: '2026-11-01T09:00:00Z' })
const irt = (n: string) => ({ raw: n, representation: 'REPRESENTABLE', money: { amount: n, unit: 'IRT' } })
const invoice = (over: Record<string, unknown> = {}) => ({ invoice_id: INV, code_raw: 'SYN-0303', lifecycle_state: 'UNKNOWN',
  customer: { id: CUST, state: 'RESOLVED', basis: 'explicit' }, case: { id: CASE, state: 'RESOLVED', basis: 'explicit' }, issued_by: { legacy_key: '8', user_id: null },
  items_source_complete: true, stages_source_complete: true, item_count: 0, stage_count: 0, unknown_state_count: 0, issued_at_raw: '2025-03-01 10:00:00', source: isrc('2002'), ...over })
const attempt = (n: number, over: Record<string, unknown> = {}) => ({ attempt_id: ID(300 + n), invoice_id: INV, invoice_link_state: 'RESOLVED', stage_no_raw: '1',
  channel: 'GATEWAY', pay_method_raw: 'gateway', payment_source_raw: 'zibal', amount: irt('600'), state_namespace: 'legacy_payment.status', state_raw: 'synthetic_secret_attempt_state',
  state_mapping: 'UNKNOWN', state: null, approval_provenance: 'GATEWAY_AUTO', uploaded_by_type_raw: 'seller', created_at_raw: '2025-03-02 11:00:00', updated_at_raw: null, version: 1,
  source: isrc(String(70 + n), 'legacy.payment.v1'), ...over })
const page = (items: unknown[], next?: string) => ({ items, page: next ? { has_more: true, next_cursor: next } : { has_more: false } })

function invoiceRoutes(over: Record<string, Handler> = {}, allowed: Record<string, unknown> = INV_CAPS): Record<string, Handler> {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { billing_read: true, sales_read: true, customers_identity: true } })),
    [`GET /api/v1/invoices/${INV}`]: () => json(env({ data: { invoice: invoice(), product_ref_raw: null, pay_method_raw: null, payment_plan_raw: null, current_stage_raw: null,
      updated_at_raw: '2025-03-05 12:00', states: [], declared: [] } }, allowed)),
    [`GET /api/v1/invoices/${INV}/items`]: () => json(env(page([]), allowed)),
    [`GET /api/v1/invoices/${INV}/stages`]: () => json(env(page([]), allowed)),
    [`GET /api/v1/invoices/${INV}/payments`]: () => json(env(page([attempt(1)], 'p-2'), allowed)),
    [`GET /api/v1/invoices/${INV}/receipts`]: () => json(env(page([]), allowed)),
    [`GET /api/v1/invoices/${INV}/totals`]: () => json(env({ data: { invoice_id: INV, compared_declared: [], payment_evidence_complete: true, unclassified_attempts: 0, units: [] } }, allowed)),
    [`GET /api/v1/payments/attempts/${ID(301)}/history`]: () => json(env(page([{ history_id: ID(91), old_state_raw: 'pending', new_state_raw: 'approved', regression: false,
      batch_id: ID(900), recorded_at: '2026-11-01T09:00:00Z' }]), allowed)),
    'GET /api/v1/finance/review/attempts': () => json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-r', page: { has_more: false },
      data: [{ attempt_id: ID(801), batch_id: ID(810), invoice_id: INV, stage_no: 1, amount: 250, unit: 'IRT', state: 'PENDING_REVIEW', review_provenance: 'NOT_REVIEWED',
        evidence_state: 'PENDING_FINALISATION', evidence_ready: false, reviewer_is_submitter: false, submitted_by_user_id: ID(706), submitted_at: '2026-11-03T08:00:00Z', receipts: [],
        gates: [] }] }),
    ...over,
  }
}

describe('RM-P03 invoice activity timeline', () => {
  it('shows invoice record times, legacy attempts and target submissions; attempt history opens on demand (RM-P02 reuse)', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, invoiceRoutes())
    const rec = await screen.findByRole('region', { name: 'رکورد فاکتور' })
    expect(await within(rec).findByText('صدور فاکتور (طبق منبع)')).toBeTruthy()
    expect(within(rec).getByText('2025-03-01 10:00:00')).toBeTruthy()
    expect(within(rec).getByText(/دقت: دقیقه/)).toBeTruthy()
    expect(within(rec).getByText('فیلدهای زمانی رکورد؛ سابقه تغییرات نیست')).toBeTruthy()
    const att = lane('تلاش‌های پرداخت (منبع قدیمی)')
    expect(await within(att).findByText('ثبت تلاش پرداخت (طبق منبع)')).toBeTruthy()
    expect(within(att).getByRole('link', { name: 'ردیف در بخش «تلاش‌های پرداخت»' }).getAttribute('href')).toBe('#inv-attempts')
    expect(within(att).getByText('فقط صفحه 1 بارگذاری شده است')).toBeTruthy()
    expect(calls.some(c => c.url.endsWith('/history'))).toBe(false)
    const d = within(att).getByText('جزئیات').closest('details') as HTMLDetailsElement
    d.open = true
    fireEvent(d, new Event('toggle'))
    expect(await within(att).findByText('pending → approved')).toBeTruthy()
    expect(within(att).getByText(/زمان مشاهده واردسازی/)).toBeTruthy()
    const sub = lane('پرداخت‌های ثبت‌شده در سامانه جدید')
    expect(await within(sub).findByRole('link', { name: 'پرونده بررسی مالی' })).toBeTruthy()
    // One review request, shared with the RM-P02 review section.
    expect(calls.filter(c => c.url === '/api/v1/finance/review/attempts')).toHaveLength(1)
    readOnlyNoIdentityJoin(calls)
  })

  it('without payments.attempt.read: attempts are never requested and nothing of them reaches the DOM', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, invoiceRoutes({}, caps('billing.invoice.read')))
    const att = await screen.findByRole('region', { name: 'تلاش‌های پرداخت (منبع قدیمی)' })
    expect(await within(att).findByText('دسترسی به شواهد پرداخت ندارید')).toBeTruthy()
    await screen.findByText('صدور فاکتور (طبق منبع)')
    expect(calls.some(c => c.url.includes('/payments') || c.url.includes('/receipts') || c.url.includes('/totals'))).toBe(false)
    expect(document.body.textContent).not.toMatch(/synthetic_secret_attempt_state/)
  })

  it('a denied Finance review read locks only its own source', async () => {
    renderAt(`/billing/invoices/${INV}`, invoiceRoutes({ 'GET /api/v1/finance/review/attempts': denied }))
    const sub = await screen.findByRole('region', { name: 'پرداخت‌های ثبت‌شده در سامانه جدید' })
    expect(await within(sub).findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(within(lane('رکورد فاکتور')).getByText('صدور فاکتور (طبق منبع)')).toBeTruthy()
    expect(screen.queryByRole('link', { name: 'پرونده بررسی مالی' })).toBeNull()
  })

  it('a failed invoice detail leaves every dependent source unrequested and awaiting', async () => {
    const { calls } = renderAt(`/billing/invoices/${INV}`, invoiceRoutes({ [`GET /api/v1/invoices/${INV}`]: denied }))
    const sub = await screen.findByRole('region', { name: 'پرداخت‌های ثبت‌شده در سامانه جدید' })
    expect(within(sub).getByText('این منبع فقط پس از دریافت موفق فاکتور خوانده می‌شود')).toBeTruthy()
    await waitFor(() => expect(within(lane('رکورد فاکتور')).getByText('دسترسی به این بخش ندارید')).toBeTruthy())
    expect(calls.some(c => c.url === '/api/v1/finance/review/attempts' || c.url.endsWith('/payments'))).toBe(false)
  })
})

/* ------------------------------ Customer ------------------------------- */

const CUS_CAPS = caps('customers.customer.read', 'customers.merge_case.read')
const prov = { source_system: 'wp', source_spec: 'legacy.invoice_issued_to.v1', source_key: '1001', batch_id: ID(900), imported_at: '2026-10-29T09:00:00Z',
  refreshed_at: '2026-10-29T09:00:00Z', presence: 'PRESENT' }
const customer = { customer_id: CUST, subject_kind: 'INVOICE_ISSUED_TO', resolution_state: 'UNKNOWN', resolution_reason: 'phone-only candidate', name_masked: 'Sy****',
  observed_raw: '2025-02-01 10:00:00', contact_count: 0, identifier_count: 0, evidence_count: 0, open_case_count: 0, conflict_case_count: 0, provenance: prov }
const merge = { merge_case_id: ID(701), reason: 'SHARED_PHONE', state: 'UNKNOWN', state_reason: 'phone match only', max_strength: 'WEAK', member_customer_ids: [CUST, ID(501)],
  evidence_kinds: ['PHONE_MATCH'], presence: 'PRESENT', created_at: '2026-10-29T09:00:00Z', refreshed_at: '2026-10-30T09:00:00Z' }

function customerRoutes(over: Record<string, Handler> = {}, allowed: Record<string, unknown> = CUS_CAPS): Record<string, Handler> {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { customers_identity: true, sales_read: true, billing_read: true } })),
    [`GET /api/v1/customers/${CUST}`]: () => json(env({ data: { customer, identifiers: [], merge_cases: [merge] } }, allowed)),
    [`GET /api/v1/customers/${CUST}/contact-points`]: () => json(env({ items: [], page: { has_more: false } }, allowed)),
    [`GET /api/v1/customers/${CUST}/evidence`]: () => json(env({ items: [], page: { has_more: false } }, allowed)),
    'GET /api/v1/cases': () => json(env({ items: [], page: { has_more: false } }, allowed)),
    'GET /api/v1/invoices': req => json(env(q(req).get('customer_id') === CUST
      ? page([invoice(), invoice({ invoice_id: ID(3), source: isrc('2003'), customer: { id: ID(999), state: 'RESOLVED', basis: 'x' }, code_raw: 'SYN-LEAK' })], 'i-2')
      : page([]), INV_CAPS)),
    ...over,
  }
}

describe('RM-P03 customer activity timeline (safe subset)', () => {
  it('shows the customer record, merge cases and RESOLVED invoice issuance with honest coverage; unconfirmed rows are dropped', async () => {
    const { calls } = renderAt(`/customers/${CUST}`, customerRoutes())
    const t = await screen.findByRole('group', { name: 'خط زمانی فعالیت مشتری' })
    expect(within(t).getByText(/این خط زمانی کامل مشتری نیست/)).toBeTruthy()
    expect(await within(lane('رکورد مشتری')).findByText('ثبت مشتری در منبع (زمان مشاهده خام)')).toBeTruthy()
    const mc = lane('پرونده‌های تطبیق هویت')
    expect(within(mc).getByRole('link', { name: 'پرونده تطبیق' }).getAttribute('href')).toBe(`/customers/merge-cases?case=${ID(701)}`)
    const inv = lane('صدور فاکتورهای با پیوند تأییدشده')
    expect(await within(inv).findByRole('link', { name: /SYN-0303/ })).toBeTruthy()
    expect(within(inv).queryByText(/SYN-LEAK/)).toBeNull()
    expect(within(inv).getByText(/1 ردیف بدون تأیید پیوند در خود ردیف کنار گذاشته شد/)).toBeTruthy()
    expect(within(inv).getByText('فقط صفحه 1 بارگذاری شده است')).toBeTruthy()
    expect(calls.filter(c => c.url === '/api/v1/invoices').every(c => c.search.get('customer_id') === CUST)).toBe(true)
    readOnlyNoIdentityJoin(calls)
  })

  it('merge-case events stay locked without customers.merge_case.read; a denied invoice list is shown as denied', async () => {
    renderAt(`/customers/${CUST}`, customerRoutes({ 'GET /api/v1/invoices': denied }, caps('customers.customer.read')))
    const mc = await screen.findByRole('region', { name: 'پرونده‌های تطبیق هویت' })
    expect(await within(mc).findByText('پرونده‌های تطبیق در خط زمانی نمی‌آیند (دسترسی ندارید)')).toBeTruthy()
    expect(within(mc).queryByRole('link')).toBeNull()
    expect(await within(lane('صدور فاکتورهای با پیوند تأییدشده')).findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(within(lane('صدور فاکتورهای با پیوند تأییدشده')).queryByText(/SYN-0303/)).toBeNull()
  })
})
