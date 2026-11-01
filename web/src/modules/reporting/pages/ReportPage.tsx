import '../reporting.css'
import { Fragment, type FormEvent, type ReactNode } from 'react'
import { useParams, useSearchParams } from 'react-router'
import { useApi, type CaseFactRow, type CustodyFactRow, type InvoiceFactRow, type PaymentFactRow, type ReportMeta, type ReportSummary } from '../../../shared/api'
import { settle, useRead, type ReadState } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { CountList, MetaBar, ReadOnlyNote } from '../components'
import { blockedDetail, reportLabel, text, timeBasisLabel } from '../labels'
import { Pager } from '../pager'

type Report = 'cases' | 'custody' | 'invoices' | 'payments'
type Row = CaseFactRow | CustodyFactRow | InvoiceFactRow | PaymentFactRow
type Answer = { meta: ReportMeta; summary?: ReportSummary | undefined; items: Row[]; next?: string | null | undefined; hasMore: boolean }
type Detail = { meta: ReportMeta; member?: boolean | undefined; data?: Row | undefined }

/** The allowlisted filters of each report (server-validated as well). */
const filterOptions: Record<Report, Record<string, string[]>> = {
  cases: { presence: ['PRESENT', 'ABSENT_IN_LATEST'], workflow_type: ['LEGACY_LEAD', 'DISTRIBUTION_ITEM'], state_mapping: ['KNOWN', 'UNKNOWN'],
    customer_link_state: ['RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE'],
    original_owner_resolution: ['RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE', 'NOT_CAPTURED'] },
  custody: { presence: ['PRESENT', 'ABSENT_IN_LATEST'], parity: ['MATCH', 'TARGET_UNKNOWN', 'LEGACY_UNKNOWN', 'CONFLICT', 'MULTIPLE_CURRENT', 'HISTORY_GAP', 'NOT_RECOVERABLE'],
    derivation: ['DERIVED', 'RELEASED', 'UNKNOWN', 'MULTIPLE', 'NONE'] },
  invoices: { presence: ['PRESENT', 'ABSENT_IN_LATEST'], has_unknown_status: ['true', 'false'], main_mapping: ['KNOWN_VOCABULARY', 'UNKNOWN', 'EMPTY'],
    customer_link_state: ['RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE'], case_link_state: ['RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE'],
    total_class: ['MATCH', 'MISMATCH', 'UNKNOWN', 'INCOMPLETE', 'CONFLICT'], unit: ['IRT', 'IRR', 'UNATTRIBUTED'] },
  payments: { presence: ['PRESENT', 'ABSENT_IN_LATEST'], state: ['PENDING', 'ACCEPTED', 'REJECTED', 'CANCELLED', 'RECONCILIATION_REQUIRED', 'UNMAPPED'],
    state_mapping: ['MAPPED', 'UNKNOWN', 'EMPTY'], channel: ['GATEWAY', 'MANUAL_CARD', 'UNKNOWN'],
    approval_provenance: ['GATEWAY_AUTO', 'LEGACY_MANUAL_APPROVAL', 'NOT_APPROVED', 'UNKNOWN'], invoice_link_state: ['RESOLVED', 'UNKNOWN'], unit: ['IRT', 'IRR', 'NONE'] },
}
const contextKeys = ['time_basis', 'from', 'to', 'as_of', 'run_id']
const idOf = (r: Row): string => ('attempt_id' in r ? r.attempt_id : 'invoice_id' in r ? r.invoice_id : r.case_id)

/** The query contract taken from the URL: filters + time basis + cutoff. */
function contractOf(report: Report, params: URLSearchParams): Record<string, string> {
  const q: Record<string, string> = {}
  for (const k of [...Object.keys(filterOptions[report]), ...contextKeys]) {
    const v = params.get(k)
    if (v) q[k] = v
  }
  return q
}

async function fetchReport(api: ReturnType<typeof useApi>, report: Report, q: Record<string, string | number>): Promise<ReadState<Answer>> {
  const pick = (e: { meta?: ReportMeta; summary?: ReportSummary; items?: Row[]; page?: { has_more: boolean; next_cursor?: string | null } }) =>
    e.meta ? { meta: e.meta, summary: e.summary, items: e.items ?? [], next: e.page?.next_cursor, hasMore: e.page?.has_more ?? false } : undefined
  const query = q as never
  switch (report) {
    case 'cases': return settle(await api.GET('/api/v1/reports/cases', { params: { query } }), pick)
    case 'custody': return settle(await api.GET('/api/v1/reports/custody', { params: { query } }), pick)
    case 'invoices': return settle(await api.GET('/api/v1/reports/invoices', { params: { query } }), pick)
    case 'payments': return settle(await api.GET('/api/v1/reports/payments', { params: { query } }), pick)
  }
}

async function fetchDetail(api: ReturnType<typeof useApi>, report: Report, id: string, q: Record<string, string>): Promise<ReadState<Detail>> {
  const pick = (e: { meta?: ReportMeta; member?: boolean; data?: Row }) => (e.meta ? { meta: e.meta, member: e.member, data: e.data } : undefined)
  const query = q as never
  switch (report) {
    case 'cases': return settle(await api.GET('/api/v1/reports/cases/{case_id}', { params: { path: { case_id: id }, query } }), pick)
    case 'custody': return settle(await api.GET('/api/v1/reports/custody/{case_id}', { params: { path: { case_id: id }, query } }), pick)
    case 'invoices': return settle(await api.GET('/api/v1/reports/invoices/{invoice_id}', { params: { path: { invoice_id: id }, query } }), pick)
    case 'payments': return settle(await api.GET('/api/v1/reports/payments/{attempt_id}', { params: { path: { attempt_id: id }, query } }), pick)
  }
}

function cells(report: Report, r: Row): ReactNode[] {
  switch (report) {
    case 'cases': { const x = r as CaseFactRow
      return [`${x.source_spec} #${x.source_key}`, x.workflow_type, `${text(x.state_raw)} (${x.state_mapping})`, x.customer_link_state,
        `${x.original_owner_resolution}${x.original_owner_key ? ` #${x.original_owner_key}` : ''}`, x.credit_owner_resolution] }
    case 'custody': { const x = r as CustodyFactRow
      return [x.case_id.slice(0, 8), x.parity, x.derivation, text(x.derived_holder_key), text(x.legacy_holder_key), String(x.finding_count)] }
    case 'invoices': { const x = r as InvoiceFactRow
      return [`${x.source_spec} #${x.source_key}`, `${text(x.main.raw)} (${x.main.mapping})`, `${text(x.payment.raw)} (${x.payment.mapping})`,
        String(x.unknown_state_count), x.units.map(u => `${u.unit}: ${u.class}`).join(' · ') || '—', x.customer_link_state] }
    case 'payments': { const x = r as PaymentFactRow
      return [x.attempt_id.slice(0, 8), `${text(x.state_raw)} → ${x.state ?? 'UNMAPPED'}`, x.channel, x.approval_provenance,
        x.money ? `${x.money.amount} ${x.money.unit}` : `${text(x.amount_raw)} (${x.amount_representation})`, x.invoice_link_state] }
  }
}

const headers: Record<Report, string[]> = {
  cases: ['منبع', 'نوع', 'وضعیت (خام / نگاشت)', 'پیوند مشتری', 'مالک اولیه (تاریخی)', 'صاحب اعتبار'],
  custody: ['پرونده', 'هم‌خوانی نگهدار', 'استنتاج', 'نگهدار فعلی (مشتق)', 'نگهدار ثبت‌شده در منبع', 'یافته‌ها'],
  invoices: ['فاکتور', 'وضعیت اصلی (خام)', 'وضعیت پرداخت (خام)', 'مقادیر ناشناخته', 'شواهد جمع به تفکیک واحد', 'پیوند مشتری'],
  payments: ['تلاش', 'وضعیت خام → تلاش', 'کانال', 'منشأ تأیید', 'مبلغ (واحد صریح)', 'پیوند فاکتور'],
}

/**
 * A report page (read only). Summary (card), table and detail all use the
 * one query contract held in the URL: filters, scope, time basis and the
 * pinned run (cutoff). Changing a filter changes that contract for all
 * three; the cursor only pages the table and never changes the summary.
 */
export function ReportPage() {
  const report = useParams().report as Report
  const api = useApi()
  const [params, setParams] = useSearchParams()
  const known = report in filterOptions
  const q = known ? contractOf(report, params) : {}
  const cursors = params.getAll('cursor')
  const cursor = cursors[cursors.length - 1]
  const row = params.get('row')
  const key = JSON.stringify({ report, q, cursor })
  const list = useRead(`rep:${key}`, async (): Promise<ReadState<Answer>> => known
    ? fetchReport(api, report, { ...q, limit: 25, ...(cursor ? { cursor } : {}) })
    : { status: 'failed', failure: { kind: 'transport', message: 'unknown report' } })
  const detail = useRead(`det:${report}:${row ?? ''}:${JSON.stringify(q)}`, async (): Promise<ReadState<Detail | null>> => known && row
    ? fetchDetail(api, report, row, q) : { status: 'ok', data: null, outcome: 'SUCCESS', allowed: undefined })

  if (!known) return <StateBlock kind="error" title="گزارش ناشناخته" />
  const update = (fn: (p: URLSearchParams) => void) => {
    const next = new URLSearchParams(params)
    fn(next)
    setParams(next)
  }
  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    update(p => {
      for (const k of Object.keys(filterOptions[report])) {
        const v = form.get(k)
        if (typeof v === 'string' && v !== '') p.set(k, v)
        else p.delete(k)
      }
      p.delete('cursor')
      p.delete('row')
      p.delete('run_id') // a changed query is evaluated at the latest cutoff
    })
  }
  return (
    <>
      <PageHeader title={`گزارش ${reportLabel[report]}`} subtitle="کارت، جدول و جزئیات با یک قرارداد پرس‌وجو.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <Card title="فیلترها">
        <form className="rp-filters" onSubmit={apply} aria-label="فیلتر گزارش">
          {Object.entries(filterOptions[report]).map(([k, values]) => (
            <label key={k} className="lbl">{k}
              <select className="input" name={k} defaultValue={params.get(k) ?? ''}>
                <option value="">همه</option>
                {values.map(v => <option key={v} value={v}>{v}</option>)}
              </select>
            </label>
          ))}
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
        <p className="muted">مبنای زمانی: {timeBasisLabel[q.time_basis ?? 'CURRENT_SNAPSHOT'] ?? q.time_basis}</p>
      </Card>
      <ReadStates state={list} what="گزارش" blockedDetail={blockedDetail} />
      {list.status === 'ok' && (
        <div className="rp-sections">
          <Card title="خلاصه (کل عضویت؛ مستقل از صفحه)">
            <MetaBar meta={list.data.meta} />
            {!list.data.meta.available || !list.data.summary
              ? <StateBlock kind="unknown" title="هیچ اجرای پروجکشنی به این پرس‌وجو پاسخ نمی‌دهد" detail="نامعلوم است، نه صفر." />
              : (
                <dl className="rp-kv" aria-label="خلاصه">
                  <dt>کل</dt><dd className="num" data-testid="summary-total">{list.data.summary.total}</dd>
                  {Object.entries(list.data.summary.counts).map(([d, c]) => <Fragment key={d}><dt>{d}</dt><dd><CountList counts={c} /></dd></Fragment>)}
                  {list.data.summary.amounts?.map(a => (
                    <Fragment key={a.unit}><dt>{`مبلغ ${a.unit}`}</dt><dd className="rp-ltr">{a.money ? `${a.money.amount} ${a.unit} (${a.rows})` : 'سرریز؛ مقداری ادعا نمی‌شود'}</dd></Fragment>
                  ))}
                  {list.data.summary.unrepresentable_amounts ? <><dt>غیرقابل نمایش دقیق</dt><dd className="num">{list.data.summary.unrepresentable_amounts}</dd></> : null}
                </dl>
              )}
          </Card>
          <Card title="جدول">
            {list.data.items.length === 0
              ? <StateBlock kind="empty" title={list.data.summary?.total === 0 ? 'صفر ردیف (نتیجه واقعی این پرس‌وجو)' : 'ردیفی در این صفحه نیست'} />
              : (
                <div className="rp-table-wrap">
                  <table className="rp-table">
                    <caption className="sr">{`گزارش ${reportLabel[report]}`}</caption>
                    <thead><tr>{headers[report].map(h => <th key={h} scope="col">{h}</th>)}<th scope="col">جزئیات</th></tr></thead>
                    <tbody>
                      {list.data.items.map(r => (
                        <tr key={idOf(r)}>
                          {cells(report, r).map((c, i) => <td key={i} className="rp-ltr">{c}</td>)}
                          <td><Button onClick={() => update(p => p.set('row', idOf(r)))}>جزئیات</Button></td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            <Pager page={{ has_more: list.data.hasMore, next_cursor: list.data.next ?? null }} depth={cursors.length}
              onBack={() => update(p => { const all = p.getAll('cursor'); p.delete('cursor'); all.slice(0, -1).forEach(c => p.append('cursor', c)) })}
              onNext={n => update(p => p.append('cursor', n))} label="صفحه‌بندی جدول" />
          </Card>
          {row && (
            <Card title="جزئیات (عضویت در همین پرس‌وجو)">
              <ReadStates state={detail} what="جزئیات" blockedDetail={blockedDetail} />
              {detail.status === 'ok' && detail.data && (
                <>
                  <p>{detail.data.member
                    ? <StatusPill tone="green" icon="check" label="عضو همین پرس‌وجو است (همان شرط کارت و جدول)" />
                    : <StatusPill tone="amber" icon="alert" label="عضو این پرس‌وجو نیست" />}</p>
                  {detail.data.data && <ul className="rp-list" aria-label="مقادیر ردیف">{cells(report, detail.data.data).map((c, i) => <li key={i}><span className="muted">{headers[report][i]}: </span><span className="rp-ltr">{c}</span></li>)}</ul>}
                </>
              )}
            </Card>
          )}
        </div>
      )}
    </>
  )
}
