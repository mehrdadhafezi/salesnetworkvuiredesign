import '../billing.css'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type InvoiceSummary } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ReadOnlyNote } from '../components'
import { blockedDetail, text } from '../labels'
import { Pager, useCursors } from '../pager'

type Link_ = 'RESOLVED' | 'RELATED' | 'UNKNOWN' | 'CONFLICT' | 'NOT_RECOVERABLE'
const filterKeys = ['presence', 'has_unknown_status', 'customer_link_state', 'case_link_state'] as const
const linkLabel: Record<string, string> = { RESOLVED: 'ثبت‌شده با شاهد', RELATED: 'مرتبط', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض', NOT_RECOVERABLE: 'قابل بازیابی نیست' }

/**
 * Invoices (Slice 7 shadow), read-only. Server filters and cursor
 * pagination; the invoice lifecycle is shown as UNKNOWN (TAD #20) and raw
 * unknown state counts are visible. No action control exists.
 */
export function InvoicesPage() {
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const c = useCursors()
  const q = Object.fromEntries(filterKeys.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : [])))
  const list = useRead(JSON.stringify({ q, cursor: c.cursor }), async () => settle(await api.GET('/api/v1/invoices', {
    params: { query: compact({ presence: q.presence as 'PRESENT' | 'ABSENT_IN_LATEST' | undefined,
      has_unknown_status: q.has_unknown_status === undefined ? undefined : q.has_unknown_status === 'true',
      customer_link_state: q.customer_link_state as Link_ | undefined, case_link_state: q.case_link_state as Link_ | undefined, cursor: c.cursor, limit: 25 }) },
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

  const linkSelect = (name: string, label: string) => (
    <label className="lbl">{label}
      <select className="input" name={name} defaultValue={params.get(name) ?? ''}>
        <option value="">همه</option>
        {Object.entries(linkLabel).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
      </select>
    </label>
  )
  return (
    <>
      <PageHeader title="فاکتورها" subtitle="مدل سایه صورت‌حساب؛ وضعیت قطعی فاکتور تعیین نشده است.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <Card title="فیلترها">
        <form className="bi-filters" onSubmit={apply} aria-label="فیلتر فاکتورها">
          <label className="lbl">وضعیت ناشناخته
            <select className="input" name="has_unknown_status" defaultValue={params.get('has_unknown_status') ?? ''}>
              <option value="">همه</option>
              <option value="true">دارای مقدار ناشناخته</option>
              <option value="false">بدون مقدار ناشناخته</option>
            </select>
          </label>
          <label className="lbl">حضور در دسته
            <select className="input" name="presence" defaultValue={params.get('presence') ?? ''}>
              <option value="">همه</option>
              <option value="PRESENT">حاضر</option>
              <option value="ABSENT_IN_LATEST">در آخرین دسته کامل نیست</option>
            </select>
          </label>
          {linkSelect('customer_link_state', 'پیوند مشتری')}
          {linkSelect('case_link_state', 'پیوند پرونده')}
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <Card title="فهرست فاکتورها">
        <ReadStates state={list} what="فاکتورها" blockedDetail={blockedDetail} />
        {list.status === 'ok' && (list.data.items.length === 0
          ? <StateBlock kind="empty" title={Object.keys(q).length ? 'نتیجه‌ای برای این فیلتر نیست' : 'فاکتوری وارد نشده است'} />
          : <InvoiceTable items={list.data.items} />)}
        {list.status === 'ok' && <Pager page={list.data.page} c={c} label="صفحه‌بندی فاکتورها" />}
      </Card>
    </>
  )
}

function InvoiceTable({ items }: { items: InvoiceSummary[] }) {
  return (
    <div className="bi-table-wrap">
      <table className="bi-table">
        <caption className="sr">فاکتورها</caption>
        <thead>
          <tr><th scope="col">فاکتور</th><th scope="col">صدور (خام)</th><th scope="col">وضعیت چرخه</th><th scope="col">مقادیر ناشناخته</th>
            <th scope="col">اقلام / مراحل</th><th scope="col">مشتری / پرونده</th></tr>
        </thead>
        <tbody>
          {items.map(x => (
            <tr key={x.invoice_id}>
              <td><Link to={`/billing/invoices/${x.invoice_id}`} className="bi-ltr">{`${x.source.source_spec} #${x.source.source_key}`}</Link>
                {x.source.presence !== 'PRESENT' && <span className="muted"> (در آخرین دسته نیست)</span>}</td>
              <td className="bi-ltr">{text(x.issued_at_raw)}</td>
              <td><StatusPill tone="slate" icon="question" label="تعیین نشده (TAD #20)" /></td>
              <td className="num">{x.unknown_state_count}</td>
              <td className="num">{`${x.item_count} / ${x.stage_count}`}</td>
              <td>{`${linkLabel[x.customer.state] ?? x.customer.state} / ${linkLabel[x.case.state] ?? x.case.state}`}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
