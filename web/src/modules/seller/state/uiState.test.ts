import { describe, expect, it } from 'vitest'
import { BEHAVIOR, CONVERSIONS, INVOICES, LEADS, REPEAT_ITEMS, WALLET } from '../fixtures/sellerFixtures'
import { collectibleTotal, filterBehavior, filterConversions, filterInvoices, filterLeads, filterRepeat, DEFAULT_LEAD_FILTER, unitOf, walletTotals } from './selectors'
import { SELLER_UI_STATES, SIMS, STATE_POLICY, SYNTHETIC_ANSWERS, classifyAnswer, stateOfSim, type SellerUiState } from './uiState'

// RM-E04C / A7 — the pure Seller UI state model. Synthetic answers only; a passing test is presentation
// evidence, never business authorization.

describe('classifyAnswer', () => {
  it('every synthetic answer classifies back to its own state (all twelve)', () => {
    expect(SELLER_UI_STATES).toHaveLength(12)
    for (const s of SELLER_UI_STATES) expect(classifyAnswer(SYNTHETIC_ANSWERS[s])).toBe(s)
  })
  it('authorization and transport come before the envelope', () => {
    expect(classifyAnswer({ status: 401, outcome: 'SUCCESS' })).toBe('session_expired')
    expect(classifyAnswer({ status: 403 })).toBe('action_denied')
    expect(classifyAnswer({ status: 403, reasons: ['ROW_SCOPE_DENIED'] })).toBe('row_scope_denied')
    expect(classifyAnswer({ status: 200, outcome: 'UNAUTHORIZED', reasons: ['ROW_SCOPE_DENIED'] })).toBe('row_scope_denied')
    expect(classifyAnswer({ status: 404 })).toBe('not_found')
  })
  it('anything unrecognised fails closed to failure — never success or empty', () => {
    for (const a of [{ status: 0 }, { status: 400 }, { status: 418 }, { status: 502 }, { status: 200 }, { status: 200, outcome: 'FAILED' as const }, { status: 200, outcome: 'CONFLICT' as const }, { status: 200, outcome: 'BLOCKED' as const }]) {
      expect(classifyAnswer(a)).toBe('failure')
    }
  })
  it('an empty page is "empty" only with an exact count; an absent count is never empty and never zero', () => {
    expect(classifyAnswer({ status: 200, outcome: 'SUCCESS', itemCount: 0, count: 'EXACT' })).toBe('empty')
    expect(classifyAnswer({ status: 200, outcome: 'SUCCESS', itemCount: 0, count: 'ABSENT' })).toBe('count_absent')
    expect(classifyAnswer({ status: 200, outcome: 'SUCCESS', itemCount: 0 })).toBe('success')
  })
  it('PARTIAL wins over FIELD_DENIED and UNKNOWN (incomplete facts are the stricter presentation)', () => {
    expect(classifyAnswer({ status: 200, outcome: 'SUCCESS', reasons: ['FIELD_DENIED', 'COVERAGE_INCOMPLETE'] })).toBe('partial')
  })
})

describe('STATE_POLICY', () => {
  const where = (k: keyof (typeof STATE_POLICY)['success']) => SELLER_UI_STATES.filter(s => STATE_POLICY[s][k])
  it('retry is offered only for transport / server failure (a safe read)', () => {
    expect(where('retry')).toEqual(['failure'])
  })
  it('denials, disabled feature, expired session and failure render no screen body', () => {
    expect(where('blocksContent').sort()).toEqual(['action_denied', 'failure', 'feature_disabled', 'row_scope_denied', 'session_expired'].sort())
  })
  it('each presentation rule belongs to exactly one state', () => {
    expect(where('countsAbsent')).toEqual(['count_absent'])
    expect(where('dependentAmountsWithheld')).toEqual(['partial'])
    expect(where('fieldsRestricted')).toEqual(['field_denied'])
    expect(where('statusUnknown')).toEqual(['unknown'])
    expect(where('detailNotFound')).toEqual(['not_found'])
  })
  it('only synthetic correlation ids are used', () => {
    for (const s of SELLER_UI_STATES) {
      const c = SYNTHETIC_ANSWERS[s].correlationId
      if (c) expect(c).toMatch(/^syn-corr-a7-/)
    }
  })
})

describe('preview simulator', () => {
  it('keeps the RM-E03 values and adds every A7 state', () => {
    for (const s of ['none', 'loading', 'empty', 'error', 'offline']) expect(SIMS).toContain(s)
    const covered = new Set<SellerUiState>(SIMS.map(stateOfSim))
    for (const s of SELLER_UI_STATES) expect(covered.has(s)).toBe(true)
    expect(stateOfSim('error')).toBe('failure')
    expect(stateOfSim('none')).toBe('success')
  })
})

describe('money units', () => {
  it('fixtures carry an IRR invoice and an IRR wallet line; unit defaults to IRT', () => {
    expect(INVOICES.filter(i => unitOf(i) === 'IRR').map(i => i.code)).toEqual(['TST-1015'])
    expect(WALLET.tx.filter(t => unitOf(t) === 'IRR').map(t => t.id)).toEqual(['w9'])
  })
  it('collectible remaining is one sum per unit and never mixes units', () => {
    const c = collectibleTotal(INVOICES)
    expect(c.byUnit.map(u => u.unit)).toEqual(['IRT', 'IRR'])
    expect(c.byUnit.find(u => u.unit === 'IRR')?.value).toBe(60_000_000)
    const irtOnly = collectibleTotal(INVOICES.filter(i => unitOf(i) === 'IRT'))
    expect(c.byUnit.find(u => u.unit === 'IRT')?.value).toBe(irtOnly.byUnit[0]?.value)
  })
  it('wallet totals per unit; unknown lines are in no sum', () => {
    const t = walletTotals(WALLET.tx)
    expect(t.byUnit.map(u => u.unit)).toEqual(['IRT', 'IRR'])
    expect(t.unknownCount).toBe(1)
    expect(walletTotals([]).byUnit).toEqual([])
  })
})

describe('FIELD_DENIED search', () => {
  it('with identity withheld, name and phone are not searchable (a search cannot confirm a hidden value)', () => {
    const lead = LEADS[0]
    const inv = INVOICES[0]
    const conv = CONVERSIONS[0]
    const rep = REPEAT_ITEMS[0]
    const beh = BEHAVIOR[0]
    if (!lead || !inv || !conv || !rep || !beh) throw new Error('fixtures')
    const f = { ...DEFAULT_LEAD_FILTER, queue: 'all' as const }
    expect(filterLeads(LEADS, { ...f, q: lead.phone }, true).length).toBeGreaterThan(0)
    expect(filterLeads(LEADS, { ...f, q: lead.phone }, false)).toEqual([])
    expect(filterLeads(LEADS, { ...f, q: lead.name }, false)).toEqual([])
    expect(filterLeads(LEADS, { ...f, q: lead.city }, false).length).toBeGreaterThan(0)
    expect(filterInvoices(INVOICES, 'all', inv.phone, null, false)).toEqual([])
    expect(filterInvoices(INVOICES, 'all', inv.code, null, false)).toHaveLength(1)
    expect(filterConversions(CONVERSIONS, 'all', conv.customer, false)).toEqual([])
    expect(filterRepeat(REPEAT_ITEMS, rep.done ? 'done' : 'open', rep.phone, false)).toEqual([])
    expect(filterBehavior(BEHAVIOR, beh.name, false, false)).toEqual([])
  })
})

describe('preview isolation (static)', () => {
  // Sources as raw text via Vite (no Node APIs in the web tsconfig). Keys are paths relative to this file.
  const ALL = import.meta.glob<string>('../../../**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true })
  const isTest = (p: string) => /\.test\.tsx?$/.test(p)
  const inSeller = (p: string) => p.startsWith('../') && !p.startsWith('../../')
  it('only the DEV-guarded lazy route imports the Seller module', () => {
    const importers = Object.entries(ALL)
      .filter(([p]) => !inSeller(p) && p !== './uiState.ts' && !p.startsWith('./') && !isTest(p))
      .filter(([, src]) => /modules\/seller/.test(src))
      .map(([p]) => p)
    expect(importers).toEqual(['../../../app/routes.tsx'])
    expect(ALL['../../../app/routes.tsx']).toMatch(/import\.meta\.env\.DEV && config\.devTools[\s\S]*modules\/seller\/SellerPreviewPage/)
  })
  it('the Seller module performs no network or storage call and imports no API client', () => {
    const seller = Object.entries(ALL).filter(([p]) => (inSeller(p) || p.startsWith('./')) && !isTest(p))
    expect(seller.length).toBeGreaterThan(15)
    for (const [p, src] of seller) {
      expect(src, p).not.toMatch(/\bfetch\s*\(|XMLHttpRequest|sendBeacon|WebSocket|EventSource|localStorage|sessionStorage|indexedDB|navigator\.clipboard/)
      expect(src, p).not.toMatch(/shared\/api\/client|openapi-fetch/)
      expect(src, p).not.toMatch(/<form[^>]*\baction=/)
    }
  })
})
