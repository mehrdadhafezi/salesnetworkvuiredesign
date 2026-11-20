import { describe, expect, it } from 'vitest'
import { toFaDigits } from '../../../shared/ui/ds'
import { REQUIRED_REASON, REQUIRED_TEXT } from './selectors'
import { EMPTY_FIELDS, draftLines, draftReducer, emptyDraft, type OutcomeDraft } from './outcomeDraft'

// RM-E04J — pure draft model of the S1 contact-outcome preview. Steps are presentation only.

const run = (d: OutcomeDraft, ...as: Parameters<typeof draftReducer>[1][]) => as.reduce(draftReducer, d)

describe('outcome draft', () => {
  it('selecting an outcome updates the draft; field edits clear their error', () => {
    const d = run(emptyDraft(), { type: 'pick', outcome: 'not_purchased' }, { type: 'review' })
    expect(d.outcome).toBe('not_purchased')
    expect(d.step).toBe('edit')
    expect(d.errors.reason).toBe(REQUIRED_REASON)
    const e = draftReducer(d, { type: 'field', key: 'reason', value: 'قیمت' })
    expect(e.errors.reason).toBeUndefined()
    expect(e.fields.reason).toBe('قیمت')
  })
  it('reuses validateOutcome: an invalid draft never reaches review; a valid one does', () => {
    const bad = run(emptyDraft('callback'), { type: 'field', key: 'date', value: ' ' }, { type: 'review' })
    expect(bad.step).toBe('edit')
    expect(bad.errors.date).toBe(REQUIRED_TEXT)
    expect(draftReducer(bad, { type: 'confirm' }).step).toBe('edit') // confirm is impossible from edit
    const ok = run(emptyDraft('callback'), { type: 'review' })
    expect(ok.step).toBe('review')
    expect(run(emptyDraft(), { type: 'review' }).step).toBe('edit') // no outcome, no review
  })
  it('edit returns to the form with nothing lost; review locks the draft', () => {
    const r = run(emptyDraft('not_purchased'), { type: 'field', key: 'reason', value: 'قیمت' }, { type: 'review' })
    const locked = run(r, { type: 'pick', outcome: 'duplicate' }, { type: 'field', key: 'reason', value: 'x' }, { type: 'payType', value: 'staged' })
    expect(locked).toBe(r)
    const back = draftReducer(r, { type: 'edit' })
    expect(back.step).toBe('edit')
    expect(back.outcome).toBe('not_purchased')
    expect(back.fields.reason).toBe('قیمت')
  })
  it('confirm produces exactly one simulated result; discard / reset clears everything', () => {
    const r = run(emptyDraft('success'), { type: 'product', value: 'p2' }, { type: 'payType', value: 'staged' }, { type: 'review' })
    const sim = draftReducer(r, { type: 'confirm' })
    expect(sim.step).toBe('simulated')
    expect(draftReducer(sim, { type: 'confirm' })).toBe(sim) // double confirm is a no-op
    expect(draftReducer(sim, { type: 'edit' })).toBe(sim)
    for (const from of [r, sim]) {
      const clean = draftReducer(from, { type: 'discard' })
      expect(clean).toEqual(emptyDraft())
      expect(clean.fields).toEqual(EMPTY_FIELDS)
    }
  })
  it('summary lines show only what was entered; no server time, id or amount is invented', () => {
    const ctx = { attemptsNext: 2, fa: toFaDigits }
    expect(draftLines(emptyDraft('no_answer'), ctx)).toEqual([{ label: 'تلاش', value: '۲ از ۳' }])
    const nr = run(emptyDraft('not_purchased'), { type: 'field', key: 'reason', value: '  قیمت ' })
    expect(draftLines(nr, ctx)).toEqual([{ label: 'دلیل عدم خرید', value: 'قیمت' }])
    const cb = draftLines(emptyDraft('callback'), ctx)
    expect(cb.find(l => l.label === 'ساعت تهران')?.value).toBe('۱۰:۳۰')
    expect(draftLines(emptyDraft(), ctx)).toEqual([])
    const all = [...(['no_answer', 'callback', 'not_purchased', 'duplicate', 'success'] as const).flatMap(o => draftLines(emptyDraft(o), ctx))]
    expect(all.some(l => /شناسه|correlation|event/i.test(l.label))).toBe(false)
  })
})
