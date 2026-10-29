import { describe, expect, expectTypeOf, it } from 'vitest'
import { configFromEnv } from '../../app/config'
import { actionHint } from '../permissions/allowedActions'
import { outcomePresentation } from '../ui/outcome'
import { fakeApi, json } from '../../test/fakeApi'
import { isProblem, toFailure } from './problem'
import type { AllowedActions, Money, Outcome, PageInfo, Problem, Unit } from './types'

// Type-level contract checks run under `tsc --noEmit` (npm run typecheck).
describe('generated contract types', () => {
  it('Money.amount is a string, never a number; IRR and IRT are the only units', () => {
    expectTypeOf<Money['amount']>().toEqualTypeOf<string>()
    expectTypeOf<Money['amount']>().not.toEqualTypeOf<number>()
    expectTypeOf<Unit>().toEqualTypeOf<'IRR' | 'IRT'>()
    // @ts-expect-error a numeric amount must not compile
    const bad: Money = { amount: 1000, unit: 'IRR' }
    expect(bad).toBeTruthy()
    const ok: Money = { amount: '-9223372036854775808', unit: 'IRT' }
    expect(ok.amount).toBe('-9223372036854775808')
  })

  it('Outcome is exactly the seven frozen values', () => {
    expectTypeOf<Outcome>().toEqualTypeOf<'SUCCESS' | 'PARTIAL' | 'FAILED' | 'UNKNOWN' | 'CONFLICT' | 'BLOCKED' | 'UNAUTHORIZED'>()
    expect(Object.keys(outcomePresentation).sort()).toEqual(['BLOCKED', 'CONFLICT', 'FAILED', 'PARTIAL', 'SUCCESS', 'UNAUTHORIZED', 'UNKNOWN'])
    expect(outcomePresentation.UNKNOWN.tone).not.toBe(outcomePresentation.FAILED.tone)
    expect(outcomePresentation.PARTIAL.tone).not.toBe(outcomePresentation.SUCCESS.tone)
  })

  it('PageInfo has a cursor and has_more but no total', () => {
    expectTypeOf<PageInfo>().toHaveProperty('has_more')
    expectTypeOf<PageInfo>().toHaveProperty('next_cursor')
    expectTypeOf<PageInfo>().not.toHaveProperty('total')
  })

  it('Problem carries a foundation-v1 reason code and correlation id', () => {
    expectTypeOf<Problem['reason_code']>().toEqualTypeOf<'DEPENDENCY_FINANCIAL_ACTIVE' | 'DEPENDENCY_PROTECTED_DOWNSTREAM' | 'DEPENDENCY_UNKNOWN' | 'LINKAGE_CONFLICT' | 'STALE_STATE' | 'ROW_SCOPE_DENIED' | 'ACTION_DENIED' | 'FIELD_DENIED' | 'RECIPIENT_INELIGIBLE' | 'OUTCOME_UNKNOWN' | 'SOURCE_UNAVAILABLE' | 'COVERAGE_INCOMPLETE' | 'POLICY_UNRESOLVED' | 'FEATURE_DISABLED' | 'CONTRACT_INVALID'>()
    expectTypeOf<Problem['correlation_id']>().toEqualTypeOf<string>()
  })
})

describe('allowed_actions hints', () => {
  it('absent actions are not allowed; the server hint is the only source', () => {
    const actions: AllowedActions = { approve: { allowed: true }, reject: { allowed: false, reason_code: 'ACTION_DENIED' } }
    expect(actionHint(actions, 'approve').allowed).toBe(true)
    expect(actionHint(actions, 'reject')).toEqual({ allowed: false, reason_code: 'ACTION_DENIED' })
    expect(actionHint(actions, 'delete').allowed).toBe(false)
    expect(actionHint(undefined, 'approve').allowed).toBe(false)
  })
})

describe('api adapter', () => {
  it('sets a correlation id, keeps a caller one, and never sends a dev header without a dev user', async () => {
    const { client, calls } = fakeApi({ 'GET /health/live': () => json({ status: 'live' }) })
    await client.GET('/health/live')
    await client.GET('/health/live', { headers: { 'X-Correlation-ID': 'caller-corr-01' } })
    expect(calls[0]?.headers.get('X-Correlation-ID')).toBe('test-corr-0001')
    expect(calls[1]?.headers.get('X-Correlation-ID')).toBe('caller-corr-01')
    expect(calls.some(c => c.headers.has('X-Dev-User'))).toBe(false)
  })

  it('classifies problems strictly', async () => {
    const { client } = fakeApi({})
    const { error, response } = await client.GET('/api/v1/session')
    const failure = toFailure(error, response)
    expect(failure.kind).toBe('problem')
    expect(isProblem({ title: 'x' })).toBe(false)
    expect(toFailure({ message: 'not a problem' }, new Response('x', { status: 502 })).kind).toBe('transport')
    expect(toFailure(new Error('down'), undefined)).toEqual({ kind: 'transport', message: 'down' })
  })
})

describe('build configuration', () => {
  it('production builds never carry a dev identity or dev tools', () => {
    const prod = configFromEnv({ PROD: true, DEV: false, VITE_DEV_AUTH_USER: 'dev-user-001' })
    expect(prod.devUser).toBeUndefined()
    expect(prod.devTools).toBe(false)
    expect(prod.displayZone).toBe('Asia/Tehran')
  })

  it('development accepts only synthetic dev-user-NNN identities', () => {
    expect(configFromEnv({ PROD: false, DEV: true, VITE_DEV_AUTH_USER: 'dev-user-002' }).devUser).toBe('dev-user-002')
    expect(configFromEnv({ PROD: false, DEV: true, VITE_DEV_AUTH_USER: 'admin' }).devUser).toBeUndefined()
    expect(configFromEnv({ PROD: false, DEV: true }).devTools).toBe(true)
  })
})
