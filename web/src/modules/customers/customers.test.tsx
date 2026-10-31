import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

// All data is SYNTHETIC. Masked values are what the server sends; the SPA
// never receives or reconstructs a raw phone or name.
const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const prov = { source_system: 'wp', source_spec: 'legacy.invoice_issued_to.v1', source_key: '1001', batch_id: ID(900), imported_at: '2026-10-29T09:00:00Z', refreshed_at: '2026-10-29T09:00:00Z', presence: 'PRESENT' }
const allowed = Object.fromEntries(['customers.customer.read', 'customers.merge_case.read', 'customers.identity_report.read'].map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-9999', allowed_actions: allowed, ...extra })

function customer(n: number, over: Record<string, unknown> = {}) {
  return { customer_id: ID(n), subject_kind: 'INVOICE_ISSUED_TO', resolution_state: 'UNKNOWN', resolution_reason: 'phone-only candidate; phone is discovery data',
    name_masked: `Sy********${n}`, observed_raw: '2025-02-01 10:00:00', contact_count: 1, identifier_count: 0, evidence_count: 1, open_case_count: 1,
    conflict_case_count: 0, provenance: { ...prov, source_key: String(1000 + n) }, ...over }
}

const mergeCase = (n: number, over: Record<string, unknown> = {}) => ({ merge_case_id: ID(700 + n), reason: 'SHARED_PHONE', state: 'UNKNOWN', state_reason: 'phone match only',
  max_strength: 'WEAK', member_customer_ids: [ID(1), ID(2)], evidence_kinds: ['PHONE_MATCH'], presence: 'PRESENT', created_at: '2026-10-29T09:00:00Z', refreshed_at: '2026-10-29T09:00:00Z', ...over })

const report = { customers: 5, customers_present: 5, source_subjects_present: 5, customers_by_state: { RELATED: 2, UNKNOWN: 2, NOT_RECOVERABLE: 1 },
  merge_cases_by_state: { RELATED: 1, UNKNOWN: 1 }, merge_cases_by_reason: { SHARED_PHONE: 1, SHARED_ACCOUNT_REF: 1 }, phone_only_candidate_count: 1,
  overlapping_phone_conflict_count: 0, duplicate_external_id_conflict_count: 0, strong_evidence_count: 0, weak_only_customer_count: 3, unresolved_count: 3,
  phone_only_resolved_count: 0, phone_only_merge_count: 0, merge_count: 0 }

function routes(over: Record<string, (req: Request) => Response> = {}, enabled = true) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: enabled } })),
    'GET /api/v1/customers': (req: Request) => {
      const cursor = new URL(req.url).searchParams.get('cursor')
      return json(env(cursor
        ? { items: [customer(3)], page: { has_more: false } }
        : { items: [customer(1), customer(2, { resolution_state: 'CONFLICT', conflict_case_count: 1 })], page: { has_more: true, next_cursor: 'c-2' } }))
    },
    'GET /api/v1/customer-identity/merge-cases': () => json(env({ items: [mergeCase(1), mergeCase(2, { reason: 'SHARED_EXCLUSIVE_IDENTIFIER', state: 'CONFLICT', max_strength: 'MEDIUM', evidence_kinds: ['EXCLUSIVE_IDENTIFIER_SHARED'] })], page: { has_more: false } })),
    [`GET /api/v1/customer-identity/merge-cases/${ID(702)}`]: () => json(env({ data: mergeCase(2, { reason: 'SHARED_EXCLUSIVE_IDENTIFIER', state: 'CONFLICT' }) })),
    'GET /api/v1/customer-identity/resolution-report': () => json(env({ data: report })),
    [`GET /api/v1/customers/${ID(1)}`]: () => json(env({ data: { customer: customer(1, { identifier_count: 1 }), identifiers: [{ identifier_id: ID(50), scheme: 'LEGACY_WP_ACCOUNT', masked_value: '7*', derivation: 'PHONE_DERIVED', status: 'ACTIVE', provenance: prov }], merge_cases: [mergeCase(1)] } })),
    [`GET /api/v1/customers/${ID(1)}/contact-points`]: () => json(env({ items: [{ contact_point_id: ID(60), kind: 'PHONE', role: 'PRIMARY', masked_value: '09*******01', normalization: 'VALID', shared_with_customers: 1, valid_from_raw: null, valid_to_raw: null, observed_raw: '2025-02-01 10:00:00', provenance: prov }], page: { has_more: false } })),
    [`GET /api/v1/customers/${ID(1)}/evidence`]: () => json(env({ items: [{ evidence_id: ID(80), kind: 'PHONE_MATCH', strength: 'WEAK', polarity: 'DISCOVERY', origin: 'DERIVED', reference: 'phone group', observed_raw: null, subject_count: 2, source_spec: 'legacy.invoice_issued_to.v1', presence: 'PRESENT', recorded_at: '2026-10-29T09:00:00Z' }], page: { has_more: false } })),
    ...over,
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls }
}

// No merge/approve/edit/delete/reassign/winner control may appear.
const writeWords = /ادغام کن|ادغام|تأیید|ویرایش|حذف|انتقال|انتخاب اصلی|ذخیره|اعمال تطبیق|merge|approve|edit|delete/i
const noWriteControls = () => expect(screen.queryAllByRole('button').filter(b => writeWords.test(b.textContent ?? ''))).toHaveLength(0)

describe('Customer identity read (Slice 5)', () => {
  it('shows customer navigation only when the server reports the feature', async () => {
    renderAt('/', routes({}, false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    await waitFor(() => expect(screen.getByText('ورود هنوز فعال نیست')).toBeTruthy())
    expect(screen.queryByRole('link', { name: 'مشتریان' })).toBeNull()
  })

  it('lists customers through the generated client with server filters and cursor pagination', async () => {
    const { calls } = renderAt('/customers', routes())
    expect(await screen.findByRole('link', { name: 'Sy********1' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'مشتریان' })).toBeTruthy()
    expect(screen.getAllByText('تعارض').length).toBeGreaterThan(0) // conflicts stay visible
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    expect(await screen.findByRole('link', { name: 'Sy********3' })).toBeTruthy()
    expect(calls.filter(c => c.url === '/api/v1/customers').at(-1)?.search.get('cursor')).toBe('c-2')

    const form = screen.getByRole('form', { name: 'فیلتر مشتریان' })
    fireEvent.change(within(form).getByLabelText('وضعیت هویت'), { target: { value: 'CONFLICT' } })
    fireEvent.submit(form)
    await waitFor(() => {
      const last = calls.filter(c => c.url === '/api/v1/customers').at(-1)
      expect(last?.search.get('state')).toBe('CONFLICT')
      expect(last?.search.has('cursor')).toBe(false)
    })
    // No phone search exists: phone is not identity.
    expect(screen.queryByLabelText(/تلفن/)).toBeNull()
    for (const c of calls) expect(c.method).toBe('GET')
    noWriteControls()
  })

  it('renders empty, unauthorized, feature-disabled and error states honestly', async () => {
    renderAt('/customers', routes({ 'GET /api/v1/customers': () => json(env({ items: [], page: { has_more: false } })) }))
    expect(await screen.findByText('مشتری‌ای وارد نشده است')).toBeTruthy()
  })

  it('renders UNAUTHORIZED as locked, never as empty', async () => {
    renderAt('/customers', routes({ 'GET /api/v1/customers': () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'test-corr-1' }, 403) }))
    expect(await screen.findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(screen.queryByText('مشتری‌ای وارد نشده است')).toBeNull()
  })

  it('renders BLOCKED (flag off) as not active and names the flag', async () => {
    renderAt('/customers', routes({ 'GET /api/v1/customers': () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'test-corr-1' }) }))
    expect(await screen.findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    expect(screen.getByText(/customers\.identity_shadow_read/)).toBeTruthy()
  })

  it('renders a problem as an error state', async () => {
    renderAt('/customers', routes({ 'GET /api/v1/customers': () => json({ type: '/problems/bad-request', title: 'Request does not match the contract', status: 400, reason_code: 'CONTRACT_INVALID', correlation_id: 'srv-corr-0001' }, 400, 'application/problem+json') }))
    expect(await screen.findByText('Request does not match the contract')).toBeTruthy()
  })

  it('shows customer detail sections with masked contacts and a phone-derived identifier', async () => {
    renderAt(`/customers/${ID(1)}`, routes())
    expect(await screen.findByRole('heading', { level: 1, name: 'مشتری: Sy********1' })).toBeTruthy()
    for (const title of ['مشتری', 'نقاط تماس', 'شناسه‌ها', 'شواهد', 'پرونده‌های تطبیق', 'منبع']) {
      expect(screen.getByRole('region', { name: title })).toBeTruthy()
    }
    expect(await screen.findByText('09*******01')).toBeTruthy()
    expect(screen.getByText(/مشترک با 1 مشتری دیگر/)).toBeTruthy()
    expect(screen.getByText(/مشتق از تلفن؛ اثبات هویت نیست/)).toBeTruthy()
    expect(screen.getByText('تطابق تلفن')).toBeTruthy()
    noWriteControls()
  })

  it('hides merge cases on the detail when the server denies merge-case read', async () => {
    const limited = { ...allowed, 'customers.merge_case.read': { allowed: false } }
    renderAt(`/customers/${ID(1)}`, routes({ [`GET /api/v1/customers/${ID(1)}`]: () => json({ ...env({ data: { customer: customer(1), identifiers: [], merge_cases: [] } }), allowed_actions: limited }) }))
    expect(await screen.findByText('دسترسی به پرونده‌های تطبیق ندارید')).toBeTruthy()
  })

  it('shows the merge-case queue with conflicts visible and server filters, without merge controls', async () => {
    const { calls } = renderAt('/customers/merge-cases', routes())
    expect(await screen.findByRole('link', { name: 'شناسه انحصاری مشترک' })).toBeTruthy()
    expect(screen.getAllByText('تعارض').length).toBeGreaterThan(0)
    const form = screen.getByRole('form', { name: 'فیلتر پرونده‌ها' })
    fireEvent.change(within(form).getByLabelText('قوت شواهد'), { target: { value: 'WEAK' } })
    fireEvent.change(within(form).getByLabelText('دلیل'), { target: { value: 'SHARED_PHONE' } })
    fireEvent.submit(form)
    await waitFor(() => {
      const last = calls.filter(c => c.url === '/api/v1/customer-identity/merge-cases').at(-1)
      expect(last?.search.get('evidence_strength')).toBe('WEAK')
      expect(last?.search.get('reason')).toBe('SHARED_PHONE')
    })
    noWriteControls()
  })

  it('shows one merge case with members kept separate', async () => {
    renderAt(`/customers/merge-cases?case=${ID(702)}`, routes())
    const region = await screen.findByRole('region', { name: 'جزئیات پرونده' })
    expect(await within(region).findByText('مشتریان عضو (همه جدا می‌مانند)')).toBeTruthy()
    expect(within(region).getAllByRole('link')).toHaveLength(2)
    noWriteControls()
  })

  it('shows the resolution report with the phone invariants first', async () => {
    renderAt('/customers/identity-report', routes())
    const invariants = await screen.findByRole('list', { name: 'قیود' })
    expect(within(invariants).getByText(/تطبیق قطعی فقط با تلفن/)).toBeTruthy()
    expect(within(invariants).getAllByText('برقرار')).toHaveLength(3)
    expect(within(invariants).queryByText('نقض شده')).toBeNull()
  })

  it('renders a server-reported invariant violation as BLOCKED, never as zero', async () => {
    renderAt('/customers/identity-report', routes({ 'GET /api/v1/customer-identity/resolution-report': () => json({ outcome: 'BLOCKED', reasons: ['LINKAGE_CONFLICT'], correlation_id: 'test-corr-2' }) }))
    expect(await screen.findByText('سرور این خواندن را مسدود کرده است')).toBeTruthy()
    expect(screen.getByText(/LINKAGE_CONFLICT/)).toBeTruthy()
    expect(screen.queryByText('این بخش هنوز فعال نشده است')).toBeNull()
    expect(screen.queryByRole('list', { name: 'قیود' })).toBeNull()
  })
})
