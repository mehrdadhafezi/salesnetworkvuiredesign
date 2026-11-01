import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

// All data is SYNTHETIC. The contract carries no phone, address, URL, path or gateway reference.
const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const allowed = Object.fromEntries(['billing.invoice.read', 'payments.attempt.read', 'billing.reconciliation.read'].map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-7777', allowed_actions: allowed, ...extra })
const src = (key: string, spec = 'legacy.invoice.v2') => ({ source_system: 'wp', source_spec: spec, source_key: key, first_batch_id: ID(900), last_batch_id: ID(900),
  presence: 'PRESENT', first_seen_at: '2026-11-01T09:00:00Z', last_seen_at: '2026-11-01T09:00:00Z' })
const link = { id: null, state: 'UNKNOWN', basis: 'no explicit reference' }
const irt = (n: string, raw = n) => ({ raw, representation: 'REPRESENTABLE', money: { amount: n, unit: 'IRT' } })

function invoice(n: number, over: Record<string, unknown> = {}) {
  return { invoice_id: ID(n), code_raw: `SYN-${n}`, lifecycle_state: 'UNKNOWN', customer: link, case: link, items_source_complete: true, stages_source_complete: true,
    item_count: 1, stage_count: 1, unknown_state_count: 0, issued_at_raw: '2025-03-01 10:00:00', source: src(String(2000 + n)), ...over }
}

const attempt = (n: number, over: Record<string, unknown> = {}) => ({ attempt_id: ID(300 + n), invoice_id: ID(1), invoice_link_state: 'RESOLVED', stage_no_raw: '1',
  channel: 'GATEWAY', pay_method_raw: 'gateway', payment_source_raw: 'zibal', amount: irt('1000', '1000.00'), state_namespace: 'legacy_payment.status',
  state_raw: 'approved', state_mapping: 'MAPPED', state: 'ACCEPTED', approval_provenance: 'GATEWAY_AUTO', uploaded_by_type_raw: 'seller', created_at_raw: null,
  updated_at_raw: null, version: 1, source: src(String(n), 'legacy.payment.v1'),
  gateway: { provider_raw: 'zibal', has_authority_reference: true, has_gateway_reference: false, state_raw: 'approved', state_mapping: 'MAPPED', state: 'VERIFIED' }, ...over })

const sum = (state: string, amount?: string, unit = 'IRT') => ({ state, rows: amount ? 1 : 0, ...(amount ? { money: { amount, unit } } : {}) })

function routes(over: Record<string, (req: Request) => Response> = {}, enabled = true) {
  const pageOf = (req: Request, first: unknown[], second: unknown[], next: string) => {
    const cursor = new URL(req.url).searchParams.get('cursor')
    return json(env(cursor ? { items: second, page: { has_more: false } } : { items: first, page: { has_more: true, next_cursor: next } }))
  }
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: false, sales_read: false, billing_read: enabled } })),
    'GET /api/v1/invoices': (req: Request) => pageOf(req, [invoice(1), invoice(2, { unknown_state_count: 2 })], [invoice(3)], 'i-2'),
    [`GET /api/v1/invoices/${ID(1)}`]: () => json(env({ data: { invoice: invoice(1), product_ref_raw: '301', pay_method_raw: null, payment_plan_raw: 'full',
      current_stage_raw: '1', updated_at_raw: null,
      states: [
        { field: 'MAIN', namespace: 'legacy_invoice.status', raw: 'synthetic_unfamiliar_status', mapping: 'UNKNOWN' },
        { field: 'PAYMENT', namespace: 'legacy_invoice.payment_status', raw: 'awaiting_payment', mapping: 'KNOWN_VOCABULARY' },
        { field: 'LEGACY_DISPLAY', namespace: 'legacy_invoice.review_display_status', raw: 'synthetic_unfamiliar_status', mapping: 'LEGACY_DERIVED' },
      ],
      declared: [{ kind: 'FINAL_TOTAL', amount: irt('1000', '1000.00') }, { kind: 'REMAINING_AMOUNT', amount: { raw: null, representation: 'ABSENT' } },
        { kind: 'DISCOUNT_TOTAL', amount: { raw: '0.50', representation: 'UNREPRESENTABLE' } }] } })),
    [`GET /api/v1/invoices/${ID(1)}/items`]: (req: Request) => pageOf(req,
      [{ item_id: ID(11), product_ref_raw: '301', title_snapshot: 'Synthetic product A', kind_raw: 'main', qty_raw: '1', unit_price: irt('600'), line_total: irt('600'),
        is_free_raw: '0', created_at_raw: null, source: src('31', 'legacy.invoice_item.v1') }],
      [{ item_id: ID(12), product_ref_raw: '302', title_snapshot: 'Synthetic product B', kind_raw: 'main', qty_raw: '1', unit_price: irt('400'), line_total: irt('400'),
        is_free_raw: '0', created_at_raw: null, source: src('32', 'legacy.invoice_item.v1') }], 'it-2'),
    [`GET /api/v1/invoices/${ID(1)}/stages`]: () => json(env({ items: [{ stage_id: ID(41), stage_no_raw: '1', stage_type_raw: 'full', requested: irt('1000'),
      state_namespace: 'legacy_invoice_payment_stage.status', state_raw: 'synthetic_stage_status', state_mapping: 'UNKNOWN', state: null, pay_method_raw: null,
      payment_source_raw: null, approved_at_raw: null, paid_at_raw: null, created_at_raw: null, source: src('41', 'legacy.invoice_payment_stage.v1') }], page: { has_more: false } })),
    [`GET /api/v1/invoices/${ID(1)}/payments`]: (req: Request) => pageOf(req, [attempt(1)],
      [attempt(2, { channel: 'MANUAL_CARD', gateway: undefined, approval_provenance: 'NOT_APPROVED', state_raw: 'reconcile_error', state: 'RECONCILIATION_REQUIRED' })], 'p-2'),
    [`GET /api/v1/invoices/${ID(1)}/receipts`]: () => json(env({ items: [{ receipt_id: ID(51), invoice_id: ID(1), owner_kind: 'STAGE', stage_no_raw: '1', ref_index: 0,
      has_reference: true, host: 'files.example.test', extension: 'jpg', state: 'REFERENCE_ONLY', source: src('41', 'legacy.invoice_payment_stage.v1') }], page: { has_more: false } })),
    [`GET /api/v1/invoices/${ID(1)}/totals`]: () => json(env({ data: { invoice_id: ID(1), compared_declared: ['FINAL_TOTAL', 'PAYMENT_TOTAL_AMOUNT'],
      payment_evidence_complete: true, unclassified_attempts: 0, units: [
        { unit: 'IRT', declared: [{ kind: 'FINAL_TOTAL', money: { amount: '1000', unit: 'IRT' } }], item_sum: sum('COMPLETE', '1000'), stage_sum: sum('COMPLETE', '1000'),
          payment_sum: sum('COMPLETE', '1000'), class: 'MATCH' },
        { unit: 'IRR', declared: [], item_sum: sum('COMPLETE', '5000', 'IRR'), stage_sum: sum('NONE'), payment_sum: sum('NONE'), class: 'UNKNOWN' },
      ] } })),
    'GET /api/v1/payments/attempts': (req: Request) => pageOf(req, [attempt(1)],
      [attempt(3, { state_raw: 'verified', state_mapping: 'UNKNOWN', state: null, approval_provenance: 'UNKNOWN', invoice_id: null, invoice_link_state: 'UNKNOWN' })], 'a-2'),
    'GET /api/v1/billing/reconciliation-report': () => json(env({ data: {
      billing: { invoices_present: 4, items_present: 3, stages_present: 3, invoices_without_items: 1, invoices_without_stages: 1,
        invoice_state_vocabulary_by_field: {}, unknown_raw_states: { MAIN: { synthetic_unfamiliar_status: 1 } }, stage_state_mapping: {},
        unknown_stage_raw_states: { synthetic_stage_status: 1 }, customer_link_states: {}, case_link_states: {}, unrepresentable_amounts: 6,
        invoices_with_raw_pre_invoice_value: 1 },
      payments: { attempts_present: 5, attempts_by_channel: {}, attempt_state_mapping: {}, attempt_states: {}, unknown_raw_attempt_states: { verified: 1 },
        approved_raw_count: 2, reconcile_error_raw_count: 1, approval_provenance: { GATEWAY_AUTO: 1 }, gateway_attempts: 3, gateway_states: {},
        attempts_without_invoice: 1, unrepresentable_amounts: 0, attempt_state_regressions: 0, receipts_referenced: 3, receipts_unknown: 1, receipts_missing: 0 },
      f02: { code: 'F02', label: 'Open Pre-invoices', status: 'NOT_EVALUATED', count: null, definition: 'synthetic', authority: 'GATE-0', coverage: 'PARTIAL', reason: 'deferred (TAD #20)' },
      notes: ['synthetic note'] } })),
    ...over,
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls }
}

// No issue/cancel/approve/pay/upload/post/edit control anywhere.
const actionWords = /صدور|لغو فاکتور|تأیید|پرداخت کن|بارگذاری|ثبت|ویرایش|حذف|issue|cancel|approve|pay now|upload|post|refund/i
const noActions = () => expect(screen.queryAllByRole('button').filter(b => actionWords.test(b.textContent ?? ''))).toHaveLength(0)
const noSensitive = () => {
  const body = document.body.textContent ?? ''
  for (const s of ['http://', 'https://', '/uploads/', 'SYN-AUTH', 'تلفن', 'phone', 'آدرس']) expect(body).not.toContain(s)
}

describe('Billing + payments read (Slice 7)', () => {
  it('shows billing navigation only when the server reports the feature', async () => {
    renderAt('/', routes({}, false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    await waitFor(() => expect(screen.getByText('ورود هنوز فعال نیست')).toBeTruthy())
    expect(screen.queryByRole('link', { name: 'فاکتورها' })).toBeNull()
  })

  it('lists invoices with server filters and pagination beyond one page; lifecycle never claimed; GET only', async () => {
    const { calls } = renderAt('/billing/invoices', routes())
    expect(await screen.findByRole('link', { name: 'legacy.invoice.v2 #2001' })).toBeTruthy()
    expect(screen.getAllByText('تعیین نشده (TAD #20)')).toHaveLength(2)
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    expect(await screen.findByRole('link', { name: 'legacy.invoice.v2 #2003' })).toBeTruthy()
    expect(calls.filter(c => c.url === '/api/v1/invoices').at(-1)?.search.get('cursor')).toBe('i-2')
    const form = screen.getByRole('form', { name: 'فیلتر فاکتورها' })
    fireEvent.change(within(form).getByLabelText('وضعیت ناشناخته'), { target: { value: 'true' } })
    fireEvent.submit(form)
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/invoices').at(-1)?.search.get('has_unknown_status')).toBe('true'))
    expect(screen.queryByLabelText(/تلفن/)).toBeNull()
    for (const c of calls) expect(c.method).toBe('GET')
    noActions()
  })

  it('separates every state, shows raw unknowns, per-unit totals side by side and pages every list', async () => {
    const { calls } = renderAt(`/billing/invoices/${ID(1)}`, routes())
    const main = await screen.findByRole('region', { name: 'وضعیت اصلی فاکتور (خام)' })
    expect(within(main).getByText('synthetic_unfamiliar_status')).toBeTruthy()
    expect(within(main).getByText('ناشناخته (مقدار خام حفظ شد)')).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'نمایش قدیمی (مشتق، فقط شاهد)' })).getByText('مشتق قدیمی (تصمیم‌گیر نیست)')).toBeTruthy()
    expect(screen.getByText('تعیین نشده — وضعیت قطعی ادعا نمی‌شود (TAD #20)')).toBeTruthy()
    // Units are separate cards; IRT and IRR are never combined.
    const irtCard = await screen.findByRole('region', { name: 'واحد IRT (تومان)' })
    const irrCard = screen.getByRole('region', { name: 'واحد IRR (ریال)' })
    expect(within(irtCard).getByText('هم‌خوان')).toBeTruthy()
    expect(within(irrCard).getByText('نامعلوم (شاهد کافی نیست)')).toBeTruthy()
    expect(within(irrCard).getByText('5000 IRR (ریال)')).toBeTruthy()
    // Absent is not zero; unrepresentable is not rounded.
    expect(screen.getByText('ثبت نشده (صفر نیست)')).toBeTruthy()
    expect(screen.getByText('(قابل نمایش دقیق نیست؛ گرد نشد)')).toBeTruthy()
    // Stage, attempt and gateway states keep their own vocabularies.
    expect(await screen.findByText('#1 · synthetic_stage_status')).toBeTruthy()
    expect(await screen.findByText('تلاش: پذیرفته‌شده (نه بررسی مالی، نه تسویه)')).toBeTruthy()
    expect(screen.getByText(/پذیرش خودکار درگاه \(بررسی مالی نیست\)/)).toBeTruthy()
    expect(screen.getByText(/درگاه: تأییدشده توسط درگاه/)).toBeTruthy()
    expect(await screen.findByText('فقط ارجاع (فایل بررسی نشده)')).toBeTruthy()
    // Items and attempts continue beyond the page size.
    const items = screen.getByRole('navigation', { name: 'صفحه‌بندی اقلام' })
    fireEvent.click(within(items).getByRole('button', { name: 'صفحه بعد' }))
    expect(await screen.findByText(/Synthetic product B/)).toBeTruthy()
    const pays = screen.getByRole('navigation', { name: 'صفحه‌بندی تلاش‌ها' })
    fireEvent.click(within(pays).getByRole('button', { name: 'صفحه بعد' }))
    expect(await screen.findByText('تلاش: نیازمند تطبیق')).toBeTruthy()
    expect(calls.filter(c => c.url === `/api/v1/invoices/${ID(1)}/items`).at(-1)?.search.get('cursor')).toBe('it-2')
    for (const c of calls) expect(c.method).toBe('GET')
    noActions()
    noSensitive()
  })

  it('hides payment evidence when the server does not allow it (locked, never empty)', async () => {
    const limited = Object.fromEntries(Object.entries(allowed).map(([k]) => [k, { allowed: k !== 'payments.attempt.read' }]))
    const { calls } = renderAt(`/billing/invoices/${ID(1)}`, routes({
      [`GET /api/v1/invoices/${ID(1)}`]: () => json({ ...env({}), allowed_actions: limited, data: { invoice: invoice(1), product_ref_raw: null, pay_method_raw: null,
        payment_plan_raw: null, current_stage_raw: null, updated_at_raw: null, states: [], declared: [] } }),
    }))
    expect((await screen.findAllByText('دسترسی به شواهد پرداخت ندارید')).length).toBe(3)
    expect(calls.some(c => c.url.endsWith('/payments') || c.url.endsWith('/totals') || c.url.endsWith('/receipts'))).toBe(false)
  })

  it('lists payment attempts with raw state beside the mapped state and pages beyond one page', async () => {
    const { calls } = renderAt('/billing/attempts', routes())
    const table = await screen.findByRole('table', { name: 'تلاش‌های پرداخت' })
    expect(within(table).getByText('approved')).toBeTruthy()
    expect(within(table).getByText('پذیرش خودکار درگاه (بررسی مالی نیست)')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    const next = await screen.findByRole('table', { name: 'تلاش‌های پرداخت' })
    expect(await within(next).findByText('verified')).toBeTruthy()
    expect(within(next).getByText('ناشناخته (خام حفظ شد)')).toBeTruthy()
    expect(within(next).getByText('نامعلوم (حل نشد)')).toBeTruthy()
    expect(calls.filter(c => c.url === '/api/v1/payments/attempts').at(-1)?.search.get('cursor')).toBe('a-2')
    noActions()
    noSensitive()
  })

  it('shows F02 as not evaluated (null is not zero) and raw approved/reconcile_error counts', async () => {
    renderAt('/billing/report', routes())
    expect(await screen.findByText('ارزیابی نشده — مقدار تهی است، صفر نیست')).toBeTruthy()
    expect(screen.getByText('«approved» خام (بررسی مالی نیست)')).toBeTruthy()
    expect(screen.getByText('«reconcile_error» خام')).toBeTruthy()
  })

  it('renders blocked and denied honestly', async () => {
    renderAt('/billing/invoices', routes({ 'GET /api/v1/invoices': () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'c-1' }) }))
    expect(await screen.findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    expect(screen.getByText(/billing_payments\.read_shadow/)).toBeTruthy()
  })
})
