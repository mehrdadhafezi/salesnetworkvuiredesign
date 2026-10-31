import '../customers.css'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type CustomerSummary, type ResolutionState } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { NoMergeNote, StatePill } from '../components'
import { blockedDetail, stateLabel, text } from '../labels'
import { Pager, useCursors } from '../pager'

const filterKeys = ['state', 'has_conflict'] as const

/**
 * Customers (Slice 5 shadow), read-only: server filters and cursor
 * pagination. One row per source subject; there is no merge, edit, delete
 * or reassign control, and no phone search (phone is not identity).
 */
export function CustomersPage() {
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const c = useCursors()
  const query = Object.fromEntries(filterKeys.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : [])))
  const list = useRead(JSON.stringify({ query, cursor: c.cursor }), async () => settle(await api.GET('/api/v1/customers', {
    params: { query: compact({ state: query.state as ResolutionState | undefined, has_conflict: query.has_conflict === 'true' ? true : undefined, cursor: c.cursor, limit: 25 }) },
  }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))

  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const next = new URLSearchParams()
    for (const k of filterKeys) {
      const v = form.get(k)
      if (typeof v === 'string' && v !== '') next.set(k, v)
    }
    c.reset()
    setParams(next)
  }

  return (
    <>
      <PageHeader title="مشتریان" subtitle="مدل سایه هویت مشتری؛ هر رکورد منبع یک مشتری جداست.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <NoMergeNote />
      <Card title="فیلترها">
        <form className="cu-filters" onSubmit={apply} aria-label="فیلتر مشتریان">
          <label className="lbl">وضعیت هویت
            <select className="input" name="state" defaultValue={params.get('state') ?? ''}>
              <option value="">همه</option>
              {(Object.keys(stateLabel) as ResolutionState[]).map(s => <option key={s} value={s}>{stateLabel[s]}</option>)}
            </select>
          </label>
          <label className="lbl cu-check"><input type="checkbox" name="has_conflict" value="true" defaultChecked={params.get('has_conflict') === 'true'} /> فقط دارای تعارض</label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <Card title="فهرست مشتریان">
        <ReadStates state={list} what="مشتریان" blockedDetail={blockedDetail} />
        {list.status === 'ok' && (list.data.items.length === 0
          ? (Object.keys(query).length
            ? <StateBlock kind="empty" title="نتیجه‌ای برای این فیلتر نیست" detail="این پاسخ درباره کل مشتریان نیست." />
            : <StateBlock kind="empty" title="مشتری‌ای وارد نشده است" />)
          : <CustomerTable items={list.data.items} />)}
        {list.status === 'ok' && <Pager page={list.data.page} c={c} />}
      </Card>
    </>
  )
}

function CustomerTable({ items }: { items: CustomerSummary[] }) {
  return (
    <div className="cu-table-wrap">
      <table className="cu-table">
        <caption className="sr">مشتریان</caption>
        <thead>
          <tr>
            <th scope="col">نام (پوشانده)</th><th scope="col">وضعیت هویت</th><th scope="col">پرونده‌های باز</th>
            <th scope="col">تعارض</th><th scope="col">منبع</th>
          </tr>
        </thead>
        <tbody>
          {items.map(cu => (
            <tr key={cu.customer_id}>
              <td><Link to={`/customers/${cu.customer_id}`}>{text(cu.name_masked) === '—' ? 'بدون نام ثبت‌شده' : text(cu.name_masked)}</Link></td>
              <td><StatePill state={cu.resolution_state} /></td>
              <td className="num">{cu.open_case_count}</td>
              <td>{cu.conflict_case_count > 0 ? <StatusPill tone="orange" icon="alert" label={`${cu.conflict_case_count} تعارض`} /> : <span className="muted">—</span>}</td>
              <td><span className="cu-ltr">{cu.provenance.source_spec} #{cu.provenance.source_key}</span>{cu.provenance.presence !== 'PRESENT' && <span className="muted"> (در آخرین دسته نیست)</span>}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
