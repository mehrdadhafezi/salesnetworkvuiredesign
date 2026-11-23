import { Link } from 'react-router'
import { useApi, type InvoiceSummary, type PaymentAttempt, type ReviewAttempt } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatFaNumber } from '../../../shared/ui/ds'
import { formatInstant } from '../../../shared/ui/time'
import { StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { AmountText } from '../../billing/components'
import { attemptStateLabel as legacyStateLabel, blockedDetail as billingBlocked, channelLabel, gatewayStateLabel, mappingLabel, provenanceLabel, text } from '../../billing/labels'
import { Pager, useCursors } from '../../billing/pager'
import { Gates, ReviewFlags } from '../../payments/review'
import { attemptStateLabel as targetStateLabel, money, receiptStateLabel, receiptStateTone } from '../../payments/labels'
import { FINANCIAL_OPERATIONS } from './operations'

export const reviewBlocked = 'صف بررسی مالی پشت پرچم finance.review_read است (FEATURE_DISABLED).'
export const invoiceHref = (id: string, section?: string) => `/billing/invoices/${id}${section ? `#${section}` : ''}`
export const shortId = (id: string) => id.slice(0, 8)

/** A count read from a named aggregate, or the honest reason it is not available (never zero). */
export function Count({ value, unavailable }: { value: number | null | undefined; unavailable?: string | undefined }) {
  if (value === null || value === undefined) {
    return <span className="fw-count fw-count-na" title={unavailable}>—<small> در دسترس نیست</small></span>
  }
  return <span className="fw-count num">{formatFaNumber(value)}</span>
}

/** Financial operations with the grounded reason each is unavailable: statuses, never controls. */
export function OperationsStatus() {
  return (
    <div>
      <p className="muted fw-hint">هیچ‌یک از عملیات مالی زیر قرارداد مصوب ندارد؛ بنابراین هیچ دکمه، پیوند یا کنترل غیرفعالی برای آن‌ها نمایش داده نمی‌شود.</p>
      <ul className="fw-ops-list" aria-label="وضعیت عملیات مالی">
        {FINANCIAL_OPERATIONS.map(o => (
          <li key={o.id}>
            <span className="fw-ops-head"><b>{o.label}</b> <StatusPill tone="slate" icon="ban" label="در دسترس نیست" /></span>
            <span className="muted">{o.reason}</span>
            <span className="muted">{`وابسته به: ${o.blockedBy.join('، ')}`}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}

/** Receipt states of one target attempt, counted (receipts, not money). */
function ReceiptStates({ a }: { a: ReviewAttempt }) {
  if (a.receipts.length === 0) return <StatusPill tone="amber" icon="alert" label="بدون فیش" />
  const by = new Map<string, number>()
  for (const r of a.receipts) by.set(r.state, (by.get(r.state) ?? 0) + 1)
  return (
    <span className="fw-pills">
      {[...by].map(([s, n]) => <StatusPill key={s} tone={receiptStateTone[s] ?? 'slate'} icon="inbox" label={`${formatFaNumber(n)} × ${receiptStateLabel[s] ?? s}`} />)}
    </span>
  )
}

/**
 * Target-submitted attempts awaiting Finance review. The amount is the
 * submitted (claimed) amount in its own unit: never a receipt of funds,
 * never summed. Each row links to the verified invoice detail (RM-P02) and
 * its legacy evidence, and to the read-only review item.
 */
export function ReviewTable({ items, zone, caption }: { items: ReviewAttempt[]; zone: string; caption: string }) {
  return (
    <div className="fw-table-wrap">
      <table className="fw-table fw-stack">
        <caption className="sr">{caption}</caption>
        <thead>
          <tr><th scope="col">مبلغ ثبت‌شده</th><th scope="col">فاکتور و مرحله</th><th scope="col">منبع و وضعیت</th><th scope="col">فیش‌ها و شواهد</th><th scope="col">تصمیم مالی</th></tr>
        </thead>
        <tbody>
          {items.map(a => (
            <tr key={a.attempt_id} data-attempt={a.attempt_id}>
              <td data-label="مبلغ ثبت‌شده">
                <span className="fw-amount">{money(a.amount, a.unit)}</span>
                <span className="fw-id">ادعای ثبت‌شده؛ وصول وجه نیست</span>
                <Link to={`/finance/review/${a.attempt_id}`}>جزئیات مورد بررسی</Link>
              </td>
              <td data-label="فاکتور و مرحله">
                <span className="fw-links">
                  <Link to={invoiceHref(a.invoice_id)}>{`فاکتور ${shortId(a.invoice_id)}…`}</Link>
                  <span className="num">{`مرحله #${a.stage_no}`}</span>
                </span>
                <span className="fw-links">
                  <Link to={invoiceHref(a.invoice_id, 'inv-attempts')}>تلاش‌های قدیمی</Link>
                  <Link to={invoiceHref(a.invoice_id, 'inv-receipts')}>ارجاع رسیدها</Link>
                  <Link to={invoiceHref(a.invoice_id, 'inv-review')}>بررسی این فاکتور</Link>
                </span>
              </td>
              <td data-label="منبع و وضعیت">
                <span className="fw-pills">
                  <StatusPill tone="blue" icon="inbox" label="ثبت در سامانه جدید (نه قدیمی)" />
                  <StatusPill tone="amber" icon="question" label={targetStateLabel[a.state] ?? a.state} />
                </span>
                <span className="fw-id">{`ثبت: ${formatInstant(a.submitted_at, zone)} · دسته ${shortId(a.batch_id)}…`}</span>
              </td>
              <td data-label="فیش‌ها و شواهد"><ReceiptStates a={a} /> <ReviewFlags a={a} /></td>
              <td data-label="تصمیم مالی"><Gates a={a} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

/** One page of the review queue for a fixed server filter (exceptions / receipts). */
export function ReviewList({ query, zone, empty }: { query: Record<string, string>; zone: string; empty: string }) {
  const api = useApi()
  const c = useCursors()
  const q = useRead(`fw-rv:${JSON.stringify(query)}:${c.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/finance/review/attempts', {
    params: { query: { ...(query as { evidence_state?: 'PENDING_FINALISATION' | 'READY' | 'NOT_RECOVERABLE'; invoice_id?: string }), limit: 25, ...(c.cursor ? { cursor: c.cursor } : {}) } } }),
  e => e.data && e.page ? { items: e.data, page: e.page } : undefined))
  return (
    <>
      <ReadStates state={q} what="صف بررسی" blockedDetail={reviewBlocked} />
      {q.status === 'ok' && (q.data.items.length === 0 ? <StateBlock kind="empty" title={empty} />
        : <ReviewTable items={q.data.items} zone={zone} caption="پرداخت‌های ثبت‌شده این صف" />)}
      {q.status === 'ok' && <Pager page={q.data.page} c={c} label="صفحه‌بندی صف" />}
    </>
  )
}

type AttemptQuery = Record<string, string>

/**
 * Legacy payment attempts for one server filter. Raw state beside the
 * mapped state, channel, approval provenance and gateway evidence; a legacy
 * "accepted" is never a Finance decision, a settlement or a posting.
 */
export function LegacyAttemptList({ query, empty }: { query: AttemptQuery; empty: string }) {
  const api = useApi()
  const c = useCursors()
  const q = useRead(`fw-at:${JSON.stringify(query)}:${c.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/payments/attempts', {
    params: { query: compact({ ...(query as Record<string, never>), has_state_regression: query.has_state_regression === undefined ? undefined : query.has_state_regression === 'true',
      cursor: c.cursor, limit: 25 }) } }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  return (
    <>
      <ReadStates state={q} what="تلاش‌های پرداخت" blockedDetail={billingBlocked} />
      {q.status === 'ok' && (q.data.items.length === 0 ? <StateBlock kind="empty" title={empty} /> : <AttemptRows items={q.data.items} />)}
      {q.status === 'ok' && <Pager page={q.data.page} c={c} label="صفحه‌بندی تلاش‌ها" />}
    </>
  )
}

function AttemptRows({ items }: { items: PaymentAttempt[] }) {
  return (
    <div className="fw-table-wrap">
      <table className="fw-table fw-stack">
        <caption className="sr">تلاش‌های پرداخت قدیمی</caption>
        <thead>
          <tr><th scope="col">منبع</th><th scope="col">فاکتور</th><th scope="col">مبلغ (خام)</th><th scope="col">وضعیت خام → نگاشت</th><th scope="col">کانال · منشأ تأیید</th><th scope="col">درگاه</th></tr>
        </thead>
        <tbody>
          {items.map(a => (
            <tr key={a.attempt_id}>
              <td data-label="منبع"><span className="fw-ltr">{`${a.source.source_spec} #${a.source.source_key}`}</span>
                {a.source.presence !== 'PRESENT' && <> <StatusPill tone="amber" icon="alert" label="غایب در آخرین دسته (حذف نشده)" /></>}
                <span className="fw-id"><StatusPill tone="violet" icon="inbox" label="شاهد قدیمی" /></span></td>
              <td data-label="فاکتور">{a.invoice_id
                ? <span className="fw-links"><Link to={invoiceHref(a.invoice_id)}>فاکتور</Link><Link to={invoiceHref(a.invoice_id, 'inv-attempts')}>سابقه تلاش‌ها</Link></span>
                : <StatusPill tone="amber" icon="question" label="فاکتور حل نشد (UNKNOWN)" />}</td>
              <td data-label="مبلغ (خام)"><AmountText amount={a.amount} /></td>
              <td data-label="وضعیت خام → نگاشت"><span className="fw-ltr">{text(a.state_raw)}</span>{' '}
                {a.state ? <StatusPill tone="blue" icon="check" label={legacyStateLabel[a.state] ?? a.state} />
                  : <StatusPill tone="amber" icon="question" label={mappingLabel[a.state_mapping]} />}</td>
              <td data-label="کانال · منشأ تأیید">{`${channelLabel[a.channel]} · ${provenanceLabel[a.approval_provenance]}`}</td>
              <td data-label="درگاه">{a.gateway ? (a.gateway.state ? gatewayStateLabel[a.gateway.state] : 'درگاه: نامعلوم') : <span className="muted">—</span>}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

const linkLabel: Record<string, string> = { RESOLVED: 'تأییدشده', RELATED: 'مرتبط (اثبات نشده)', UNKNOWN: 'نامعلوم', CONFLICT: 'تعارض', NOT_RECOVERABLE: 'غیرقابل بازیابی' }
const linkTone = (s: string) => (s === 'RESOLVED' ? 'green' : s === 'CONFLICT' || s === 'NOT_RECOVERABLE' ? 'red' : 'amber') as 'green' | 'red' | 'amber'

type InvoiceQuery = Record<string, string>

/** Invoices for one server filter; the lifecycle stays UNKNOWN (TAD #20). */
export function InvoiceList({ query, empty }: { query: InvoiceQuery; empty: string }) {
  const api = useApi()
  const c = useCursors()
  const q = useRead(`fw-inv:${JSON.stringify(query)}:${c.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/invoices', {
    params: { query: compact({ ...(query as Record<string, never>), has_unknown_status: query.has_unknown_status === undefined ? undefined : query.has_unknown_status === 'true',
      cursor: c.cursor, limit: 25 }) } }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  return (
    <>
      <ReadStates state={q} what="فاکتورها" blockedDetail={billingBlocked} />
      {q.status === 'ok' && (q.data.items.length === 0 ? <StateBlock kind="empty" title={empty} /> : <InvoiceRows items={q.data.items} />)}
      {q.status === 'ok' && <Pager page={q.data.page} c={c} label="صفحه‌بندی فاکتورها" />}
    </>
  )
}

function InvoiceRows({ items }: { items: InvoiceSummary[] }) {
  return (
    <div className="fw-table-wrap">
      <table className="fw-table fw-stack">
        <caption className="sr">فاکتورها</caption>
        <thead>
          <tr><th scope="col">فاکتور (منبع)</th><th scope="col">پیوند مشتری · پرونده</th><th scope="col">وضعیت‌های خام ناشناخته</th><th scope="col">اقلام · مراحل</th><th scope="col">شواهد</th></tr>
        </thead>
        <tbody>
          {items.map(x => (
            <tr key={x.invoice_id}>
              <td data-label="فاکتور (منبع)"><Link to={invoiceHref(x.invoice_id)} className="fw-ltr">{`${x.source.source_spec} #${x.source.source_key}`}</Link>
                {x.source.presence !== 'PRESENT' && <> <StatusPill tone="amber" icon="alert" label="غایب در آخرین دسته (حذف نشده)" /></>}</td>
              <td data-label="پیوند مشتری · پرونده"><span className="fw-pills">
                <StatusPill tone={linkTone(x.customer.state)} icon={x.customer.state === 'RESOLVED' ? 'check' : 'question'} label={`مشتری: ${linkLabel[x.customer.state] ?? x.customer.state}`} />
                <StatusPill tone={linkTone(x.case.state)} icon={x.case.state === 'RESOLVED' ? 'check' : 'question'} label={`پرونده: ${linkLabel[x.case.state] ?? x.case.state}`} />
              </span></td>
              <td data-label="وضعیت‌های خام ناشناخته" className="num">{formatFaNumber(x.unknown_state_count)}</td>
              <td data-label="اقلام · مراحل" className="num">{`${formatFaNumber(x.item_count)} · ${formatFaNumber(x.stage_count)}`}</td>
              <td data-label="شواهد"><span className="fw-links">
                <Link to={invoiceHref(x.invoice_id, 'inv-provenance')}>منشأ</Link>
                <Link to={invoiceHref(x.invoice_id, 'inv-stages')}>مراحل</Link>
                <Link to={invoiceHref(x.invoice_id, 'inv-attempts')}>تلاش‌ها</Link>
              </span></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
