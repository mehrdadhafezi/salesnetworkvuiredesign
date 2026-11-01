import '../finance.css'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { MoneyCell, Pager, ReconPill, RunBar, ShadowBanner } from '../components'
import { Filters } from '../Filters'
import { basisLabel, blockedDetail, coverageLabel, figureLabel, text } from '../labels'
import { useQueryContext } from '../useQueryContext'

const options = {
  basis: ['ACCRUAL_SHADOW', 'CASH_SHADOW', 'WALLET_MIRROR_SHADOW'],
  unit: ['IRT', 'IRR', 'UNKNOWN'],
  state: ['MATCH', 'MISMATCH', 'UNKNOWN', 'INCOMPLETE', 'CONFLICT', 'NOT_RECOVERABLE'],
  subject_type: ['legacy_wallet', 'invoice'],
} as const

/**
 * Legacy vs SHADOW reconciliation (read only): each row compares one legacy
 * figure with one shadow amount for one subject, basis and unit of the same
 * run. The difference exists only when both amounts are complete; every
 * other state is explicit with its reasons — never a zero.
 */
export function FinanceReconciliationPage() {
  const api = useApi()
  const ctx = useQueryContext(Object.keys(options))
  const q = { ...ctx.filters, run_id: ctx.runId, cursor: ctx.cursor, limit: 50 }
  const st = useRead(`fn-rec:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/finance/reconciliation', { params: { query: compact(q) as never } }),
    e => e.meta && e.items && e.page ? { meta: e.meta, items: e.items, page: e.page, summary: e.summary ?? [] } : undefined))
  return (
    <>
      <PageHeader title="تطبیق با مانده‌های قدیمی" subtitle="رقم قدیمی، مبلغ سایه و اختلاف؛ به تفکیک مبنا و واحد.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <Filters options={options} values={ctx.filters} onApply={ctx.setFilters} label="فیلتر تطبیق" />
      <ReadStates state={st} what="تطبیق" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <div className="fn-sections">
          <RunBar run={st.data.meta.run} />
          <Card title="خلاصه (کل عضویت؛ مستقل از صفحه)">
            <ul className="fn-list" aria-label="خلاصه تطبیق">
              {st.data.summary.map((c, i) => (
                <li key={i}><span className="fn-ltr">{`${c.basis ?? ''} · ${c.unit ?? ''}`}</span> <ReconPill state={c.state ?? 'UNKNOWN'} /> <span className="num">{c.count}</span></li>
              ))}
            </ul>
          </Card>
          <Card title="ردیف‌های تطبیق">
            {st.data.items.length === 0
              ? <StateBlock kind="empty" title="هیچ ردیفی با این فیلتر در این اجرا نیست" />
              : (
                <div className="fn-table-wrap">
                  <table className="fn-table">
                    <caption className="sr">تطبیق</caption>
                    <thead><tr>
                      <th scope="col">موضوع</th><th scope="col">مبنا</th><th scope="col">واحد</th><th scope="col">رقم قدیمی</th><th scope="col">مبلغ سایه</th>
                      <th scope="col">اختلاف</th><th scope="col">وضعیت</th><th scope="col">دلایل</th><th scope="col">پوشش</th>
                    </tr></thead>
                    <tbody>
                      {st.data.items.map(r => (
                        <tr key={`${r.subject_domain}|${r.subject_ref}|${r.basis}|${r.legacy_figure}|${r.unit}`}>
                          <td><span className="fn-ltr">{`${r.subject_domain}:${r.subject_ref}`}</span><br /><span className="muted">{figureLabel[r.legacy_figure]}</span></td>
                          <td>{basisLabel[r.basis] ?? r.basis}</td>
                          <td className="fn-ltr">{r.unit}</td>
                          <td><MoneyCell m={r.legacy_amount} absent={r.legacy_repr} /></td>
                          <td><MoneyCell m={r.shadow_amount} /></td>
                          <td><MoneyCell m={r.difference} absent="—" /></td>
                          <td><ReconPill state={r.state} /></td>
                          <td className="fn-ltr">{text(r.reasons.join(', '))}</td>
                          <td>{coverageLabel[r.coverage] ?? r.coverage}{r.source_snapshot ? <span className="muted fn-ltr">{` · ${r.source_snapshot}`}</span> : null}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            <Pager page={st.data.page} depth={ctx.cursors.length} onBack={ctx.back}
              onNext={n => ctx.pinned(st.data.meta.query.run_id, p => p.append('cursor', n))} label="صفحه‌بندی تطبیق" />
          </Card>
        </div>
      )}
    </>
  )
}
