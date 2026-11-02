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
const NOTICE = 'SHADOW: derived for architecture validation and reconciliation only. Not the financial system of record; not settled, bank-confirmed or Finance-approved money; not an official or accounting balance.'
const allowed = Object.fromEntries(['finance.shadow.read', 'finance.ledger.read', 'finance.reconciliation.read'].map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-9999', allowed_actions: allowed, ...extra })
const run = {
  record_kind: 'SHADOW', run_id: RUN, run_seq: 1, derivation_version: 1, bases: ['ACCRUAL_SHADOW', 'CASH_SHADOW', 'WALLET_MIRROR_SHADOW'],
  source_cutoff: '2026-11-01T09:00:00Z', started_at: '2026-11-01T09:00:00Z', completed_at: '2026-11-01T09:00:00Z', outcome: 'PARTIAL', reasons: ['COVERAGE_INCOMPLETE'],
  fact_counts: { ledger_entry: 3 }, facts_sha256: 'ab', freshness: 'FRESH',
  sources: [{ source: 'billing', status: 'AVAILABLE', read_at: '2026-11-01T09:00:00Z', reason: 'published', freshness: 'FRESH' },
    { source: 'legacy_wallet_transactions', status: 'PARTIAL', snapshot_id: 'syn-fin', read_at: '2026-11-01T09:00:00Z', reason: 'staging', freshness: 'FRESH' }],
}
const meta = (filters: Record<string, string> = {}) => ({ record_kind: 'SHADOW', notice: NOTICE, query: { run_id: RUN, filters }, run })
const money = (amount: string, unit = 'IRT') => ({ amount, unit })
const account = (n: number, category: string, key: string) => ({ account_id: ID(n), account_key: key, category, normal_side: 'CREDIT', subject_domain: 'legacy_wallet', subject_ref: String(n) })
const recon = (ref: string, basis: string, state: string, extra: Record<string, unknown> = {}) => ({ subject_domain: 'legacy_wallet', subject_ref: ref, basis, unit: 'IRT',
  legacy_figure: 'WALLET_BALANCE', legacy_repr: 'REPRESENTABLE', state, reasons: [], coverage: 'AVAILABLE', ...extra })
const entry = (n: number, extra: Record<string, unknown> = {}) => ({ record_kind: 'SHADOW', entry_id: ID(n), entry_seq: n, business_key: `legacy_wallet/transaction/${n}/wallet_effect/IRT/WALLET_MIRROR_SHADOW/v1`,
  basis: 'WALLET_MIRROR_SHADOW', unit: 'IRT', effect_kind: 'LEGACY_WALLET_CREDIT', provenance: 'GATEWAY_AUTO', finance_review: 'NOT_FINANCE_REVIEWED',
  source_refs: [{ domain: 'legacy.wallet_transaction', ref: String(n), role: 'subject' }], time_evidence: 'SOURCE_TIME_ZONE_UNASSERTED', source_time_raw: '2025-03-03 10:00:00',
  reconciliation_status: 'CONSISTENT', lines: [], debits: money('100'), credits: money('100'), ...extra })

function routes(over: Record<string, (req: Request) => Response> = {}, enabled = true) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: false, customers_identity: false, sales_read: false, billing_read: false, reporting_read: false,
      finance_shadow_read: enabled } })),
    'GET /api/v1/finance/shadow-runs': () => json(env({ notice: NOTICE, items: [run], page: { has_more: false } })),
    [`GET /api/v1/finance/shadow-runs/${RUN}`]: () => json(env({ outcome: 'PARTIAL', reasons: ['COVERAGE_INCOMPLETE'], notice: NOTICE, run,
      reconciliation_summary: [{ basis: 'WALLET_MIRROR_SHADOW', unit: 'IRT', state: 'MATCH', count: 2 }, { basis: 'WALLET_MIRROR_SHADOW', unit: 'IRT', state: 'MISMATCH', count: 1 },
        { basis: 'ACCRUAL_SHADOW', unit: 'IRT', state: 'UNKNOWN', count: 1 }, { basis: 'CASH_SHADOW', unit: 'IRR', state: 'MATCH', count: 1 }],
      finding_summary: [{ basis: 'WALLET_MIRROR_SHADOW', state: 'INCOMPLETE', reason: 'UNREPRESENTABLE_AMOUNT', count: 1 }] })),
    'GET /api/v1/finance/balances': (req: Request) => {
      const p = new URL(req.url).searchParams
      const page = p.get('cursor')
        ? { items: [{ account: account(3, 'customer_receivable', 'customer_receivable|invoice:x'), basis: 'CASH_SHADOW', unit: 'IRR', debits: { state: 'REPRESENTABLE', money: money('5000', 'IRR') },
          credits: { state: 'REPRESENTABLE', money: money('0', 'IRR') }, net: { state: 'REPRESENTABLE', money: money('5000', 'IRR') }, lines: 1 }], page: { has_more: false } }
        : { items: [
          { account: account(1, 'commission_payable', 'commission_payable|legacy_wallet:1'), basis: 'WALLET_MIRROR_SHADOW', unit: 'IRT', debits: { state: 'REPRESENTABLE', money: money('100') },
            credits: { state: 'REPRESENTABLE', money: money('150') }, net: { state: 'REPRESENTABLE', money: money('50') }, lines: 3,
            legacy: recon('1', 'WALLET_MIRROR_SHADOW', 'MATCH', { legacy_amount: money('50'), shadow_amount: money('50'), difference: money('0') }) },
          { account: account(2, 'commission_payable', 'commission_payable|legacy_wallet:2'), basis: 'WALLET_MIRROR_SHADOW', unit: 'IRT', debits: { state: 'REPRESENTABLE', money: money('0') },
            credits: { state: 'REPRESENTABLE', money: money('40') }, net: { state: 'REPRESENTABLE', money: money('40') }, lines: 1,
            legacy: recon('2', 'WALLET_MIRROR_SHADOW', 'INCOMPLETE', { legacy_repr: 'UNREPRESENTABLE', shadow_amount: money('40'), reasons: ['UNREPRESENTABLE_AMOUNT'] }) },
          { account: account(4, 'cash_bank_clearing', 'cash_bank_clearing|provenance:GATEWAY_AUTO'), basis: 'ACCRUAL_SHADOW', unit: 'IRT', debits: { state: 'OVERFLOW' },
            credits: { state: 'REPRESENTABLE', money: money('0') }, net: { state: 'OVERFLOW' }, lines: 2 },
        ], page: { has_more: true, next_cursor: 'b-2' } }
      return json(env({ meta: meta(), summary: [{ basis: 'WALLET_MIRROR_SHADOW', unit: 'IRT', state: 'MATCH', count: 1 }], ...page }))
    },
    'GET /api/v1/finance/reconciliation': () => json(env({ meta: meta(), summary: [{ basis: 'WALLET_MIRROR_SHADOW', unit: 'IRT', state: 'MISMATCH', count: 1 }],
      items: [recon('4', 'WALLET_MIRROR_SHADOW', 'MISMATCH', { legacy_amount: money('999'), shadow_amount: money('25'), difference: money('-974'), reasons: ['MISSING_LEGACY_TRANSACTION'] }),
        recon('9', 'ACCRUAL_SHADOW', 'UNKNOWN', { subject_domain: 'invoice', legacy_figure: 'INVOICE_REMAINING', legacy_repr: 'ABSENT', reasons: ['MISSING_SOURCE'] })],
      page: { has_more: false } })),
    'GET /api/v1/finance/ledger': () => json(env({ meta: meta(), summary: [{ basis: 'WALLET_MIRROR_SHADOW', unit: 'IRT', entries: 2, debits: { state: 'REPRESENTABLE', money: money('200') }, unknown_effective_at: 2 }],
      items: [entry(10), entry(11, { reverses_entry_id: ID(10), effect_kind: 'LEGACY_WALLET_DEBIT' })], page: { has_more: true, next_cursor: 'l-2' } })),
    [`GET /api/v1/finance/ledger/${ID(11)}`]: () => json(env({ meta: meta(), entry: entry(11, { reverses_entry_id: ID(10), lines: [
      { line_no: 1, account_id: ID(1), account_key: 'commission_payable|legacy_wallet:1', category: 'commission_payable', side: 'DEBIT', amount: money('100') },
      { line_no: 2, account_id: ID(5), account_key: 'legacy_wallet_contra|route:LEGACY_RULES', category: 'legacy_wallet_contra', side: 'CREDIT', amount: money('100') }] }) })),
    ...over,
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response | Promise<Response>>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

// No control may imply a financial mutation.
const mutationWords = /ثبت سند|تأیید|برگشت دادن|تسویه|استرداد|بستانکار کن|بدهکار کن|post|approve|reverse|settle|refund|credit|debit|edit|delete|export|csv|xlsx/i
const noMutation = () => {
  expect(screen.queryAllByRole('button').filter(b => mutationWords.test(b.textContent ?? ''))).toHaveLength(0)
  expect(screen.queryAllByRole('link').filter(a => /\/finance\/.*(post|approve|reverse|settle|refund|credit|debit)/i.test(a.getAttribute('href') ?? ''))).toHaveLength(0)
  expect(screen.queryAllByRole('textbox')).toHaveLength(0)
}

describe('Finance SHADOW ledger read (Slice 9)', () => {
  it('shows finance navigation only when the server reports the feature', async () => {
    renderAt('/', routes({}, false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    await waitFor(() => expect(screen.getByText('ورود هنوز فعال نیست')).toBeTruthy())
    expect(screen.queryByRole('link', { name: 'دفتر سایه مالی' })).toBeNull()
  })

  it('dashboard: SHADOW banner, run coverage, counts per basis and unit, no rate and no mutation control', async () => {
    const { calls } = renderAt('/finance', routes())
    expect(await screen.findByText('SHADOW · سایه')).toBeTruthy()
    expect(await screen.findByTestId('count-WALLET_MIRROR_SHADOW-IRT-MISMATCH')).toBeTruthy()
    expect(screen.getByTestId('count-WALLET_MIRROR_SHADOW-IRT-MISMATCH').textContent).toBe('1')
    expect(screen.getByTestId('count-CASH_SHADOW-IRR-MATCH').textContent).toBe('1')
    expect(screen.getByText('اجرای فنی ناقص')).toBeTruthy()
    expect(screen.getAllByText(/مبنای تعهدی \(سایه؛ نهایی نیست\)/).length).toBeGreaterThan(0)
    expect(screen.queryByText(/%|نرخ تطبیق|امتیاز/)).toBeNull()
    noMutation()
    for (const c of calls) expect(c.method).toBe('GET')
  })

  it('dashboard: no run yet is UNKNOWN, never zero', async () => {
    renderAt('/finance', routes({ 'GET /api/v1/finance/shadow-runs': () => json(env({ notice: NOTICE, items: [], page: { has_more: false } })) }))
    expect(await screen.findByText('هنوز هیچ اجرای سایه‌ای ثبت نشده است')).toBeTruthy()
  })

  it('balances: per basis and unit, legacy comparison, distinct states, overflow is not zero, paging pins the run', async () => {
    const { calls } = renderAt('/finance/balances', routes())
    const table = await screen.findByRole('table', { name: 'مانده‌های سایه' })
    expect(within(table).getAllByText('50 IRT').length).toBeGreaterThan(0)
    expect(within(table).getByText('منطبق')).toBeTruthy()
    expect(within(table).getByText('ناقص')).toBeTruthy()
    expect(within(table).getByText('UNREPRESENTABLE')).toBeTruthy()
    expect(within(table).getByText('رقم قدیمی قابل مقایسه ندارد')).toBeTruthy()
    expect(within(table).getByText('سرریز؛ مقداری ادعا نمی‌شود')).toBeTruthy()
    expect(within(table).queryByText(/جمع کل|مانده نهایی|مانده رسمی/)).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/finance/balances').at(-1)?.search.get('cursor')).toBe('b-2'))
    expect(calls.filter(c => c.url === '/api/v1/finance/balances').at(-1)?.search.get('run_id')).toBe(RUN)
    expect(await screen.findByText('5000 IRR')).toBeTruthy()
    noMutation()
  })

  it('reconciliation: legacy, shadow, difference and reasons; unknown is never zero', async () => {
    renderAt('/finance/reconciliation', routes())
    const table = await screen.findByRole('table', { name: 'تطبیق' })
    expect(within(table).getByText('-974 IRT')).toBeTruthy()
    expect(within(table).getByText('MISSING_LEGACY_TRANSACTION')).toBeTruthy()
    expect(within(table).getByText('نامعلوم (صفر نیست)')).toBeTruthy()
    expect(within(table).getByText('ABSENT')).toBeTruthy()
    expect(within(table).queryByText('0 IRT')).toBeNull()
    noMutation()
  })

  it('ledger: entries with reversal relation and provenance; detail lines from the same run; no edit control', async () => {
    const { calls } = renderAt('/finance/ledger', routes())
    const table = await screen.findByRole('table', { name: 'ثبت‌های سایه' })
    expect(within(table).getByText(`↩ ${ID(10).slice(0, 8)}`)).toBeTruthy()
    expect(within(table).getAllByText('GATEWAY_AUTO').length).toBe(2)
    expect(screen.queryByText('FINANCE_REVIEWED')).toBeNull()
    const buttons = within(table).getAllByRole('button', { name: 'سطرها' })
    fireEvent.click(buttons[1] as HTMLElement)
    expect(await screen.findByRole('list', { name: 'سطرهای ثبت' })).toBeTruthy()
    const detail = calls.filter(c => c.url === `/api/v1/finance/ledger/${ID(11)}`).at(-1)
    expect(detail?.search.get('run_id')).toBe(RUN)
    noMutation()
  })

  it('a filter change re-evaluates at the latest run (run_id dropped); unknown filters never come from the UI', async () => {
    const { calls } = renderAt(`/finance/reconciliation?run_id=${RUN}`, routes())
    await screen.findByRole('table', { name: 'تطبیق' })
    const form = screen.getByRole('form', { name: 'فیلتر تطبیق' })
    fireEvent.change(within(form).getByLabelText('basis'), { target: { value: 'CASH_SHADOW' } })
    fireEvent.submit(form)
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/finance/reconciliation').at(-1)?.search.get('basis')).toBe('CASH_SHADOW'))
    const last = calls.filter(c => c.url === '/api/v1/finance/reconciliation').at(-1)
    expect(last?.search.get('run_id')).toBeNull()
    for (const k of last?.search.keys() ?? []) expect(['basis', 'unit', 'state', 'subject_type', 'limit', 'cursor', 'run_id']).toContain(k)
  })

  it('no shadow run yet (UNKNOWN envelope) is its own state on every list page — never a transport failure, never zero', async () => {
    const unknown = () => json({ outcome: 'UNKNOWN', reasons: ['SOURCE_UNAVAILABLE'], correlation_id: 'test-corr-9999' })
    for (const [path, url] of [['/finance/balances', 'GET /api/v1/finance/balances'], ['/finance/ledger', 'GET /api/v1/finance/ledger'],
      ['/finance/reconciliation', 'GET /api/v1/finance/reconciliation']] as const) {
      const { unmount } = render(<></>)
      unmount()
      document.body.innerHTML = ''
      renderAt(path, routes({ [url]: unknown }))
      expect(await screen.findByText('هنوز هیچ اجرای سایه‌ای ثبت نشده است')).toBeTruthy()
      expect(screen.queryByText('ارتباط با سرویس برقرار نشد')).toBeNull()
      expect(screen.queryByRole('table')).toBeNull()
    }
  })

  it('blocked and denied are distinct states', async () => {
    renderAt('/finance/balances', routes({ 'GET /api/v1/finance/balances': () => json(env({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'] })) }))
    expect(await screen.findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
  })
})
