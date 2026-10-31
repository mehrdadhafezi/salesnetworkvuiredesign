import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import { createMemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'
import { App } from '../../app/App'
import { configFromEnv } from '../../app/config'
import { buildRoutes } from '../../app/routes'
import { fakeApi, healthy, json, session } from '../../test/fakeApi'

const config = configFromEnv({ PROD: true, DEV: false })
const ID = (n: number) => `00000000-0000-4000-8000-${String(n).padStart(12, '0')}`
const prov = { source_system: 'wp', source_spec: 'legacy.synthetic.v1', source_key: '101', batch_id: ID(900), imported_at: '2026-10-29T09:00:00Z', refreshed_at: '2026-10-29T09:00:00Z', presence: 'PRESENT' }
const allAllowed = Object.fromEntries(['identity.user.read', 'org.person.read', 'org.employment.read', 'org.position.read', 'org.unit.read', 'org.assignment.read'].map(c => [c, { allowed: true }]))

function person(n: number, over: Record<string, unknown> = {}) {
  return {
    person_id: ID(n), display_name: `Synthetic Person ${n}`, finding_count: 0, provenance: prov, manager_state: 'NONE',
    employment: { employment_id: ID(n + 100), status: 'ACTIVE', status_raw: 'active', type_raw: null, employee_code: `SYN-${n}`, hire_date: '2025-01-15',
      hire_date_raw: '2025-01-15', termination_date: null, termination_date_raw: null, is_active: true,
      user: { user_id: ID(n + 200), link_state: 'LINKED' }, position: { id: ID(500), slug: 'seller', label: 'Synthetic Seller' }, position_state: 'RESOLVED', provenance: prov },
    ...over,
  }
}

const env = (extra: Record<string, unknown>) => ({ outcome: 'SUCCESS', reasons: [], correlation_id: 'test-corr-9999', allowed_actions: allAllowed, ...extra })

function routes(over: Record<string, (req: Request) => Response> = {}, hrRead = true) {
  return {
    ...healthy,
    'GET /api/v1/session': () => json(session({ features: { hr_read: hrRead } })),
    'GET /api/v1/organization/persons': (req: Request) => {
      const cursor = new URL(req.url).searchParams.get('cursor')
      return json(env(cursor
        ? { items: [person(3)], page: { has_more: false } }
        : { items: [person(1), person(2, { manager_state: 'MULTIPLE', finding_count: 2 })], page: { has_more: true, next_cursor: 'c-2' } }))
    },
    'GET /api/v1/organization/positions': () => json(env({ items: [{ id: ID(500), slug: 'seller', label: 'Synthetic Seller', description: null, sort_order: 1, is_active: true, is_system: false, provenance: prov }], page: { has_more: false } })),
    'GET /api/v1/organization/org-units': () => json(env({
      items: [
        { org_unit_id: ID(10), kind: 'DEPARTMENT', slug: 'd', label: 'Synthetic Dept', sort_order: 1, is_active: true, parent_id: null, parent_state: 'NOT_APPLICABLE', member_count: 2, finding_kinds: [], provenance: prov },
        { org_unit_id: ID(11), kind: 'UNIT', slug: 'u', label: 'Synthetic Unit', sort_order: 1, is_active: true, parent_id: ID(10), parent_state: 'RESOLVED', member_count: 1, finding_kinds: [], provenance: prov },
        { org_unit_id: ID(12), kind: 'UNIT', slug: 'o', label: 'Synthetic Orphan Unit', sort_order: 2, is_active: true, parent_id: null, parent_state: 'MISSING', member_count: 0, finding_kinds: ['ORG_UNIT_PARENT_MISSING'], provenance: prov },
      ], page: { has_more: false },
    })),
    ...over,
  }
}

function renderAt(path: string, r: Record<string, (req: Request) => Response>) {
  const { client, calls } = fakeApi(r)
  const router = createMemoryRouter(buildRoutes(config), { initialEntries: [path] })
  render(<App config={config} client={client} router={router} />)
  return { calls }
}

const writeWords = /افزودن|ویرایش|حذف|انتقال|انتساب|ارتقا|ذخیره|ثبت درخواست/

describe('HR read (Slice 4)', () => {
  it('shows HR navigation only when the server reports the feature', async () => {
    renderAt('/', routes({}, false))
    await screen.findByRole('heading', { level: 1, name: 'خانه' })
    await waitFor(() => expect(screen.getByText('ورود هنوز فعال نیست')).toBeTruthy())
    expect(screen.queryByRole('link', { name: 'نیروها' })).toBeNull()
  })

  it('lists persons through the generated client, with server filters and cursor pagination', async () => {
    const { calls } = renderAt('/hr', routes())
    expect(await screen.findByRole('link', { name: 'Synthetic Person 1' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'نیروها' })).toBeTruthy()
    expect(screen.getByText('تعارض: چند مدیر جاری')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'صفحه بعد' }))
    expect(await screen.findByRole('link', { name: 'Synthetic Person 3' })).toBeTruthy()
    const persons = calls.filter(c => c.url === '/api/v1/organization/persons')
    expect(persons.at(-1)?.search.get('cursor')).toBe('c-2')

    const form = screen.getByRole('form', { name: 'فیلتر نیروها' })
    fireEvent.change(within(form).getByLabelText('وضعیت همکاری'), { target: { value: 'UNMAPPED' } })
    fireEvent.change(within(form).getByLabelText('جست‌وجو (نام یا کد)'), { target: { value: 'Synthetic' } })
    fireEvent.submit(form)
    await waitFor(() => {
      const last = calls.filter(c => c.url === '/api/v1/organization/persons').at(-1)
      expect(last?.search.get('employment_status')).toBe('UNMAPPED')
      expect(last?.search.get('q')).toBe('Synthetic')
      expect(last?.search.has('cursor')).toBe(false)
    })
    // Read-only: only GETs, no write controls.
    for (const c of calls) expect(c.method).toBe('GET')
    expect(screen.queryAllByRole('button').filter(b => writeWords.test(b.textContent ?? ''))).toHaveLength(0)
  })

  it('renders empty, unauthorized, feature-disabled and error states honestly', async () => {
    renderAt('/hr', routes({ 'GET /api/v1/organization/persons': () => json(env({ items: [], page: { has_more: false } })) }))
    expect(await screen.findByText('نیرویی وارد نشده است')).toBeTruthy()
  })

  it('renders UNAUTHORIZED as locked, never as empty', async () => {
    renderAt('/hr', routes({ 'GET /api/v1/organization/persons': () => json({ outcome: 'UNAUTHORIZED', reasons: ['ACTION_DENIED'], correlation_id: 'test-corr-1' }, 403) }))
    expect(await screen.findByText('دسترسی به این بخش ندارید')).toBeTruthy()
    expect(screen.queryByText('نیرویی وارد نشده است')).toBeNull()
  })

  it('renders BLOCKED (flag off) as not active', async () => {
    renderAt('/hr', routes({ 'GET /api/v1/organization/persons': () => json({ outcome: 'BLOCKED', reasons: ['FEATURE_DISABLED'], correlation_id: 'test-corr-1' }) }))
    expect(await screen.findByText('این بخش هنوز فعال نشده است')).toBeTruthy()
  })

  it('renders a problem as an error state', async () => {
    renderAt('/hr', routes({ 'GET /api/v1/organization/persons': () => json({ type: '/problems/bad-request', title: 'Request does not match the contract', status: 400, reason_code: 'CONTRACT_INVALID', correlation_id: 'srv-corr-0001' }, 400, 'application/problem+json') }))
    expect(await screen.findByText('Request does not match the contract')).toBeTruthy()
  })

  it('never infers permission: masked sections stay hidden when the server denies them', async () => {
    const limited = { ...allAllowed, 'org.employment.read': { allowed: false, reason_code: 'ACTION_DENIED' }, 'org.assignment.read': { allowed: false } }
    renderAt('/hr', routes({ 'GET /api/v1/organization/persons': () => json({ ...env({ items: [{ ...person(1), employment: undefined, manager_state: undefined }], page: { has_more: false } }), allowed_actions: limited }) }))
    await screen.findByRole('link', { name: 'Synthetic Person 1' })
    expect(screen.queryByRole('columnheader', { name: 'وضعیت همکاری' })).toBeNull()
    expect(screen.queryByRole('columnheader', { name: /مدیر جاری/ })).toBeNull()
  })

  it('renders the org tree from the OrgUnit model and labels orphans instead of attaching them', async () => {
    renderAt('/hr/structure', routes())
    const tree = await screen.findByRole('list', { name: 'درخت سازمانی' })
    expect(within(tree).getByText('Synthetic Dept')).toBeTruthy()
    expect(within(tree).getByText('Synthetic Unit')).toBeTruthy()
    expect(within(tree).queryByText('Synthetic Orphan Unit')).toBeNull()
    const orphans = screen.getByRole('list', { name: 'واحدهای بدون والد' })
    expect(within(orphans).getByText('Synthetic Orphan Unit')).toBeTruthy()
    expect(within(orphans).getByText('تعارض: والد ناموجود')).toBeTruthy()
  })

  it('shows person detail sections separately with raw interval evidence', async () => {
    const detail = {
      person: person(2, { manager_state: 'MULTIPLE', finding_count: 1, employment: { ...person(2).employment, status: 'UNMAPPED', status_raw: 'synthetic_x' } }),
      assignments: [{ assignment_id: ID(300), child: { state: 'RESOLVED' }, parent: { state: 'MISSING' }, relationship_kind: 'REPORTS_TO', relationship_raw: 'reports_to',
        effective_from_raw: '2026-01-01 00:00:00', effective_to_raw: null, interval_zone: 'UNKNOWN', is_current: true, current_raw: '1', finding_kinds: ['MISSING_PARENT'], provenance: prov }],
      direct_reports: [], history: [],
      findings: [{ finding_id: ID(400), kind: 'MULTIPLE_CURRENT_PARENTS', severity: 'CONFLICT', subject_type: 'ASSIGNMENT', subject_id: ID(300), subject_source_key: '203', related_id: null, related_source_key: null, detail: 'synthetic detail' }],
    }
    renderAt(`/hr/people/${ID(2)}`, routes({ [`GET /api/v1/organization/persons/${ID(2)}`]: () => json(env({ data: detail })) }))
    expect(await screen.findByRole('heading', { level: 1, name: 'Synthetic Person 2' })).toBeTruthy()
    for (const title of ['شخص', 'همکاری', 'سمت و سطح', 'واحد سازمانی', 'مدیر جاری', 'بازه‌های گزارش‌دهی']) {
      expect(screen.getByRole('region', { name: title })).toBeTruthy()
    }
    expect(screen.getByText(/مقدار قدیمی: synthetic_x/)).toBeTruthy()
    expect(screen.getByText(/منطقه زمانی: نامعلوم/)).toBeTruthy()
    expect(screen.getByText('سابقه‌ای ثبت نشده')).toBeTruthy()
    expect(screen.getAllByText(/تعارض: چند مدیر جاری/).length).toBeGreaterThan(0)
    expect(screen.queryAllByRole('button').filter(b => writeWords.test(b.textContent ?? ''))).toHaveLength(0)
  })
})
