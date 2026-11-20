import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { NOT_FOUND_DETAIL } from './components/states'
import { LEADS } from './fixtures/sellerFixtures'
import { LEAD_OUTCOMES } from './labels'
import { SellerPreviewPage } from './SellerPreviewPage'
import { SIM_LABEL, type Sim } from './state/uiState'

// RM-E04J — S1 contact-outcome preview: draft → review → simulated result, over SYNTHETIC fixtures in the
// DEV/TEST-only preview. A passing test is presentation evidence only: no request is made, nothing is
// recorded, and `cases.contact_outcome.record` is not wired (Seller authority remains blocked, D-42).

function renderAt(url: string) {
  const router = createMemoryRouter([
    { path: '/dev/seller', Component: SellerPreviewPage },
    { path: '/dev/seller/:section', Component: SellerPreviewPage },
  ], { initialEntries: [url] })
  return { router, ...render(<RouterProvider router={router} />) }
}

function must<T>(v: T | null | undefined): T { if (v == null) throw new Error('missing'); return v }
const LEAD = must(LEADS.find(l => l.id === 900104))
const bodyText = () => document.body.textContent ?? ''

let fetchSpy: ReturnType<typeof vi.spyOn>
let xhrSpy: ReturnType<typeof vi.spyOn>
let beaconSpy: ReturnType<typeof vi.fn>
beforeEach(() => {
  fetchSpy = vi.spyOn(globalThis, 'fetch')
  xhrSpy = vi.spyOn(XMLHttpRequest.prototype, 'open')
  beaconSpy = vi.fn(() => true)
  Object.defineProperty(navigator, 'sendBeacon', { value: beaconSpy, configurable: true })
})
afterEach(() => {
  expect(fetchSpy).not.toHaveBeenCalled()
  expect(xhrSpy).not.toHaveBeenCalled()
  expect(beaconSpy).not.toHaveBeenCalled()
  fetchSpy.mockRestore(); xhrSpy.mockRestore()
})

async function openLead(extra = '') {
  renderAt(`/dev/seller/leads?queue=all&lead=900104${extra}`)
  return screen.findByRole('dialog', { name: extra.includes('field_denied') ? 'مشتری (محدود)' : LEAD.name })
}

/** Fill a valid «عدم خرید» draft and go to review. */
function toReview(d: HTMLElement, reason = 'قیمت بالاتر از بودجه (نمونه)') {
  fireEvent.click(within(d).getByRole('button', { name: /^عدم خرید/ }))
  fireEvent.change(within(d).getByLabelText(/دلیل عدم خرید/), { target: { value: reason } })
  fireEvent.click(within(d).getByRole('button', { name: 'ثبت عدم خرید' }))
}

function switchSim(sim: Sim) {
  fireEvent.click(screen.getByRole('button', { name: /منوی کاربر/ }))
  fireEvent.click(screen.getByRole('menuitem', { name: new RegExp(SIM_LABEL[sim].replace(/[()]/g, '\\$&')) }))
}

describe('S1 contact-outcome preview workflow', () => {
  it('the existing outcome options render unchanged, with the step indicator on «انتخاب و تکمیل»', async () => {
    const d = await openLead()
    const group = within(d).getByRole('group', { name: 'نتیجه تماس' })
    expect(within(group).getAllByRole('button').map(b => b.getAttribute('data-outcome'))).toEqual(LEAD_OUTCOMES.map(o => o.id))
    const steps = within(d).getByRole('list', { name: 'مراحل پیش‌نمایش ثبت نتیجه' })
    expect(within(steps).getAllByRole('listitem')[0]?.getAttribute('aria-current')).toBe('step')
  })

  it('invalid drafts cannot enter review; the error is announced and the field focused', async () => {
    const d = await openLead()
    fireEvent.click(within(d).getByRole('button', { name: /^عدم خرید/ }))
    fireEvent.click(within(d).getByRole('button', { name: 'ثبت عدم خرید' }))
    expect(within(d).getByRole('alert').textContent).toContain('دلیل')
    expect(document.activeElement).toBe(within(d).getByLabelText(/دلیل عدم خرید/))
    expect(within(d).queryByRole('heading', { name: 'بازبینی پیش از تأیید' })).toBeNull()
    // callback: emptied required time is blocked too
    fireEvent.click(within(d).getByRole('button', { name: /^تماس مجدد/ }))
    fireEvent.change(within(d).getByLabelText('ساعت'), { target: { value: '' } })
    fireEvent.click(within(d).getByRole('button', { name: /^ثبت تماس مجدد/ }))
    expect(within(d).queryByRole('heading', { name: 'بازبینی پیش از تأیید' })).toBeNull()
    expect(within(d).getByLabelText('ساعت').getAttribute('aria-invalid')).toBe('true')
  })

  it('valid draft → review (focused, labelled simulation, entered facts) → edit keeps data → discard clears', async () => {
    const d = await openLead()
    toReview(d)
    const h = within(d).getByRole('heading', { name: 'بازبینی پیش از تأیید' })
    expect(document.activeElement).toBe(h)
    const review = must(h.closest('section'))
    expect(review.getAttribute('aria-labelledby')).toBe(h.id)
    expect(review.textContent).toContain('شبیه‌سازی')
    expect(review.textContent).toContain('قیمت بالاتر از بودجه (نمونه)')
    expect(review.textContent).toContain('عدم خرید')
    expect(review.textContent).toContain('(بدون تغییر)')
    // outcome grid is gone in review; digit keys cannot change the outcome
    expect(within(d).queryByRole('group', { name: 'نتیجه تماس' })).toBeNull()
    fireEvent.keyDown(document.body, { key: '4' })
    expect(review.textContent).toContain('عدم خرید')
    // Edit: back to the form, nothing lost, focus on the chosen outcome
    fireEvent.click(within(d).getByRole('button', { name: 'ویرایش' }))
    const chosen = within(d).getByRole('button', { name: /^عدم خرید/ })
    expect(chosen.getAttribute('aria-pressed')).toBe('true')
    expect(document.activeElement).toBe(chosen)
    expect((within(d).getByLabelText(/دلیل عدم خرید/) as HTMLTextAreaElement).value).toBe('قیمت بالاتر از بودجه (نمونه)')
    // Discard from review: a clean draft
    fireEvent.click(within(d).getByRole('button', { name: 'ثبت عدم خرید' }))
    fireEvent.click(within(d).getByRole('button', { name: 'دورانداختن پیش‌نویس' }))
    expect(within(d).getByRole('button', { name: 'یک نتیجه انتخاب کنید' })).toBeTruthy()
    expect(within(d).queryAllByRole('button', { pressed: true }).filter(b => b.hasAttribute('data-outcome'))).toHaveLength(0)
    expect(bodyText()).not.toContain('قیمت بالاتر از بودجه (نمونه)')
  })

  it('confirm creates only a labelled simulated result (no time, no id, history unchanged); reset clears it', async () => {
    const d = await openLead()
    const historyBefore = within(d).getByRole('list', { name: 'تاریخچه تعامل' }).textContent
    const statusBefore = LEAD.status
    const pillBefore = d.querySelector('.sl-dr-title')?.textContent
    toReview(d)
    const confirm = within(d).getByRole('button', { name: 'تأیید شبیه‌سازی' })
    fireEvent.click(confirm)
    fireEvent.click(confirm) // double submit: a no-op
    // The confirm slot now holds a disabled button, so a second click cannot hit «شروع دوباره».
    expect((within(d).getByRole('button', { name: 'شبیه‌سازی انجام شد' }) as HTMLButtonElement).disabled).toBe(true)
    const h = within(d).getByRole('heading', { name: 'نتیجه شبیه‌سازی‌شده' })
    expect(document.activeElement).toBe(h)
    const result = must(h.closest('section'))
    expect(result.getAttribute('data-simulation')).toBe('result')
    expect(result.textContent).toContain('چیزی در CRM ثبت نشد')
    expect(within(result).getByRole('list', { name: 'رویداد شبیه‌سازی‌شده (ثبت نشده)' }).textContent).toContain('بدون زمان سرور')
    expect(document.querySelectorAll('[data-simulated="true"]')).toHaveLength(1)
    // The persisted (fixture) history and status are untouched; the simulated entry is not in it.
    expect(within(d).getByRole('list', { name: 'تاریخچه تعامل' }).textContent).toBe(historyBefore)
    expect(LEAD.history.some(e => e.text.includes('شبیه‌سازی'))).toBe(false)
    expect(LEAD.status).toBe(statusBefore)
    expect(d.querySelector('.sl-dr-title')?.textContent).toBe(pillBefore)
    // No message claims a real save
    expect(bodyText()).not.toMatch(/با موفقیت (ثبت|ذخیره) شد/)
    const toasts = Array.from(document.querySelectorAll('.sl-toast')).map(t => t.textContent ?? '')
    expect(toasts).toHaveLength(1)
    expect(toasts[0]).toContain('شبیه‌سازی')
    expect(toasts[0]).not.toContain(LEAD.name) // toasts never carry identity
    fireEvent.click(within(d).getByRole('button', { name: 'شروع دوباره' }))
    expect(within(d).queryByRole('heading', { name: 'نتیجه شبیه‌سازی‌شده' })).toBeNull()
    expect(document.querySelectorAll('[data-simulated="true"]')).toHaveLength(0)
    expect(within(d).getByRole('button', { name: 'یک نتیجه انتخاب کنید' })).toBeTruthy()
  })

  it('success: the product and payment type entered are what the review shows; nothing issued', async () => {
    const d = await openLead()
    fireEvent.click(within(d).getByRole('button', { name: /^موفق/ }))
    fireEvent.change(within(d).getByLabelText(/محصول/), { target: { value: 'p2' } })
    fireEvent.click(within(d).getByRole('button', { name: 'مرحله‌ای' }))
    fireEvent.click(within(d).getByRole('button', { name: 'صدور پیش‌فاکتور' }))
    const review = must(within(d).getByRole('heading', { name: 'بازبینی پیش از تأیید' }).closest('section'))
    expect(review.textContent).toContain('اشتراک نمونه نقره‌ای')
    expect(review.textContent).toContain('مرحله‌ای')
    expect(review.textContent).toContain('صادر یا پیامک نمی‌شود')
  })

  it('Escape closes the drawer and the draft does not survive reopening', async () => {
    const { router } = renderAt('/dev/seller/leads?queue=all&lead=900104')
    const d = await screen.findByRole('dialog', { name: LEAD.name })
    toReview(d)
    fireEvent.keyDown(d, { key: 'Escape' })
    expect(screen.queryByRole('dialog', { name: LEAD.name })).toBeNull()
    await act(async () => { await router.navigate('/dev/seller/leads?queue=all&lead=900104') })
    const again = await screen.findByRole('dialog', { name: LEAD.name })
    expect(within(again).queryByRole('heading', { name: 'بازبینی پیش از تأیید' })).toBeNull()
    expect(bodyText()).not.toContain('قیمت بالاتر از بودجه (نمونه)')
  })
})

describe('RM-E04J security regressions (fail closed)', () => {
  for (const sim of ['session_expired', 'action_denied', 'row_scope_denied', 'feature_disabled'] as const) {
    it(`${sim}: a review / simulated draft is dropped and cannot reappear after recovery`, async () => {
      const d = await openLead()
      toReview(d, 'محرمانه-پیش‌نویس')
      fireEvent.click(within(d).getByRole('button', { name: 'تأیید شبیه‌سازی' }))
      expect(document.querySelectorAll('.sl-toast').length).toBeGreaterThan(0)
      switchSim(sim)
      expect(screen.queryByRole('dialog')).toBeNull()
      const t = bodyText()
      expect(t).not.toContain('محرمانه-پیش‌نویس')
      expect(t).not.toContain(LEAD.name)
      expect(t).not.toMatch(/0900000\d{4}|۰۹۰۰۰۰۰[۰-۹]{4}/)
      expect(t).not.toContain('TST-')
      expect(document.querySelectorAll('.sl-toast, [data-simulated], [data-simulation]')).toHaveLength(0)
      // no hidden or accessible remnant either
      expect(document.body.innerHTML).not.toContain('محرمانه-پیش‌نویس')
      // Recovering does not resurrect the stale draft
      switchSim('none')
      const again = await screen.findByRole('dialog', { name: LEAD.name })
      expect(within(again).queryByRole('heading', { name: /بازبینی|شبیه‌سازی‌شده/ })).toBeNull()
      expect(document.body.innerHTML).not.toContain('محرمانه-پیش‌نویس')
    })
  }

  it('FIELD_DENIED: review and simulated result show «محدود» — never the name or phone', async () => {
    const d = await openLead('&sim=field_denied')
    toReview(d)
    const review = must(within(d).getByRole('heading', { name: 'بازبینی پیش از تأیید' }).closest('section'))
    expect(review.querySelector('[data-restricted="true"]')).not.toBeNull()
    fireEvent.click(within(d).getByRole('button', { name: 'تأیید شبیه‌سازی' }))
    const html = document.body.innerHTML
    expect(html).not.toContain(LEAD.name)
    expect(html).not.toContain(LEAD.phone)
    expect(bodyText()).not.toMatch(/۰۹۰۰۰۰۰[۰-۹]{4}/)
  })

  it('404: missing and out-of-scope leads both give the generic «یافت نشد» — no draft form at all', async () => {
    for (const url of ['/dev/seller/leads?queue=all&lead=1', '/dev/seller/leads?queue=all&lead=900104&sim=not_found']) {
      const { unmount } = renderAt(url)
      const d = await screen.findByRole('dialog', { name: 'یافت نشد' })
      expect(d.textContent).toContain(NOT_FOUND_DETAIL)
      expect(within(d).queryByRole('group', { name: 'نتیجه تماس' })).toBeNull()
      expect(d.querySelector('[data-draft-step]')).toBeNull()
      unmount()
    }
  })

  it('FEATURE_DISABLED from the start: no lead, no outcome grid, nothing to interact with', async () => {
    renderAt('/dev/seller/leads?queue=all&lead=900104&sim=feature_disabled')
    await screen.findByRole('heading', { level: 1, name: 'شماره‌های من' })
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.querySelector('[data-draft-step], [data-outcome]')).toBeNull()
  })
})
