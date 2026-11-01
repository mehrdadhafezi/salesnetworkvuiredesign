import '../finance.css'
import { useApi, type GroupCount, type ShadowRun } from '../../../shared/api'
import { compact, settle, useRead, type ReadState } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ReconPill, RunBar, ShadowBanner } from '../components'
import { basisLabel, blockedDetail, coverageLabel, sourceLabel } from '../labels'

const states = ['MATCH', 'MISMATCH', 'UNKNOWN', 'INCOMPLETE', 'CONFLICT', 'NOT_RECOVERABLE'] as const

/**
 * Finance SHADOW dashboard (read only): the latest shadow run, its source
 * coverage and the reconciliation counts per basis and unit. Counts only —
 * no rate, score or combined money total. The rebuild is an offline job.
 */
export function FinanceDashboardPage() {
  const api = useApi()
  const runs = useRead('fn-runs', async () => settle(await api.GET('/api/v1/finance/shadow-runs', { params: { query: compact({ limit: 10 }) } }),
    e => e.items ? { items: e.items } : undefined))
  const latest = runs.status === 'ok' ? runs.data.items[0]?.run_id : undefined
  const detail = useRead(`fn-run:${latest ?? ''}`, async (): Promise<ReadState<RunDetail | null>> => latest
    ? settle(await api.GET('/api/v1/finance/shadow-runs/{run_id}', { params: { path: { run_id: latest } } }),
      e => e.run ? { run: e.run, rec: e.reconciliation_summary ?? [], findings: e.finding_summary ?? [] } : undefined)
    : { status: 'ok', data: null, outcome: 'UNKNOWN', allowed: undefined })
  return (
    <>
      <PageHeader title="دفتر سایه مالی" subtitle="وضعیت اجرای سایه، پوشش منابع و خلاصه تطبیق با مانده‌های قدیمی.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <ReadStates state={runs} what="اجراهای سایه" blockedDetail={blockedDetail} />
      {runs.status === 'ok' && runs.data.items.length === 0 && (
        <StateBlock kind="unknown" title="هنوز هیچ اجرای سایه‌ای ثبت نشده است" detail="نامعلوم است، نه صفر. اجرا فقط با کار آفلاین crm-finance-shadow ساخته می‌شود." />
      )}
      <ReadStates state={detail} what="اجرای سایه" blockedDetail={blockedDetail} />
      {detail.status === 'ok' && detail.data && (
        <div className="fn-sections">
          <RunBar run={detail.data.run} />
          <Card title="خلاصه تطبیق (تعداد؛ بدون نرخ و بدون جمع ارزی)">
            <div className="fn-counts">
              {groups(detail.data.rec).map(g => (
                <section key={g.key} className="fn-count" aria-label={`${g.basis} ${g.unit}`}>
                  <h3>{basisLabel[g.basis] ?? g.basis} · <span className="fn-ltr">{g.unit}</span></h3>
                  <ul className="fn-list">
                    {states.map(s => <li key={s}><ReconPill state={s} /> <span className="num" data-testid={`count-${g.basis}-${g.unit}-${s}`}>{g.counts[s] ?? 0}</span></li>)}
                  </ul>
                </section>
              ))}
            </div>
          </Card>
          <Card title="منابع اجرا">
            <ul className="fn-list" aria-label="منابع اجرا">
              {detail.data.run.sources.map(s => (
                <li key={s.source}>{sourceLabel[s.source] ?? s.source}: {coverageLabel[s.status] ?? s.status}
                  {s.snapshot_id ? <span className="muted fn-ltr">{` · snapshot ${s.snapshot_id}`}</span> : null}
                  <span className="muted">{` · ${s.reason}`}</span></li>
              ))}
            </ul>
          </Card>
          <Card title="یافته‌های بدون ثبت (هیچ ثبتی جعل نشد)">
            <ul className="fn-list" aria-label="یافته‌ها">
              {detail.data.findings.map((f, i) => <li key={i}><span className="fn-ltr">{`${f.reason ?? ''} · ${f.state ?? ''}`}</span> <span className="num">{f.count}</span></li>)}
            </ul>
          </Card>
        </div>
      )}
    </>
  )
}

type RunDetail = { run: ShadowRun; rec: GroupCount[]; findings: GroupCount[] }

type Group = { key: string; basis: string; unit: string; counts: Record<string, number> }

/** Groups the reconciliation counts per basis and unit (counts of rows). */
function groups(rows: { basis?: string; unit?: string; state?: string; count: number }[]): Group[] {
  const m = new Map<string, Group>()
  for (const r of rows) {
    const key = `${r.basis ?? ''}|${r.unit ?? ''}`
    const g = m.get(key) ?? { key, basis: r.basis ?? '', unit: r.unit ?? '', counts: {} }
    if (r.state) g.counts[r.state] = (g.counts[r.state] ?? 0) + r.count
    m.set(key, g)
  }
  return [...m.values()]
}
