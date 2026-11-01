import '../billing.css'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type ApprovalProvenance, type AttemptState, type PaymentAttempt, type PaymentChannel, type TargetMapping } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { AmountText, ReadOnlyNote } from '../components'
import { attemptStateLabel, blockedDetail, channelLabel, gatewayStateLabel, mappingLabel, provenanceLabel, text } from '../labels'
import { Pager, useCursors } from '../pager'

const filterKeys = ['channel', 'state', 'state_mapping', 'approval_provenance'] as const

/**
 * Payment attempts (Slice 7 shadow), read-only. Raw legacy state beside the
 * explicitly mapped attempt state, gateway state and approval provenance —
 * never one generic status. 'approved' is not Finance-reviewed, settled or
 * posted; gateway approval is GATEWAY_AUTO. No action control exists.
 */
export function AttemptsPage() {
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const c = useCursors()
  const q = Object.fromEntries(filterKeys.flatMap(k => (params.get(k) ? [[k, params.get(k) as string]] : [])))
  const list = useRead(JSON.stringify({ q, cursor: c.cursor }), async () => settle(await api.GET('/api/v1/payments/attempts', {
    params: { query: compact({ channel: q.channel as PaymentChannel | undefined, state: q.state as AttemptState | undefined,
      state_mapping: q.state_mapping as TargetMapping | undefined, approval_provenance: q.approval_provenance as ApprovalProvenance | undefined,
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
  const select = (name: string, label: string, options: Record<string, string>) => (
    <label className="lbl">{label}
      <select className="input" name={name} defaultValue={params.get(name) ?? ''}>
        <option value="">همه</option>
        {Object.entries(options).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
      </select>
    </label>
  )
  return (
    <>
      <PageHeader title="تلاش‌های پرداخت" subtitle="مدل سایه پرداخت؛ وضعیت خام، وضعیت تلاش، درگاه و منشأ تأیید جدا هستند.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <Card title="فیلترها">
        <form className="bi-filters" onSubmit={apply} aria-label="فیلتر تلاش‌ها">
          {select('channel', 'کانال', channelLabel)}
          {select('state', 'وضعیت تلاش', attemptStateLabel)}
          {select('state_mapping', 'نگاشت وضعیت', mappingLabel)}
          {select('approval_provenance', 'منشأ تأیید', provenanceLabel)}
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <Card title="فهرست تلاش‌ها">
        <ReadStates state={list} what="تلاش‌ها" blockedDetail={blockedDetail} />
        {list.status === 'ok' && (list.data.items.length === 0
          ? <StateBlock kind="empty" title={Object.keys(q).length ? 'نتیجه‌ای برای این فیلتر نیست' : 'تلاشی وارد نشده است'} />
          : <AttemptTable items={list.data.items} />)}
        {list.status === 'ok' && <Pager page={list.data.page} c={c} label="صفحه‌بندی تلاش‌ها" />}
      </Card>
    </>
  )
}

function AttemptTable({ items }: { items: PaymentAttempt[] }) {
  return (
    <div className="bi-table-wrap">
      <table className="bi-table">
        <caption className="sr">تلاش‌های پرداخت</caption>
        <thead>
          <tr><th scope="col">منبع</th><th scope="col">فاکتور</th><th scope="col">مبلغ</th><th scope="col">وضعیت خام</th><th scope="col">وضعیت تلاش</th>
            <th scope="col">کانال</th><th scope="col">منشأ تأیید</th><th scope="col">درگاه</th></tr>
        </thead>
        <tbody>
          {items.map(a => (
            <tr key={a.attempt_id}>
              <td className="bi-ltr">{`${a.source.source_spec} #${a.source.source_key}`}</td>
              <td>{a.invoice_id ? <Link to={`/billing/invoices/${a.invoice_id}`}>فاکتور</Link> : <span className="muted">نامعلوم (حل نشد)</span>}</td>
              <td><AmountText amount={a.amount} /></td>
              <td className="bi-ltr">{text(a.state_raw)}</td>
              <td>{a.state ? attemptStateLabel[a.state] : <StatusPill tone="amber" icon="question" label={mappingLabel[a.state_mapping]} />}</td>
              <td>{channelLabel[a.channel]}</td>
              <td>{provenanceLabel[a.approval_provenance]}</td>
              <td>{a.gateway ? (a.gateway.state ? gatewayStateLabel[a.gateway.state] : 'نامعلوم') : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
