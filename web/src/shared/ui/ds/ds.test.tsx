import { act, fireEvent, render, screen, within } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import {
  Banner, ContextHint, CountBadge, Drawer, DsAppShell, AppHeader, Glossary, Menu, MenuButton, Metric, MobileBottomNavigation,
  PageHead, Popover, QueueTabs, RoleNavigation, StatePanel, StatusIndicator, computeNavOverflow, countLabel, countText,
  dataStatePresentation, isKnownCount, toFaDigits, type DataState, type DsNavItem, type MenuEntry, type PanelStateKind,
} from '.'

// RM-E01 shared design-system components: behaviour, RTL, keyboard, focus and truthful states.
// All data is synthetic.

const NAV: DsNavItem[] = [
  { id: 'a', label: 'الف', href: '#a', icon: 'users', count: 3, alert: true, mobileLabel: 'الف' },
  { id: 'b', label: 'ب', href: '#b', icon: 'swap', count: 0, mobileLabel: 'ب' },
  { id: 'c', label: 'پ', href: '#c', icon: 'receipt', count: 'unknown', mobileLabel: 'پ' },
  { id: 'd', label: 'ت', href: '#d', icon: 'activity', group: 'g2' },
  { id: 'e', label: 'ث', href: '#e', icon: 'wallet', group: 'g2', mobileLabel: 'ث' },
  { id: 'f', label: 'ج', href: '#f', icon: 'plus', group: 'g2', mobileLabel: 'ج' },
]

describe('formatting: unknown is never zero', () => {
  it('only finite non-negative numbers are known counts', () => {
    expect(isKnownCount(0)).toBe(true)
    expect(isKnownCount(12)).toBe(true)
    for (const v of ['unknown', -1, Number.NaN, Number.POSITIVE_INFINITY, null, undefined] as const) expect(isKnownCount(v as never)).toBe(false)
  })
  it('renders Persian digits for known counts and «—» / «تعداد نامعلوم» for unknown', () => {
    expect(countText(0)).toBe('۰')
    expect(countText(1234)).toBe('۱٬۲۳۴')
    expect(countText('unknown')).toBe('—')
    expect(countText(Number.NaN)).toBe('—')
    expect(countLabel('unknown')).toBe('تعداد نامعلوم')
    expect(toFaDigits('A-104')).toBe('A-۱۰۴')
  })
  it('CountBadge exposes unknown accessibly and never as 0', () => {
    render(<CountBadge count="unknown" />)
    expect(screen.getByText('—')).toBeTruthy()
    expect(screen.getByText('تعداد نامعلوم')).toBeTruthy()
    expect(screen.queryByText('۰')).toBeNull()
  })
})

describe('computeNavOverflow (priority+)', () => {
  const base = { itemWidths: [100, 100, 100, 100], separatorWidths: [0, 0, 17, 0], moreWidth: 80, gap: 2 }
  it('hides nothing when everything fits', () => {
    expect(computeNavOverflow({ ...base, available: 500, currentIndex: 0 })).toEqual([])
  })
  it('moves trailing items into «بیشتر» first', () => {
    expect(computeNavOverflow({ ...base, available: 300, currentIndex: 0 })).toEqual([2, 3])
  })
  it('never hides the current item', () => {
    const hidden = computeNavOverflow({ ...base, available: 300, currentIndex: 3 })
    expect(hidden).not.toContain(3)
    expect(hidden).toEqual([1, 2])
  })
})

describe('RoleNavigation', () => {
  it('is a labelled nav with one current link, group separators and truthful counts', () => {
    render(<RoleNavigation items={NAV} currentId="c" label="بخش‌های نمونه" />)
    const nav = screen.getByRole('navigation', { name: 'بخش‌های نمونه' })
    const links = within(nav).getAllByRole('link')
    expect(links).toHaveLength(6)
    expect(nav.querySelectorAll('[aria-current="page"]')).toHaveLength(1)
    expect(within(nav).getByRole('link', { name: /پ/ }).getAttribute('aria-current')).toBe('page')
    // unknown → «—» with an accessible "unknown" name; known 0 → no badge; alert tone is a class, not the only signal
    expect(within(nav).getByRole('link', { name: /پ، تعداد نامعلوم/ }).textContent).toContain('—')
    expect(within(nav).getByRole('link', { name: 'ب' }).textContent).not.toContain('۰')
    expect(nav.querySelector('.ds-count-alert')?.textContent).toContain('۳')
    expect(nav.querySelectorAll('[data-sep]')).toHaveLength(1)
  })
  it('reports navigation intent to the caller and grants nothing itself', () => {
    const onNavigate = vi.fn((_item: DsNavItem, e?: { preventDefault: () => void }) => e?.preventDefault())
    render(<RoleNavigation items={NAV} currentId="a" label="ن" onNavigate={onNavigate} />)
    fireEvent.click(screen.getByRole('link', { name: /^ت$/ }))
    expect(onNavigate).toHaveBeenCalledTimes(1)
    expect(onNavigate.mock.calls[0]?.[0].id).toBe('d')
  })
  it('accepts a custom link renderer (e.g. router links in RM-E02)', () => {
    render(<RoleNavigation items={NAV} currentId="a" label="ن" renderLink={(item, p) => <a href={`/x${item.href.slice(1)}`} {...p} data-custom="1" />} />)
    expect(screen.getByRole('link', { name: /^ت$/ }).getAttribute('href')).toBe('/xd')
    expect(document.querySelectorAll('[data-custom]')).toHaveLength(6)
  })
})

describe('MobileBottomNavigation', () => {
  it('shows at most four primary items and puts the rest under «بیشتر»', () => {
    render(<MobileBottomNavigation items={NAV} currentId="d" />)
    const nav = screen.getByRole('navigation', { name: 'ناوبری موبایل' })
    expect(within(nav).getAllByRole('link')).toHaveLength(4)
    const more = within(nav).getByRole('button', { name: 'بیشتر' })
    expect(more.getAttribute('aria-current')).toBe('page') // current item lives in the overflow
    expect(more.getAttribute('aria-expanded')).toBe('false')
    fireEvent.click(more)
    const menu = screen.getByRole('menu', { name: 'بیشتر' })
    expect(within(menu).getAllByRole('menuitem').map(m => m.textContent)).toEqual(['ت', 'ج'])
    expect(within(menu).getByRole('menuitem', { name: 'ت' }).getAttribute('aria-current')).toBe('page')
    expect(document.activeElement).toBe(within(menu).getAllByRole('menuitem')[0])
    fireEvent.keyDown(menu, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
    expect(document.activeElement).toBe(more)
  })
  it('selecting an overflow item calls onNavigate', () => {
    const onNavigate = vi.fn()
    render(<MobileBottomNavigation items={NAV} currentId="a" onNavigate={onNavigate} />)
    fireEvent.click(screen.getByRole('button', { name: 'بیشتر' }))
    fireEvent.click(screen.getByRole('menuitem', { name: 'ج' }))
    expect(onNavigate.mock.calls[0]?.[0].id).toBe('f')
    expect(screen.queryByRole('menu')).toBeNull()
  })
})

describe('QueueTabs', () => {
  const queues = [
    { id: 'q1', label: 'یک', count: 2, tone: 'amber' as const, shortcut: '1' },
    { id: 'q2', label: 'دو', count: 0 },
    { id: 'q3', label: 'سه', count: 'unknown' as const },
  ]
  function Harness({ dir }: { dir: 'rtl' | 'ltr' }) {
    const [v, setV] = useState('q1')
    return <div dir={dir}><QueueTabs queues={queues} value={v} onChange={setV} /></div>
  }
  it('is a labelled group of toggle buttons; a known 0 is shown, unknown is «—»', () => {
    render(<Harness dir="rtl" />)
    const group = screen.getByRole('group', { name: 'صف‌ها' })
    const [q1, q2, q3] = within(group).getAllByRole('button')
    expect(q1?.getAttribute('aria-pressed')).toBe('true')
    expect(q1?.getAttribute('aria-keyshortcuts')).toBe('1')
    expect(q2?.textContent).toContain('۰')
    expect(q3?.textContent).toContain('—')
    expect(q3?.textContent).toContain('تعداد نامعلوم')
    fireEvent.click(q3 as HTMLElement)
    expect(q3?.getAttribute('aria-pressed')).toBe('true')
    expect(q1?.getAttribute('aria-pressed')).toBe('false')
  })
  it('arrow keys follow the reading direction (RTL: ← is next)', () => {
    render(<Harness dir="rtl" />)
    const [q1, q2, q3] = screen.getAllByRole('button')
    q1?.focus()
    fireEvent.keyDown(q1 as HTMLElement, { key: 'ArrowLeft' })
    expect(document.activeElement).toBe(q2)
    fireEvent.keyDown(q2 as HTMLElement, { key: 'ArrowRight' })
    expect(document.activeElement).toBe(q1)
    fireEvent.keyDown(q1 as HTMLElement, { key: 'ArrowRight' }) // wraps
    expect(document.activeElement).toBe(q3)
    fireEvent.keyDown(q3 as HTMLElement, { key: 'Home' })
    expect(document.activeElement).toBe(q1)
    // focus moves; selection does not change without Enter/Space/click
    expect(q1?.getAttribute('aria-pressed')).toBe('true')
  })
  it('LTR: → is next', () => {
    render(<Harness dir="ltr" />)
    const [q1, q2] = screen.getAllByRole('button')
    q1?.focus()
    fireEvent.keyDown(q1 as HTMLElement, { key: 'ArrowRight' })
    expect(document.activeElement).toBe(q2)
  })
})

describe('Drawer', () => {
  function Harness({ onClose }: { onClose?: () => void }) {
    const [open, setOpen] = useState(false)
    return (
      <>
        <button type="button" onClick={() => setOpen(true)}>باز کن</button>
        <Drawer open={open} onClose={() => { onClose?.(); setOpen(false) }} title="جزئیات نمونه" footer={<button type="button">اقدام</button>}>
          <p>بدنه</p>
        </Drawer>
      </>
    )
  }
  it('is hidden and inert while closed', () => {
    render(<Harness />)
    const dialog = screen.getByRole('dialog', { hidden: true })
    expect(dialog.getAttribute('aria-hidden')).toBe('true')
    expect(dialog.hasAttribute('inert')).toBe(true)
    expect(screen.queryByRole('dialog')).toBeNull()
  })
  it('opens as a labelled non-modal dialog, takes focus, Esc closes and restores focus', () => {
    const onClose = vi.fn()
    render(<Harness onClose={onClose} />)
    const opener = screen.getByRole('button', { name: 'باز کن' })
    opener.focus()
    fireEvent.click(opener)
    const dialog = screen.getByRole('dialog', { name: 'جزئیات نمونه' })
    expect(dialog.getAttribute('aria-modal')).toBe('false')
    expect(dialog.hasAttribute('inert')).toBe(false)
    expect(document.activeElement).toBe(dialog)
    expect(within(dialog).getByRole('heading', { level: 2, name: 'جزئیات نمونه' })).toBeTruthy()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.activeElement).toBe(opener)
  })
  it('closes from the close button and the scrim', () => {
    const onClose = vi.fn()
    const { container } = render(<Harness onClose={onClose} />)
    fireEvent.click(screen.getByRole('button', { name: 'باز کن' }))
    fireEvent.click(screen.getByRole('button', { name: 'بستن' }))
    expect(onClose).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: 'باز کن' }))
    fireEvent.click(container.querySelector('.ds-scrim') as HTMLElement)
    expect(onClose).toHaveBeenCalledTimes(2)
  })
})

describe('Menu / MenuButton / Popover', () => {
  const onSelect = vi.fn()
  const entries: MenuEntry[] = [
    { kind: 'heading', id: 'h', label: 'گروه' },
    { kind: 'item', id: 'one', label: 'یک', icon: 'file', onSelect },
    { kind: 'separator', id: 's' },
    { kind: 'item', id: 'two', label: 'دو', disabled: true, disabledReason: 'مجوز ندارید', onSelect },
    { kind: 'item', id: 'three', label: 'سه', danger: true, onSelect },
  ]
  it('opens on click with focus on the first item; ↑/↓/Home/End rove; Esc returns focus', () => {
    onSelect.mockClear()
    render(<MenuButton label="اقدام‌ها" entries={entries}>اقدام‌ها</MenuButton>)
    const trigger = screen.getByRole('button', { name: 'اقدام‌ها' })
    expect(trigger.getAttribute('aria-haspopup')).toBe('menu')
    fireEvent.click(trigger)
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    const menu = screen.getByRole('menu', { name: 'اقدام‌ها' })
    expect(trigger.getAttribute('aria-controls')).toBe(menu.id)
    const items = within(menu).getAllByRole('menuitem')
    expect(items).toHaveLength(3)
    expect(document.activeElement).toBe(items[0])
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[1])
    fireEvent.keyDown(menu, { key: 'End' })
    expect(document.activeElement).toBe(items[2])
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(document.activeElement).toBe(items[0])
    fireEvent.keyDown(menu, { key: 'ArrowUp' })
    expect(document.activeElement).toBe(items[2])
    fireEvent.keyDown(menu, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
    expect(document.activeElement).toBe(trigger)
    expect(onSelect).not.toHaveBeenCalled()
  })
  it('a disabled item stays visible with its reason and never fires', () => {
    onSelect.mockClear()
    render(<MenuButton label="اقدام‌ها" entries={entries}>اقدام‌ها</MenuButton>)
    fireEvent.click(screen.getByRole('button', { name: 'اقدام‌ها' }))
    const two = screen.getByRole('menuitem', { name: /دو/ })
    expect(two.getAttribute('aria-disabled')).toBe('true')
    const reason = document.getElementById(two.getAttribute('aria-describedby') ?? '')
    expect(reason?.textContent).toBe('مجوز ندارید')
    fireEvent.click(two)
    expect(onSelect).not.toHaveBeenCalled()
    expect(screen.getByRole('menu')).toBeTruthy()
    fireEvent.click(screen.getByRole('menuitem', { name: 'یک' }))
    expect(onSelect).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('menu')).toBeNull()
  })
  it('ArrowUp on the trigger opens on the last item; outside pointerdown closes', () => {
    render(<><MenuButton label="اقدام‌ها" entries={entries}>اقدام‌ها</MenuButton><p>بیرون</p></>)
    fireEvent.keyDown(screen.getByRole('button', { name: 'اقدام‌ها' }), { key: 'ArrowUp' })
    expect(document.activeElement).toBe(screen.getByRole('menuitem', { name: 'سه' }))
    fireEvent.pointerDown(screen.getByText('بیرون'))
    expect(screen.queryByRole('menu')).toBeNull()
  })
  it('Menu renders nothing while closed', () => {
    const ref = { current: null }
    render(<Menu open={false} onClose={() => undefined} anchorRef={ref} label="م" entries={entries} />)
    expect(screen.queryByRole('menu')).toBeNull()
  })
  it('Popover is a labelled non-modal dialog; Esc closes and returns focus', () => {
    function P() {
      const [open, setOpen] = useState(false)
      const [btn, setBtn] = useState<HTMLButtonElement | null>(null)
      return (
        <>
          <button ref={setBtn} type="button" onClick={() => setOpen(true)}>توضیح</button>
          <Popover open={open} onClose={() => setOpen(false)} anchorRef={{ current: btn }} label="توضیح نمونه"><p>متن</p></Popover>
        </>
      )
    }
    render(<P />)
    const btn = screen.getByRole('button', { name: 'توضیح' })
    btn.focus()
    fireEvent.click(btn)
    const dlg = screen.getByRole('dialog', { name: 'توضیح نمونه' })
    expect(dlg.getAttribute('aria-modal')).toBe('false')
    expect(document.activeElement).toBe(dlg)
    fireEvent.keyDown(dlg, { key: 'Escape' })
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.activeElement).toBe(btn)
  })
})

describe('status and state indicators', () => {
  it('every data state has a distinct label, an icon and a tone class (never colour alone)', () => {
    const states = Object.keys(dataStatePresentation) as DataState[]
    expect(states.sort()).toEqual(['conflict', 'incomplete', 'not_evaluated', 'partial', 'stale', 'unknown'])
    const { container } = render(<>{states.map(s => <StatusIndicator key={s} state={s} />)}</>)
    const pills = Array.from(container.querySelectorAll('.pill'))
    expect(pills).toHaveLength(states.length)
    const labels = pills.map(p => p.textContent)
    expect(new Set(labels).size).toBe(states.length)
    for (const p of pills) {
      expect(p.querySelector('svg')).toBeTruthy()
      expect(p.className).toMatch(/t-(amber|orange|slate)/)
      expect(p.textContent).not.toMatch(/۰|موفق/)
    }
    expect(container.querySelector('[data-state="unknown"]')?.className).toContain('ds-dashed')
    expect(container.querySelector('[data-state="not_evaluated"]')?.textContent).toBe('ارزیابی نشده')
  })
  it('a caller may refine the label but not the kind', () => {
    const { container } = render(<StatusIndicator state="conflict" label="تعارض مبلغ" detail="۲ منبع" />)
    const pill = container.querySelector('[data-state="conflict"]')
    expect(pill?.className).toContain('t-orange')
    expect(pill?.textContent).toBe('تعارض مبلغ· ۲ منبع')
  })
  it('StatePanel: error/conflict alert, others status; loading is busy with a text alternative', () => {
    const kinds: PanelStateKind[] = ['empty', 'noresult', 'error', 'locked', 'stale', 'incomplete', 'conflict', 'partial', 'retry', 'unknown', 'offline', 'info']
    for (const k of kinds) {
      const { container, unmount } = render(<StatePanel kind={k} title={`عنوان ${k}`} detail="توضیح" />)
      const el = container.querySelector('[data-kind]') as HTMLElement
      expect(el.getAttribute('role')).toBe(k === 'error' || k === 'conflict' ? 'alert' : 'status')
      expect(screen.getByRole('heading', { level: 2, name: `عنوان ${k}` })).toBeTruthy()
      expect(el.querySelector('svg')).toBeTruthy()
      unmount()
    }
    render(<StatePanel kind="loading" title="در حال دریافت" />)
    const loading = screen.getByRole('status')
    expect(loading.getAttribute('aria-busy')).toBe('true')
    expect(loading.textContent).toBe('در حال دریافت')
  })
  it('Banner: conflict interrupts, the rest are polite', () => {
    render(<><Banner kind="conflict">ت</Banner><Banner kind="stale">ق</Banner></>)
    expect(screen.getByRole('alert').textContent).toBe('ت')
    expect(screen.getByRole('status').textContent).toBe('ق')
  })
  it('Metric never renders an unknown, unavailable or undefined value as zero', () => {
    const { container } = render(
      <>
        <Metric label="m1" value="۴" />
        <Metric label="m2" value="0" state="unknown" />
        <Metric label="m3" state="unavailable" />
        <Metric label="m4" state="undefined" />
        <Metric label="m5" />
        <Metric label="m6" value="۱" state="stale" />
      </>,
    )
    const text = (s: string) => container.querySelector(`.ds-kpi:nth-child(${s}) .ds-kpi-value`)?.textContent
    expect(text('1')).toBe('۴')
    expect(text('2')).toBe('—نامعلوم') // a value passed with state unknown is NOT shown
    expect(text('3')).toBe('—نامعتبر')
    expect(text('4')).toBe('—تعریف نشده')
    expect(text('5')).toBe('—نامعلوم') // missing value is unknown, not 0
    expect(text('6')).toBe('۱قدیمی')
    expect(container.textContent).not.toMatch(/(^|[^۰-۹])0([^۰-۹]|$)/)
  })
})

describe('layout and help primitives', () => {
  it('DsAppShell: skip link → header → nav → main, one h1 from PageHead', () => {
    const onOpen = vi.fn()
    render(
      <DsAppShell
        header={<AppHeader brandMark="C" brandName="CRM" crumb="خانه" search={{ label: 'جستجو', placeholder: 'جستجو…', shortcut: 'Ctrl K', onOpen }} />}
        nav={<RoleNavigation items={NAV} currentId="a" label="بخش‌ها" />}
      >
        <PageHead title="عنوان صفحه" sub="زیرعنوان" />
      </DsAppShell>,
    )
    expect(screen.getByRole('link', { name: 'پرش به محتوای اصلی' }).getAttribute('href')).toBe('#ws')
    expect(screen.getByRole('main').id).toBe('ws')
    expect(screen.getByRole('banner')).toBeTruthy()
    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1)
    fireEvent.click(screen.getAllByRole('button', { name: 'جستجو' })[0] as HTMLElement)
    expect(onOpen).toHaveBeenCalledTimes(1)
  })
  it('ContextHint: text is the accessible name; tooltip shows on focus and Esc dismisses it', () => {
    render(<ContextHint text="پرونده الان نزد کیست." />)
    const btn = screen.getByRole('button', { name: 'پرونده الان نزد کیست.' })
    expect(screen.getByRole('tooltip', { hidden: true }).hidden).toBe(true)
    act(() => btn.focus())
    expect(screen.getByRole('tooltip').hidden).toBe(false)
    fireEvent.keyDown(btn, { key: 'Escape' })
    expect(screen.getByRole('tooltip', { hidden: true }).hidden).toBe(true)
  })
  it('Glossary is a description list', () => {
    const { container } = render(<Glossary entries={[{ term: 'مسئول فعلی', definition: 'پرونده الان نزد کیست.' }]} />)
    expect(container.querySelector('dl dt')?.textContent).toBe('مسئول فعلی')
    expect(container.querySelector('dl dd')?.textContent).toBe('پرونده الان نزد کیست.')
  })
})
