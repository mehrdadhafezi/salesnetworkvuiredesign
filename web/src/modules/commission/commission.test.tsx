import { render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

// All data is SYNTHETIC.
const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const RUN = ID(1300)
const NOTICE = 'SHADOW only: no commission was calculated, approved, posted or credited by this read; legacy remains the system of record'
const allowed = Object.fromEntries(['commission.shadow.read', 'commission.policy.read', 'commission.own.read'].map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>) => ({ outcome: 'PARTIAL', reasons: ['COVERAGE_INCOMPLETE'], correlation_id: 'test-corr-1313', notice: NOTICE,
  allowed_actions: allowed, ...extra })
const run = {
  record_kind: 'SHADOW', run_id: RUN, run_seq: 1, state: 'CALCULATED', normalization_version: 1, rounding: 'LEGACY_PHP_ROUND_HALF_AWAY_2DP',
  snapshot_id: 'syn-commission-1', source_cutoff: '2026-11-01T09:00:00Z', completed_at: '2026-11-01T09:05:00Z', outcome: 'PARTIAL',
  reasons: ['COVERAGE_INCOMPLETE'], counts: { calculation: 4 }, sources: [{ source: 'commission_dry_run_items', status: 'MISSING', rows_read: 0 }],
}
const decisions = [
  { id: 'OPD-09', state: 'REQUIRES_HUMAN_OWNER_DECISION', effect: 'approval and posting blocked' },
  { id: 'D-36', state: 'IMPLEMENTATION_BLOCKED', effect: 'component allocation unsupported' },
]
const calc = (n: number, cls: string, amount: string | null, extra: Record<string, unknown> = {}) => ({
  calc_id: ID(n), source_key: `tx:${n}`, entitlement_key: `inv:2001|stage:1|purpose:product_sale|facet:seller|ctx:PURPOSE_MATRIX#${n}`, engine: 'PURPOSE_MATRIX',
  function_id: 'purpose_matrix_stage_percent_v1', policy_key: 'purpose_matrix:product_sale:seller', policy_version: 2, event_ref: `legacy.wallet_transaction.v1:${n}`,
  input_sha256: 'ab'.repeat(32), share_facet: 'seller', recipient_state: 'SNAPSHOT', recipient_key: '5', base: '1000.00', rate_percent: '2.5000', amount,
  legacy_amount: '25.00', unit: 'IRT', eligibility: amount === null ? 'BLOCKED' : 'ELIGIBLE', coverage: amount === null ? 'PARTIAL' : 'FULL',
  blocks: amount === null ? ['ATTRIBUTION_UNKNOWN'] : [], parity_class: cls, parity_fields: [], reasons: [], ...extra,
})

function routes(enabled = true) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: false, sales_read: false, billing_read: false, reporting_read: false,
      finance_shadow_read: false, biawin_fulfillment_read: false, commission_read: enabled } })),
    'GET /api/v1/commission/overview': () => json(env({ overview: { run, decisions, posting_state: 'BLOCKED_NOT_AUTHORIZED', findings: { DUPLICATE_RISK: 2 },
      parity: { cohort: 'synthetic cohort', window_from_raw: '2025-03-01 10:00:00', window_to_raw: '2025-03-31 10:00:00', legacy_rows: 11, compared_rows: 6,
        coverage: 'PARTIAL', classes: { MATCH: 4, AMOUNT_MISMATCH: 1, NOT_COMPARABLE: 2 }, engines: { PURPOSE_MATRIX: 7 }, legacy_totals: { IRT: '200.50' },
        target_totals: { IRT: '160.00' }, legacy_amount_missing: 0, target_amount_missing: 5, reasons: {} } } })),
    'GET /api/v1/commission/historical-runs': () => json(env({ run, items: [{ source_key: '1', run_code: 'RUN-SYN-1', state: 'APPROVED', approval_raw: 'approved',
      rejected_then_approved: true, item_count: 2, posted_items: 1, failed_items: 0 }] })),
    'GET /api/v1/commission/calculations': () => json(env({ run, items: [calc(100, 'MATCH', '25.00'), calc(106, 'ATTRIBUTION_UNKNOWN', null,
      { recipient_state: 'UNKNOWN', recipient_key: null })], page: { has_more: false } })),
    'GET /api/v1/commission/findings': () => json(env({ run, items: [{ finding_id: ID(9), kind: 'DUPLICATE_RISK', severity: 'BLOCKING', source_key: 'invoice:2002',
      detail: 'live commission credits from more than one engine' }], page: { has_more: false } })),
    'GET /api/v1/commission/policies': () => json(env({ outcome: 'SUCCESS', reasons: [], items: [{ policy_key: 'purpose_matrix:product_sale:seller', version: 1,
      purpose: 'product_sale', share_facet: 'seller', function_id: 'purpose_matrix_stage_percent_v1', rate_percent: '2.5000', unit: 'IRT', evidence: 'OBSERVED_UNVERSIONED',
      approval_state: 'NOT_APPROVED', provenance: 'synthetic', source_ref: '1', first_run_id: RUN }], page: { has_more: false } })),
    'GET /api/v1/commission/me/earnings': () => json(env({ run, items: [{ source_key: '100', engine: 'PURPOSE_MATRIX', direction: 'CREDIT', type_raw: 'purpose_commission',
      amount: '25.00', unit: 'IRT', reversed: false, purpose_raw: 'product_sale', share_facet_raw: 'seller', stage_raw: '1' }], totals: { IRT: '25.00' }, page: { has_more: false } })),
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

// No control may imply calculating, approving, posting, crediting or settling.
const commandWords = /تأیید کن|رد کن|ثبت کن|واریز|تسویه کن|برگشت بده|محاسبه کن|اعمال در کیف|approve|reject|post|credit|settle|reverse|recalculate|calculate|backfill|withdraw/i
const noCommand = () => {
  expect(screen.queryAllByRole('button').filter(b => commandWords.test(b.textContent ?? ''))).toHaveLength(0)
  expect(screen.queryAllByRole('textbox')).toHaveLength(0)
}

describe('Commission controlled SHADOW read (Slice 13)', () => {
  it('shows navigation only when the server reports the feature', async () => {
    renderAt('/', routes(false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    expect(screen.queryByRole('link', { name: 'پورسانت · نمای سایه' })).toBeNull()
  })

  it('overview: posting blocked, decisions, honest coverage and totals by unit, rejected-then-approved preserved, GET only', async () => {
    const { calls } = renderAt('/commission', routes())
    expect(await screen.findByText('ثبت پورسانت: مسدود — مجاز نیست')).toBeTruthy()
    expect(screen.getByText('SHADOW · سایه')).toBeTruthy()
    expect(screen.getByText('OPD-09')).toBeTruthy()
    expect(screen.getByText('نیازمند تصمیم مالک')).toBeTruthy()
    expect(screen.getByText('پوشش مقایسه: 6 از 11 (PARTIAL)')).toBeTruthy()
    expect(screen.getByTestId('missing').textContent).toBe('0 / 5')
    expect(screen.getByText('commission_dry_run_items: منبع موجود نیست')).toBeTruthy()
    await waitFor(() => expect(screen.getByText('بله (هر دو حفظ شد)')).toBeTruthy())
    noCommand()
    for (const c of calls) expect(c.method).toBe('GET')
  })

  it('calculations: stored results only; a missing amount is never zero; unknown recipient stays unknown', async () => {
    const { calls } = renderAt('/commission/calculations', routes())
    const table = await screen.findByRole('table', { name: 'محاسبات سایه' })
    expect(within(table).getByText('مبلغ ثبت نشده (صفر نیست)')).toBeTruthy()
    expect(within(table).getByText('نامعلوم')).toBeTruthy()
    expect(within(table).getAllByText('25.00 IRT').length).toBe(3)
    expect(screen.getByText('live commission credits from more than one engine')).toBeTruthy()
    noCommand()
    for (const c of calls) expect(c.method).toBe('GET')
  })

  it('policies: observed versions, never approved', async () => {
    renderAt('/commission/policies', routes())
    expect(await screen.findByText('تأیید نشده')).toBeTruthy()
    expect(screen.getByText('نامعلوم (همیشگی فرض نمی‌شود)')).toBeTruthy()
    noCommand()
  })

  it('own earnings: own records and totals only; no withdrawal action', async () => {
    renderAt('/commission/me', routes())
    expect(await screen.findByTestId('total-IRT')).toBeTruthy()
    expect(screen.getByTestId('total-IRT').textContent).toBe('25.00')
    noCommand()
  })

  it('no run yet is UNKNOWN, never zero', async () => {
    const r = routes()
    r['GET /api/v1/commission/calculations'] = () => json({ outcome: 'UNKNOWN', reasons: ['SOURCE_UNAVAILABLE'], correlation_id: 'c', notice: NOTICE })
    renderAt('/commission/calculations', r)
    expect(await screen.findByText('هنوز هیچ اجرای سایه پورسانتی ثبت نشده است')).toBeTruthy()
  })
})
