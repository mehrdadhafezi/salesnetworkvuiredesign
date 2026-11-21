import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import type { EffectiveAccess } from '../../shared/api'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'
import { HR_NAV_LABEL } from './workspace/SectionNav'
import { lifecycle } from './workspace/requests'

// RM-P04 HR operational workspace. Synthetic fixtures only; a passing test is
// presentation evidence, never proof that production authorizes these reads.

const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const prov = { source_system: 'wp', source_spec: 'legacy.synthetic.v1', source_key: '101', batch_id: ID(900), imported_at: '2026-10-29T09:00:00Z', refreshed_at: '2026-10-29T09:00:00Z', presence: 'PRESENT' }
const HR_CAPS = ['identity.user.read', 'org.person.read', 'org.employment.read', 'org.position.read', 'org.unit.read', 'org.assignment.read']
const REQ_CAPS = ['org.hr_request.read', 'org.hr_request.create', 'org.hr_request.review']
const allowedActions = (caps: string[]) => Object.fromEntries(caps.map(c => [c, { allowed: true }]))
const env = (extra: Record<string, unknown>, caps = HR_CAPS) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-9999', allowed_actions: allowedActions(caps), ...extra })
const denied = () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'test-corr-dn01' }, 403)

type Route = (req: Request) => Response | Promise<Response>

function evaluation(allow: Record<string, string[]>, extra: string[] = []): EffectiveAccess {
  const caps = [...new Set([...HR_CAPS, ...REQ_CAPS, 'billing.invoice.read', ...extra, ...Object.keys(allow)])]
  return {
    authenticated: true, hr_read_enabled: true,
    decisions: caps.map(c => allow[c]
      ? { capability: c as never, allowed: true, outcome: 'SUCCESS' as const, scope: 'ORGANIZATION' as const, matched_roles: allow[c] ?? [] }
      : { capability: c as never, allowed: false, outcome: 'UNAUTHORIZED' as const, reason_code: 'ACTION_DENIED' as const, scope: 'NONE' as const, matched_roles: [] }),
  }
}
const hrAll = (roles = ['hr']) => evaluation(Object.fromEntries([...HR_CAPS, ...REQ_CAPS].map(c => [c, roles])))

const features = { hr_read: true, customers_identity: false, sales_read: false, billing_read: false, reporting_read: false, finance_shadow_read: false,
  biawin_fulfillment_read: false, contact_outcome_write: false, hr_request_write: true, writeback_read: false, payment_submission: false,
  finance_review_read: false, commission_read: false, fulfillment_work: false }

const ME = ID(201)
const sessionOn = (over: Record<string, unknown> = {}) => session({
  authenticated: true, auth_mode: 'session', session_state: 'active', login_enabled: true,
  session: { user_id: ME, display_name: 'کاربر ساختگی', csrf_token: 'a'.repeat(43), idle_expires_at: '2099-01-01T00:00:00Z', absolute_expires_at: '2099-01-01T00:00:00Z' },
  features, ...over,
})

function person(n: number, over: Record<string, unknown> = {}, user: Record<string, unknown> = {}) {
  return {
    person_id: ID(n), display_name: `Synthetic Person ${n}`, finding_count: 0, provenance: prov, manager_state: 'NONE',
    employment: { employment_id: ID(n + 100), status: 'ACTIVE', status_raw: 'active', type_raw: null, employee_code: `SYN-${n}`, hire_date: '2025-01-15',
      hire_date_raw: '2025-01-15', termination_date: null, termination_date_raw: null, is_active: true,
      user: { user_id: ID(n + 200), link_state: 'LINKED', ...user }, position: { id: ID(500), slug: 'hr', label: 'Synthetic HR Position' }, position_state: 'RESOLVED',
      level: { id: ID(600), slug: 'l1', label: 'Synthetic Level 1' }, level_state: 'RESOLVED', provenance: prov },
    ...over,
  }
}
const lookup = (id: number, label: string, over: Record<string, unknown> = {}) => ({ id: ID(id), slug: label.toLowerCase().replace(/ /g, '_'), label, description: null, sort_order: 1, is_active: true, is_system: false, provenance: prov, ...over })

/** Persons by user id: user 201 → person 1; 202 → two persons (shared); anything else → none. */
function personsRoute(req: Request): Response {
  const u = new URL(req.url).searchParams.get('user_id')
  if (u === ID(201)) return json(env({ items: [person(1)], page: { has_more: false } }))
  if (u === ID(202)) return json(env({ items: [person(2, {}, { user_id: ID(202), link_state: 'SHARED_USER' }), person(3, {}, { user_id: ID(202), link_state: 'SHARED_USER' })], page: { has_more: false } }))
  if (u) return json(env({ items: [], page: { has_more: false } }))
  return json(env({ items: [person(1), person(2)], page: { has_more: false } }))
}

const hrRequest = (n: number, status: string, over: Record<string, unknown> = {}) => ({
  request_id: ID(700 + n), request_type: 'transfer_seller', status, subject_user_id: ID(202), subject_parent_user_id: ID(201), target_parent_user_id: ID(209),
  requested_by_user_id: ID(201), current_reviewer_user_id: status === 'pending_review' ? ID(201) : null, reason: 'synthetic reason', reviewer_notes: null, version: 1,
  created_at: '2026-11-02T09:00:00Z', updated_at: '2026-11-02T09:00:00Z', blocked_actions: status.startsWith('pending')
    ? [{ action: 'apply_final', blocked_by: ['HR_APPLY_IMPLEMENTATION_BLOCKED', 'OPD-05', 'SD-04'] }] : [],
  allowed_actions: { approve_step: { allowed: false, reason_code: 'ROW_SCOPE_DENIED' }, reject: { allowed: false, reason_code: 'ROW_SCOPE_DENIED' } }, ...over,
})

function base(ev: EffectiveAccess = hrAll(), over: Record<string, Route> = {}, sess = sessionOn()): Record<string, Route> {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(sess),
    'GET /api/v1/authorization/effective-access': () => json({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-ea01', data: ev }),
    'GET /api/v1/organization/persons': personsRoute,
    'GET /api/v1/organization/positions': () => json(env({ items: [lookup(500, 'Synthetic HR Position'), lookup(501, 'Synthetic Retired Position', { is_active: false })], page: { has_more: false } })),
    'GET /api/v1/organization/levels': () => json(env({ items: [lookup(600, 'Synthetic Level 1')], page: { has_more: false } })),
    'GET /api/v1/organization/org-units': () => json(env({ items: [], page: { has_more: false } })),
    ...over,
  }
}

function renderAt(path: string, routes: Record<string, Route>) {
  const { client, calls } = fakeApi(routes)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls, router }
}

const hrNav = () => screen.getByRole('navigation', { name: HR_NAV_LABEL })
const navLabels = () => within(hrNav()).getAllByRole('link').map(a => a.textContent)
const onlyGets = (calls: { method: string }[]) => calls.every(c => c.method === 'GET')

describe('RM-P04 workspace navigation', () => {
  it('lists every section the server allows, marks the current one, and navigates between sections', async () => {
    const { calls } = renderAt('/hr', base())
    expect(await screen.findByRole('link', { name: 'Synthetic Person 1' })).toBeTruthy()
    await waitFor(() => expect(navLabels()).toEqual(['نیروها', 'ساختار سازمانی', 'انتساب‌ها و گزارش‌دهی', 'سمت‌ها و سطوح', 'درخواست‌ها', 'استثناها', 'دسترسی من']))
    expect(within(hrNav()).getByRole('link', { current: 'page' }).textContent).toBe('نیروها')
    // The design status is stated, never implied frozen.
    expect(screen.getByText('طراحی منابع انسانی: آماده بازبینی (منجمد نشده)')).toBeTruthy()
    fireEvent.click(within(hrNav()).getByRole('link', { name: 'سمت‌ها و سطوح' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'سمت‌ها و سطوح' })).toBeTruthy()
    expect(within(hrNav()).getByRole('link', { current: 'page' }).textContent).toBe('سمت‌ها و سطوح')
    fireEvent.click(within(hrNav()).getByRole('link', { name: 'دسترسی من' }))
    expect(await screen.findByRole('heading', { level: 1, name: 'دسترسی من' })).toBeTruthy()
    expect(onlyGets(calls)).toBe(true)
  })

  it('fails closed: without ALLOWED decisions only the current section and "دسترسی من" are offered', async () => {
    renderAt('/hr/assignments', base(evaluation({}), { 'GET /api/v1/organization/assignments': denied }))
    expect(await screen.findByRole('heading', { level: 1, name: 'انتساب‌ها و گزارش‌دهی' })).toBeTruthy()
    await waitFor(() => expect(navLabels()).toEqual(['انتساب‌ها و گزارش‌دهی', 'دسترسی من']))
    expect(await screen.findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(screen.queryByRole('table')).toBeNull()
  })

  it('requests section is hidden when its feature switch is off, even with the capability', async () => {
    renderAt('/hr', base(hrAll(), {}, sessionOn({ features: { ...features, hr_request_write: false } })))
    expect(await screen.findByRole('link', { name: 'Synthetic Person 1' })).toBeTruthy()
    await waitFor(() => expect(navLabels()).toContain('استثناها'))
    expect(navLabels()).not.toContain('درخواست‌ها')
  })
})

describe('RM-P04 personnel directory', () => {
  it('sends level and active-flag filters to the server and keeps a shared-account view honest', async () => {
    const { calls } = renderAt('/hr', base())
    expect(await screen.findByRole('link', { name: 'Synthetic Person 1' })).toBeTruthy()
    const form = screen.getByRole('form', { name: 'فیلتر نیروها' })
    await waitFor(() => expect(within(form).getByLabelText('سطح')).toBeTruthy())
    fireEvent.change(within(form).getByLabelText('سطح'), { target: { value: ID(600) } })
    fireEvent.change(within(form).getByLabelText('پرچم فعال (قدیمی)'), { target: { value: 'false' } })
    fireEvent.submit(form)
    await waitFor(() => {
      const last = calls.filter(c => c.url === '/api/v1/organization/persons').at(-1)
      expect(last?.search.get('level_id')).toBe(ID(600))
      expect(last?.search.get('is_active')).toBe('false')
    })
  })

  it('a shared account lists every linked person as a conflict, never one merged person', async () => {
    const { calls } = renderAt(`/hr?user_id=${ID(202)}`, base())
    expect(await screen.findByText('فقط پرونده‌های متصل به یک حساب کاربری')).toBeTruthy()
    expect(await screen.findByRole('link', { name: 'Synthetic Person 2' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Synthetic Person 3' })).toBeTruthy()
    expect(screen.getAllByText('تعارض: حساب مشترک')).toHaveLength(2)
    expect(calls.find(c => c.url === '/api/v1/organization/persons')?.search.get('user_id')).toBe(ID(202))
  })
})

describe('RM-P04 organisation: assignments and positions', () => {
  const assignment = (n: number, over: Record<string, unknown> = {}) => ({
    assignment_id: ID(800 + n), child: { state: 'RESOLVED', person: { person_id: ID(1), employment_id: ID(101), display_name: 'Synthetic Person 1' } },
    parent: { state: 'RESOLVED', person: { person_id: ID(2), employment_id: ID(102), display_name: 'Synthetic Person 2' } },
    relationship_kind: 'REPORTS_TO', relationship_raw: 'reports_to', effective_from_raw: '2025-01-01 00:00:00', effective_to_raw: null, interval_zone: 'UNKNOWN',
    is_current: true, current_raw: '1', finding_kinds: [], provenance: prov, ...over,
  })

  it('assignments: person scope, relation filter, missing ends and unknown current flags stay visible', async () => {
    const { calls } = renderAt(`/hr/assignments?person_id=${ID(1)}`, base(hrAll(), {
      'GET /api/v1/organization/assignments': (req: Request) => {
        const c = new URL(req.url).searchParams.get('cursor')
        return json(env(c ? { items: [assignment(3, { is_current: false })], page: { has_more: false } } : {
          items: [assignment(1), assignment(2, { parent: { state: 'MISSING' }, is_current: null, current_raw: 'x', finding_kinds: ['MISSING_PARENT'] })],
          page: { has_more: true, next_cursor: 'a-2' },
        }))
      },
      [`GET /api/v1/organization/persons/${ID(1)}`]: () => json(env({ data: { person: person(1), assignments: [], direct_reports: [], history: [], findings: [] } })),
    }))
    const table = await screen.findByRole('table', { name: 'انتساب‌های گزارش‌دهی' })
    expect(within(table).getByText('مدیر ناموجود')).toBeTruthy()
    expect(within(table).getByText('نامعلوم (x)')).toBeTruthy()
    expect(within(table).getByText('مدیر ثبت‌شده وجود ندارد')).toBeTruthy()
    await waitFor(() => expect(document.querySelector('.hr-scope a')?.textContent).toBe('Synthetic Person 1'))
    let last = calls.filter(c => c.url === '/api/v1/organization/assignments').at(-1)
    expect(last?.search.get('person_id')).toBe(ID(1))
    fireEvent.change(screen.getByLabelText('نقش این نیرو در انتساب'), { target: { value: 'PARENT' } })
    fireEvent.submit(screen.getByRole('form', { name: 'فیلتر انتساب‌ها' }))
    await waitFor(() => {
      last = calls.filter(c => c.url === '/api/v1/organization/assignments').at(-1)
      expect(last?.search.get('relation')).toBe('PARENT')
      expect(last?.search.get('person_id')).toBe(ID(1))
    })
    fireEvent.click(await screen.findByRole('button', { name: 'صفحه بعد' }))
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/organization/assignments').at(-1)?.search.get('cursor')).toBe('a-2'))
    expect(onlyGets(calls)).toBe(true)
    expect(screen.queryAllByRole('button').filter(b => /افزودن|ویرایش|حذف|انتقال|انتساب جدید|ذخیره/.test(b.textContent ?? ''))).toHaveLength(0)
  })

  it('positions and levels: read only, linked to the filtered directory, never presented as roles', async () => {
    renderAt('/hr/positions', base())
    const positions = await screen.findByRole('table', { name: 'سمت‌ها' })
    expect(within(positions).getByText('Synthetic Retired Position')).toBeTruthy()
    expect(within(positions).getByText('غیرفعال')).toBeTruthy()
    const link = within(positions).getAllByRole('link', { name: 'مشاهده نیروها' })[0]
    expect(link?.getAttribute('href')).toBe(`/hr?position_id=${ID(500)}`)
    expect(within(await screen.findByRole('table', { name: 'سطوح' })).getByText('Synthetic Level 1')).toBeTruthy()
    expect(screen.getByText(/سمت سازمانی ≠ نقش ≠ مجوز ≠ اعتبارنامه/)).toBeTruthy()
    expect(screen.queryAllByRole('button').filter(b => /افزودن|ویرایش|حذف|غیرفعال‌سازی|نگاشت/.test(b.textContent ?? ''))).toHaveLength(0)
  })
})

describe('RM-P04 person profile', () => {
  const detail = (user: Record<string, unknown> = {}) => () => json(env({ data: { person: person(2, {}, user), assignments: [], direct_reports: [], history: [], findings: [] } }))

  it('shows the account separately from the person and the caller-scoped requests of that account', async () => {
    const { calls } = renderAt(`/hr/people/${ID(2)}`, base(hrAll(), {
      [`GET /api/v1/organization/persons/${ID(2)}`]: detail({ user_id: ID(202), link_state: 'SHARED_USER' }),
      [`GET /api/v1/identity/users/${ID(202)}`]: () => json(env({ data: { user_id: ID(202), display_name: 'Synthetic Account 202', account_state: 'SHADOW_IMPORTED', login_enabled: false, provenance: prov } })),
      'GET /api/v1/hr/requests': () => json(env({ rollout_mode: 'CONTROLLED_WRITEBACK', items: [hrRequest(1, 'pending_hr')], page: { has_more: false } }, REQ_CAPS)),
    }))
    const account = await screen.findByRole('region', { name: 'حساب کاربری (هویت ورود)' })
    expect(await within(account).findByText('Synthetic Account 202')).toBeTruthy()
    expect(within(account).getByText('ورود فعال نیست')).toBeTruthy()
    expect(within(account).getByRole('link', { name: 'همه پرونده‌های این حساب' }).getAttribute('href')).toBe(`/hr?user_id=${ID(202)}`)
    const reqs = screen.getByRole('region', { name: 'درخواست‌های منابع انسانی این نیرو' })
    expect(await within(reqs).findByRole('link', { name: 'جابجایی فروشنده' })).toBeTruthy()
    expect(within(reqs).getByText('در انتظار منابع انسانی (اعمال نشده)')).toBeTruthy()
    expect(calls.find(c => c.url === '/api/v1/hr/requests')?.search.get('subject_user_id')).toBe(ID(202))
    expect(screen.getByRole('link', { name: 'همه انتساب‌های این نیرو' }).getAttribute('href')).toBe(`/hr/assignments?person_id=${ID(2)}`)
    expect(screen.queryAllByRole('button').filter(b => /رمز|بازنشانی|ورود به‌جای|ساخت حساب|غیرفعال/.test(b.textContent ?? ''))).toHaveLength(0)
  })

  it('never requests HR requests or account details the evaluation does not allow (no data in the DOM)', async () => {
    const { calls } = renderAt(`/hr/people/${ID(2)}`, base(evaluation(Object.fromEntries(['org.person.read', 'org.employment.read', 'org.assignment.read'].map(c => [c, ['hr']]))), {
      [`GET /api/v1/organization/persons/${ID(2)}`]: () => json(env({ data: { person: person(2), assignments: [], direct_reports: [], history: [], findings: [] } },
        ['org.person.read', 'org.employment.read', 'org.assignment.read'])),
    }))
    const reqs = await screen.findByRole('region', { name: 'درخواست‌های منابع انسانی این نیرو' })
    await waitFor(() => expect(within(reqs).getByText('دسترسی به درخواست‌ها ندارید')).toBeTruthy())
    expect(screen.getByText('جزئیات حساب برای شما قابل نمایش نیست')).toBeTruthy()
    expect(calls.some(c => c.url === '/api/v1/hr/requests')).toBe(false)
    expect(calls.some(c => c.url.startsWith('/api/v1/identity/users'))).toBe(false)
  })
})

describe('RM-P04 HR requests', () => {
  it('status queues are server filters; people are resolved, shared accounts stay a conflict; nothing is written', async () => {
    const { calls } = renderAt('/hr/requests', base(hrAll(), {
      'GET /api/v1/hr/requests': (req: Request) => {
        const st = new URL(req.url).searchParams.get('status')
        const items = st === 'pending_hr' ? [hrRequest(2, 'pending_hr')] : [hrRequest(1, 'pending_review'), hrRequest(2, 'pending_hr')]
        return json(env({ rollout_mode: 'CONTROLLED_WRITEBACK', items, page: { has_more: false } }, REQ_CAPS))
      },
    }))
    const table = await screen.findByRole('table', { name: 'درخواست‌های منابع انسانی' })
    await waitFor(() => expect(within(table).getAllByRole('link', { name: 'Synthetic Person 1' }).length).toBeGreaterThan(0))
    expect(within(table).getAllByText('تعارض: حساب مشترک بین 2 پرونده').length).toBeGreaterThan(0)
    // Target 209 has no person: shown raw, flagged, never guessed. (Not in the list columns; checked on detail.)
    fireEvent.click(screen.getByRole('button', { name: 'در انتظار منابع انسانی' }))
    await waitFor(() => expect(calls.filter(c => c.url === '/api/v1/hr/requests').at(-1)?.search.get('status')).toBe('pending_hr'))
    await waitFor(() => expect(within(screen.getByRole('table', { name: 'درخواست‌های منابع انسانی' })).getAllByRole('row')).toHaveLength(2))
    // One lookup per distinct user, not per cell.
    const lookups = calls.filter(c => c.url === '/api/v1/organization/persons').map(c => c.search.get('user_id'))
    expect(new Set(lookups).size).toBe(lookups.length)
    expect(onlyGets(calls)).toBe(true)
  })

  it('without the employment read, accounts are shown as raw ids and no person lookup is made', async () => {
    const { calls } = renderAt('/hr/requests', base(evaluation(Object.fromEntries(REQ_CAPS.map(c => [c, ['supervisor']]))), {
      'GET /api/v1/hr/requests': () => json(env({ rollout_mode: 'CONTROLLED_WRITEBACK', items: [hrRequest(1, 'pending_review')], page: { has_more: false } }, REQ_CAPS)),
    }))
    const table = await screen.findByRole('table', { name: 'درخواست‌های منابع انسانی' })
    expect(within(table).getAllByText(ID(201)).length).toBeGreaterThan(0)
    expect(calls.some(c => c.url === '/api/v1/organization/persons')).toBe(false)
  })

  it('detail: lifecycle, requested vs applied, unresolved target flagged, final apply never a control', async () => {
    renderAt(`/hr/requests/${ID(702)}`, base(hrAll(), {
      [`GET /api/v1/hr/requests/${ID(702)}`]: () => json(env({ rollout_mode: 'CONTROLLED_WRITEBACK', request: hrRequest(2, 'pending_hr'),
        history: [
          { event_id: ID(720), command_id: ID(721), action: 'created', to_status: 'pending_review', after_version: 1, actor_user_id: ID(201), correlation_id: 'c', occurred_at: '2026-11-02T09:00:00Z' },
          { event_id: ID(722), command_id: ID(723), action: 'approved_step', from_status: 'pending_review', to_status: 'pending_hr', after_version: 2, actor_user_id: ID(201), notes: 'synthetic note', correlation_id: 'c', occurred_at: '2026-11-02T10:00:00Z' },
        ] }, REQ_CAPS)),
    }))
    const steps = await screen.findByRole('list', { name: 'مراحل درخواست' })
    expect(within(steps).getByText('بررسی منابع انسانی').closest('li')?.getAttribute('aria-current')).toBe('step')
    expect(within(steps).getByText('اعمال نهایی').closest('li')?.getAttribute('data-state')).toBe('blocked')
    expect(screen.getByText('اعمال نشده. سلسله‌مراتب، همکاری، حساب و دسترسی تغییری نکرده‌اند.')).toBeTruthy()
    expect(await screen.findAllByText('پرونده نیروی متصل یافت نشد')).not.toHaveLength(0)
    expect(screen.getByText(/synthetic note/)).toBeTruthy()
    expect(screen.getByText('اعمال نهایی منابع انسانی')).toBeTruthy()
    expect(screen.queryAllByRole('button').filter(b => /اعمال|تایید مرحله/.test(b.textContent ?? ''))).toHaveLength(0)
  })

  it('lifecycle never shows apply as done unless the server reports it', () => {
    for (const s of ['pending_review', 'pending_hr', 'rejected', 'cancelled'] as const) expect(lifecycle(s).at(-1)?.state).not.toBe('done')
    expect(lifecycle('approved').at(-1)?.state).toBe('done')
    expect(lifecycle('failed').at(-1)?.state).toBe('stopped')
    expect(lifecycle('rejected')[1]?.state).toBe('stopped')
  })
})

describe('RM-P04 access visibility (دسترسی من)', () => {
  it('keeps credential, grant, role, position and capability apart; position grants nothing', async () => {
    const ev = evaluation({ ...Object.fromEntries(HR_CAPS.map(c => [c, ['hr']])), 'org.hr_request.read': ['synthetic_dev_stub'] })
    ev.decisions.push({ capability: 'finance.review.read', allowed: false, outcome: 'BLOCKED', reason_code: 'FEATURE_DISABLED', scope: 'NONE', matched_roles: [] })
    const { calls } = renderAt('/hr/access', base(ev))
    expect(await screen.findByRole('heading', { level: 1, name: 'دسترسی من' })).toBeTruthy()
    const concepts = await screen.findByLabelText('مفاهیم دسترسی')
    expect(within(concepts).getByText('نشست ورود با حساب کاربری')).toBeTruthy()
    expect(within(concepts).getByText('منابع انسانی')).toBeTruthy()
    expect(within(concepts).getByText(/واگذاری مستقیم بدون نقش: synthetic_dev_stub/)).toBeTruthy()
    // Own position: looked up by the session user id, shown as an HR fact only.
    const pos = within(concepts).getByRole('region', { name: 'سمت سازمانی' })
    expect(await within(pos).findByRole('link', { name: 'Synthetic Person 1' })).toBeTruthy()
    expect(within(pos).getByText(/Synthetic HR Position/)).toBeTruthy()
    expect(within(pos).getByText('سمت یک واقعیت منابع انسانی است؛ هیچ مجوزی از آن مشتق نمی‌شود.')).toBeTruthy()
    expect(calls.find(c => c.url === '/api/v1/organization/persons')?.search.get('user_id')).toBe(ME)
    const table = screen.getByRole('table', { name: 'ارزیابی مجوزها' })
    expect(within(table).getByText('finance.review.read').closest('tr')?.textContent).toContain('مسدود (پرچم ویژگی)')
    fireEvent.click(screen.getByRole('button', { name: /^مسدود/ }))
    await waitFor(() => expect(within(screen.getByRole('table', { name: 'ارزیابی مجوزها' })).getAllByRole('row')).toHaveLength(2))
    expect(screen.queryAllByRole('button').filter(b => /اعطا|لغو|افزودن نقش|حذف نقش|ذخیره/.test(b.textContent ?? ''))).toHaveLength(0)
    expect(onlyGets(calls)).toBe(true)
    // The evaluation is the shell's own (one request), never a second fetch.
    expect(calls.filter(c => c.url === '/api/v1/authorization/effective-access')).toHaveLength(1)
  })

  it('a failed evaluation is unknown, never allowed; anonymous visitors see no evaluation', async () => {
    renderAt('/hr/access', base(hrAll(), { 'GET /api/v1/authorization/effective-access': () => json({ type: '/problems/x', title: 'x', status: 500, reason_code: 'CONTRACT_INVALID', correlation_id: 'c-1' }, 500, 'application/problem+json') }))
    expect(await screen.findByText('ارزیابی دسترسی دریافت نشد')).toBeTruthy()
    expect(screen.queryByRole('table')).toBeNull()
    document.body.innerHTML = ''
    const { calls } = renderAt('/hr/access', base(hrAll(), {}, session()))
    expect(await screen.findByText('ابتدا وارد شوید')).toBeTruthy()
    expect(calls.some(c => c.url === '/api/v1/authorization/effective-access')).toBe(false)
  })
})
