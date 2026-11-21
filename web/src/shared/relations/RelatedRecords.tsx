import './relations.css'
import { useState } from 'react'
import { Link } from 'react-router'
import { useApi, type CaseSummary, type InvoiceSummary, type PageInfo } from '../api'
import { settle, useRead } from '../read/load'
import { ReadStates } from '../read/ReadStates'
import { Button, StateBlock } from '../ui/primitives'
import { casesBlockedDetail, invoicesBlockedDetail, text } from './labels'
import { RelationLink, RelationPill } from './RelationLink'

/**
 * RM-P01 read-only relationship sections. Each one reads a module-owned
 * list endpoint with an exact native-id filter (customer_id / case_id), so
 * the server returns only rows whose own link is RESOLVED to that id; the
 * section never widens, joins or infers anything itself. Every non-data
 * state (loading, denied, blocked, failed) is rendered as such, never as
 * empty, and pages follow the server cursor only.
 */

const PAGE = 10

function useCursorStack() {
  const [stack, setStack] = useState<string[]>([])
  return { cursor: stack[stack.length - 1], depth: stack.length,
    back: () => setStack(s => s.slice(0, -1)), next: (c: string) => setStack(s => [...s, c]) }
}

function RelationPager({ page, c, label }: { page: PageInfo; c: ReturnType<typeof useCursorStack>; label: string }) {
  if (c.depth === 0 && !page.has_more) return null
  return (
    <nav className="rel-pager" aria-label={label}>
      <span className="muted">{`صفحه ${c.depth + 1}`}</span>
      <Button small disabled={c.depth === 0} onClick={c.back}>صفحه قبل</Button>
      <Button small disabled={!page.has_more || !page.next_cursor} onClick={() => { if (page.next_cursor) c.next(page.next_cursor) }}>صفحه بعد</Button>
    </nav>
  )
}

const presenceText = (p: string) => (p === 'PRESENT' ? 'حاضر' : 'در آخرین دسته نیست (حذف نشده)')

/** Cases whose customer link is RESOLVED to this customer. */
export function CustomerCases({ customerId }: { customerId: string }) {
  const api = useApi()
  const c = useCursorStack()
  const cases = useRead(`rel-cases:${customerId}:${c.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/cases', {
    params: { query: { customer_id: customerId, limit: PAGE, ...(c.cursor ? { cursor: c.cursor } : {}) } } }),
  e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  return (
    <>
      <ReadStates state={cases} what="پرونده‌های مرتبط" blockedDetail={casesBlockedDetail} />
      {cases.status === 'ok' && (cases.data.items.length === 0
        ? <StateBlock kind="empty" title="پرونده‌ای با پیوند تأییدشده به این مشتری ثبت نشده است"
          detail="پرونده‌هایی که پیوند مشتری‌شان مرتبط، نامعلوم یا در تعارض است به این مشتری نسبت داده نمی‌شوند؛ تلفن و کلید قدیمی هرگز پیوند نمی‌سازند." />
        : <CaseTable items={cases.data.items} customerId={customerId} />)}
      {cases.status === 'ok' && <RelationPager page={cases.data.page} c={c} label="صفحه‌بندی پرونده‌های مشتری" />}
    </>
  )
}

function CaseTable({ items, customerId }: { items: CaseSummary[]; customerId: string }) {
  return (
    <div className="rel-table-wrap">
      <table className="rel-table">
        <caption className="sr">پرونده‌های دارای پیوند تأییدشده به این مشتری</caption>
        <thead><tr><th scope="col">پرونده</th><th scope="col">وضعیت (خام)</th><th scope="col">پیوند مشتری</th><th scope="col">حضور</th></tr></thead>
        <tbody>
          {items.map(x => {
            // Defensive: navigate only when the row itself confirms the link.
            const confirmed = x.customer_link_state === 'RESOLVED' && x.customer_id === customerId
            const name = `${x.source.source_spec} #${x.source.source_key}`
            return (
              <tr key={x.case_id}>
                <td data-label="پرونده">{confirmed ? <Link to={`/sales/cases/${x.case_id}`}><span className="rel-ltr">{name}</span></Link> : <span className="rel-ltr">{name}</span>}</td>
                <td data-label="وضعیت (خام)"><span className="rel-ltr">{`${x.state_namespace}: ${text(x.state_raw)}`}</span></td>
                <td data-label="پیوند مشتری"><RelationPill state={x.customer_link_state} /></td>
                <td data-label="حضور">{presenceText(x.presence)}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

/**
 * Invoices whose own link is RESOLVED to the given customer and/or case
 * (both filters are conjunctive on the server). The current record's own
 * column shows only its state; the other relationship links out only when
 * RESOLVED.
 */
export function RelatedInvoices({ customerId, caseId }: { customerId?: string; caseId?: string }) {
  const api = useApi()
  const c = useCursorStack()
  const invoices = useRead(`rel-inv:${customerId ?? ''}:${caseId ?? ''}:${c.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/invoices', {
    params: { query: { ...(customerId ? { customer_id: customerId } : {}), ...(caseId ? { case_id: caseId } : {}), limit: PAGE,
      ...(c.cursor ? { cursor: c.cursor } : {}) } } }),
  e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  const owner = caseId ? 'این پرونده' : 'این مشتری'
  return (
    <>
      <ReadStates state={invoices} what="فاکتورهای مرتبط" blockedDetail={invoicesBlockedDetail} />
      {invoices.status === 'ok' && (invoices.data.items.length === 0
        ? <StateBlock kind="empty" title={`فاکتوری با پیوند تأییدشده به ${owner} ثبت نشده است`}
          detail="فاکتورهای دارای پیوند مرتبط، نامعلوم یا در تعارض نمایش داده نمی‌شوند و از پیوندهای ماژول دیگر استنتاج نمی‌شوند." />
        : <InvoiceTable items={invoices.data.items} customerId={customerId} caseId={caseId} />)}
      {invoices.status === 'ok' && <RelationPager page={invoices.data.page} c={c} label={`صفحه‌بندی فاکتورهای ${owner}`} />}
    </>
  )
}

function InvoiceTable({ items, customerId, caseId }: { items: InvoiceSummary[]; customerId: string | undefined; caseId: string | undefined }) {
  return (
    <div className="rel-table-wrap">
      <table className="rel-table">
        <caption className="sr">فاکتورهای دارای پیوند تأییدشده</caption>
        <thead><tr><th scope="col">فاکتور</th><th scope="col">پیوند مشتری</th><th scope="col">پیوند پرونده</th><th scope="col">اقلام · مراحل</th><th scope="col">وضعیت نامعلوم</th></tr></thead>
        <tbody>
          {items.map(v => {
            const custOK = !customerId || (v.customer.state === 'RESOLVED' && v.customer.id === customerId)
            const caseOK = !caseId || (v.case.state === 'RESOLVED' && v.case.id === caseId)
            const name = `${v.source.source_spec} #${v.source.source_key}`
            return (
              <tr key={v.invoice_id}>
                <td data-label="فاکتور">
                  {custOK && caseOK ? <Link to={`/billing/invoices/${v.invoice_id}`}><span className="rel-ltr">{name}</span></Link> : <span className="rel-ltr">{name}</span>}
                  {v.code_raw ? <span className="muted rel-ltr"> · {v.code_raw}</span> : null}
                </td>
                <td data-label="پیوند مشتری">{customerId
                  ? <RelationPill state={v.customer.state} />
                  : <RelationLink state={v.customer.state} id={v.customer.id} to={id => `/customers/${id}`}>مشتری</RelationLink>}</td>
                <td data-label="پیوند پرونده">{caseId
                  ? <RelationPill state={v.case.state} />
                  : <RelationLink state={v.case.state} id={v.case.id} to={id => `/sales/cases/${id}`}>پرونده</RelationLink>}</td>
                <td data-label="اقلام · مراحل" className="num">{`${v.item_count} · ${v.stage_count}`}</td>
                <td data-label="وضعیت نامعلوم" className="num">{v.unknown_state_count}</td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
