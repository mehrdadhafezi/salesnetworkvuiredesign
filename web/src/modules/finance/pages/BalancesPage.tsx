import '../finance.css'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { AmountCell, MoneyCell, Pager, ReconPill, RunBar, ShadowBanner } from '../components'
import { Filters } from '../Filters'
import { NoRun, noRun } from '../noRun'
import { basisLabel, blockedDetail, categoryLabel, coverageLabel, figureLabel } from '../labels'
import { useQueryContext } from '../useQueryContext'

const options = {
  basis: ['ACCRUAL_SHADOW', 'CASH_SHADOW', 'WALLET_MIRROR_SHADOW'],
  unit: ['IRT', 'IRR'],
  account_category: ['customer_receivable', 'revenue_by_product_kind', 'cash_bank_clearing', 'commission_payable', 'legacy_wallet_unclassified', 'legacy_wallet_contra'],
  state: ['MATCH', 'MISMATCH', 'UNKNOWN', 'INCOMPLETE', 'CONFLICT', 'NOT_RECOVERABLE', 'NOT_COMPARABLE'],
} as const

/**
 * Derived SHADOW balances (read only): one row per account, basis and unit,
 * with the comparable legacy figure of the same run. Accrual and cash are
 * separate rows with equal weight (D-41 open); no total across units,
 * accounts or bases exists; no row is a final or official balance.
 */
export function BalancesPage() {
  const api = useApi()
  const ctx = useQueryContext(Object.keys(options))
  const q = { ...ctx.filters, run_id: ctx.runId, cursor: ctx.cursor, limit: 50 }
  const st = useRead(`fn-bal:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/finance/balances', { params: { query: compact(q) as never } }),
    e => e.meta && e.items && e.page ? { meta: e.meta, items: e.items, page: e.page, summary: e.summary ?? [] } : noRun(e)))
  const resolvedRun = st.status === 'ok' ? st.data?.meta.query.run_id : undefined
  return (
    <>
      <PageHeader title="مانده‌های سایه" subtitle="مانده مشتق از سطرهای دفتر، به تفکیک حساب، مبنا و واحد.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <Filters options={options} values={ctx.filters} onApply={ctx.setFilters} label="فیلتر مانده‌ها" />
      <ReadStates state={st} what="مانده‌ها" blockedDetail={blockedDetail} />
      {st.status === 'ok' && st.data === null && <NoRun />}
      {st.status === 'ok' && st.data && (
        <div className="fn-sections">
          <RunBar run={st.data.meta.run} />
          <Card title="مانده‌ها (بدون جمع کل؛ IRR و IRT جدا)">
            {st.data.items.length === 0
              ? <StateBlock kind="empty" title="هیچ مانده‌ای با این فیلتر در این اجرا نیست" />
              : (
                <div className="fn-table-wrap">
                  <table className="fn-table">
                    <caption className="sr">مانده‌های سایه</caption>
                    <thead><tr>
                      <th scope="col">حساب</th><th scope="col">مبنا</th><th scope="col">واحد</th><th scope="col">مانده سایه</th>
                      <th scope="col">رقم قدیمی</th><th scope="col">اختلاف</th><th scope="col">پوشش</th><th scope="col">وضعیت تطبیق</th>
                    </tr></thead>
                    <tbody>
                      {st.data.items.map(b => (
                        <tr key={`${b.account.account_id}|${b.basis}|${b.unit}`}>
                          <td><span className="fn-ltr">{b.account.account_key}</span><br /><span className="muted">{categoryLabel[b.account.category] ?? b.account.category}</span></td>
                          <td>{basisLabel[b.basis] ?? b.basis}</td>
                          <td className="fn-ltr">{b.unit}</td>
                          <td><AmountCell a={b.net} /></td>
                          <td>{b.legacy ? <><MoneyCell m={b.legacy.legacy_amount} absent={b.legacy.legacy_repr} /><br /><span className="muted">{figureLabel[b.legacy.legacy_figure]}</span></> : <span className="muted">قابل مقایسه نیست</span>}</td>
                          <td>{b.legacy ? <MoneyCell m={b.legacy.difference} absent="—" /> : '—'}</td>
                          <td>{b.legacy ? coverageLabel[b.legacy.coverage] ?? b.legacy.coverage : '—'}</td>
                          <td>{b.legacy ? <ReconPill state={b.legacy.state} /> : <ReconPill state="NOT_COMPARABLE" />}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            <Pager page={st.data.page} depth={ctx.cursors.length} onBack={ctx.back}
              onNext={n => ctx.pinned(resolvedRun, p => p.append('cursor', n))} label="صفحه‌بندی مانده‌ها" />
          </Card>
        </div>
      )}
    </>
  )
}
