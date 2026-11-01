import '../reporting.css'
import { useApi } from '../../../shared/api'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { MetricCard, ReadOnlyNote } from '../components'
import { blockedDetail, coverageLabel, coverageTone, freshnessLabel, freshnessTone } from '../labels'

/**
 * MIS Today (read only). Only metrics with a FINAL contract are shown with a
 * value; deferred contracts (OPD-07, F02/TAD #20) are listed without one. No
 * target, SLA or score is invented; health is the run's real freshness and
 * coverage, never a cached green.
 */
export function MisOverviewPage() {
  const api = useApi()
  const st = useRead('mis-overview', async () => settle(await api.GET('/api/v1/mis/overview'), e => e.data))
  return (
    <>
      <PageHeader title="امروز MIS" subtitle="سلامت و پوشش داده؛ فقط شاخص‌های دارای قرارداد نهایی.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <ReadStates state={st} what="نمای کلی" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <div className="rp-sections">
          <Card title="سلامت پروجکشن">
            <div className="rp-meta">
              <StatusPill tone={freshnessTone[st.data.freshness] ?? 'slate'} icon={st.data.freshness === 'FRESH' ? 'check' : 'alert'} label={freshnessLabel[st.data.freshness] ?? st.data.freshness} />
              <StatusPill tone={coverageTone[st.data.coverage] ?? 'slate'} icon={st.data.coverage === 'AVAILABLE' ? 'check' : 'alert'} label={coverageLabel[st.data.coverage] ?? st.data.coverage} />
              {st.data.run ? <span className="muted rp-ltr">{`cutoff ${st.data.run.source_cutoff} · ${st.data.run.outcome}`}</span> : <span className="muted">هنوز هیچ اجرای پروجکشنی ثبت نشده است</span>}
            </div>
          </Card>
          <Card title="شاخص‌ها">
            {st.data.metrics.length === 0 ? <StateBlock kind="empty" title="شاخص نهایی‌ای تعریف نشده است" /> : (
              <div className="rp-cards">{st.data.metrics.map(m => <MetricCard key={m.contract.metric_id} m={m} />)}</div>
            )}
          </Card>
          <Card title="شاخص‌های معوق (NOT FINAL)">
            <ul className="rp-list" aria-label="شاخص‌های معوق">
              {st.data.deferred.map(c => <li key={c.metric_id}><StatusPill tone="violet" icon="ban" label={c.name} /> <span className="muted">{c.status_reason}</span></li>)}
            </ul>
          </Card>
        </div>
      )}
    </>
  )
}
