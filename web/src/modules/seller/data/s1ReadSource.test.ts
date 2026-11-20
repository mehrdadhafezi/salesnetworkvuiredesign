import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { INVOICES, LEADS } from '../fixtures/sellerFixtures'
import { SIMS } from '../state/uiState'
import sourceText from './s1ReadSource.ts?raw'
import readsText from './s1Reads.tsx?raw'
import { ReadSequencer, createSyntheticS1Source, isAbortError, isAuthorizationBlock, mayShowData } from './s1ReadSource'

// RM-E04K — S1 read boundary over SYNTHETIC fixtures. A passing test is presentation evidence only: the
// synthetic source decides nothing about real row scope, and no proposed Seller endpoint exists or is called.

const lead = (id: number) => {
  const l = LEADS.find(x => x.id === id)
  if (!l) throw new Error(`fixture ${id} missing`)
  return l
}
const src = (scenario: Parameters<typeof createSyntheticS1Source>[0]['scenario']) => createSyntheticS1Source({ scenario, latencyMs: 0 })

let fetchSpy: ReturnType<typeof vi.spyOn>
let xhrSpy: ReturnType<typeof vi.spyOn>
beforeEach(() => {
  fetchSpy = vi.spyOn(globalThis, 'fetch')
  xhrSpy = vi.spyOn(XMLHttpRequest.prototype, 'open')
  vi.spyOn(Storage.prototype, 'setItem')
  vi.spyOn(Storage.prototype, 'getItem')
})
afterEach(() => {
  // 10. No real network usage and no browser storage, for every read below.
  expect(fetchSpy).not.toHaveBeenCalled()
  expect(xhrSpy).not.toHaveBeenCalled()
  expect(Storage.prototype.setItem).not.toHaveBeenCalled()
  expect(Storage.prototype.getItem).not.toHaveBeenCalled()
  vi.restoreAllMocks()
})

describe('synthetic S1 read source', () => {
  it('1. list: the synthetic set, an EXACT count of its rows, no next page (no pagination contract)', async () => {
    const r = await src('none').listCases()
    expect(r.state).toBe('success')
    expect(r.answer).toMatchObject({ status: 200, outcome: 'SUCCESS', count: 'EXACT', itemCount: LEADS.length })
    expect(r.data?.rows.map(l => l.id)).toEqual(LEADS.map(l => l.id))
    expect(r.data?.nextPage).toBeNull()
  })

  it('2. detail: the record plus only its explicitly referenced invoices (never a phone join)', async () => {
    const r = await src('none').getCase('900104')
    expect(r.state).toBe('success')
    expect(r.data?.record.id).toBe(900104)
    expect(r.data?.invoices.map(i => i.code)).toEqual(lead(900104).invoiceRefs)
    // A lead without references has no invoices, even if an invoice shares its phone or name.
    const none = await src('none').getCase('900102')
    expect(none.data?.invoices).toEqual([])
    expect(INVOICES.length).toBeGreaterThan(0)
  })

  it('3. exact empty: zero rows with an EXACT count — the only case where 0 is a fact', async () => {
    const r = await src('empty').listCases()
    expect(r.state).toBe('empty')
    expect(r.answer).toMatchObject({ itemCount: 0, count: 'EXACT' })
    expect(r.data?.rows).toEqual([])
  })

  it('4. absent count: rows render, the count is ABSENT (presented «—», never 0)', async () => {
    const r = await src('count_absent').listCases()
    expect(r.state).toBe('count_absent')
    expect(r.answer.count).toBe('ABSENT')
    expect(r.data?.rows.length).toBe(LEADS.length)
  })

  it('5. partial history: truncated, marked incomplete, answered PARTIAL', async () => {
    const r = await src('partial').listCaseHistory('900104')
    expect(r.state).toBe('partial')
    expect(r.answer.reasons).toContain('COVERAGE_INCOMPLETE')
    expect(r.data?.complete).toBe(false)
    expect(r.data?.events.length).toBeLessThan(lead(900104).history.length)
    const full = await src('none').listCaseHistory('900104')
    expect(full.data).toMatchObject({ caseId: '900104', complete: true, events: lead(900104).history })
  })

  it('6. FIELD_DENIED: name and phone are withheld in list and detail answers (not partially kept)', async () => {
    const s = src('field_denied')
    const list = await s.listCases()
    expect(list.state).toBe('field_denied')
    expect(list.data?.rows.every(l => l.name === '' && l.phone === '')).toBe(true)
    const detail = await s.getCase('900104')
    expect(detail.answer.reasons).toContain('FIELD_DENIED')
    expect(detail.data?.record).toMatchObject({ name: '', phone: '' })
    expect(JSON.stringify([list.data, detail.data])).not.toMatch(/0900000\d{4}|مشتری نمونه/)
    // The fixtures themselves are untouched.
    expect(lead(900104).phone).toMatch(/^0900000/)
  })

  it('7. missing and out-of-scope detail share one generic 404 with no data', async () => {
    const missing = await src('none').getCase('999999')
    const hidden = await src('not_found').getCase('900104')
    expect(missing).toEqual(hidden)
    expect(missing).toMatchObject({ state: 'not_found', data: null, answer: { status: 404 } })
    expect(JSON.stringify(missing)).not.toContain('900104')
    // The list itself still answers normally in the not-found scenario.
    expect((await src('not_found').listCases()).state).toBe('success')
  })

  it('8. transport failures carry no data — never an empty list', async () => {
    for (const [scenario, status] of [['error', 500], ['offline', 0]] as const) {
      const s = src(scenario)
      for (const r of [await s.listCases(), await s.getCase('900104'), await s.listCaseHistory('900104')]) {
        expect(r.state).toBe('failure')
        expect(r.answer.status).toBe(status)
        expect(r.data).toBeNull()
      }
    }
  })

  it('authorization answers block every read with no data', async () => {
    for (const scenario of ['action_denied', 'row_scope_denied', 'session_expired', 'feature_disabled'] as const) {
      const s = src(scenario)
      for (const r of [await s.listCases(), await s.getCase('900104'), await s.listCaseHistory('900104'), await s.listCaseContactOutcomes('900104')]) {
        expect(isAuthorizationBlock(r.state)).toBe(true)
        expect(r.data).toBeNull()
      }
    }
  })

  it('9. retry: a read is idempotent — repeating it on a fresh source answers the same, nothing changes', async () => {
    const before = JSON.stringify(LEADS)
    const failed = await src('error').listCases()
    expect(failed.state).toBe('failure')
    const a = await src('none').listCases()
    const b = await src('none').listCases()
    expect(a.data).toEqual(b.data)
    expect(JSON.stringify(LEADS)).toBe(before)
  })

  it('contact-outcome history is not derived from status: answered UNKNOWN with no outcome records', async () => {
    const r = await src('none').listCaseContactOutcomes('900104')
    expect(r.state).toBe('unknown')
    expect(r.data).toEqual({ caseId: '900104', available: false })
  })

  it('every scenario answers deterministically; the loading scenario never settles until aborted', async () => {
    for (const scenario of SIMS.filter(s => s !== 'loading')) {
      const one = await src(scenario).listCases()
      const two = await src(scenario).listCases()
      expect(one).toEqual(two)
      expect(mayShowData(one.state)).toBe(one.data !== null)
    }
    const ctl = new AbortController()
    const pending = src('loading').listCases({ signal: ctl.signal })
    let settled = false
    void pending.then(() => { settled = true }, () => { settled = true })
    await new Promise(r => setTimeout(r, 20))
    expect(settled).toBe(false)
    ctl.abort()
    await expect(pending).rejects.toSatisfy(isAbortError)
  })

  it('aborting a pending read rejects it with AbortError; an already-aborted signal never starts', async () => {
    const s = createSyntheticS1Source({ scenario: 'none', latencyMs: 50 })
    const ctl = new AbortController()
    const p = s.getCase('900104', { signal: ctl.signal })
    ctl.abort()
    await expect(p).rejects.toSatisfy(isAbortError)
    await expect(s.listCases({ signal: AbortSignal.abort() })).rejects.toSatisfy(isAbortError)
  })

  it('refuses to run outside DEV/TEST (a production build never gets a synthetic source)', () => {
    vi.stubEnv('DEV', false)
    try {
      expect(() => createSyntheticS1Source({ scenario: 'none' })).toThrow(/DEV\/TEST only/)
    } finally {
      vi.unstubAllEnvs()
    }
    expect(() => createSyntheticS1Source({ scenario: 'none' })).not.toThrow()
  })

  it('each source has its own identity (a new scenario restarts every read)', () => {
    expect(src('none').id).not.toBe(src('none').id)
  })

  it('the boundary has no network, storage or write path in its source text', () => {
    for (const text of [sourceText, readsText]) {
      expect(text).not.toMatch(/\bfetch\(|XMLHttpRequest|sendBeacon|localStorage|sessionStorage|indexedDB|shared\/api/)
      expect(text).not.toMatch(/contact_outcome\.record|\/api\/v1/)
    }
    // Read-only interface: no create / update / record / post method names.
    expect(sourceText).not.toMatch(/^\s+(create|update|record|post|put|delete)\w*\(/m)
  })
})

describe('ReadSequencer (request ordering)', () => {
  it('only the newest generation is current; begin() aborts the previous request', () => {
    const seq = new ReadSequencer()
    const a = seq.begin()
    const b = seq.begin()
    expect(a.signal.aborted).toBe(true)
    expect(seq.isCurrent(a.generation)).toBe(false)
    expect(seq.isCurrent(b.generation)).toBe(true)
  })

  it('cancel() (close / unmount) makes every pending request stale and aborts it', () => {
    const seq = new ReadSequencer()
    const a = seq.begin()
    seq.cancel()
    expect(a.signal.aborted).toBe(true)
    expect(seq.isCurrent(a.generation)).toBe(false)
  })

  it('a late answer for an earlier query is dropped while the later one applies', async () => {
    const seq = new ReadSequencer()
    const s = createSyntheticS1Source({ scenario: 'none', latencyMs: 30 })
    const applied: string[] = []
    const run = (id: string) => {
      const { generation, signal } = seq.begin()
      return s.getCase(id, { signal }).then(r => { if (seq.isCurrent(generation)) applied.push(String(r.data?.record.id)) }, () => undefined)
    }
    await Promise.all([run('900104'), run('900105')])
    expect(applied).toEqual(['900105'])
  })
})
