import '../hr.css'
import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type EmploymentStatus, type Lookup, type OrgUnit, type PersonSummary } from '../../../shared/api'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ReadStates } from '../components/States'
import { kindLabel, linkLabel, statusLabel, text } from '../labels'
import { compact, settle, useRead } from '../load'

type Sort = 'display_name' | '-display_name' | 'employee_code' | '-employee_code' | 'employment_status' | '-employment_status'

const sorts: { value: Sort; label: string }[] = [
  { value: 'display_name', label: 'نام (صعودی)' }, { value: '-display_name', label: 'نام (نزولی)' },
  { value: 'employee_code', label: 'کد کارمندی (صعودی)' }, { value: '-employee_code', label: 'کد کارمندی (نزولی)' },
  { value: 'employment_status', label: 'وضعیت همکاری' },
]

const filterKeys = ['q', 'employment_status', 'position_id', 'level_id', 'org_unit_id', 'has_findings', 'sort'] as const

/**
 * Workforce Explorer (SN-205 §3), read-only: server-side filters, sorting
 * and cursor pagination. There is no add, edit, delete, move, assign or
 * promote control; sections render only what the server allows.
 */
export function WorkforcePage() {
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const [cursors, setCursors] = useState<string[]>([])
  const cursor = cursors[cursors.length - 1]
  const query = Object.fromEntries(filterKeys.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : [])))
  const key = JSON.stringify({ query, cursor })

  const list = useRead(key, async () => settle(await api.GET('/api/v1/organization/persons', {
    params: { query: compact({ ...query, has_findings: query.has_findings === 'true' ? true : undefined, sort: query.sort as Sort | undefined,
      employment_status: query.employment_status as EmploymentStatus | undefined, cursor, limit: 25 }) },
  }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))

  const allowed = list.status === 'ok' ? list.allowed : undefined
  const positions = useRead(`positions:${actionHint(allowed, 'org.position.read').allowed}`, async () =>
    actionHint(allowed, 'org.position.read').allowed
      ? settle(await api.GET('/api/v1/organization/positions', { params: { query: { limit: 200 } } }), e => e.items)
      : { status: 'ok', data: [] as Lookup[], outcome: 'SUCCESS', allowed: undefined })
  const units = useRead(`units:${actionHint(allowed, 'org.unit.read').allowed}`, async () =>
    actionHint(allowed, 'org.unit.read').allowed
      ? settle(await api.GET('/api/v1/organization/org-units', { params: { query: { limit: 200 } } }), e => e.items)
      : { status: 'ok', data: [] as OrgUnit[], outcome: 'SUCCESS', allowed: undefined })

  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const next = new URLSearchParams()
    for (const k of filterKeys) {
      const v = form.get(k)
      if (typeof v === 'string' && v !== '') next.set(k, v)
    }
    setCursors([])
    setParams(next)
  }

  return (
    <>
      <PageHeader title="نیروها" subtitle="نمای فقط‌خواندنی مدل سایه؛ هیچ تغییری از این صفحه انجام نمی‌شود.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <Card title="فیلترها">
        <form className="hr-filters" onSubmit={apply} aria-label="فیلتر نیروها">
          <label className="lbl">جست‌وجو (نام یا کد)<input className="input" name="q" defaultValue={params.get('q') ?? ''} maxLength={100} /></label>
          <label className="lbl">وضعیت همکاری
            <select className="input" name="employment_status" defaultValue={params.get('employment_status') ?? ''}>
              <option value="">همه</option>
              {(Object.keys(statusLabel) as EmploymentStatus[]).map(s => <option key={s} value={s}>{statusLabel[s]}</option>)}
            </select>
          </label>
          {positions.status === 'ok' && positions.data.length > 0 && (
            <label className="lbl">سمت
              <select className="input" name="position_id" defaultValue={params.get('position_id') ?? ''}>
                <option value="">همه</option>
                {positions.data.map(p => <option key={p.id} value={p.id}>{text(p.label)}</option>)}
              </select>
            </label>
          )}
          {units.status === 'ok' && units.data.length > 0 && (
            <label className="lbl">واحد سازمانی
              <select className="input" name="org_unit_id" defaultValue={params.get('org_unit_id') ?? ''}>
                <option value="">همه</option>
                {units.data.map(u => <option key={u.org_unit_id} value={u.org_unit_id}>{`${kindLabel[u.kind]}: ${text(u.label)}`}</option>)}
              </select>
            </label>
          )}
          <label className="lbl">ترتیب
            <select className="input" name="sort" defaultValue={params.get('sort') ?? 'display_name'}>
              {sorts.map(s => <option key={s.value} value={s.value}>{s.label}</option>)}
            </select>
          </label>
          <label className="lbl hr-check"><input type="checkbox" name="has_findings" value="true" defaultChecked={params.get('has_findings') === 'true'} /> فقط دارای یافته</label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <Card title="فهرست نیروها">
        <ReadStates state={list} what="نیروها" />
        {list.status === 'ok' && (list.data.items.length === 0
          ? (Object.keys(query).length
            ? <StateBlock kind="empty" title="نتیجه‌ای برای این فیلتر نیست" detail="فیلترها را تغییر دهید؛ این پاسخ درباره کل سازمان نیست." />
            : <StateBlock kind="empty" title="نیرویی وارد نشده است" />)
          : <PersonTable items={list.data.items} canEmployment={actionHint(list.allowed, 'org.employment.read').allowed}
            canAssignment={actionHint(list.allowed, 'org.assignment.read').allowed} />)}
        {list.status === 'ok' && (
          <nav className="hr-pager" aria-label="صفحه‌بندی">
            <Button disabled={cursors.length === 0} onClick={() => setCursors(c => c.slice(0, -1))}>صفحه قبل</Button>
            <Button disabled={!list.data.page.has_more || !list.data.page.next_cursor}
              onClick={() => { const n = list.data.page.next_cursor; if (n) setCursors(c => [...c, n]) }}>صفحه بعد</Button>
          </nav>
        )}
      </Card>
    </>
  )
}

function PersonTable({ items, canEmployment, canAssignment }: { items: PersonSummary[]; canEmployment: boolean; canAssignment: boolean }) {
  return (
    <div className="hr-table-wrap">
      <table className="hr-table">
        <caption className="sr">نیروها</caption>
        <thead>
          <tr>
            <th scope="col">نام</th>
            {canEmployment && <><th scope="col">کد کارمندی</th><th scope="col">وضعیت همکاری</th><th scope="col">سمت · سطح</th><th scope="col">پیوند حساب</th></>}
            {canAssignment && <th scope="col">مدیر جاری <span className="muted">(اعتبار تاریخی نیست)</span></th>}
            <th scope="col">یافته‌ها</th>
          </tr>
        </thead>
        <tbody>
          {items.map(p => {
            const e = p.employment
            return (
              <tr key={p.person_id}>
                <td><Link to={`/hr/people/${p.person_id}`}>{text(p.display_name)}</Link></td>
                {canEmployment && (
                  <>
                    <td className="num">{text(e?.employee_code)}</td>
                    <td>{e ? <>{statusLabel[e.status]}{e.status === 'UNMAPPED' && <span className="muted"> ({text(e.status_raw)})</span>}</> : '—'}</td>
                    <td>{text(e?.position?.label)} · {text(e?.level?.label)}</td>
                    <td>{e ? <StatusPill tone={e.user.link_state === 'LINKED' ? 'green' : 'orange'} icon={e.user.link_state === 'LINKED' ? 'check' : 'alert'} label={linkLabel[e.user.link_state]} /> : '—'}</td>
                  </>
                )}
                {canAssignment && <td><Manager p={p} /></td>}
                <td>{p.finding_count > 0 ? <StatusPill tone="orange" icon="alert" label={`${p.finding_count} یافته`} /> : <span className="muted">—</span>}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function Manager({ p }: { p: PersonSummary }) {
  switch (p.manager_state) {
    case 'RESOLVED':
      return p.manager ? <Link to={`/hr/people/${p.manager.person_id}`}>{text(p.manager.display_name)}</Link> : <span>—</span>
    case 'MULTIPLE':
      return <StatusPill tone="orange" icon="alert" label="تعارض: چند مدیر جاری" />
    case 'MISSING':
      return <StatusPill tone="orange" icon="alert" label="مدیر ثبت‌شده وجود ندارد" />
    default:
      return <span className="muted">بدون مدیر جاری</span>
  }
}
