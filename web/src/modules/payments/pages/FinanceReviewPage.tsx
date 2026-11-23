import '../../writes/writes.css'
import '../payments.css'
import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type PageInfo, type ReviewAttempt } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { settle, useRead, type ReadState } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatFaNumber } from '../../../shared/ui/ds'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { Pager, useCursors } from '../../billing/pager'
import { ReviewTable } from '../../finance/workspace/components'
import { FinanceSectionNav } from '../../finance/workspace/sections'
import { evidenceStateLabel, evidenceStateTone } from '../labels'
import { Gates, ReviewFlags } from '../review'

const blockedDetail = 'صف بررسی مالی پشت پرچم finance.review_read است.'

// Re-exported for existing importers (RM-P02 invoice workspace).
export { Gates, ReviewFlags }

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const EVIDENCE = ['PENDING_FINALISATION', 'READY', 'NOT_RECOVERABLE'] as const
type Evidence = (typeof EVIDENCE)[number]
const isEvidence = (v: string | null): v is Evidence => !!v && (EVIDENCE as readonly string[]).includes(v)

/**
 * The Finance review queue (read only; RM-P07 workspace section). Server
 * filters: invoice_id (exact native id) and evidence_state; keyset paging.
 * Each row is a target-submitted payment: its amount is the submitted claim
 * in its own unit (never summed, never a receipt of funds), its source is
 * the new system (never legacy), and it links to the verified invoice
 * detail (RM-P02) and its legacy evidence. Evidence readiness (D-43) and
 * the maker / checker gate are shown; there is no approve or reject control
 * because the decision policy is not approved (D-41, OPD-08, OPD-09).
 */
export function FinanceReviewPage() {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [params, setParams] = useSearchParams()
  const [error, setError] = useState('')
  const c = useCursors()
  const invoice = params.get('invoice_id') ?? ''
  const rawEvidence = params.get('evidence_state')
  const evidence = isEvidence(rawEvidence) ? rawEvidence : undefined
  const invoiceOk = invoice === '' || UUID.test(invoice)
  const query = { ...(invoice && invoiceOk ? { invoice_id: invoice.toLowerCase() } : {}), ...(evidence ? { evidence_state: evidence } : {}) }
  const filtered = Object.keys(query).length > 0
  const q = useRead(`frq:${JSON.stringify(query)}:${c.cursor ?? ''}:${invoiceOk}`, async (): Promise<ReadState<{ items: ReviewAttempt[]; page: PageInfo }>> => invoiceOk
    ? settle(await api.GET('/api/v1/finance/review/attempts', {
      params: { query: { ...query, limit: 50, ...(c.cursor ? { cursor: c.cursor } : {}) } } }), e => (e.data && e.page ? { items: e.data, page: e.page } : undefined))
    : { status: 'ok', data: { items: [], page: { has_more: false } }, outcome: 'SUCCESS', allowed: undefined })

  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const inv = String(form.get('invoice_id') ?? '').trim()
    const ev = String(form.get('evidence_state') ?? '')
    if (inv !== '' && !UUID.test(inv)) {
      setError('شناسه فاکتور باید شناسه بومی کامل (UUID) باشد؛ کد، کلید منبع یا شماره تلفن پذیرفته نمی‌شود.')
      return
    }
    setError('')
    const next = new URLSearchParams()
    if (inv) next.set('invoice_id', inv.toLowerCase())
    if (isEvidence(ev)) next.set('evidence_state', ev)
    c.reset()
    setParams(next)
  }
  const items = q.status === 'ok' ? q.data.items : []
  return (
    <>
      <PageHeader title="صف بررسی مالی" subtitle="پرداخت‌های ثبت‌شده در سامانه جدید؛ فقط خواندنی، تایید و رد مالی در این نسخه وجود ندارد.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <FinanceSectionNav />
      <StateBlock kind="locked" title="تصمیم مالی در دسترس نیست"
        detail="مبنای ثبت در دفتر (D-41)، سیاست تایید و تفکیک وظایف (OPD-08) و مسیر استحقاق (OPD-09) هنوز تصویب نشده‌اند." />
      <Card title="فیلترهای سرور">
        <form className="fw-filters" onSubmit={apply} aria-label="فیلتر صف بررسی مالی" key={params.toString()}>
          <label className="lbl">شناسه فاکتور (دقیق)
            <input className="input fw-ltr" name="invoice_id" defaultValue={invoice} inputMode="text" autoComplete="off" spellCheck={false}
              aria-invalid={error ? true : undefined} aria-describedby={error ? 'frq-error' : undefined} />
          </label>
          <label className="lbl">آمادگی فیش
            <select className="input" name="evidence_state" defaultValue={evidence ?? ''}>
              <option value="">همه</option>
              {EVIDENCE.map(k => <option key={k} value={k}>{evidenceStateLabel[k] ?? k}</option>)}
            </select>
          </label>
          <div className="fw-actions">
            <Button type="submit" variant="primary">اعمال فیلتر</Button>
            {filtered || params.toString() ? <Link to="/finance/review">پاک کردن فیلترها</Link> : null}
          </div>
        </form>
        {error ? <p className="fw-error" id="frq-error" role="alert">{error}</p> : null}
        {!invoiceOk ? <p className="fw-error" role="alert">شناسه فاکتور نشانی معتبر نیست؛ درخواستی به سرور ارسال نشد.</p> : null}
        {rawEvidence && !evidence ? <p className="fw-error" role="alert">مقدار آمادگی فیش نشانی معتبر نیست و نادیده گرفته شد.</p> : null}
      </Card>
      <Card title="پرداخت‌های در انتظار بررسی">
        <ReadStates state={q} what="صف بررسی" blockedDetail={blockedDetail} />
        {q.status === 'ok' && items.length > 0 && <PageSummary items={items} hasMore={q.data.page.has_more} depth={c.depth} />}
        {q.status === 'ok' && (items.length === 0
          ? (filtered
            ? <StateBlock kind="empty" title="نتیجه‌ای برای این فیلتر نیست" detail="نبودِ نتیجه برای فیلتر، صفر بودن کل صف را ادعا نمی‌کند." />
            : <StateBlock kind="empty" title="پرداختی در انتظار بررسی نیست" />)
          : <ReviewTable items={items} zone={zone} caption="پرداخت‌های در انتظار بررسی مالی" />)}
        {q.status === 'ok' && <Pager page={q.data.page} c={c} label="صفحه‌بندی صف بررسی" />}
        <p className="muted fw-hint" role="note">ثبت پرداخت یا آماده بودن فیش به معنای وصول قطعی وجه، پرداخت شدن فاکتور یا تسویه نیست. تلاش‌های قدیمی جدا هستند و در «تلاش‌های پرداخت (قدیمی)» و جزئیات هر فاکتور دیده می‌شوند.</p>
      </Card>
    </>
  )
}

/**
 * Summary of the visible page only: counts of rows by evidence state and
 * by maker flag. No total of the queue (the server returns none) and no
 * money sum (amounts stay per row, in their own unit).
 */
function PageSummary({ items, hasMore, depth }: { items: ReviewAttempt[]; hasMore: boolean; depth: number }) {
  const by = new Map<string, number>()
  for (const a of items) by.set(a.evidence_state, (by.get(a.evidence_state) ?? 0) + 1)
  const mine = items.filter(a => a.reviewer_is_submitter).length
  return (
    <div className="fw-hint" aria-label="خلاصه صفحه جاری" role="group">
      <p className="fw-pills">
        <StatusPill tone="slate" icon="inbox" label={`صفحه ${formatFaNumber(depth + 1)}: ${formatFaNumber(items.length)} مورد${hasMore ? ' (صفحه‌های بیشتری هست)' : ''}`} />
        {[...by].map(([k, n]) => <StatusPill key={k} tone={evidenceStateTone[k] ?? 'slate'} icon="inbox" label={`${evidenceStateLabel[k] ?? k}: ${formatFaNumber(n)}`} />)}
        {mine > 0 && <StatusPill tone="red" icon="ban" label={`بررسی‌کننده مستقل نیستید (ثبت‌کننده یا هویت نامعلوم): ${formatFaNumber(mine)}`} />}
      </p>
      <p className="muted">فقط صفحه جاری شمرده شده است؛ شمار کل صف در قرارداد وجود ندارد و مبالغ جمع زده نمی‌شوند.</p>
    </div>
  )
}
