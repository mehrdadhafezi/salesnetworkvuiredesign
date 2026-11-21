import '../hr.css'
import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type Assignment, type PersonDetail } from '../../../shared/api'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ReadStates } from '../components/States'
import { findingLabel, text } from '../labels'
import { compact, settle, useRead } from '../load'
import { HRSectionNav } from '../workspace/SectionNav'

const filterKeys = ['person_id', 'relation', 'is_current', 'relationship_kind', 'has_findings'] as const

/**
 * Reporting assignments (REPORTS_TO lines), read only. Each row keeps both
 * ends' own evidence state; a missing or unrecorded end is never replaced by
 * today's manager. A reporting line is organisation evidence, not a
 * permission: being someone's manager grants nothing.
 */
export function AssignmentsPage() {
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const [cursors, setCursors] = useState<string[]>([])
  const cursor = cursors[cursors.length - 1]
  const query = Object.fromEntries(filterKeys.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : [])))
  const personId = query.person_id
  const list = useRead(JSON.stringify({ query, cursor }), async () => settle(await api.GET('/api/v1/organization/assignments', {
    params: { query: compact({
      person_id: personId, relation: personId ? (query.relation as 'CHILD' | 'PARENT' | undefined) : undefined,
      is_current: query.is_current === 'true' ? true : query.is_current === 'false' ? false : undefined,
      relationship_kind: query.relationship_kind as 'REPORTS_TO' | 'UNMAPPED' | undefined,
      has_findings: query.has_findings === 'true' ? true : undefined, cursor, limit: 50,
    }) },
  }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))

  const subject = useRead<PersonDetail | null>(`subject:${personId ?? ''}`, async () => personId
    ? settle(await api.GET('/api/v1/organization/persons/{person_id}', { params: { path: { person_id: personId } } }), e => e.data)
    : { status: 'ok', data: null, outcome: 'SUCCESS', allowed: undefined })

  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const next = new URLSearchParams()
    if (personId) next.set('person_id', personId)
    for (const k of filterKeys) {
      if (k === 'person_id') continue
      const v = form.get(k)
      if (typeof v === 'string' && v !== '') next.set(k, v)
    }
    setCursors([])
    setParams(next)
  }

  return (
    <>
      <HRSectionNav />
      <PageHeader title="انتساب‌ها و گزارش‌دهی" subtitle="خطوط گزارش‌دهی ثبت‌شده؛ مدیر بودن هیچ مجوزی نمی‌دهد و سابقه تاریخی بازنویسی نمی‌شود.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      {personId && (
        <div className="hr-scope" role="status">
          <span>انتساب‌های </span>
          {subject.status === 'ok' && subject.data
            ? <Link to={`/hr/people/${personId}`}>{text(subject.data.person.display_name)}</Link>
            : <span className="hr-ltr num">{personId}</span>}
          <span className="muted"> {query.relation === 'PARENT' ? '(به‌عنوان مدیر)' : query.relation === 'CHILD' ? '(به‌عنوان زیرمجموعه)' : '(هر دو سو)'}</span>
          {' '}<Link to="/hr/assignments">نمایش همه انتساب‌ها</Link>
        </div>
      )}
      <Card title="فیلترها">
        <form className="hr-filters" onSubmit={apply} aria-label="فیلتر انتساب‌ها">
          {personId && (
            <label className="lbl">نقش این نیرو در انتساب
              <select className="input" name="relation" defaultValue={params.get('relation') ?? ''}>
                <option value="">هر دو</option>
                <option value="CHILD">زیرمجموعه (گزارش‌دهنده)</option>
                <option value="PARENT">مدیر (گزارش‌گیرنده)</option>
              </select>
            </label>
          )}
          <label className="lbl">جاری بودن (پرچم منبع)
            <select className="input" name="is_current" defaultValue={params.get('is_current') ?? ''}>
              <option value="">همه</option>
              <option value="true">جاری</option>
              <option value="false">غیرجاری</option>
            </select>
          </label>
          <label className="lbl">نوع رابطه
            <select className="input" name="relationship_kind" defaultValue={params.get('relationship_kind') ?? ''}>
              <option value="">همه</option>
              <option value="REPORTS_TO">گزارش به مدیر</option>
              <option value="UNMAPPED">نگاشت‌نشده (مقدار قدیمی)</option>
            </select>
          </label>
          <label className="lbl hr-check"><input type="checkbox" name="has_findings" value="true" defaultChecked={params.get('has_findings') === 'true'} /> فقط دارای یافته</label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <Card title="انتساب‌ها">
        <p className="muted">تاریخ‌ها مقدار خام قدیمی‌اند و منطقه زمانی منبع نامعلوم است؛ بازه گم‌شده پر نمی‌شود.</p>
        <ReadStates state={list} what="انتساب‌ها" />
        {list.status === 'ok' && (list.data.items.length === 0
          ? (Object.keys(query).length
            ? <StateBlock kind="empty" title="انتسابی برای این فیلتر نیست" detail="این پاسخ درباره کل سازمان نیست." />
            : <StateBlock kind="empty" title="انتسابی وارد نشده است" />)
          : <AssignmentTable items={list.data.items} />)}
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

function End({ end, missing, none }: { end: Assignment['child']; missing: string; none: string }) {
  if (end.person) return <Link to={`/hr/people/${end.person.person_id}`}>{text(end.person.display_name)}</Link>
  return end.state === 'MISSING'
    ? <StatusPill tone="orange" icon="alert" label={missing} />
    : <span className="muted">{none}</span>
}

function current(a: Assignment) {
  if (a.is_current === null) return <StatusPill tone="amber" icon="question" label={`نامعلوم${a.current_raw ? ` (${a.current_raw})` : ''}`} />
  return a.is_current ? <StatusPill tone="green" icon="check" label="جاری" /> : <span className="muted">غیرجاری</span>
}

function AssignmentTable({ items }: { items: Assignment[] }) {
  return (
    <div className="hr-table-wrap">
      <table className="hr-table hr-stack">
        <caption className="sr">انتساب‌های گزارش‌دهی</caption>
        <thead>
          <tr>
            <th scope="col">نیرو (گزارش‌دهنده)</th><th scope="col">مدیر (گزارش‌گیرنده)</th><th scope="col">نوع رابطه</th>
            <th scope="col">بازه (خام)</th><th scope="col">جاری</th><th scope="col">یافته‌ها</th><th scope="col">منبع</th>
          </tr>
        </thead>
        <tbody>
          {items.map(a => (
            <tr key={a.assignment_id}>
              <td data-label="نیرو"><End end={a.child} missing="نیروی ناموجود" none="ثبت نشده" /></td>
              <td data-label="مدیر"><End end={a.parent} missing="مدیر ناموجود" none="بدون مدیر" /></td>
              <td data-label="نوع رابطه">{a.relationship_kind === 'REPORTS_TO' ? 'گزارش به مدیر' : <>نگاشت‌نشده <span className="muted">({text(a.relationship_raw)})</span></>}</td>
              <td data-label="بازه" className="num">{a.effective_from_raw ?? 'شروع نامعلوم'} ← {a.effective_to_raw ?? (a.is_current ? 'جاری' : 'پایان نامعلوم')}</td>
              <td data-label="جاری">{current(a)}</td>
              <td data-label="یافته‌ها">{a.finding_kinds.length === 0 ? <span className="muted">—</span>
                : a.finding_kinds.map(k => <StatusPill key={k} tone="orange" icon="alert" label={findingLabel[k]} />)}</td>
              <td data-label="منبع" className="num">{`${a.provenance.source_spec} #${a.provenance.source_key}`}{a.provenance.presence !== 'PRESENT' && <span className="muted"> (در آخرین ورود نبود)</span>}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
