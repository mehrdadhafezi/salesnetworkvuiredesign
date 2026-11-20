import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { RouterProvider, createMemoryRouter } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { S1ReadSourceProvider } from './data/s1Reads'
import {
  abortError, createSyntheticS1Source, type S1CaseDetail, type S1CaseHistory, type S1CaseList, type S1Read, type S1ReadSource,
} from './data/s1ReadSource'
import { LEADS } from './fixtures/sellerFixtures'
import { SellerPreviewPage } from './SellerPreviewPage'
import { SIM_LABEL, classifyAnswer, type ServerAnswer, type Sim } from './state/uiState'

// RM-E04K — S1 read lifecycle in the rendered preview: loading, ordering, cancellation and fail-closed
// authorization transitions. Driven by a hand-controlled source whose reads settle only when the test says
// so. SYNTHETIC only: a passing test proves presentation behaviour, never real row-level authorization.

function must<T>(v: T | null | undefined): T { if (v == null) throw new Error('missing'); return v }
const A = must(LEADS.find(l => l.id === 900104))
const B = must(LEADS.find(l => l.id === 900105))
const bodyText = () => document.body.textContent ?? ''

type Kind = 'list' | 'detail' | 'history' | 'outcomes'
interface Pending { kind: Kind; caseId: string | null; signal: AbortSignal | undefined; settle: (r: S1Read<unknown>) => void }

/** A source whose every read stays pending until `answer()` settles it. */
function controlled() {
  const pending: Pending[] = []
  const make = <T,>(kind: Kind, caseId: string | null, signal: AbortSignal | undefined) => new Promise<S1Read<T>>((resolve, reject) => {
    if (signal?.aborted) { reject(abortError()); return }
    signal?.addEventListener('abort', () => reject(abortError()), { once: true })
    pending.push({ kind, caseId, signal, settle: r => resolve(r as S1Read<T>) })
  })
  const source: S1ReadSource = {
    id: 'ctl',
    listCases: o => make<S1CaseList>('list', null, o?.signal),
    getCase: (id, o) => make<S1CaseDetail>('detail', id, o?.signal),
    listCaseHistory: (id, o) => make<S1CaseHistory>('history', id, o?.signal),
    listCaseContactOutcomes: (id, o) => make('outcomes', id, o?.signal),
  }
  const find = (kind: Kind, caseId: string | null = null) => must(pending.filter(p => p.kind === kind && (caseId === null || p.caseId === caseId)).at(-1))
  /** Settle the newest pending read of `kind` (and case) — even an aborted one, to prove late answers are ignored. */
  const answer = async (kind: Kind, caseId: string | null, answer: ServerAnswer, data: unknown) => {
    const state = classifyAnswer(answer)
    await act(async () => { find(kind, caseId).settle({ answer, state, data }) })
  }
  return { source, pending, find, answer }
}

const OK: ServerAnswer = { status: 200, outcome: 'SUCCESS', reasons: [], itemCount: LEADS.length, count: 'EXACT' }
const DENIED: ServerAnswer = { status: 403, outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'] }
const listData: S1CaseList = { rows: LEADS, nextPage: null }
const detailOf = (l: typeof A): S1CaseDetail => ({ record: l, invoices: [] })
const historyOf = (l: typeof A): S1CaseHistory => ({ caseId: String(l.id), events: l.history, complete: true })

function renderAt(url: string, source?: S1ReadSource) {
  const router = createMemoryRouter([
    { path: '/dev/seller', Component: SellerPreviewPage },
    { path: '/dev/seller/:section', Component: SellerPreviewPage },
  ], { initialEntries: [url] })
  const app = <RouterProvider router={router} />
  return { router, ...render(source ? <S1ReadSourceProvider source={source}>{app}</S1ReadSourceProvider> : app) }
}

const go = (router: ReturnType<typeof renderAt>['router'], search: string) => act(async () => { await router.navigate(`/dev/seller/leads${search}`) })

function switchSim(sim: Sim) {
  fireEvent.click(screen.getByRole('button', { name: /منوی کاربر/ }))
  fireEvent.click(screen.getByRole('menuitem', { name: new RegExp(SIM_LABEL[sim].replace(/[()]/g, '\\$&')) }))
}

async function readsSettled() {
  await waitFor(() => expect(document.querySelector('[data-read-phase="loading"]')).toBeNull())
}

let fetchSpy: ReturnType<typeof vi.spyOn>
let xhrSpy: ReturnType<typeof vi.spyOn>
beforeEach(() => {
  fetchSpy = vi.spyOn(globalThis, 'fetch')
  xhrSpy = vi.spyOn(XMLHttpRequest.prototype, 'open')
})
afterEach(() => {
  expect(fetchSpy).not.toHaveBeenCalled()
  expect(xhrSpy).not.toHaveBeenCalled()
  fetchSpy.mockRestore(); xhrSpy.mockRestore()
})

describe('S1 read lifecycle', () => {
  it('loading: no rows, no counts and no drawer until the list answer arrives; then the existing screen renders', async () => {
    const c = controlled()
    renderAt('/dev/seller/leads?queue=all&lead=900104', c.source)
    // Announced politely and busy (accessible loading state).
    const loading = must(document.querySelector('[data-read-phase="loading"] [data-kind="loading"]'))
    expect(loading.getAttribute('role')).toBe('status')
    expect(loading.getAttribute('aria-busy')).toBe('true')
    expect(document.querySelector('.sl-tbl, .sl-lcard, [data-queue]')).toBeNull()
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(bodyText()).not.toContain(A.name)
    await c.answer('list', null, OK, listData)
    expect(screen.getByRole('table', { name: 'فهرست شماره‌ها' })).toBeTruthy()
    // The detail read is still pending: the drawer is open with a loading panel and no record content.
    const d = screen.getByRole('dialog')
    expect(d.querySelector('[data-read-phase="loading"]')).not.toBeNull()
    expect(d.textContent).not.toContain(A.name)
    await c.answer('detail', '900104', { status: 200, outcome: 'SUCCESS', reasons: [] }, detailOf(A))
    expect(screen.getByRole('dialog', { name: A.name })).toBeTruthy()
  })

  it('1. a late answer for case A cannot overwrite the newer request for case B', async () => {
    const c = controlled()
    const { router } = renderAt('/dev/seller/leads?queue=all&lead=900104', c.source)
    await c.answer('list', null, OK, listData)
    const reqA = c.find('detail', '900104')
    await go(router, '?queue=all&lead=900105')
    expect(reqA.signal?.aborted).toBe(true)
    await c.answer('detail', '900105', { status: 200, outcome: 'SUCCESS', reasons: [] }, detailOf(B))
    expect(screen.getByRole('dialog', { name: B.name })).toBeTruthy()
    // A's success arrives after B's: ignored.
    await act(async () => { reqA.settle({ answer: { status: 200, outcome: 'SUCCESS', reasons: [] }, state: 'success', data: detailOf(A) }) })
    expect(screen.getByRole('dialog', { name: B.name })).toBeTruthy()
    expect(screen.getByRole('dialog').textContent).not.toContain(A.name)
  })

  it('2. a delayed success cannot restore protected content after a 403', async () => {
    const c = controlled()
    renderAt('/dev/seller/leads?queue=all&lead=900104', c.source)
    // The detail read is denied while the list read is still pending.
    await c.answer('detail', '900104', DENIED, null)
    expect(await screen.findByRole('heading', { level: 2, name: 'دسترسی به این بخش ندارید' })).toBeTruthy()
    // The list success arrives late: the screen stays closed.
    await c.answer('list', null, OK, listData)
    expect(screen.getByRole('heading', { level: 2, name: 'دسترسی به این بخش ندارید' })).toBeTruthy()
    expect(document.querySelector('.sl-tbl, .sl-lcard, [role="dialog"]')).toBeNull()
    for (const l of LEADS) expect(bodyText()).not.toContain(l.name)
  })

  it('3. a drawer closed while its read is pending does not reopen when the answer arrives', async () => {
    const c = controlled()
    const { router } = renderAt('/dev/seller/leads?queue=all&lead=900104', c.source)
    await c.answer('list', null, OK, listData)
    expect(screen.getByRole('dialog')).toBeTruthy()
    const req = c.find('detail', '900104')
    fireEvent.click(within(screen.getByRole('dialog')).getAllByRole('button', { name: /بستن/ })[0] as HTMLElement)
    await waitFor(() => expect(router.state.location.search).not.toContain('lead='))
    expect(req.signal?.aborted).toBe(true)
    await act(async () => { req.settle({ answer: { status: 200, outcome: 'SUCCESS', reasons: [] }, state: 'success', data: detailOf(A) }) })
    expect(screen.queryByRole('dialog', { name: A.name })).toBeNull()
    expect(bodyText()).not.toContain('آخرین فعالیت')
  })

  it('4. a stale history answer never appears under another case', async () => {
    const c = controlled()
    const { router } = renderAt('/dev/seller/leads?queue=all&lead=900104', c.source)
    await c.answer('list', null, OK, listData)
    await c.answer('detail', '900104', { status: 200, outcome: 'SUCCESS', reasons: [] }, detailOf(A))
    const histA = c.find('history', '900104')
    await go(router, '?queue=all&lead=900105')
    expect(histA.signal?.aborted).toBe(true)
    await c.answer('detail', '900105', { status: 200, outcome: 'SUCCESS', reasons: [] }, detailOf(B))
    // A's history (with A-only events) arrives late, while B's is pending.
    await act(async () => { histA.settle({ answer: { status: 200, outcome: 'SUCCESS', reasons: [] }, state: 'success', data: historyOf(A) }) })
    const d = screen.getByRole('dialog', { name: B.name })
    expect(within(d).queryByRole('list', { name: 'تاریخچه تعامل' })).toBeNull()
    expect(d.querySelector('[data-read-phase="loading"]')).not.toBeNull()
    await c.answer('history', '900105', { status: 200, outcome: 'SUCCESS', reasons: [] }, historyOf(B))
    const tl = within(d).getByRole('list', { name: 'تاریخچه تعامل' })
    expect(tl.textContent).toContain(must(B.history[0]).text)
    expect(tl.textContent).not.toContain(must(A.history[0]).text)
  })

  it('5. unmounting with pending reads aborts them; late answers are harmless', async () => {
    const c = controlled()
    const errors = vi.spyOn(console, 'error')
    const { unmount } = renderAt('/dev/seller/leads?queue=all&lead=900104', c.source)
    const list = c.find('list'); const detail = c.find('detail')
    unmount()
    expect(list.signal?.aborted).toBe(true)
    expect(detail.signal?.aborted).toBe(true)
    await act(async () => { list.settle({ answer: OK, state: 'success', data: listData }) })
    expect(errors).not.toHaveBeenCalled()
    expect(bodyText()).toBe('')
    errors.mockRestore()
  })

  it('a rejected (non-abort) read is a controlled failure — never an empty list', async () => {
    const c = controlled()
    const broken: S1ReadSource = { ...c.source, listCases: () => Promise.reject(new TypeError('network down')) }
    renderAt('/dev/seller/leads?queue=all', broken)
    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toContain('امکان دریافت لیست وجود ندارد')
    expect(screen.queryByText('در این صف موردی وجود ندارد')).toBeNull()
    expect(document.querySelector('[data-queue]')).toBeNull()
  })

  it('history failure: inline message and a retry that repeats only the history read', async () => {
    const c = controlled()
    renderAt('/dev/seller/leads?queue=all&lead=900104', c.source)
    await c.answer('list', null, OK, listData)
    await c.answer('detail', '900104', { status: 200, outcome: 'SUCCESS', reasons: [] }, detailOf(A))
    await c.answer('history', '900104', { status: 500, correlationId: 'syn' }, null)
    const d = screen.getByRole('dialog', { name: A.name })
    expect(d.querySelector('[data-read-phase="failed"]')?.textContent).toContain('تاریخچه دریافت نشد')
    const before = c.pending.length
    fireEvent.click(within(d).getByRole('button', { name: 'تلاش مجدد تاریخچه' }))
    expect(c.pending.slice(before).map(p => p.kind)).toEqual(['history'])
    await c.answer('history', '900104', { status: 200, outcome: 'SUCCESS', reasons: [] }, historyOf(A))
    expect(within(d).getByRole('list', { name: 'تاریخچه تعامل' })).toBeTruthy()
  })
})

describe('S1 lifecycle over the synthetic source (scenario switches)', () => {
  it('6. session expiry mid-draft removes the record and the draft; recovering starts a fresh draft', async () => {
    renderAt('/dev/seller/leads?queue=all&lead=900104')
    const d = await screen.findByRole('dialog', { name: A.name })
    await readsSettled()
    fireEvent.click(within(d).getByRole('button', { name: /^عدم خرید/ }))
    fireEvent.change(within(d).getByLabelText(/دلیل عدم خرید/), { target: { value: 'دلیل محرمانه نمونه' } })
    fireEvent.click(within(d).getByRole('button', { name: 'ثبت عدم خرید' }))
    expect(within(d).getByRole('heading', { name: 'بازبینی پیش از تأیید' })).toBeTruthy()
    switchSim('session_expired')
    expect(await screen.findByRole('heading', { level: 2, name: 'نشست شما به پایان رسیده است' })).toBeTruthy()
    expect(screen.queryByRole('dialog', { name: A.name })).toBeNull()
    expect(document.querySelector('[data-draft-step], [data-outcome], [data-simulation]')).toBeNull()
    expect(bodyText()).not.toContain('دلیل محرمانه نمونه')
    expect(bodyText()).not.toContain(A.name)
    switchSim('none')
    const again = await screen.findByRole('dialog', { name: A.name })
    await readsSettled()
    expect(again.getAttribute('data-draft-step') ?? again.querySelector('[data-draft-step]')?.getAttribute('data-draft-step')).toBe('edit')
    expect(bodyText()).not.toContain('دلیل محرمانه نمونه')
  })

  it('stepping to another case (J) discards the previous case draft; the next case loads fresh', async () => {
    renderAt('/dev/seller/leads?queue=all&lead=900104')
    const d = await screen.findByRole('dialog', { name: A.name })
    await readsSettled()
    fireEvent.click(within(d).getByRole('button', { name: /^عدم خرید/ }))
    fireEvent.change(within(d).getByLabelText(/دلیل عدم خرید/), { target: { value: 'پیش‌نویس الف' } })
    fireEvent.click(within(d).getByRole('button', { name: 'مشتری بعدی (J)' }))
    expect(screen.getByRole('dialog').textContent).not.toContain('پیش‌نویس الف')
    await readsSettled()
    const next = screen.getByRole('dialog')
    expect(next.querySelector('[data-draft-step]')).not.toBeNull()
    expect(screen.queryByRole('dialog', { name: A.name })).toBeNull()
    expect(next.querySelector('[aria-pressed="true"][data-outcome]')).toBeNull()
    expect(bodyText()).not.toContain('پیش‌نویس الف')
  })

  it('failure → retry repeats only the read: loading, then rows (no write, no request)', async () => {
    renderAt('/dev/seller/leads?queue=all&sim=error')
    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toContain('syn-corr-a7-500')
    fireEvent.click(within(alert).getByRole('button', { name: 'تلاش مجدد' }))
    expect(document.querySelector('[data-read-phase="loading"]')).not.toBeNull()
    expect(document.querySelector('.sl-tbl')).toBeNull()
    await readsSettled()
    expect(screen.getByRole('table', { name: 'فهرست شماره‌ها' })).toBeTruthy()
  })

  it('offline: S1 presents a network failure, not cached rows and not an empty list', async () => {
    renderAt('/dev/seller/leads?queue=all&sim=offline')
    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toContain('امکان دریافت لیست وجود ندارد')
    expect(document.querySelector('[data-kind="offline"]')).not.toBeNull()
    expect(document.querySelector('.sl-tbl, .sl-lcard')).toBeNull()
    expect(screen.queryByText('در این صف موردی وجود ندارد')).toBeNull()
  })

  it('the synthetic source is the default; an injected source replaces it without touching the screen', async () => {
    const injected = createSyntheticS1Source({ scenario: 'empty', latencyMs: 0 })
    renderAt('/dev/seller/leads?queue=all', injected)
    expect(await screen.findByText('در این صف موردی وجود ندارد')).toBeTruthy()
  })
})
