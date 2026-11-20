import './workspace.css'
import { Fragment, useState, type ReactNode } from 'react'
import { Link } from 'react-router'
import { useApi, type AttemptHistoryEntry, type InvoiceDetail, type InvoiceTotals, type PaymentAttempt, type ReviewAttempt } from '../../shared/api'
import { settle, useRead, type ReadState } from '../../shared/read/load'
import { ReadStates } from '../../shared/read/ReadStates'
import { RelationLink } from '../../shared/relations/RelationLink'
import { formatInstant } from '../../shared/ui/time'
import { Button, StateBlock, StatusPill } from '../../shared/ui/primitives'
import { Gates, ReviewFlags } from '../payments/pages/FinanceReviewPage'
import { money } from '../payments/labels'
import { AmountText, ClassPill, SourceText } from './components'
import { attemptStateLabel, channelLabel, gatewayStateLabel, mappingLabel, provenanceLabel, text, unitLabel } from './labels'

/**
 * RM-P02 invoice workspace building blocks (read only). Every section owns
 * its read and its failure state; nothing here sums across units, invents a
 * canonical total or treats a gateway / legacy acceptance as a Finance
 * decision. Relationships navigate only when RESOLVED.
 */

/**
 * Placeholder of an invoice-dependent section while the invoice detail is
 * not successfully available. Dependent reads (items, stages, Finance
 * review) are neither mounted nor requested until the detail read returns
 * SUCCESS; if it is denied, blocked, not found or failed, they never run.
 */
export function AwaitInvoice({ detail }: { detail: ReadState<unknown> }) {
  if (detail.status === 'loading') return <StateBlock kind="loading" title="در انتظار دریافت فاکتور…" />
  return <StateBlock kind="unknown" title="این بخش فقط پس از دریافت موفق فاکتور نمایش داده می‌شود" />
}

/** In-page section navigation (native anchors; no client-side routing). */
export function WorkspaceNav({ sections }: { sections: [string, string][] }) {
  return (
    <nav className="ws-nav" aria-label="بخش‌های فاکتور">
      <ul>{sections.map(([id, label]) => <li key={id}><a href={`#${id}`}>{label}</a></li>)}</ul>
    </nav>
  )
}

/** A navigation target wrapping one section card. */
export function Anchor({ id, children }: { id: string; children: ReactNode }) {
  return <div id={id} className="ws-anchor">{children}</div>
}

/**
 * The overview: identity, relationships, completeness and the per-unit
 * reconciliation class. It states explicitly that no canonical total or
 * final lifecycle exists; unknown or missing evidence is shown as such.
 */
export function InvoiceOverview({ d, totals, canPay }: { d: InvoiceDetail; totals: ReadState<InvoiceTotals | null>; canPay: boolean }) {
  const v = d.invoice
  return (
    <div className="ws-overview">
      <dl className="ws-facts">
        <div><dt>کد (خام)</dt><dd className="bi-ltr">{text(v.code_raw)}</dd></div>
        <div><dt>منبع</dt><dd><SourceText source={v.source} /></dd></div>
        <div><dt>وضعیت چرخه</dt><dd><StatusPill tone="slate" icon="question" label="تعیین نشده (TAD #20)" /></dd></div>
        <div><dt>پیوند مشتری</dt><dd><RelationLink state={v.customer.state} id={v.customer.id} to={id => `/customers/${id}`}>مشاهده مشتری</RelationLink></dd></div>
        <div><dt>پیوند پرونده</dt><dd><RelationLink state={v.case.state} id={v.case.id} to={id => `/sales/cases/${id}`}>مشاهده پرونده</RelationLink></dd></div>
        <div><dt>اقلام · مراحل</dt><dd><span className="num">{`${v.item_count} · ${v.stage_count}`}</span>
          {(!v.items_source_complete || !v.stages_source_complete) && <> <StatusPill tone="amber" icon="alert" label="منبع ناقص" /></>}</dd></div>
        <div><dt>وضعیت‌های ناشناخته</dt><dd>{v.unknown_state_count > 0
          ? <StatusPill tone="amber" icon="question" label={`${v.unknown_state_count} مقدار خام نگاشت‌نشده`} />
          : <StatusPill tone="green" icon="check" label="بدون مقدار ناشناخته" />}</dd></div>
        <div><dt>تطبیق مبالغ (به تفکیک واحد)</dt><dd><UnitClasses totals={totals} canPay={canPay} /></dd></div>
      </dl>
      <p className="muted ws-note" role="note">جمع قطعی فاکتور تعیین نشده است (TAD #19). مبالغ هر واحد جدا نمایش داده می‌شوند و هرگز با هم جمع یا تبدیل نمی‌شوند.</p>
    </div>
  )
}

function UnitClasses({ totals, canPay }: { totals: ReadState<InvoiceTotals | null>; canPay: boolean }) {
  if (!canPay) return <StatusPill tone="slate" icon="lock" label="دسترسی به شواهد پرداخت ندارید" />
  if (totals.status === 'loading') return <span className="muted">…</span>
  if (totals.status !== 'ok' || !totals.data) return <StatusPill tone="slate" icon="question" label="در دسترس نیست" />
  return (
    <span className="ws-pills">
      {totals.data.units.map(u => <span key={u.unit} className="ws-unit"><span className="bi-ltr">{unitLabel[u.unit] ?? u.unit}</span> <ClassPill cls={u.class} /></span>)}
      {!totals.data.payment_evidence_complete && <StatusPill tone="amber" icon="alert" label="شواهد پرداخت ناقص" />}
    </span>
  )
}

/** Payment attempts with gateway evidence and an on-demand state history per attempt. */
export function AttemptsTable({ items, zone }: { items: PaymentAttempt[]; zone: string }) {
  const [open, setOpen] = useState<Record<string, boolean>>({})
  return (
    <div className="ws-table-wrap">
      <table className="ws-table">
        <caption className="sr">تلاش‌های پرداخت این فاکتور</caption>
        <thead><tr><th scope="col">وضعیت (خام → نگاشت)</th><th scope="col">کانال · منشأ تأیید</th><th scope="col">مبلغ</th><th scope="col">درگاه</th><th scope="col">سابقه</th></tr></thead>
        <tbody>
          {items.map(a => (
            <Fragment key={a.attempt_id}>
              <tr>
                <td data-label="وضعیت (خام → نگاشت)"><span className="bi-ltr">{text(a.state_raw)}</span>{' '}
                  {a.state ? <StatusPill tone="blue" icon="check" label={attemptStateLabel[a.state] ?? a.state} />
                    : <StatusPill tone="amber" icon="question" label={mappingLabel[a.state_mapping]} />}</td>
                <td data-label="کانال · منشأ تأیید">{channelLabel[a.channel]} · {provenanceLabel[a.approval_provenance]}</td>
                <td data-label="مبلغ"><AmountText amount={a.amount} /></td>
                <td data-label="درگاه">{a.gateway
                  ? <>{a.gateway.state ? gatewayStateLabel[a.gateway.state] : 'درگاه: نامعلوم'} <span className="muted bi-ltr">({text(a.gateway.provider_raw)})</span></>
                  : <span className="muted">—</span>}</td>
                <td data-label="سابقه">
                  <Button small aria-expanded={!!open[a.attempt_id]} aria-controls={`hist-${a.attempt_id}`}
                    onClick={() => setOpen(o => ({ ...o, [a.attempt_id]: !o[a.attempt_id] }))}>
                    {open[a.attempt_id] ? 'بستن سابقه' : 'نمایش سابقه'}
                  </Button>
                </td>
              </tr>
              {open[a.attempt_id] && (
                <tr className="ws-sub"><td colSpan={5} id={`hist-${a.attempt_id}`}><AttemptHistory attemptId={a.attempt_id} zone={zone} /></td></tr>
              )}
            </Fragment>
          ))}
        </tbody>
      </table>
      <p className="muted ws-note" role="note">پذیرش درگاه یا تأیید قدیمی، تأیید نهایی مالی یا ثبت حسابداری نیست.</p>
    </div>
  )
}

function useCursorStack() {
  const [stack, setStack] = useState<string[]>([])
  return { cursor: stack[stack.length - 1], depth: stack.length, back: () => setStack(s => s.slice(0, -1)), next: (c: string) => setStack(s => [...s, c]) }
}

function MiniPager({ hasMore, next, c, label }: { hasMore: boolean; next: string | null | undefined; c: ReturnType<typeof useCursorStack>; label: string }) {
  if (c.depth === 0 && !hasMore) return null
  return (
    <nav className="ws-pager" aria-label={label}>
      <span className="muted">{`صفحه ${c.depth + 1}`}</span>
      <Button small disabled={c.depth === 0} onClick={c.back}>صفحه قبل</Button>
      <Button small disabled={!hasMore || !next} onClick={() => { if (next) c.next(next) }}>صفحه بعد</Button>
    </nav>
  )
}

const historyBlocked = 'خواندن صورت‌حساب و پرداخت پشت پرچم ویژگی است (billing_payments.read_shadow · FEATURE_DISABLED).'

/** The append-only state history of one legacy payment attempt. */
export function AttemptHistory({ attemptId, zone }: { attemptId: string; zone: string }) {
  const api = useApi()
  const c = useCursorStack()
  const h = useRead(`ah:${attemptId}:${c.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/payments/attempts/{attempt_id}/history', {
    params: { path: { attempt_id: attemptId }, query: { limit: 10, ...(c.cursor ? { cursor: c.cursor } : {}) } } }),
  e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  return (
    <div className="ws-history" aria-label="سابقه وضعیت تلاش">
      <ReadStates state={h} what="سابقه تلاش" blockedDetail={historyBlocked} />
      {h.status === 'ok' && (h.data.items.length === 0 ? <StateBlock kind="empty" title="تغییر وضعیتی ثبت نشده است" /> : (
        <ol className="ws-list">{h.data.items.map((x: AttemptHistoryEntry) => (
          <li key={x.history_id}><span className="bi-ltr">{`${text(x.old_state_raw)} → ${text(x.new_state_raw)}`}</span>
            {x.regression && <> <StatusPill tone="red" icon="alert" label="بازگشت از پذیرفته‌شده" /></>}
            <span className="muted"> · {formatInstant(x.recorded_at, zone)}</span></li>
        ))}</ol>
      ))}
      {h.status === 'ok' && <MiniPager hasMore={h.data.page.has_more} next={h.data.page.next_cursor} c={c} label="صفحه‌بندی سابقه تلاش" />}
    </div>
  )
}

const reviewBlocked = 'بررسی مالی پشت پرچم ویژگی است (finance.review_read · FEATURE_DISABLED).'

/**
 * Target-submitted attempts of this invoice awaiting Finance review, read
 * through the review queue filtered by the attempt's own invoice id. Shows
 * evidence readiness (D-43) and the maker / checker gate; there is no
 * approve or reject control (D-41, OPD-08, OPD-09).
 */
export function FinanceReviewForInvoice({ invoiceId, zone }: { invoiceId: string; zone: string }) {
  const api = useApi()
  const c = useCursorStack()
  const q = useRead(`frq-inv:${invoiceId}:${c.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/finance/review/attempts', {
    params: { query: { invoice_id: invoiceId, limit: 10, ...(c.cursor ? { cursor: c.cursor } : {}) } } }),
  e => e.data && e.page ? { items: e.data, page: e.page } : undefined))
  return (
    <>
      <p className="muted ws-note" role="note">تصمیم مالی (تأیید یا رد) در این نسخه وجود ندارد؛ این بخش فقط وضعیت شواهد و دلیل مسدود بودن تصمیم را نشان می‌دهد.</p>
      <ReadStates state={q} what="بررسی مالی" blockedDetail={reviewBlocked} />
      {q.status === 'ok' && (q.data.items.length === 0
        ? <StateBlock kind="empty" title="پرداخت ثبت‌شده‌ای از این فاکتور در انتظار بررسی مالی نیست"
          detail="فقط پرداخت‌هایی که در سامانه جدید ثبت شده‌اند در صف بررسی قرار می‌گیرند؛ تلاش‌های قدیمی در بخش تلاش‌های پرداخت هستند." />
        : <ReviewTable items={q.data.items} zone={zone} />)}
      {q.status === 'ok' && <MiniPager hasMore={q.data.page.has_more} next={q.data.page.next_cursor} c={c} label="صفحه‌بندی بررسی مالی" />}
    </>
  )
}

function ReviewTable({ items, zone }: { items: ReviewAttempt[]; zone: string }) {
  return (
    <div className="ws-table-wrap">
      <table className="ws-table">
        <caption className="sr">پرداخت‌های این فاکتور در انتظار بررسی مالی</caption>
        <thead><tr><th scope="col">مبلغ</th><th scope="col">مرحله · ثبت</th><th scope="col">شواهد و تفکیک وظایف</th><th scope="col">تصمیم</th></tr></thead>
        <tbody>
          {items.map(a => (
            <tr key={a.attempt_id}>
              <td data-label="مبلغ"><Link to={`/finance/review/${a.attempt_id}`}>{money(a.amount, a.unit)}</Link></td>
              <td data-label="مرحله · ثبت"><span className="num">{`#${a.stage_no}`}</span> · {formatInstant(a.submitted_at, zone)}</td>
              <td data-label="شواهد و تفکیک وظایف"><ReviewFlags a={a} /></td>
              <td data-label="تصمیم"><Gates a={a} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
