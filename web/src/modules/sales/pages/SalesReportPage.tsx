import '../sales.css'
import { Fragment } from 'react'
import { useApi, type HonestMetric } from '../../../shared/api'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { ReadOnlyNote } from '../components'
import { blockedDetail, parityLabel } from '../labels'

const statusLabel: Record<HonestMetric['status'], string> = { EVALUATED: 'ارزیابی‌شده', PARTIAL: 'ناقص', NOT_EVALUATED: 'ارزیابی نشده' }
const statusTone = { EVALUATED: 'green', PARTIAL: 'amber', NOT_EVALUATED: 'slate' } as const

/** A metric with its exact label and definition; a null count is shown as not evaluated, never as zero. */
function Metric({ m }: { m: HonestMetric }) {
  return (
    <section className="sa-owner" aria-label={m.label}>
      <h3>{`${m.code} · ${m.label}`}</h3>
      <StatusPill tone={statusTone[m.status]} icon={m.status === 'EVALUATED' ? 'check' : 'question'} label={statusLabel[m.status]} />
      <p className="num">{m.count === null ? 'بدون عدد (صفر نیست)' : m.count}</p>
      <p className="muted">{m.definition}</p>
      <p className="muted">پوشش: {m.coverage}</p>
      <p className="muted">{m.reason}</p>
      <p className="muted sa-ltr">{m.authority}</p>
    </section>
  )
}

function Counts({ m, label }: { m: Record<string, number>; label?: Record<string, string> }) {
  const entries = Object.entries(m)
  if (entries.length === 0) return <p className="muted">—</p>
  return <dl className="sa-kv">{entries.map(([k, v]) => <Fragment key={k}><dt>{label?.[k] ?? k}</dt><dd className="num">{v}</dd></Fragment>)}</dl>
}

/** Source coverage, honest F02/F03 labels and custody parity (read only). */
export function SalesReportPage() {
  const api = useApi()
  const cov = useRead('cov', async () => settle(await api.GET('/api/v1/sales/coverage-report'), e => e.data))
  const par = useRead('par', async () => settle(await api.GET('/api/v1/sales/custody-parity'), e => e.data))
  return (
    <>
      <PageHeader title="گزارش پوشش و تطابق نگهداری" subtitle="شمارش‌ها با برچسب صادقانه؛ عدد نامعلوم صفر نیست.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <div className="sa-sections">
        <Card title="F02 / F03">
          <ReadStates state={cov} what="گزارش پوشش" blockedDetail={blockedDetail} />
          {cov.status === 'ok' && (
            <div className="sa-owners">
              <Metric m={cov.data.legacy_leads_only} />
              <Metric m={cov.data.total_cases} />
              <Metric m={cov.data.f02} />
            </div>
          )}
        </Card>
        {cov.status === 'ok' && (
          <Card title="پوشش منابع">
            <Counts m={cov.data.cases_by_source_spec} />
            <dl className="sa-kv">{Object.entries(cov.data.source_coverage).map(([k, v]) => <Fragment key={k}><dt className="sa-ltr">{k}</dt><dd>{v}</dd></Fragment>)}</dl>
          </Card>
        )}
        <Card title="تطابق نگهدار فعلی">
          <ReadStates state={par} what="گزارش تطابق" blockedDetail={blockedDetail} />
          {par.status === 'ok' && (
            <>
              <p>{par.data.full_parity ? <StatusPill tone="green" icon="check" label="تطابق کامل" /> : <StatusPill tone="amber" icon="alert" label="تطابق کامل ادعا نمی‌شود" />}</p>
              <Counts m={par.data.current_holder_parity} label={parityLabel} />
              <ul className="sa-list">{par.data.notes.map(n => <li key={n} className="muted">{n}</li>)}</ul>
            </>
          )}
        </Card>
      </div>
    </>
  )
}
