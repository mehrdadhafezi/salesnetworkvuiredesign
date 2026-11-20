import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { RouterProvider, createMemoryRouter, type RouteObject } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import { configFromEnv } from '../../app/config'
import { DESTINATIONS } from '../../app/navigation/destinations'
import { buildRoutes } from '../../app/routes'
import { BEHAVIOR, CONVERSIONS, EXTRA_REQUESTS, INVOICES, LEADS, REPEAT_ITEMS, WALLET } from './fixtures/sellerFixtures'
import { SellerPreviewPage } from './SellerPreviewPage'
import {
  DEFAULT_LEAD_FILTER, NOTE_TOO_LONG, PHONE_INVALID, PHONE_REQUIRED, REQUIRED_REASON, REQUIRED_TEXT, collectibleTotal, filterLeads,
  filterWallet, followUpRate, invoiceAmounts, leadCounts, normalizeSearch, paginate, validateExtraRequest, validateOutcome, walletTotals,
} from './state/selectors'
import type { Invoice } from './types'

// RM-E03 Seller V1 preview: route isolation, synthetic-fixture policy, pure presentation logic and
// rendered behaviour. All data is synthetic; a passing test is presentation evidence only.

const SECTIONS = [
  ['leads', 'شماره‌های من'], ['conversions', 'آماده‌های تبدیل'], ['repeat', 'نیاز به اقدام مجدد'], ['invoices', 'فاکتورها'],
  ['behavior', 'رفتار مشتریان'], ['request', 'درخواست اضافه کردن شماره'], ['wallet', 'کیف پول من'],
] as const

function paths(routes: RouteObject[]): string[] {
  return routes.flatMap(r => [r.path ?? '', ...paths(r.children ?? [])])
}

function renderAt(url: string) {
  const router = createMemoryRouter([
    { path: '/dev/seller', Component: SellerPreviewPage },
    { path: '/dev/seller/:section', Component: SellerPreviewPage },
  ], { initialEntries: [url] })
  return { router, ...render(<RouterProvider router={router} />) }
}

describe('route isolation (DEV/TEST only)', () => {
  it('production config mounts no /dev/seller route; a dev build does', () => {
    const prod = paths(buildRoutes(configFromEnv({ PROD: true, DEV: false })))
    expect(prod.some(p => p.includes('dev/seller'))).toBe(false)
    expect(prod.some(p => p.startsWith('dev') || p.startsWith('/dev'))).toBe(false)
    const dev = paths(buildRoutes(configFromEnv({ PROD: false, DEV: true })))
    expect(dev).toContain('/dev/seller')
    expect(dev).toContain('/dev/seller/:section')
  })
  it('existing production URLs are unchanged and the role navigation gains no preview destination', () => {
    const prod = paths(buildRoutes(configFromEnv({ PROD: true, DEV: false })))
    for (const p of ['sales/cases', 'sales/cases/:caseId', 'billing/invoices', 'billing/invoices/:invoiceId', 'customers', 'customers/:customerId', 'commission', 'commission/me']) expect(prod).toContain(p)
    expect(DESTINATIONS.some(d => d.to.includes('/dev'))).toBe(false)
  })
})

describe('synthetic fixture policy', () => {
  it('phones use the unallocated 0900-000 range, names are visibly synthetic, invoice codes carry TST-', () => {
    const phones = [...LEADS, ...INVOICES, ...CONVERSIONS, ...REPEAT_ITEMS, ...BEHAVIOR, ...EXTRA_REQUESTS].map(x => x.phone)
    for (const p of phones) expect(p).toMatch(/^0900000\d{4}$/)
    for (const n of [...LEADS.map(l => l.name), ...INVOICES.map(i => i.customer), ...CONVERSIONS.map(c => c.customer)]) expect(n).toContain('نمونه')
    for (const c of [...INVOICES.map(i => i.code), ...WALLET.tx.map(t => t.invoice)]) expect(c).toMatch(/^TST-\d{4}$/)
  })
  it('lead → invoice links are explicit references, never phone matches', () => {
    for (const l of LEADS) for (const ref of l.invoiceRefs ?? []) expect(INVOICES.some(i => i.code === ref)).toBe(true)
  })
})

describe('presentation logic', () => {
  it('search normalises Persian digits and spacing', () => {
    expect(normalizeSearch('۰۹۰۰ ۰۰۰ ۰۱۰۴')).toBe('0900 000 0104')
    expect(filterLeads(LEADS, { ...DEFAULT_LEAD_FILTER, queue: 'all', q: '۰۹۰۰۰۰۰۰۱۰۴' }).map(l => l.id)).toEqual([900104])
  })
  it('queues, filters and sort work on fixtures', () => {
    const c = leadCounts(LEADS)
    expect(c.all).toBe(LEADS.length)
    expect(c.none + c.no_answer + c.callback + c.not_purchased + c.duplicate + c.invoiced).toBe(c.all)
    const req = filterLeads(LEADS, { ...DEFAULT_LEAD_FILTER, queue: 'all', source: 'درخواستی' })
    expect(req.length).toBeGreaterThan(0)
    expect(req.every(l => l.source === 'درخواستی')).toBe(true)
    const recent = filterLeads(LEADS, { ...DEFAULT_LEAD_FILTER, queue: 'all', range: '7' })
    expect(recent.every(l => l.assignedDays <= 7)).toBe(true)
    const oldest = filterLeads(LEADS, { ...DEFAULT_LEAD_FILTER, queue: 'all', sort: 'oldest' })
    expect(oldest[0]?.assignedDays).toBe(Math.max(...LEADS.map(l => l.assignedDays)))
    expect(followUpRate([])).toBeNull()
  })
  it('pagination clamps the page and reports the range', () => {
    const p = paginate(Array.from({ length: 25 }, (_, i) => i), 9, 10)
    expect(p).toMatchObject({ page: 3, pages: 3, from: 21, to: 25, total: 25 })
    expect(p.items).toHaveLength(5)
    expect(paginate([], 1, 10)).toMatchObject({ page: 1, pages: 1, from: 0, to: 0, total: 0 })
  })
  it('invoice money: submitted is not paid, unknown is its own payable state, cancelled has no remaining', () => {
    const byCode = (c: string) => INVOICES.find(i => i.code === c) as Invoice
    expect(invoiceAmounts(byCode('TST-1004'))).toMatchObject({ confirmed: 0, submitted: 125_000_000, remaining: 125_000_000, payable: { kind: 'awaiting_finance' } })
    expect(invoiceAmounts(byCode('TST-1007'))).toMatchObject({ confirmed: 32_000_000, unknown: 32_000_000, remaining: 32_000_000, payable: { kind: 'unknown' } })
    expect(invoiceAmounts(byCode('TST-1003')).payable).toEqual({ kind: 'amount', value: 19_000_000, stage: 2 })
    expect(invoiceAmounts(byCode('TST-1002')).payable).toEqual({ kind: 'settled' })
    expect(invoiceAmounts(byCode('TST-1008'))).toMatchObject({ remaining: null, payable: { kind: 'none' } })
    expect(collectibleTotal(INVOICES).hasUnknown).toBe(true)
  })
  it('wallet: unknown lines are excluded from both sums and counted separately — never as zero; units never mix', () => {
    const t = walletTotals(WALLET.tx)
    expect(t.unknownCount).toBe(1)
    const irt = t.byUnit.find(u => u.unit === 'IRT')
    const irr = t.byUnit.find(u => u.unit === 'IRR')
    expect(irt?.pending).toBe(600_000)
    expect(irt?.confirmed).toBe(WALLET.tx.filter(x => x.state === 'confirmed' && (x.unit ?? 'IRT') === 'IRT').reduce((s, x) => s + (x.amount ?? 0), 0))
    expect(irr).toEqual({ unit: 'IRR', confirmed: 2_400_000, pending: 0 })
    expect(filterWallet(WALLET.tx, 'online', 'all').every(x => x.channel === 'online')).toBe(true)
  })
  it('outcome and request validation mirror the prototype required fields', () => {
    const f = { date: '', hour: '', minute: '1', reason: '', amount: '' }
    expect(validateOutcome('callback', f)).toEqual({ date: REQUIRED_TEXT, hour: REQUIRED_TEXT })
    expect(validateOutcome('not_purchased', f)).toEqual({ reason: REQUIRED_REASON })
    expect(validateOutcome('customer_declined', f)).toEqual({ reason: REQUIRED_REASON })
    expect(validateOutcome('no_answer', f)).toEqual({})
    expect(validateExtraRequest({ phone: '', name: '', note: '' })).toEqual({ phone: PHONE_REQUIRED })
    expect(validateExtraRequest({ phone: '0912', name: '', note: '' })).toEqual({ phone: PHONE_INVALID })
    expect(validateExtraRequest({ phone: '۰۹۰۰۰۰۰۱۲۳۴', name: '', note: 'x'.repeat(301) })).toEqual({ note: NOTE_TOO_LONG })
  })
})

describe('rendered preview', () => {
  it('renders all seven sections without any network request', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
    for (const [path, title] of SECTIONS) {
      const { unmount } = renderAt(`/dev/seller/${path}`)
      expect(await screen.findByRole('heading', { level: 1, name: title })).toBeTruthy()
      expect(screen.getByRole('navigation', { name: 'بخش‌های پنل فروشنده' })).toBeTruthy()
      unmount()
    }
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('/dev/seller redirects to leads; leads queue tabs filter the table', async () => {
    const { router } = renderAt('/dev/seller')
    await screen.findByRole('heading', { level: 1, name: 'شماره‌های من' })
    // RM-E04K: the list arrives through the asynchronous synthetic read boundary.
    await waitFor(() => expect(document.querySelector('[data-read-phase="loading"]')).toBeNull())
    expect(router.state.location.pathname).toBe('/dev/seller/leads')
    const table = screen.getByRole('table', { name: 'فهرست شماره‌ها' })
    expect(within(table).getAllByRole('row')).toHaveLength(1 + leadCounts(LEADS).none)
    fireEvent.click(document.querySelector('[data-queue="all"]') as HTMLElement)
    expect(within(table).getAllByRole('row')).toHaveLength(1 + 10)
  })

  it('lead drawer: commit is disabled until an outcome is chosen; a required reason blocks review; confirming only simulates', async () => {
    renderAt('/dev/seller/leads?queue=all&lead=900104')
    const dialog = await screen.findByRole('dialog', { name: 'مشتری نمونه ت' })
    expect((within(dialog).getByRole('button', { name: 'یک نتیجه انتخاب کنید' }) as HTMLButtonElement).disabled).toBe(true)
    fireEvent.click(within(dialog).getByRole('button', { name: /^عدم خرید/ }))
    fireEvent.click(within(dialog).getByRole('button', { name: 'ثبت عدم خرید' }))
    expect(within(dialog).getByText(REQUIRED_REASON)).toBeTruthy()
    expect(document.querySelector('.sl-toast')).toBeNull()
    fireEvent.change(within(dialog).getByLabelText(/دلیل عدم خرید/), { target: { value: 'نمونه' } })
    fireEvent.click(within(dialog).getByRole('button', { name: 'ثبت عدم خرید' }))
    // RM-E04J: a valid draft opens the review step first; still nothing is recorded.
    expect(within(dialog).getByRole('heading', { name: 'بازبینی پیش از تأیید' })).toBeTruthy()
    expect(document.querySelector('.sl-toast')).toBeNull()
    fireEvent.click(within(dialog).getByRole('button', { name: 'تأیید شبیه‌سازی' }))
    expect(document.querySelector('.sl-toast')?.textContent).toContain('چیزی ثبت یا ارسال نشد')
  })

  it('unknown values render «—» / نامعلوم, never 0', async () => {
    renderAt('/dev/seller/invoices?inv=TST-1007')
    const dialog = await screen.findByRole('dialog', { name: 'فاکتور TST-1007' })
    expect(within(dialog).getAllByText(/نامعلوم/).length).toBeGreaterThan(0)
  })
})
