import '../sales.css'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type CaseSummary, type WorkflowType } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ReadOnlyNote, ResolutionPill } from '../components'
import { blockedDetail, text, workflowLabel } from '../labels'
import { Pager, useCursors } from '../pager'

const filterKeys = ['workflow_type', 'state_mapping'] as const

/**
 * Sales cases (Slice 6 shadow), read-only: one Case per staged legacy row,
 * server filters and cursor pagination. One surface for seller, supervisor
 * and manager reads: no frozen per-role row-scope policy exists (OPD-01 and
 * scope decisions open), so the server grants it only explicitly. No action
 * control exists.
 */
export function CasesPage() {
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const c = useCursors()
  const query = Object.fromEntries(filterKeys.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : [])))
  const list = useRead(JSON.stringify({ query, cursor: c.cursor }), async () => settle(await api.GET('/api/v1/cases', {
    params: { query: compact({ workflow_type: query.workflow_type as WorkflowType | undefined, state_mapping: query.state_mapping as 'KNOWN' | 'UNKNOWN' | undefined,
      cursor: c.cursor, limit: 25 }) },
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
      <PageHeader title="پرونده‌های فروش" subtitle="مدل سایه؛ هر رکورد منبع یک پرونده جداست.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <Card title="فیلترها">
        <form className="sa-filters" onSubmit={apply} aria-label="فیلتر پرونده‌ها">
          <label className="lbl">نوع منبع
            <select className="input" name="workflow_type" defaultValue={params.get('workflow_type') ?? ''}>
              <option value="">همه</option>
              {(Object.keys(workflowLabel) as WorkflowType[]).map(w => <option key={w} value={w}>{workflowLabel[w]}</option>)}
            </select>
          </label>
          <label className="lbl">نگاشت وضعیت
            <select className="input" name="state_mapping" defaultValue={params.get('state_mapping') ?? ''}>
              <option value="">همه</option>
              <option value="KNOWN">شناخته‌شده</option>
              <option value="UNKNOWN">نامعلوم (مقدار خام حفظ شد)</option>
            </select>
          </label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <Card title="فهرست پرونده‌ها">
        <ReadStates state={list} what="پرونده‌ها" blockedDetail={blockedDetail} />
        {list.status === 'ok' && (list.data.items.length === 0
          ? <StateBlock kind="empty" title={Object.keys(query).length ? 'نتیجه‌ای برای این فیلتر نیست' : 'پرونده‌ای وارد نشده است'} />
          : <CaseTable items={list.data.items} />)}
        {list.status === 'ok' && <Pager page={list.data.page} c={c} />}
      </Card>
    </>
  )
}

function CaseTable({ items }: { items: CaseSummary[] }) {
  return (
    <div className="sa-table-wrap">
      <table className="sa-table">
        <caption className="sr">پرونده‌ها</caption>
        <thead>
          <tr><th scope="col">پرونده</th><th scope="col">نوع منبع</th><th scope="col">وضعیت (خام)</th><th scope="col">پیوند مشتری</th><th scope="col">پیوندها</th></tr>
        </thead>
        <tbody>
          {items.map(x => (
            <tr key={x.case_id}>
              <td><Link to={`/sales/cases/${x.case_id}`} className="sa-ltr">{`${x.source.source_spec} #${x.source.source_key}`}</Link>
                {x.presence !== 'PRESENT' && <span className="muted"> (در آخرین دسته نیست)</span>}</td>
              <td>{workflowLabel[x.workflow_type]}</td>
              <td><span className="sa-ltr">{text(x.state_raw)}</span>{x.state_mapping === 'UNKNOWN' && <span className="muted"> (نگاشت‌نشده)</span>}</td>
              <td><ResolutionPill resolution={x.customer_link_state} /></td>
              <td className="num">{x.alias_count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
