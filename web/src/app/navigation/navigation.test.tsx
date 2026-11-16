import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter, matchRoutes } from 'react-router'
import { afterEach, describe, expect, it } from 'vitest'
import type { EffectiveAccess, SessionContext } from '../../shared/api'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'
import { App } from '../App'
import { configFromEnv, type AppConfig } from '../config'
import { buildRoutes } from '../routes'
import { destinationForPath, resolveNavigation } from './access'
import { DESTINATIONS, SAFE_DESTINATIONS, type ContractCapability, type FeatureKey } from './destinations'
import { ROLE_CONTRACTS, roleDestinationIds } from './roles'

// RM-E02 fail-closed navigation. Synthetic fixtures only; a passing test here is
// presentation evidence, never proof of business authorization.

const prod: AppConfig = configFromEnv({ PROD: true, DEV: false })
const dev: AppConfig = configFromEnv({ PROD: false, DEV: true, VITE_DEV_AUTH_USER: 'dev-user-001' })

const ALL_CAPS: ContractCapability[] = [
  'identity.user.read', 'org.person.read', 'org.employment.read', 'org.position.read', 'org.unit.read', 'org.assignment.read',
  'customers.customer.read', 'customers.merge_case.read', 'customers.identity_report.read', 'cases.case.read', 'custody.custody.read',
  'sales.report.read', 'billing.invoice.read', 'payments.attempt.read', 'billing.reconciliation.read', 'reporting.metrics.read',
  'reporting.mis.read', 'reporting.invoice_review.read', 'finance.shadow.read', 'finance.ledger.read', 'finance.reconciliation.read',
  'biawin.subscription.read', 'fulfillment.case.read', 'biawin.parity.read', 'cases.contact_outcome.record', 'org.hr_request.create',
  'org.hr_request.review', 'org.hr_request.read', 'integration.writeback.read',
  // RM-ACCESS-01: the 14 registry capabilities the contract previously omitted.
  'payments.upload.stage', 'payments.batch.submit', 'payments.submission.read', 'payments.evidence.read', 'finance.review.read',
  'commission.shadow.read', 'commission.policy.read', 'commission.own.read',
  'fulfillment.work.read', 'fulfillment.work.assign', 'fulfillment.work.execute', 'biawin.activation.read',
  'integration.provider_operation.read', 'integration.provider_operation.reconcile',
]

/** The nine destinations RM-E02 had to hide because their capability was missing from the contract. */
const FORMERLY_HIDDEN: { id: string; capability: ContractCapability; feature: FeatureKey }[] = [
  { id: 'commission.overview', capability: 'commission.shadow.read', feature: 'commission_read' },
  { id: 'commission.calculations', capability: 'commission.shadow.read', feature: 'commission_read' },
  { id: 'commission.policies', capability: 'commission.policy.read', feature: 'commission_read' },
  { id: 'commission.me', capability: 'commission.own.read', feature: 'commission_read' },
  { id: 'operations.work', capability: 'fulfillment.work.read', feature: 'fulfillment_work' },
  { id: 'operations.activations', capability: 'biawin.activation.read', feature: 'fulfillment_work' },
  { id: 'operations.providers', capability: 'integration.provider_operation.read', feature: 'fulfillment_work' },
  { id: 'payments.evidence', capability: 'payments.evidence.read', feature: 'payment_submission' },
  { id: 'finance.review', capability: 'finance.review.read', feature: 'finance_review_read' },
]

const FEATURES_ON: Record<FeatureKey, boolean> = {
  hr_read: true, customers_identity: true, sales_read: true, billing_read: true, reporting_read: true, finance_shadow_read: true,
  biawin_fulfillment_read: true, contact_outcome_write: true, hr_request_write: true, writeback_read: true, payment_submission: true,
  finance_review_read: true, commission_read: true, fulfillment_work: true,
}
const FEATURES_OFF = Object.fromEntries(Object.keys(FEATURES_ON).map(k => [k, false])) as Record<FeatureKey, boolean>

const sessionOn = (features: Partial<typeof FEATURES_ON> = FEATURES_ON, over: Record<string, unknown> = {}) => session({
  authenticated: true, auth_mode: 'session', session_state: 'active', login_enabled: true,
  session: { user_id: '00000000-0000-4000-8000-000000000001', display_name: 'کاربر ساختگی', csrf_token: 'a'.repeat(43), idle_expires_at: '2099-01-01T00:00:00Z', absolute_expires_at: '2099-01-01T00:00:00Z' },
  features: { ...FEATURES_OFF, ...features }, ...over,
})

/** Effective-access with `allow` ALLOWED (matched to `roles`) and every other capability UNAUTHORIZED. */
function evaluation(allow: Partial<Record<string, string[]>> = {}, over: Partial<EffectiveAccess> = {}): EffectiveAccess {
  const caps = [...new Set([...ALL_CAPS, ...Object.keys(allow)])]
  return {
    authenticated: true, hr_read_enabled: true,
    decisions: caps.map(c => {
      const roles = allow[c]
      return roles
        ? { capability: c as ContractCapability, allowed: true, outcome: 'SUCCESS' as const, scope: 'ORGANIZATION' as const, matched_roles: roles }
        : { capability: c as ContractCapability, allowed: false, outcome: 'UNAUTHORIZED' as const, reason_code: 'ACTION_DENIED' as const, scope: 'NONE' as const, matched_roles: [] }
    }),
    ...over,
  }
}
const blocked = (): EffectiveAccess => ({
  authenticated: true, hr_read_enabled: false,
  decisions: ALL_CAPS.map(c => ({ capability: c, allowed: false, outcome: 'BLOCKED' as const, reason_code: 'FEATURE_DISABLED' as const, scope: 'NONE' as const, matched_roles: [] })),
})
const envelope = (data: EffectiveAccess) => json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-ea01', data })
const denied = () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'test-corr-dn01' }, 403)

type Route = (req: Request) => Response | Promise<Response>
function renderAt(path: string, routes: Record<string, Route>, config = prod) {
  const { client, calls } = fakeApi({ ...healthy, ...routes }, config.devUser)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}
const nav = () => screen.getByRole('navigation', { name: 'ناوبری اصلی' })
const navLabels = () => within(nav()).queryAllByRole('link').map(a => a.textContent?.trim())
const SAFE = [SAFE_DESTINATIONS.home.label, SAFE_DESTINATIONS.system.label]

afterEach(() => { try { localStorage.clear() } catch { /* jsdom only */ } })

describe('RM-E02 resolver (pure)', () => {
  const ready = (s: ReturnType<typeof sessionOn>) => ({ status: 'ready' as const, session: s.data as SessionContext })

  it('feature ON + capability ALLOWED is the only way a destination appears', () => {
    const a = resolveNavigation(ready(sessionOn({ billing_read: true })), { state: 'READY', data: evaluation({ 'billing.invoice.read': ['seller'] }) })
    expect(a.status === 'GRANTED' && a.entries.map(e => e.id)).toEqual(['billing.invoices'])
  })

  it('RM-ACCESS-01: every destination capability is in the contract enum (no UNKNOWN left)', () => {
    expect(DESTINATIONS.filter(x => x.capability.kind === 'UNKNOWN')).toEqual([])
    for (const d of DESTINATIONS) if (d.capability.kind === 'CONTRACT') expect(ALL_CAPS).toContain(d.capability.capability)
    for (const h of FORMERLY_HIDDEN) {
      const d = DESTINATIONS.find(x => x.id === h.id)
      expect(d?.capability, h.id).toEqual({ kind: 'CONTRACT', capability: h.capability })
      expect(d?.feature, h.id).toBe(h.feature)
    }
  })

  it('M. the nine formerly hidden destinations: neither contract membership nor a feature flag alone shows them', () => {
    for (const h of FORMERLY_HIDDEN) {
      const ids = (a: ReturnType<typeof resolveNavigation>) => (a.status === 'GRANTED' ? a.entries.map(e => e.id) : [])
      // Every feature on, capability not allowed (UNAUTHORIZED): hidden.
      expect(ids(resolveNavigation(ready(sessionOn()), { state: 'READY', data: evaluation() })), h.id).not.toContain(h.id)
      // Capability allowed, its own feature off (every other feature on): hidden.
      const off = sessionOn({ ...FEATURES_ON, [h.feature]: false })
      expect(ids(resolveNavigation(ready(off), { state: 'READY', data: evaluation({ [h.capability]: ['synthetic_dev_stub'] }) })), h.id).not.toContain(h.id)
      // Server BLOCKED its capability (own flag off server-side) while the session says on: hidden.
      const ev = evaluation({ 'cases.case.read': ['seller'] })
      ev.decisions = ev.decisions.map(d => d.capability === h.capability ? { ...d, allowed: false, outcome: 'BLOCKED', reason_code: 'FEATURE_DISABLED' } : d)
      expect(ids(resolveNavigation(ready(sessionOn()), { state: 'READY', data: ev })), h.id).not.toContain(h.id)
      // Allowed by the server AND feature on: the existing route link appears.
      expect(ids(resolveNavigation(ready(sessionOn({ [h.feature]: true })), { state: 'READY', data: evaluation({ [h.capability]: ['synthetic_dev_stub'] }) })), h.id).toContain(h.id)
    }
  })

  it('a capability string outside the contract never shows anything, even when ALLOWED', () => {
    const ev = evaluation()
    ev.decisions = [...ev.decisions, { capability: 'org.assignment.apply' as ContractCapability, allowed: true, outcome: 'SUCCESS', scope: 'ORGANIZATION', matched_roles: ['hr'] }]
    expect(resolveNavigation(ready(sessionOn()), { state: 'READY', data: ev })).toEqual({ status: 'NONE', reason: 'NO_GRANT' })
  })

  it('B (navigation). HR flag off server-side: HR decisions BLOCKED, another module still allowed → only that module links', () => {
    const hr = ['identity.user.read', 'org.person.read', 'org.employment.read', 'org.position.read', 'org.unit.read', 'org.assignment.read']
    const ev = evaluation({ 'customers.customer.read': ['synthetic_dev_stub'], 'org.person.read': ['hr'] }, { hr_read_enabled: false })
    ev.decisions = ev.decisions.map(d => hr.includes(d.capability) ? { ...d, allowed: false, outcome: 'BLOCKED', reason_code: 'FEATURE_DISABLED', scope: 'NONE' } : d)
    const a = resolveNavigation(ready(sessionOn({ customers_identity: true, hr_read: false })), { state: 'READY', data: ev })
    expect(a.status === 'GRANTED' && a.entries.map(e => e.id)).toEqual(['customers.list'])
  })

  it('an ALLOWED flag without outcome SUCCESS, or SUCCESS without allowed, never counts', () => {
    const ev = evaluation()
    ev.decisions = ev.decisions.map(d => d.capability === 'billing.invoice.read' ? { ...d, allowed: true, outcome: 'UNKNOWN' } : d.capability === 'cases.case.read' ? { ...d, allowed: false, outcome: 'SUCCESS' } : d)
    expect(resolveNavigation(ready(sessionOn()), { state: 'READY', data: ev })).toEqual({ status: 'NONE', reason: 'NO_GRANT' })
  })

  it('every decision BLOCKED (e.g. every guarded feature off) fails closed with its reason', () => {
    expect(resolveNavigation(ready(sessionOn()), { state: 'READY', data: blocked() })).toEqual({ status: 'BLOCKED', reasons: ['FEATURE_DISABLED'] })
  })

  it('loading, failed, conflicting and anonymous answers expose nothing', () => {
    expect(resolveNavigation({ status: 'loading' }, { state: 'IDLE' }).status).toBe('LOADING')
    expect(resolveNavigation({ status: 'failed' }, { state: 'IDLE' }).status).toBe('ANONYMOUS')
    expect(resolveNavigation(ready(session() as ReturnType<typeof sessionOn>), { state: 'READY', data: evaluation({ 'cases.case.read': ['seller'] }) }).status).toBe('ANONYMOUS')
    expect(resolveNavigation(ready(sessionOn()), { state: 'LOADING' }).status).toBe('LOADING')
    expect(resolveNavigation(ready(sessionOn()), { state: 'FAILED' }).status).toBe('FAILED')
    expect(resolveNavigation(ready(sessionOn()), { state: 'READY', data: evaluation({ 'cases.case.read': ['seller'] }, { authenticated: false }) }).status).toBe('CONFLICT')
  })

  it('role context: one known role → SINGLE (role-mapped first); several → MULTIPLE; synthetic / none → UNKNOWN', () => {
    const single = resolveNavigation(ready(sessionOn()), { state: 'READY', data: evaluation({ 'billing.invoice.read': ['seller'], 'cases.case.read': ['seller'], 'customers.customer.read': ['seller'] }) })
    expect(single.status === 'GRANTED' && single.role).toEqual({ kind: 'SINGLE', role: 'seller' })
    expect(single.status === 'GRANTED' && single.entries.map(e => `${e.id}:${e.placement}`)).toEqual(['sales.cases:role', 'billing.invoices:role', 'customers.list:other'])
    const multi = resolveNavigation(ready(sessionOn()), { state: 'READY', data: evaluation({ 'org.person.read': ['hr'], 'billing.invoice.read': ['finance'] }) })
    expect(multi.status === 'GRANTED' && multi.role).toEqual({ kind: 'MULTIPLE', roles: ['finance', 'hr'] })
    const synthetic = resolveNavigation(ready(sessionOn()), { state: 'READY', data: evaluation({ 'org.person.read': ['synthetic_dev_stub'] }) })
    expect(synthetic.status === 'GRANTED' && synthetic.role).toEqual({ kind: 'UNKNOWN', roles: ['synthetic_dev_stub'] })
  })

  it('aria-current follows the longest matching route prefix (detail pages keep their list current)', () => {
    const items = [SAFE_DESTINATIONS.home, ...DESTINATIONS]
    expect(destinationForPath(items, '/finance/review/abc')?.to).toBe('/finance/review')
    expect(destinationForPath(items, '/finance/ledger')?.to).toBe('/finance/ledger')
    expect(destinationForPath(items, '/hr/people/p-1')?.to).toBe('/hr')
    expect(destinationForPath(items, '/')?.to).toBe('/')
    expect(destinationForPath(items, '/nope')).toBeUndefined()
  })

  it('role contracts reference only registered destinations', () => {
    for (const role of Object.values(ROLE_CONTRACTS)) for (const id of roleDestinationIds(role.key)) expect(DESTINATIONS.some(d => d.id === id), `${role.key}:${id}`).toBe(true)
  })
})

describe('RM-E02 routes and deep links', () => {
  it('every registered destination and existing detail URL still resolves to its page, not 404', () => {
    const routes = buildRoutes(prod)
    const urls = [...DESTINATIONS.map(d => d.to), SAFE_DESTINATIONS.system.to, '/hr/people/p-1', '/customers/c-1', '/sales/cases/k-1', '/billing/invoices/i-1',
      '/payments/batches/b-1', '/finance/review/a-1', '/biawin/subscriptions/s-1', '/fulfillment/cases/f-1', '/operations/work/w-1', '/hr/requests/r-1']
    for (const u of urls) {
      const m = matchRoutes(routes, u)
      expect(m?.at(-1)?.route.path, u).not.toBe('*')
    }
  })

  it('direct URL to a page the user may not read: the route renders the server refusal; the navigation offers no link to it', async () => {
    const { calls } = renderAt('/billing/invoices', {
      'GET /api/v1/session': () => json(sessionOn()),
      'GET /api/v1/authorization/effective-access': () => envelope(evaluation()),
      'GET /api/v1/invoices': denied,
    })
    expect(await screen.findByRole('heading', { level: 1, name: 'فاکتورها' })).toBeTruthy()
    expect(await screen.findAllByText('دسترسی به این بخش ندارید')).not.toHaveLength(0)
    expect(navLabels()).toEqual(SAFE)
    expect(within(nav()).queryByRole('link', { current: 'page' })).toBeNull()
    expect(calls.every(c => c.method === 'GET')).toBe(true)
  })
})

describe('RM-E02 fail-closed shell', () => {
  it('1. no ACTIVE grant: only safe links and an explicit Persian no-access state', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(sessionOn()), 'GET /api/v1/authorization/effective-access': () => envelope(evaluation()) })
    expect(await screen.findByText('دسترسی فعالی برای شما ثبت نشده است')).toBeTruthy()
    expect(navLabels()).toEqual(SAFE)
    expect(within(screen.getByRole('navigation', { name: 'ناوبری موبایل' })).queryAllByRole('link').map(a => a.textContent)).toEqual(['خانه', 'سامانه'])
  })

  it('2. feature enabled but not authorised: no link', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(sessionOn({ finance_shadow_read: true, sales_read: true })), 'GET /api/v1/authorization/effective-access': () => envelope(evaluation({ 'cases.case.read': ['seller'] })) })
    await waitFor(() => expect(navLabels()).toEqual(['خانه', 'پرونده‌های فروش', 'وضعیت سامانه']))
    expect(within(nav()).queryByRole('link', { name: /دفتر سایه مالی/ })).toBeNull()
  })

  it('3. capability allowed but feature disabled: no link, and the reason is honest', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(sessionOn({ billing_read: false })), 'GET /api/v1/authorization/effective-access': () => envelope(evaluation({ 'billing.invoice.read': ['seller'] })) })
    expect(await screen.findByText('بخش‌های مجاز شما هنوز فعال نشده‌اند')).toBeTruthy()
    expect(navLabels()).toEqual(SAFE)
  })

  it('4a. single verified role: role crumb, role-mapped links first, deep link marked current', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(sessionOn()), 'GET /api/v1/authorization/effective-access': () => envelope(evaluation({ 'billing.invoice.read': ['seller'], 'cases.case.read': ['seller'] })) })
    expect(await screen.findByText('نقش: فروشنده')).toBeTruthy()
    await waitFor(() => expect(navLabels()).toEqual(['خانه', 'پرونده‌های فروش', 'فاکتورها', 'وضعیت سامانه']))
    expect(screen.getByRole('heading', { level: 2, name: 'بخش‌های نقش شما' })).toBeTruthy()
    fireEvent.click(within(nav()).getByRole('link', { name: 'فاکتورها' }))
    await waitFor(() => expect(within(nav()).getByRole('link', { name: 'فاکتورها' }).getAttribute('aria-current')).toBe('page'))
  })

  it('4b. unknown role context (synthetic grant): no role named, exact allowed set', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(sessionOn()), 'GET /api/v1/authorization/effective-access': () => envelope(evaluation({ 'org.person.read': ['synthetic_dev_stub'] })) })
    expect(await screen.findByText(/نقش شما از پاسخ سرور مشخص نیست/)).toBeTruthy()
    expect(screen.queryByText(/^نقش:/)).toBeNull()
    expect(navLabels()).toEqual(['خانه', 'نیروها', 'وضعیت سامانه'])
  })

  it('4c. evaluation failed, BLOCKED or contradicting the session: fail closed with a retry', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(sessionOn()), 'GET /api/v1/authorization/effective-access': () => json({ type: '/problems/internal', title: 'x', status: 500, reason_code: 'OUTCOME_UNKNOWN', correlation_id: 'c-1' }, 500, 'application/problem+json') })
    expect(await screen.findByText('ارزیابی دسترسی انجام نشد')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'بررسی دوباره' })).toBeTruthy()
    expect(navLabels()).toEqual(SAFE)
  })

  it('4d. effective-access BLOCKED for every capability (every feature off) shows the server reason and no business link', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(sessionOn()), 'GET /api/v1/authorization/effective-access': () => envelope(blocked()) })
    expect(await screen.findByText('ارزیابی دسترسی در سرور در دسترس نیست')).toBeTruthy()
    expect(screen.getByText('FEATURE_DISABLED')).toBeTruthy()
    expect(navLabels()).toEqual(SAFE)
  })

  it('5. several roles: no role is picked, no switcher, the exact server-allowed set is listed', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(sessionOn()), 'GET /api/v1/authorization/effective-access': () => envelope(evaluation({ 'org.person.read': ['hr'], 'billing.invoice.read': ['finance'] })) })
    expect(await screen.findByText(/سیاست انتخاب نقش هنوز تعریف نشده است/)).toBeTruthy()
    expect(screen.queryByText(/^نقش:/)).toBeNull()
    expect(screen.queryByRole('combobox')).toBeNull()
    expect(screen.queryByRole('button', { name: /نقش/ })).toBeNull()
    expect(navLabels()).toEqual(['خانه', 'نیروها', 'فاکتورها', 'وضعیت سامانه'])
  })

  it('6. session expired or revoked after a denial: login page with the expiry notice, business links gone', async () => {
    let live = true
    renderAt('/hr', {
      'GET /api/v1/session': () => json(live ? sessionOn() : session({ auth_mode: 'session', authenticated: false, session_state: 'invalid', login_enabled: true, features: FEATURES_ON })),
      'GET /api/v1/authorization/effective-access': () => envelope(evaluation({ 'org.person.read': ['hr'] })),
      'GET /api/v1/organization/persons': () => { live = false; return json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'c-401' }, 401) },
    })
    expect(await screen.findByText('نشست شما پایان یافته است')).toBeTruthy()
    expect(navLabels()).toEqual(SAFE)
  })

  it('12. navigating through the shell never issues a write', async () => {
    const { calls } = renderAt('/', { 'GET /api/v1/session': () => json(sessionOn()), 'GET /api/v1/authorization/effective-access': () => envelope(evaluation({ 'billing.invoice.read': ['seller'], 'org.person.read': ['seller'] })) })
    await waitFor(() => expect(navLabels()).toContain('فاکتورها'))
    for (const name of ['فاکتورها', 'نیروها', 'وضعیت سامانه', 'خانه']) {
      fireEvent.click(within(nav()).getByRole('link', { name }))
      await waitFor(() => expect(within(nav()).getByRole('link', { name }).getAttribute('aria-current')).toBe('page'))
    }
    expect(calls.filter(c => c.method !== 'GET')).toEqual([])
  })

  it('13. production: no DEV link or identity header; dev capability_hints, localStorage and query parameters add nothing', async () => {
    localStorage.setItem('role', 'finance')
    localStorage.setItem('crm.role', 'finance')
    const { calls } = renderAt('/?role=finance&capability=finance.ledger.read', {
      'GET /api/v1/session': () => json(sessionOn(FEATURES_ON, { capability_hints: ['dev.finance.ledger.read'] })),
      'GET /api/v1/authorization/effective-access': () => envelope(evaluation()),
    })
    expect(await screen.findByText('دسترسی فعالی برای شما ثبت نشده است')).toBeTruthy()
    expect(navLabels()).toEqual(SAFE)
    expect(screen.queryByText(/DEV/)).toBeNull()
    for (const c of calls) expect(c.headers.has('X-Dev-User')).toBe(false)
    expect(matchRoutes(buildRoutes(prod), '/dev/design-system')?.at(-1)?.route.path).toBe('*')
  })

  it('13b. DEV build: DEV tool links are marked and never mix with authorised destinations', async () => {
    renderAt('/', { 'GET /api/v1/session': () => json(session({ auth_mode: 'dev_stub', authenticated: true, principal: { principal_id: 'dev-user-001', display_name: 'Dev Operator', synthetic: true }, features: FEATURES_OFF })), 'GET /api/v1/authorization/effective-access': () => envelope(evaluation()) }, dev)
    expect(await screen.findByText('دسترسی فعالی برای شما ثبت نشده است')).toBeTruthy()
    expect(navLabels()).toEqual(['خانه', 'وضعیت سامانه', 'نمونه اجزا · DEV', 'سامانه طراحی · DEV'])
  })

  it('14. while the session or the evaluation is pending nothing protected is requested or shown', async () => {
    let release: (() => void) | undefined
    const { calls } = renderAt('/', {
      'GET /api/v1/session': () => json(sessionOn()),
      'GET /api/v1/authorization/effective-access': () => new Promise<Response>(r => { release = () => r(envelope(evaluation({ 'billing.invoice.read': ['seller'] }))) }),
    })
    expect(await screen.findByText('در حال بررسی دسترسی‌ها…')).toBeTruthy()
    expect(navLabels()).toEqual(SAFE)
    expect(calls.map(c => c.url).sort()).toEqual(['/api/v1/authorization/effective-access', '/api/v1/session'])
    release?.()
    await waitFor(() => expect(navLabels()).toContain('فاکتورها'))
  })

  it('14b. anonymous visitors never trigger the evaluation call', async () => {
    const { calls } = renderAt('/', {})
    expect(await screen.findByText('هویتی برای این نشست ثبت نشده است')).toBeTruthy()
    expect(calls.some(c => c.url === '/api/v1/authorization/effective-access')).toBe(false)
  })
})
