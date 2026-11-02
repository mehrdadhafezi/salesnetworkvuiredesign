import '../biawin.css'
import { Link } from 'react-router'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { Count, Filters, LinkCell, NoRun, noRun, Pager, RawTime, RunBar, ShadowBanner } from '../components'
import { activationLabel, blockedDetail, subStateIcon, subStateLabel, subStateTone, text } from '../labels'
import { useQueryContext } from '../useQueryContext'

const options = {
  state: ['ACTIVE', 'SUSPENDED', 'UNKNOWN', 'EMPTY'],
  customer_link_state: ['RESOLVED', 'RELATED', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE'],
  invoice_link_state: ['RESOLVED', 'UNKNOWN'],
  activation_evidence: ['LEGACY_ACTIVATION_RECORDED', 'NOT_EVIDENCED'],
} as const

const STATES = ['ACTIVE', 'SUSPENDED', 'UNKNOWN', 'EMPTY'] as const

/**
 * Shadow subscriptions of one import run (read only). The state is the
 * evidenced legacy status (raw value always shown); nothing is activated,
 * suspended or terminated here, and no provider is consulted.
 */
export function SubscriptionsPage() {
  const api = useApi()
  const ctx = useQueryContext(Object.keys(options))
  const q = { ...ctx.filters, run_id: ctx.runId, cursor: ctx.cursor, limit: 50 }
  const st = useRead(`bw-subs:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/biawin/subscriptions', { params: { query: compact(q) as never } }),
    e => e.meta && e.items && e.page ? { meta: e.meta, items: e.items, page: e.page, summary: e.summary ?? [] } : noRun(e)))
  const resolvedRun = st.status === 'ok' ? st.data?.meta.query.run_id : undefined
  return (
    <>
      <PageHeader title="اشتراک‌های بیاوین" subtitle="اشتراک‌های واردشده از نسخه قدیمی؛ یک اجرای ورود، یک برش منبع.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <Filters options={options} values={ctx.filters} onApply={ctx.setFilters} label="فیلتر اشتراک‌ها" />
      <ReadStates state={st} what="اشتراک‌ها" blockedDetail={blockedDetail} />
      {st.status === 'ok' && st.data === null && <NoRun />}
      {st.status === 'ok' && st.data && (
        <div className="bw-sections">
          <RunBar run={st.data.meta.run} />
          <Card title="شمار وضعیت‌ها (کل مجموعه فیلترشده، مستقل از صفحه)">
            <div className="bw-counts">
              {STATES.map(s => (
                <Count key={s} label={subStateLabel[s] ?? s} testId={`sub-count-${s}`} n={st.data?.summary.find(c => c.key === s)?.count ?? 0} />
              ))}
            </div>
          </Card>
          <Card title="اشتراک‌ها">
            {st.data.items.length === 0
              ? <StateBlock kind="empty" title="هیچ اشتراکی با این فیلتر در این اجرا نیست" />
              : (
                <div className="bw-table-wrap">
                  <table className="bw-table">
                    <caption className="sr">اشتراک‌های سایه</caption>
                    <thead><tr>
                      <th scope="col">منبع</th><th scope="col">محصول (عنوان تاریخی)</th><th scope="col">وضعیت</th><th scope="col">فعال‌سازی</th>
                      <th scope="col">مشتری</th><th scope="col">فاکتور</th><th scope="col">پرونده</th><th scope="col">آیتم‌ها</th>
                    </tr></thead>
                    <tbody>
                      {st.data.items.map(s => (
                        <tr key={s.subscription_id}>
                          <td><Link to={`/biawin/subscriptions/${s.subscription_id}?run_id=${st.data?.meta.query.run_id ?? ''}`}>
                            <span className="bw-ltr">{`#${s.source_key}`}</span></Link></td>
                          <td>{text(s.product_title_snapshot)}<br /><span className="muted">نسخه محصول: نامعلوم</span></td>
                          <td>
                            <StatusPill tone={subStateTone[s.state_mapped] ?? 'slate'} icon={subStateIcon[s.state_mapped] ?? 'question'} label={subStateLabel[s.state_mapped] ?? s.state_mapped} />
                            <br /><span className="muted bw-ltr">{`raw: ${text(s.state_raw)}`}</span>
                          </td>
                          <td>{activationLabel[s.activation_evidence] ?? s.activation_evidence}<br /><RawTime v={s.activated_at_raw} /></td>
                          <td><LinkCell l={s.customer} /></td>
                          <td><LinkCell l={s.invoice} /></td>
                          <td><LinkCell l={s.case} /></td>
                          <td className="num">{s.item_count}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            <Pager page={st.data.page} depth={ctx.cursors.length} onBack={ctx.back}
              onNext={n => ctx.pinned(resolvedRun, p => p.append('cursor', n))} label="صفحه‌بندی اشتراک‌ها" />
          </Card>
        </div>
      )}
    </>
  )
}
