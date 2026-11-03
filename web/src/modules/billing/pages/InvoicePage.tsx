import '../billing.css'
import { Fragment } from 'react'
import { Link, useParams } from 'react-router'
import { useApi, type InvoiceUnitTotals } from '../../../shared/api'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { settle, useRead, type ReadState } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { AmountText, ClassPill, MoneyText, ReadOnlyNote, ReceiptPill, SourceText, UserRefText, VocabularyPill } from '../components'
import {
  attemptStateLabel, blockedDetail, channelLabel, declaredLabel, gatewayStateLabel, mappingLabel, provenanceLabel, stageStateLabel, stateFieldLabel,
  sumStateLabel, text, unitLabel,
} from '../labels'
import { Pager, useCursors } from '../pager'
import { useSession } from '../../../shared/auth/currentSession'
import { PaymentSubmissionPanel } from '../../payments/PaymentSubmissionPanel'

/** Runs a read only when the server allows it; otherwise a settled null. */
async function gated<T>(on: boolean, run: () => Promise<ReadState<T>>): Promise<ReadState<T | null>> {
  return on ? run() : { status: 'ok', data: null, outcome: 'SUCCESS', allowed: undefined }
}

/**
 * Invoice detail (read only). Invoice status fields, payment stages, payment
 * attempts, gateway evidence and receipts are separate sections with their
 * own vocabularies; totals are per unit, side by side, with no winner. Every
 * list pages with the server cursor. No action control exists.
 */
export function InvoicePage() {
  const { invoiceId = '' } = useParams()
  const api = useApi()
  const session = useSession()
  const paymentFeature = session.status === 'ready' && session.session.features?.payment_submission === true
  const path = { invoice_id: invoiceId }
  const detail = useRead(`i:${invoiceId}`, async () => settle(await api.GET('/api/v1/invoices/{invoice_id}', { params: { path } }), e => e.data))
  const ic = useCursors()
  const items = useRead(`it:${invoiceId}:${ic.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/invoices/{invoice_id}/items', {
    params: { path, query: { limit: 25, ...(ic.cursor ? { cursor: ic.cursor } : {}) } } }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  const sc = useCursors()
  const stages = useRead(`st:${invoiceId}:${sc.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/invoices/{invoice_id}/stages', {
    params: { path, query: { limit: 25, ...(sc.cursor ? { cursor: sc.cursor } : {}) } } }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))
  const canPay = detail.status === 'ok' && actionHint(detail.allowed, 'payments.attempt.read').allowed
  const pc = useCursors()
  const pays = useRead(`p:${invoiceId}:${pc.cursor ?? ''}:${canPay}`, () => gated(canPay, async () => settle(await api.GET('/api/v1/invoices/{invoice_id}/payments', {
    params: { path, query: { limit: 25, ...(pc.cursor ? { cursor: pc.cursor } : {}) } } }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined)))
  const rc = useCursors()
  const receipts = useRead(`r:${invoiceId}:${rc.cursor ?? ''}:${canPay}`, () => gated(canPay, async () => settle(await api.GET('/api/v1/invoices/{invoice_id}/receipts', {
    params: { path, query: { limit: 25, ...(rc.cursor ? { cursor: rc.cursor } : {}) } } }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined)))
  const totals = useRead(`t:${invoiceId}:${canPay}`, () => gated(canPay, async () => settle(await api.GET('/api/v1/invoices/{invoice_id}/totals', { params: { path } }), e => e.data)))

  const title = detail.status === 'ok' ? `${detail.data.invoice.source.source_spec} #${detail.data.invoice.source.source_key}` : '…'
  const locked = <StateBlock kind="locked" title="دسترسی به شواهد پرداخت ندارید" />
  return (
    <>
      <PageHeader title={`فاکتور: ${title}`} subtitle="مدل سایه صورت‌حساب">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <p><Link to="/billing/invoices">بازگشت به فهرست فاکتورها</Link></p>
      <ReadOnlyNote />
      {paymentFeature && <PaymentSubmissionPanel invoiceId={invoiceId} />}
      <div className="bi-sections">
        <Card title="فاکتور">
          <ReadStates state={detail} what="فاکتور" blockedDetail={blockedDetail} />
          {detail.status === 'ok' && (
            <dl className="bi-kv">
              <dt>کد (خام)</dt><dd className="bi-ltr">{text(detail.data.invoice.code_raw)}</dd>
              <dt>وضعیت چرخه فاکتور</dt><dd><StatusPill tone="slate" icon="question" label="تعیین نشده — وضعیت قطعی ادعا نمی‌شود (TAD #20)" /></dd>
              <dt>فروشنده / صادرکننده</dt><dd><UserRefText user={detail.data.invoice.seller} /> · <UserRefText user={detail.data.invoice.issued_by} /></dd>
              <dt>طرح پرداخت (خام)</dt><dd className="bi-ltr">{text(detail.data.payment_plan_raw)}</dd>
              <dt>پیوند مشتری</dt><dd>{detail.data.invoice.customer.state} <span className="muted">— {detail.data.invoice.customer.basis}</span></dd>
              <dt>پیوند پرونده</dt><dd>{detail.data.invoice.case.state} <span className="muted">— {detail.data.invoice.case.basis}</span></dd>
            </dl>
          )}
        </Card>
        <Card title="شواهد وضعیت (هر فیلد جدا)">
          {detail.status === 'ok' && (
            <div className="bi-states">
              {detail.data.states.map(s => (
                <section key={s.field} className="bi-state" aria-label={stateFieldLabel[s.field] ?? s.field}>
                  <h3>{stateFieldLabel[s.field] ?? s.field}</h3>
                  <p className="bi-ltr">{text(s.raw)}</p>
                  <VocabularyPill mapping={s.mapping} />
                </section>
              ))}
            </div>
          )}
        </Card>
        <Card title="شواهد جمع (به تفکیک واحد؛ بدون برنده)">
          {!canPay ? locked : (
            <>
              <ReadStates state={totals} what="جمع‌ها" blockedDetail={blockedDetail} />
              {totals.status === 'ok' && totals.data && (
                <>
                  {!totals.data.payment_evidence_complete && <p className="muted">{`شواهد پرداخت ناقص است (تلاش‌های طبقه‌بندی‌نشده: ${totals.data.unclassified_attempts}).`}</p>}
                  <div className="bi-units">{totals.data.units.map(u => <UnitCard key={u.unit} u={u} />)}</div>
                </>
              )}
            </>
          )}
          {detail.status === 'ok' && (
            <dl className="bi-kv" aria-label="مبالغ اعلامی">
              {detail.data.declared.map(d => <Fragment key={d.kind}><dt>{declaredLabel[d.kind] ?? d.kind}</dt><dd><AmountText amount={d.amount} /></dd></Fragment>)}
            </dl>
          )}
        </Card>
        <Card title="اقلام (اسنپ‌شات صدور)">
          <ReadStates state={items} what="اقلام" blockedDetail={blockedDetail} />
          {items.status === 'ok' && (items.data.items.length === 0 ? <StateBlock kind="empty" title="قلمی ثبت نشده است" /> : (
            <ul className="bi-list" aria-label="اقلام">
              {items.data.items.map(it => (
                <li key={it.item_id}>{text(it.title_snapshot)} · <span className="bi-ltr">{`×${text(it.qty_raw)}`}</span> · <AmountText amount={it.line_total} /></li>
              ))}
            </ul>
          ))}
          {items.status === 'ok' && <Pager page={items.data.page} c={ic} label="صفحه‌بندی اقلام" />}
        </Card>
        <Card title="مراحل پرداخت">
          <ReadStates state={stages} what="مراحل" blockedDetail={blockedDetail} />
          {stages.status === 'ok' && (stages.data.items.length === 0 ? <StateBlock kind="empty" title="مرحله‌ای ثبت نشده است" /> : (
            <ul className="bi-list" aria-label="مراحل پرداخت">
              {stages.data.items.map(s => (
                <li key={s.stage_id}>
                  <span className="bi-ltr">{`#${text(s.stage_no_raw)} · ${text(s.state_raw)}`}</span>{' '}
                  {s.state ? <StatusPill tone="blue" icon="check" label={stageStateLabel[s.state] ?? s.state} /> : <StatusPill tone="amber" icon="question" label={mappingLabel[s.state_mapping]} />}
                  {' '}· <AmountText amount={s.requested} />
                </li>
              ))}
            </ul>
          ))}
          {stages.status === 'ok' && <Pager page={stages.data.page} c={sc} label="صفحه‌بندی مراحل" />}
        </Card>
        <Card title="تلاش‌های پرداخت">
          {!canPay ? locked : (
            <>
              <ReadStates state={pays} what="تلاش‌ها" blockedDetail={blockedDetail} />
              {pays.status === 'ok' && pays.data && (pays.data.items.length === 0 ? <StateBlock kind="empty" title="تلاش پرداختی ثبت نشده است" /> : (
                <ul className="bi-list" aria-label="تلاش‌های پرداخت">
                  {pays.data.items.map(a => (
                    <li key={a.attempt_id}>
                      <span className="bi-ltr">{text(a.state_raw)}</span>{' '}
                      {a.state ? <StatusPill tone="blue" icon="check" label={attemptStateLabel[a.state] ?? a.state} /> : <StatusPill tone="amber" icon="question" label={mappingLabel[a.state_mapping]} />}
                      {' '}· {channelLabel[a.channel]} · {provenanceLabel[a.approval_provenance]} · <AmountText amount={a.amount} />
                      {a.gateway && <> · {a.gateway.state ? gatewayStateLabel[a.gateway.state] : 'درگاه: نامعلوم'} <span className="muted bi-ltr">({text(a.gateway.provider_raw)})</span></>}
                    </li>
                  ))}
                </ul>
              ))}
              {pays.status === 'ok' && pays.data && <Pager page={pays.data.page} c={pc} label="صفحه‌بندی تلاش‌ها" />}
            </>
          )}
        </Card>
        <Card title="رسیدها (فقط فراداده ارجاع)">
          {!canPay ? locked : (
            <>
              <ReadStates state={receipts} what="رسیدها" blockedDetail={blockedDetail} />
              {receipts.status === 'ok' && receipts.data && (receipts.data.items.length === 0 ? <StateBlock kind="empty" title="ارجاع رسیدی ثبت نشده است" /> : (
                <ul className="bi-list" aria-label="رسیدها">
                  {receipts.data.items.map(r => (
                    <li key={r.receipt_id}><ReceiptPill state={r.state} /> <span className="bi-ltr">{`${r.owner_kind} · ${text(r.host)} · ${text(r.extension)}`}</span></li>
                  ))}
                </ul>
              ))}
              {receipts.status === 'ok' && receipts.data && <Pager page={receipts.data.page} c={rc} label="صفحه‌بندی رسیدها" />}
            </>
          )}
        </Card>
        <Card title="منشأ">
          {detail.status === 'ok' && (
            <dl className="bi-kv">
              <dt>منبع</dt><dd><SourceText source={detail.data.invoice.source} /></dd>
              <dt>دسته اول / آخر</dt><dd className="bi-ltr">{`${detail.data.invoice.source.first_batch_id} / ${detail.data.invoice.source.last_batch_id}`}</dd>
              <dt>کامل بودن منبع اقلام / مراحل</dt><dd>{`${detail.data.invoice.items_source_complete ? 'کامل' : 'ناقص'} / ${detail.data.invoice.stages_source_complete ? 'کامل' : 'ناقص'}`}</dd>
            </dl>
          )}
        </Card>
      </div>
    </>
  )
}

function UnitCard({ u }: { u: InvoiceUnitTotals }) {
  const sum = (label: string, s: InvoiceUnitTotals['item_sum']) => (
    <><dt>{label}</dt><dd>{s.money ? <MoneyText money={s.money} /> : <span className="muted">—</span>} <span className="muted">({sumStateLabel[s.state] ?? s.state})</span></dd></>
  )
  return (
    <section className="bi-unit" aria-label={`واحد ${unitLabel[u.unit] ?? u.unit}`}>
      <h3>{unitLabel[u.unit] ?? u.unit} <ClassPill cls={u.class} /></h3>
      <dl className="bi-kv">
        {u.declared.map(d => <Fragment key={d.kind}><dt>{declaredLabel[d.kind] ?? d.kind}</dt><dd><MoneyText money={d.money} /></dd></Fragment>)}
        {sum('جمع اقلام', u.item_sum)}
        {sum('جمع مراحل', u.stage_sum)}
        {sum('جمع تلاش‌های پذیرفته‌شده', u.payment_sum)}
      </dl>
    </section>
  )
}
