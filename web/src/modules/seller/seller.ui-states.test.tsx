import { fireEvent, render, screen, within } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { NOT_FOUND_DETAIL, NOT_FOUND_TITLE, SETTLEMENT_UNDEFINED } from './components/states'
import { BEHAVIOR, CONVERSIONS, INVOICES, LEADS, REPEAT_ITEMS } from './fixtures/sellerFixtures'
import { SellerPreviewPage } from './SellerPreviewPage'
import { SIMS } from './state/uiState'

// RM-E04C / A7 — rendered Seller UI states over SYNTHETIC fixtures (DEV/TEST preview). A passing test is
// presentation evidence only: the server decides scope; these tests prove what the screen does with an answer.

const SECTIONS = [
  ['leads', 'شماره‌های من'], ['conversions', 'آماده‌های تبدیل'], ['repeat', 'نیاز به اقدام مجدد'], ['invoices', 'فاکتورها'],
  ['behavior', 'رفتار مشتریان'], ['request', 'درخواست اضافه کردن شماره'], ['wallet', 'کیف پول من'],
] as const

function renderAt(url: string) {
  const router = createMemoryRouter([
    { path: '/dev/seller', Component: SellerPreviewPage },
    { path: '/dev/seller/:section', Component: SellerPreviewPage },
  ], { initialEntries: [url] })
  return { router, ...render(<RouterProvider router={router} />) }
}

const ALL_NAMES = [...LEADS.map(l => l.name), ...INVOICES.map(i => i.customer), ...CONVERSIONS.map(c => c.customer), ...REPEAT_ITEMS.map(r => r.customer), ...BEHAVIOR.map(b => b.name)]
const bodyText = () => document.body.textContent ?? ''
const hasSyntheticPhone = () => /0900000\d{4}|۰۹۰۰۰۰۰[۰-۹]{4}/.test(bodyText())

let fetchSpy: ReturnType<typeof vi.spyOn>
beforeEach(() => { fetchSpy = vi.spyOn(globalThis, 'fetch') })
afterEach(() => { expect(fetchSpy).not.toHaveBeenCalled(); fetchSpy.mockRestore() })

describe('every state × every screen', () => {
  it('renders one h1, an accessible state, and makes no network request', async () => {
    for (const sim of SIMS) {
      for (const [path, title] of SECTIONS) {
        const { unmount } = renderAt(`/dev/seller/${path}?sim=${sim}`)
        expect(await screen.findByRole('heading', { level: 1, name: title })).toBeTruthy()
        expect(document.querySelectorAll('h1')).toHaveLength(1)
        unmount()
      }
    }
  })
})

describe('blocking answers reveal nothing', () => {
  const BLOCKING = [
    ['action_denied', 'دسترسی به این بخش ندارید'],
    ['row_scope_denied', 'هویت کاربری شما برای این بخش تأیید نشده است'],
    ['feature_disabled', 'این بخش در حال حاضر فعال نیست'],
    ['session_expired', 'نشست شما به پایان رسیده است'],
  ] as const
  it.each(BLOCKING)('%s: generic panel, no rows, no counts, no names, no phones, no drawer, no retry', async (sim, title) => {
    for (const [path] of SECTIONS) {
      // Ask for a detail record too: it must not open (no stale or sensitive data after denial / expiry).
      const { unmount } = renderAt(`/dev/seller/${path}?sim=${sim}&lead=900104&inv=TST-1003&case=104&item=301&b=b1`)
      const panel = await screen.findByRole('heading', { level: 2, name: title })
      expect(panel.closest('[data-kind="locked"]')?.getAttribute('role')).toBe('status')
      expect(screen.queryByRole('table')).toBeNull()
      expect(screen.queryByRole('dialog')).toBeNull()
      expect(screen.queryByRole('button', { name: 'تلاش مجدد' })).toBeNull()
      expect(document.querySelector('.ds-kpi')).toBeNull()
      expect(document.querySelector('.ds-count')).toBeNull()
      for (const n of ALL_NAMES) expect(bodyText()).not.toContain(n)
      expect(hasSyntheticPhone()).toBe(false)
      expect(bodyText()).not.toMatch(/TST-\d{4}/)
      unmount()
    }
  })
  it('the command palette lists no record under a blocking answer', async () => {
    renderAt('/dev/seller/leads?sim=action_denied')
    await screen.findByRole('heading', { level: 2, name: 'دسترسی به این بخش ندارید' })
    fireEvent.keyDown(document, { key: 'k', ctrlKey: true })
    const dialog = await screen.findByRole('dialog')
    for (const n of ALL_NAMES) expect(dialog.textContent).not.toContain(n)
    expect(dialog.textContent).not.toMatch(/TST-\d{4}/)
  })
})

describe('failure', () => {
  it('shows the correlation id and a retry that only re-reads (no write, no request)', async () => {
    renderAt('/dev/seller/invoices?sim=error')
    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toContain('امکان دریافت لیست وجود ندارد')
    expect(alert.textContent).toContain('syn-corr-a7-500')
    expect(screen.queryByRole('table')).toBeNull()
    fireEvent.click(within(alert).getByRole('button', { name: 'تلاش مجدد' }))
    expect(await screen.findByRole('table', { name: 'فهرست فاکتورها' })).toBeTruthy()
  })
})

describe('exact empty versus absent count', () => {
  it('exact empty: counts are a real ۰ and the empty panel shows; never «—»', async () => {
    renderAt('/dev/seller/leads?sim=empty&queue=all')
    await screen.findByRole('heading', { level: 1, name: 'شماره‌های من' })
    const all = document.querySelector('[data-queue="all"]') as HTMLElement
    expect(all.textContent).toContain('۰')
    expect(all.querySelector('.ds-count-unknown')).toBeNull()
    expect(screen.getByText('در این صف موردی وجود ندارد')).toBeTruthy()
  })
  it('absent count: «—» with an accessible «تعداد نامعلوم», never ۰; rows still render', async () => {
    renderAt('/dev/seller/leads?sim=count_absent&queue=all')
    await screen.findByRole('heading', { level: 1, name: 'شماره‌های من' })
    const all = document.querySelector('[data-queue="all"]') as HTMLElement
    expect(all.querySelector('.ds-count-unknown')).not.toBeNull()
    expect(all.textContent).toContain('—')
    expect(all.textContent).toContain('تعداد نامعلوم')
    expect(all.textContent).not.toContain('۰')
    expect(screen.getAllByRole('row').length).toBeGreaterThan(1)
    expect(document.querySelector('.sl-tfoot')?.textContent).toContain('تعداد کل نامعلوم')
    const kpis = Array.from(document.querySelectorAll('.ds-kpi'))
    expect(kpis.some(k => k.getAttribute('data-state') === 'unknown')).toBe(true)
    expect(kpis.every(k => !k.textContent?.includes('نامعتبر'))).toBe(true)
  })
})

describe('PARTIAL', () => {
  it('invoices: banner; remaining, payable and collectible withheld; declared totals stay', async () => {
    renderAt('/dev/seller/invoices?sim=partial')
    await screen.findByRole('heading', { level: 1, name: 'فاکتورها' })
    expect(document.querySelector('[data-kind="incomplete"]')?.textContent).toContain('COVERAGE_INCOMPLETE')
    const table = screen.getByRole('table', { name: 'فهرست فاکتورها' })
    expect(table.querySelectorAll('[data-withheld="true"]').length).toBeGreaterThanOrEqual(2 * 9)
    expect(table.querySelectorAll('.sl-amount:not([data-withheld])').length).toBeGreaterThan(0) // declared totals
    expect(document.querySelector('.ds-kpi[data-state="unknown"]')?.textContent).toContain('مانده قابل وصول')
  })
  it('invoice drawer: confirmed, remaining and payable withheld — never 0', async () => {
    renderAt('/dev/seller/invoices?sim=partial&inv=TST-1003')
    const d = await screen.findByRole('dialog', { name: 'فاکتور TST-1003' })
    expect(d.querySelectorAll('.sl-sum-grid [data-withheld="true"]')).toHaveLength(3)
  })
  it('lead history is marked incomplete', async () => {
    renderAt('/dev/seller/leads?sim=partial&queue=all&lead=900104')
    const d = await screen.findByRole('dialog')
    expect(d.querySelector('[data-partial-history="true"]')).not.toBeNull()
  })
})

describe('FIELD_DENIED', () => {
  it('lists and drawers show «محدود» with no partial name or phone; palette lists no identity', async () => {
    for (const path of ['leads?queue=all', 'invoices', 'conversions', 'repeat', 'behavior', 'request']) {
      const { unmount } = renderAt(`/dev/seller/${path}${path.includes('?') ? '&' : '?'}sim=field_denied`)
      await screen.findByRole('heading', { level: 1 })
      expect(document.querySelectorAll('[data-restricted="true"]').length).toBeGreaterThan(0)
      for (const n of ALL_NAMES) expect(bodyText()).not.toContain(n)
      expect(hasSyntheticPhone()).toBe(false)
      unmount()
    }
    renderAt('/dev/seller/leads?sim=field_denied&queue=all&lead=900104')
    const d = await screen.findByRole('dialog', { name: 'مشتری (محدود)' })
    expect(d.textContent).not.toContain('کپی شماره')
    expect(within(d).getAllByText('محدود').length).toBeGreaterThan(0)
  })
  it('a search by a hidden phone finds nothing', async () => {
    renderAt('/dev/seller/leads?sim=field_denied&queue=all')
    await screen.findByRole('heading', { level: 1, name: 'شماره‌های من' })
    fireEvent.change(screen.getByRole('searchbox', { name: 'جستجو در این صف' }), { target: { value: LEADS[0]?.phone ?? '' } })
    expect(screen.getByText('نتیجه‌ای پیدا نشد')).toBeTruthy()
  })
})

describe('UNKNOWN mapping', () => {
  it('statuses and stages render «نامعلوم»; payable is unknown, never a guessed amount', async () => {
    renderAt('/dev/seller/invoices?sim=unknown&inv=TST-1003')
    const d = await screen.findByRole('dialog', { name: 'فاکتور TST-1003' })
    const stages = within(d).getByRole('list', { name: 'مراحل پرداخت' })
    expect(stages.querySelectorAll('[data-state="unknown"]')).toHaveLength(3)
    expect(stages.textContent).not.toContain('تایید مالی شده')
    expect(d.querySelector('.sl-sum-grid')?.textContent).toContain('نامعلوم')
    const table = screen.getByRole('table', { name: 'فهرست فاکتورها' })
    expect(table.textContent).not.toMatch(/پیش‌فاکتور|مرحله‌ای|تایید شده مالی|رد شده|لغو شده/)
  })
})

describe('404 parity', () => {
  const texts = (d: HTMLElement) => d.textContent
  it('a missing id and an out-of-scope answer render the identical generic «یافت نشد»', async () => {
    const pairs = [
      ['invoices?inv=TST-9999', 'invoices?inv=TST-1001&sim=not_found'],
      ['leads?lead=999999', 'leads?lead=900104&sim=not_found'],
      ['conversions?case=999', 'conversions?case=104&sim=not_found'],
      ['repeat?item=999', 'repeat?item=301&sim=not_found'],
      ['behavior?b=zz', 'behavior?b=b1&sim=not_found'],
    ]
    for (const [missing, hidden] of pairs) {
      const a = renderAt(`/dev/seller/${missing}`)
      const da = await screen.findByRole('dialog', { name: NOT_FOUND_TITLE })
      const ta = texts(da)
      a.unmount()
      const b = renderAt(`/dev/seller/${hidden}`)
      const db = await screen.findByRole('dialog', { name: NOT_FOUND_TITLE })
      expect(texts(db)).toBe(ta)
      expect(ta).toContain(NOT_FOUND_DETAIL)
      expect(ta).not.toMatch(/TST-\d{4}|900104|999999/)
      b.unmount()
    }
  })
})

describe('S7 wallet', () => {
  it('historical credit, not cash; settlement «— تعریف نشده»; units kept apart', async () => {
    renderAt('/dev/seller/wallet')
    await screen.findByRole('heading', { level: 1, name: 'کیف پول من' })
    expect(screen.getByTestId('wallet-settlement').textContent).toBe(SETTLEMENT_UNDEFINED)
    expect(screen.getByTestId('wallet-not-cash').textContent).toContain('نه وجه نقد قابل برداشت')
    const hero = document.querySelector('.sl-wcard.sl-hero') as HTMLElement
    expect(hero.querySelector('b[data-unit="IRT"]')?.textContent).toContain('تومان')
    expect(hero.querySelector('b[data-unit="IRR"]')?.textContent).toContain('ریال')
    // The only mention of withdrawal is the negation; no sum is labelled withdrawable.
    expect(bodyText().match(/قابل برداشت/g)).toHaveLength(1)
    expect(bodyText()).toContain('نه وجه نقد قابل برداشت')
  })
  it('unknown run: no sum at all; PARTIAL: sums withheld', async () => {
    const a = renderAt('/dev/seller/wallet?sim=unknown')
    await screen.findByRole('heading', { level: 1, name: 'کیف پول من' })
    expect(Array.from(document.querySelectorAll('[data-kind="unknown"]')).some(b => b.textContent?.includes('هیچ جمعی'))).toBe(true)
    expect(document.querySelector('.sl-wcard.sl-hero b[data-unit]')).toBeNull()
    a.unmount()
    renderAt('/dev/seller/wallet?sim=partial')
    await screen.findByRole('heading', { level: 1, name: 'کیف پول من' })
    expect(document.querySelectorAll('.sl-wallet-sum [data-withheld="true"]')).toHaveLength(2)
    expect(screen.getByTestId('wallet-settlement').textContent).toBe(SETTLEMENT_UNDEFINED)
  })
})

describe('S2 / S3 / S5 / S6 previews', () => {
  it('S2, S3 and S5 say the server contract is not there; S2 never shows an invented confirmed 0', async () => {
    for (const [path, notice] of [['conversions', 'قرارداد فهرست'], ['repeat', 'هیچ منبع داده‌ای'], ['behavior', 'هیچ منبع داده‌ای']] as const) {
      const { unmount } = renderAt(`/dev/seller/${path}`)
      await screen.findByRole('heading', { level: 1 })
      expect(document.querySelector('[data-kind="info"]')?.textContent).toContain(notice)
      unmount()
    }
    renderAt('/dev/seller/conversions?case=104')
    const d = await screen.findByRole('dialog')
    const confirmed = Array.from(d.querySelectorAll('.sl-sum')).find(x => x.textContent?.includes('پرداخت تایید‌شده'))
    expect(confirmed?.textContent).toContain('مبلغ نامعلوم')
    expect(confirmed?.textContent).not.toContain('۰')
  })
  it('S6: not-activated notice; a valid form submits nothing anywhere', async () => {
    renderAt('/dev/seller/request')
    await screen.findByRole('heading', { level: 1, name: 'درخواست اضافه کردن شماره' })
    expect(document.querySelector('[data-kind="locked"]')?.textContent).toContain('این گردش‌کار فعال نشده است')
    const form = document.querySelector('form[data-local-only="true"]') as HTMLFormElement
    expect(form.getAttribute('action')).toBeNull()
    fireEvent.change(form.querySelector('[name="phone"]') as HTMLInputElement, { target: { value: '09000001234' } })
    fireEvent.click(screen.getByRole('button', { name: 'بررسی فرم (نمایشی)' }))
    expect(document.querySelector('.sl-toast')?.textContent).toContain('درخواست ثبت نشد')
    expect(document.querySelectorAll('.sl-mini-list > li')).toHaveLength(3)
  })
})
