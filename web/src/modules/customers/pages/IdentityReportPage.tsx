import '../customers.css'
import { Fragment } from 'react'
import { useApi, type ResolutionReport } from '../../../shared/api'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { NoMergeNote } from '../components'
import { blockedDetail, reasonLabel, stateLabel } from '../labels'

/**
 * Identity resolution report. The phone invariants are shown first; a
 * violated invariant arrives from the server as BLOCKED / LINKAGE_CONFLICT
 * and is rendered as such, never as zero.
 */
export function IdentityReportPage() {
  const api = useApi()
  const report = useRead('report', async () => settle(await api.GET('/api/v1/customer-identity/resolution-report'), e => e.data))
  return (
    <>
      <PageHeader title="گزارش تطبیق هویت" subtitle="شمارش‌ها از مدل سایه؛ ادغامی وجود ندارد.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <NoMergeNote />
      <ReadStates state={report} what="گزارش" blockedDetail={blockedDetail} />
      {report.status === 'ok' && <Report r={report.data} />}
    </>
  )
}

function Invariant({ label, value }: { label: string; value: number }) {
  return (
    <li className="cu-invariant">
      <StatusPill tone={value === 0 ? 'green' : 'red'} icon={value === 0 ? 'check' : 'alert'} label={value === 0 ? 'برقرار' : 'نقض شده'} />
      <span>{label}: <b className="num">{value}</b></span>
    </li>
  )
}

function Counts({ title, m, label }: { title: string; m: Record<string, number>; label: Record<string, string> }) {
  const entries = Object.entries(m)
  return (
    <Card title={title}>
      {entries.length === 0 ? <p className="muted">—</p>
        : <dl className="cu-kv">{entries.map(([k, v]) => <Fragment key={k}><dt>{label[k] ?? k}</dt><dd className="num">{v}</dd></Fragment>)}</dl>}
    </Card>
  )
}

function Report({ r }: { r: ResolutionReport }) {
  return (
    <div className="cu-sections">
      <Card title="قیود (باید صفر باشند)">
        <ul className="cu-list" aria-label="قیود">
          <Invariant label="تطبیق قطعی فقط با تلفن" value={r.phone_only_resolved_count} />
          <Invariant label="ادغام فقط با تلفن" value={r.phone_only_merge_count} />
          <Invariant label="ادغام" value={r.merge_count} />
        </ul>
      </Card>
      <Card title="خلاصه">
        <dl className="cu-kv">
          <dt>مشتریان (حاضر / کل)</dt><dd className="num">{r.customers_present} / {r.customers}</dd>
          <dt>رکوردهای منبع حاضر</dt><dd className="num">{r.source_subjects_present}</dd>
          <dt>نامزدهای فقط‌تلفن</dt><dd className="num">{r.phone_only_candidate_count}</dd>
          <dt>تعارض مالکیت هم‌زمان تلفن</dt><dd className="num">{r.overlapping_phone_conflict_count}</dd>
          <dt>تعارض شناسه خارجی تکراری</dt><dd className="num">{r.duplicate_external_id_conflict_count}</dd>
          <dt>شواهد قوی</dt><dd className="num">{r.strong_evidence_count}</dd>
          <dt>مشتریان فقط با شواهد ضعیف</dt><dd className="num">{r.weak_only_customer_count}</dd>
          <dt>حل‌نشده</dt><dd className="num">{r.unresolved_count}</dd>
        </dl>
      </Card>
      <Counts title="مشتریان بر اساس وضعیت" m={r.customers_by_state} label={stateLabel} />
      <Counts title="پرونده‌ها بر اساس وضعیت" m={r.merge_cases_by_state} label={stateLabel} />
      <Counts title="پرونده‌ها بر اساس دلیل" m={r.merge_cases_by_reason} label={reasonLabel} />
    </div>
  )
}
