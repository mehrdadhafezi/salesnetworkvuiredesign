import '../reporting.css'
import { useState } from 'react'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { MetricCard, ReadOnlyNote } from '../components'
import { blockedDetail, coverageLabel, sourceLabel } from '../labels'
import { Pager } from '../pager'

/**
 * Reconciliation (read only): every projection run with its source cutoff
 * and per-source coverage, the honest F02/F03 metrics and the versioned
 * metric contracts. A rebuild is an offline job, never triggered here.
 */
export function ReconciliationPage() {
  const api = useApi()
  const [cursors, setCursors] = useState<string[]>([])
  const cursor = cursors[cursors.length - 1]
  const st = useRead(`mis-rec:${cursor ?? ''}`, async () => settle(await api.GET('/api/v1/mis/reconciliation', { params: { query: compact({ cursor, limit: 20 }) } }),
    e => e.data && e.page ? { data: e.data, page: e.page } : undefined))
  return (
    <>
      <PageHeader title="تطبیق و پوشش" subtitle="اجراهای پروجکشن، برش منبع و پوشش هر منبع.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <ReadStates state={st} what="تطبیق" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <div className="rp-sections">
          <div className="rp-cards">
            <MetricCard m={st.data.data.f02} />
            <MetricCard m={st.data.data.f03_total_cases} />
            <MetricCard m={st.data.data.f03_legacy_leads_only} />
          </div>
          <Card title="اجراهای پروجکشن">
            <div className="rp-table-wrap">
              <table className="rp-table">
                <caption className="sr">اجراهای پروجکشن</caption>
                <thead><tr><th scope="col">اجرا</th><th scope="col">برش منبع</th><th scope="col">نتیجه</th><th scope="col">نسخه</th><th scope="col">منابع</th></tr></thead>
                <tbody>
                  {st.data.data.runs.map(r => (
                    <tr key={r.run_id}>
                      <td className="rp-ltr">{r.run_id.slice(0, 8)}</td>
                      <td className="rp-ltr">{r.source_cutoff}</td>
                      <td>{r.outcome}</td>
                      <td>{r.projection_version}</td>
                      <td>{r.sources.map(s => `${sourceLabel[s.source] ?? s.source}: ${coverageLabel[s.status] ?? s.status}`).join(' · ')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pager page={st.data.page} depth={cursors.length} onBack={() => setCursors(c => c.slice(0, -1))} onNext={n => setCursors(c => [...c, n])}
              label="صفحه‌بندی اجراها" />
          </Card>
          <Card title="قراردادهای شاخص">
            <ul className="rp-list" aria-label="قراردادهای شاخص">
              {st.data.data.contracts.map(c => <li key={c.metric_id}><span className="rp-ltr">{`${c.metric_id} v${c.version}`}</span> — {c.name} · {c.status}</li>)}
            </ul>
          </Card>
        </div>
      )}
    </>
  )
}
