import { fireEvent, render, screen, within } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { describe, expect, it, vi } from 'vitest'
import { ActivityTimeline, type TimelineEvent, type TimelineSource } from './ActivityTimeline'
import { combine, coverageOf, instantTime, legacyTime, type EventTime, type OrderedInput, type SourceOrder } from './model'

// RM-P03 timeline model and shared component. All data is SYNTHETIC.

describe('timestamp evidence (never invented, never shifted)', () => {
  it('recognises legacy wall-clock texts with their precision', () => {
    expect(legacyTime('2025-03-01')).toMatchObject({ kind: 'legacy', precision: 'day' })
    expect(legacyTime('2025-03-01 10:00')).toMatchObject({ kind: 'legacy', precision: 'minute' })
    expect(legacyTime('2025-03-01 10:00:05')).toMatchObject({ kind: 'legacy', precision: 'second', raw: '2025-03-01 10:00:05' })
    expect(legacyTime('2025-03-01T10:00:05.25')).toMatchObject({ kind: 'legacy', precision: 'fraction' })
    const day = legacyTime('2025-03-01') as Extract<EventTime, { kind: 'legacy' }>
    expect(day.end - day.start).toBe(86_400_000_000)
    const sec = legacyTime('2025-03-01 10:00:05') as Extract<EventTime, { kind: 'legacy' }>
    expect(sec.end - sec.start).toBe(1_000_000)
  })

  it('keeps unrecognised or impossible values as unparsed raw text, and absence as missing', () => {
    for (const raw of ['1403/12/11 10:00', '0000-00-00 00:00:00', '2025-02-30 10:00:00', '1740823200', 'yesterday', '2025-13-01']) {
      expect(legacyTime(raw)).toEqual({ kind: 'unparsed', raw })
    }
    for (const v of [null, undefined, '', '   ']) expect(legacyTime(v)).toEqual({ kind: 'missing' })
    expect(instantTime('2026-11-01 09:00', 'TARGET_UTC')).toEqual({ kind: 'unparsed', raw: '2026-11-01 09:00' })
    expect(instantTime(null, 'IMPORT_OBSERVED')).toEqual({ kind: 'missing' })
  })

  it('reads RFC 3339 instants with their offset and fractional precision', () => {
    const a = instantTime('2026-11-01T09:00:00Z', 'TARGET_UTC')
    const b = instantTime('2026-11-01T12:30:00+03:30', 'TARGET_UTC')
    expect(a).toMatchObject({ kind: 'instant', precision: 'second' })
    expect((a as { start: number }).start).toBe((b as { start: number }).start)
    expect(instantTime('2026-11-01T09:00:00.123456789Z', 'TARGET_UTC')).toMatchObject({ precision: 'fraction' })
  })
})

const inp = (source: string, sourceRank: number, index: number, time: EventTime, sourceOrder: SourceOrder = 'SOURCE_ORDINAL'): OrderedInput =>
  ({ key: `${source}:${index}`, source, sourceRank, index, time, sourceOrder })

describe('combined ordering', () => {
  it('orders one clock chronologically across sources and keeps clocks apart', () => {
    const items = [
      inp('history', 0, 0, legacyTime('2025-01-03 10:00:00')),
      inp('custody', 1, 0, legacyTime('2025-01-01 10:00:00')),
      inp('outcomes', 2, 0, instantTime('2025-01-02T10:00:00Z', 'TARGET_UTC'), 'NEWEST_FIRST'),
      inp('history', 0, 1, legacyTime('2025-01-04 10:00:00')),
    ]
    const c = combine(items)
    expect(c.groups.map(g => g.clock)).toEqual(['LEGACY_SOURCE', 'TARGET_UTC'])
    expect(c.groups[0]?.items.map(i => i.item.key)).toEqual(['custody:0', 'history:0', 'history:1'])
    expect(c.groups[1]?.items.map(i => i.item.key)).toEqual(['outcomes:0'])
    // The UTC instant is never placed between legacy wall-clock values.
    expect(c.groups[0]?.items.some(i => i.item.source === 'outcomes')).toBe(false)
  })

  it('breaks equal timestamps deterministically by source rank then source index, and flags the tie', () => {
    const t = '2025-01-02 10:00:00'
    const items = [inp('custody', 1, 0, legacyTime(t)), inp('history', 0, 1, legacyTime(t)), inp('history', 0, 0, legacyTime(t))]
    const c = combine(items)
    expect(c.groups[0]?.items.map(i => i.item.key)).toEqual(['history:0', 'history:1', 'custody:0'])
    for (const i of c.groups[0]?.items ?? []) expect(i.notes).toContain('TIE')
    // Same result for any input order.
    expect(combine([...items].reverse()).groups[0]?.items.map(i => i.item.key)).toEqual(['history:0', 'history:1', 'custody:0'])
  })

  it('flags different precisions whose intervals overlap instead of pretending an order', () => {
    const c = combine([
      inp('a', 0, 0, legacyTime('2025-01-02 09:00:00')),
      inp('b', 1, 0, legacyTime('2025-01-02')),
      inp('c', 2, 0, legacyTime('2025-01-03 09:00:00')),
    ])
    const by = Object.fromEntries((c.groups[0]?.items ?? []).map(i => [i.item.key, i.notes]))
    expect(by['a:0']).toContain('PRECISION_OVERLAP')
    expect(by['b:0']).toContain('PRECISION_OVERLAP')
    expect(by['c:0']).toEqual([])
    // Day interval starts at 00:00 so it is placed first; the overlap note says the order is uncertain.
    expect(c.groups[0]?.items.map(i => i.item.key)).toEqual(['b:0', 'a:0', 'c:0'])
  })

  it('lists missing and unparsed timestamps apart, in source order, never in the chronology', () => {
    const c = combine([
      inp('custody', 1, 0, { kind: 'missing' }),
      inp('history', 0, 1, legacyTime('31/12/2024')),
      inp('history', 0, 0, legacyTime('2025-01-01 00:00:00')),
    ])
    expect(c.unordered.map(i => i.key)).toEqual(['history:1', 'custody:0'])
    expect(c.groups[0]?.items.map(i => i.item.key)).toEqual(['history:0'])
  })

  it('reports when the time order disagrees with a source\'s own insertion order', () => {
    const c = combine([inp('custody', 0, 0, legacyTime('2025-01-05 10:00:00')), inp('custody', 0, 1, legacyTime('2025-01-01 10:00:00'))])
    for (const i of c.groups[0]?.items ?? []) expect(i.notes).toContain('SOURCE_ORDER_DIFFERS')
  })

  it('states coverage honestly: complete, one page, or capped (unknown)', () => {
    expect(coverageOf(3, { depth: 0, hasMore: false }, undefined)).toBe('COMPLETE')
    expect(coverageOf(3, { depth: 0, hasMore: true }, undefined)).toBe('PAGE')
    expect(coverageOf(3, { depth: 1, hasMore: false }, undefined)).toBe('PAGE')
    expect(coverageOf(200, undefined, 200)).toBe('CAPPED')
    expect(coverageOf(199, undefined, 200)).toBe('COMPLETE')
  })
})

const ev = (id: string, time: EventTime, over: Partial<TimelineEvent> = {}): TimelineEvent => ({ id, type: 'synthetic.event', label: `رویداد ${id}`, icon: 'activity', time, provenance: `syn #${id}`, ...over })
const ok = (data: TimelineEvent[]) => ({ status: 'ok' as const, data, outcome: 'SUCCESS' as const, allowed: undefined })
const src = (id: string, read: TimelineSource['read'], over: Partial<TimelineSource> = {}): TimelineSource =>
  ({ id, label: `منبع ${id}`, icon: 'history', provenance: `module ${id}`, order: 'SOURCE_ORDINAL', blockedDetail: 'flag off', read, ...over })

const labels = (el: HTMLElement) => [...el.querySelectorAll('.tl-label')].map(n => n.textContent)

function show(sources: TimelineSource[]) {
  const router = createMemoryRouter([{ path: '/', element: <ActivityTimeline label="خط زمانی آزمایشی" coverage="پوشش آزمایشی" sources={sources} zone="Asia/Tehran" /> }])
  return render(<RouterProvider router={router} />)
}

describe('ActivityTimeline', () => {
  it('per source: keeps server order, shows raw time, transition, actor and Jalali date; only verified links navigate', () => {
    show([src('a', ok([
      ev('2', legacyTime('2025-03-02 10:00:00'), { transition: { from: 'new', to: 'won' }, actor: <span>کاربر قدیمی #9</span>, related: { label: 'فاکتور تأییدشده', href: '/billing/invoices/x' } }),
      ev('1', legacyTime('2025-03-01 10:00:00'), { actor: null, related: { label: 'پیوند تأییدنشده' } }),
    ]))])
    const lane = screen.getByRole('region', { name: 'منبع a' })
    const items = within(within(lane).getByRole('list', { name: 'رویدادهای منبع a' })).getAllByRole('listitem')
    expect(labels(lane)).toEqual(['رویداد 2', 'رویداد 1'])
    expect(items).toHaveLength(2)
    expect(within(items[0] as HTMLElement).getByText('new → won')).toBeTruthy()
    expect(within(items[0] as HTMLElement).getByText('2025-03-02 10:00:00')).toBeTruthy()
    expect((items[0] as HTMLElement).querySelector('time')?.getAttribute('dateTime')).toBe('2025-03-02T10:00:00')
    expect((items[0] as HTMLElement).querySelector('time')?.textContent).toMatch(/۱۴۰۳/)
    expect(within(items[0] as HTMLElement).getByRole('link', { name: 'فاکتور تأییدشده' }).getAttribute('href')).toBe('/billing/invoices/x')
    expect(within(items[1] as HTMLElement).queryByRole('link')).toBeNull()
    expect(within(items[1] as HTMLElement).getByText('ثبت نشده')).toBeTruthy()
  })

  it('renders denied, blocked, locked, awaiting, failed and empty sources honestly and never as events', () => {
    show([
      src('denied', { status: 'denied', reasons: ['ACTION_DENIED'] }),
      src('blocked', { status: 'blocked', reasons: ['FEATURE_DISABLED'] }),
      src('locked', { status: 'locked', title: 'دسترسی به این منبع ندارید' }),
      src('awaiting', { status: 'awaiting', title: 'در انتظار رکورد اصلی' }),
      src('failed', { status: 'failed', failure: { kind: 'transport', message: 'x' } }),
      src('empty', ok([])),
    ])
    expect(within(screen.getByRole('region', { name: 'منبع denied' })).getByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'منبع blocked' })).getByText('این بخش هنوز فعال نشده است')).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'منبع locked' })).getByText('دسترسی به این منبع ندارید')).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'منبع failed' })).getByText('ارتباط با سرویس برقرار نشد')).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'منبع empty' })).getByText('رویدادی در این منبع ثبت نشده است')).toBeTruthy()
    expect(screen.queryAllByRole('listitem').filter(li => li.classList.contains('tl-event'))).toHaveLength(0)
  })

  it('combined view: groups by clock, separates missing times, excludes denied sources and says it is partial', () => {
    show([
      src('h', ok([ev('h1', legacyTime('2025-01-03 10:00:00')), ev('h2', { kind: 'missing' })]), { paging: { depth: 0, page: { has_more: true, next_cursor: 'n' }, back: vi.fn(), next: vi.fn() } }),
      src('k', ok([ev('k1', legacyTime('2025-01-01 10:00:00'))])),
      src('o', ok([ev('o1', instantTime('2025-01-02T10:00:00Z', 'TARGET_UTC'))]), { order: 'NEWEST_FIRST' }),
      src('x', { status: 'denied', reasons: ['ACTION_DENIED'] }),
    ])
    fireEvent.click(screen.getByRole('button', { name: 'ترکیب زمانی' }))
    const combined = screen.getByRole('region', { name: 'ترکیب زمانی' })
    expect(within(combined).getByText(/این ترکیب کامل نیست/).textContent).toMatch(/منبع x/)
    expect(within(combined).getByText(/این ترکیب کامل نیست/).textContent).toMatch(/فقط صفحه بارگذاری‌شده از این منابع آمده است: منبع h/)
    const legacy = within(combined).getByRole('region', { name: 'زمان‌های منبع قدیمی' })
    expect(labels(legacy)).toEqual(['رویداد k1', 'رویداد h1'])
    expect(labels(within(combined).getByRole('region', { name: 'زمان‌های سامانه جدید (لحظه رخداد)' }))).toEqual(['رویداد o1'])
    expect(labels(within(combined).getByRole('region', { name: 'رویدادهای بدون زمان قابل مقایسه' }))).toEqual(['رویداد h2'])
  })

  it('combined view of complete sources says so (no partial claim)', () => {
    show([src('a', ok([ev('a1', legacyTime('2025-01-01'))])), src('b', ok([]))])
    fireEvent.click(screen.getByRole('button', { name: 'ترکیب زمانی' }))
    expect(screen.queryByText(/این ترکیب کامل نیست/)).toBeNull()
    expect(screen.getByText(/همه رویدادهای منابع بالا بارگذاری شده است/)).toBeTruthy()
  })

  it('pages with the server cursor and marks a capped source as of unknown completeness', () => {
    const next = vi.fn()
    show([
      src('p', ok([ev('p1', legacyTime('2025-01-01'))]), { paging: { depth: 0, page: { has_more: true, next_cursor: 'cur-2' }, back: vi.fn(), next } }),
      src('c', ok([ev('c1', legacyTime('2025-01-01')), ev('c2', legacyTime('2025-01-02'))]), { cap: 2 }),
    ])
    const lane = screen.getByRole('region', { name: 'منبع p' })
    expect(within(lane).getByText('فقط صفحه 1 بارگذاری شده است')).toBeTruthy()
    fireEvent.click(within(lane).getByRole('button', { name: 'صفحه بعد' }))
    expect(next).toHaveBeenCalledWith('cur-2')
    expect(within(screen.getByRole('region', { name: 'منبع c' })).getByText(/کامل بودن نامعلوم است/)).toBeTruthy()
  })

  it('renders lazy details only once opened', () => {
    const details = vi.fn(() => <p>جزئیات خوانده شد</p>)
    show([src('d', ok([ev('d1', legacyTime('2025-01-01'), { details })]))])
    expect(details).not.toHaveBeenCalled()
    expect(screen.queryByText('جزئیات خوانده شد')).toBeNull()
    const el = screen.getByText('جزئیات').closest('details') as HTMLDetailsElement
    el.open = true
    fireEvent(el, new Event('toggle'))
    expect(screen.getByText('جزئیات خوانده شد')).toBeTruthy()
  })
})
