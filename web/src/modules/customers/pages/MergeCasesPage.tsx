import '../customers.css'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type CaseReason, type MergeCase, type ResolutionState, type Strength } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { NoMergeNote, StatePill } from '../components'
import { blockedDetail, evidenceLabel, reasonLabel, stateLabel, strengthLabel } from '../labels'
import { Pager, useCursors } from '../pager'

const filterKeys = ['state', 'evidence_strength', 'reason'] as const

/**
 * Merge-case queue (read-only). A case is a candidate with its evidence;
 * there is no merge, approve, dismiss or winner control (D-30 is open).
 * Conflicts stay visible as conflicts.
 */
export function MergeCasesPage() {
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const c = useCursors()
  const selected = params.get('case')
  const query = Object.fromEntries(filterKeys.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : [])))
  const list = useRead(JSON.stringify({ query, cursor: c.cursor }), async () => settle(await api.GET('/api/v1/customer-identity/merge-cases', {
    params: { query: compact({ state: query.state as ResolutionState | undefined, evidence_strength: query.evidence_strength as Strength | undefined,
      reason: query.reason as CaseReason | undefined, cursor: c.cursor, limit: 25 }) },
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
      <PageHeader title="صف پرونده‌های تطبیق" subtitle="نامزدهای هم‌ارزی با شواهدشان؛ هیچ ادغامی انجام نمی‌شود.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <NoMergeNote />
      <Card title="فیلترها">
        <form className="cu-filters" onSubmit={apply} aria-label="فیلتر پرونده‌ها">
          <label className="lbl">وضعیت
            <select className="input" name="state" defaultValue={params.get('state') ?? ''}>
              <option value="">همه</option>
              {(Object.keys(stateLabel) as ResolutionState[]).map(s => <option key={s} value={s}>{stateLabel[s]}</option>)}
            </select>
          </label>
          <label className="lbl">قوت شواهد
            <select className="input" name="evidence_strength" defaultValue={params.get('evidence_strength') ?? ''}>
              <option value="">همه</option>
              {(Object.keys(strengthLabel) as Strength[]).map(s => <option key={s} value={s}>{strengthLabel[s]}</option>)}
            </select>
          </label>
          <label className="lbl">دلیل
            <select className="input" name="reason" defaultValue={params.get('reason') ?? ''}>
              <option value="">همه</option>
              {(Object.keys(reasonLabel) as CaseReason[]).map(s => <option key={s} value={s}>{reasonLabel[s]}</option>)}
            </select>
          </label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      {selected && <CaseDetail id={selected} />}
      <Card title="پرونده‌ها">
        <ReadStates state={list} what="پرونده‌ها" blockedDetail={blockedDetail} />
        {list.status === 'ok' && (list.data.items.length === 0
          ? <StateBlock kind="empty" title={Object.keys(query).length ? 'نتیجه‌ای برای این فیلتر نیست' : 'پرونده تطبیقی وجود ندارد'} />
          : <CaseTable items={list.data.items} params={params} />)}
        {list.status === 'ok' && <Pager page={list.data.page} c={c} />}
      </Card>
    </>
  )
}

function CaseTable({ items, params }: { items: MergeCase[]; params: URLSearchParams }) {
  const linkTo = (id: string) => { const p = new URLSearchParams(params); p.set('case', id); return `?${p.toString()}` }
  return (
    <div className="cu-table-wrap">
      <table className="cu-table">
        <caption className="sr">پرونده‌های تطبیق</caption>
        <thead><tr><th scope="col">دلیل</th><th scope="col">وضعیت</th><th scope="col">قوی‌ترین شاهد</th><th scope="col">اعضا</th><th scope="col">شواهد</th></tr></thead>
        <tbody>
          {items.map(m => (
            <tr key={m.merge_case_id}>
              <td><Link to={linkTo(m.merge_case_id)}>{reasonLabel[m.reason]}</Link></td>
              <td><StatePill state={m.state} /></td>
              <td>{strengthLabel[m.max_strength]}</td>
              <td className="num">{m.member_customer_ids.length}</td>
              <td>{m.evidence_kinds.map(k => evidenceLabel[k]).join('، ')}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function CaseDetail({ id }: { id: string }) {
  const api = useApi()
  const one = useRead(`case:${id}`, async () => settle(await api.GET('/api/v1/customer-identity/merge-cases/{merge_case_id}', {
    params: { path: { merge_case_id: id } } }), e => e.data))
  return (
    <Card title="جزئیات پرونده">
      <ReadStates state={one} what="پرونده" blockedDetail={blockedDetail} />
      {one.status === 'ok' && (
        <dl className="cu-kv">
          <dt>دلیل</dt><dd>{reasonLabel[one.data.reason]}</dd>
          <dt>وضعیت</dt><dd><StatePill state={one.data.state} /> <span className="muted">{one.data.state_reason}</span></dd>
          <dt>شواهد</dt><dd>{one.data.evidence_kinds.map(k => evidenceLabel[k]).join('، ')}</dd>
          <dt>مشتریان عضو (همه جدا می‌مانند)</dt>
          <dd><ul className="cu-list">{one.data.member_customer_ids.map((m, i) => <li key={m}><Link to={`/customers/${m}`}>{`مشتری ${i + 1}`}</Link></li>)}</ul></dd>
        </dl>
      )}
    </Card>
  )
}
