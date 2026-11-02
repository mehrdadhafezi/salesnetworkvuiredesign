import '../biawin.css'
import { Link } from 'react-router'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { Filters, NoRun, noRun, Pager, RunBar, ShadowBanner } from '../components'
import { blockedDetail, caseStatusLabel, caseStatusTone, kindLabel, roleLabel, text } from '../labels'
import { useQueryContext } from '../useQueryContext'

const options = {
  kind: ['SERVICE_FOLLOWUP', 'EXECUTION', 'UPGRADE_EXECUTION', 'SHIPPING'],
  status: ['NOT_STARTED', 'IN_PROGRESS', 'AWAITING_PAYMENT', 'AWAITING_PROVIDER', 'PROVIDER_ACTION_RECORDED', 'OUTCOME_RECORDED', 'SHIPPED', 'DELIVERED',
    'RETURNED', 'NOT_EXECUTED', 'CANCELLED', 'COMPLETED', 'FAILED_AT_PROVIDER', 'UNKNOWN', 'EMPTY'],
} as const

/**
 * Shadow fulfillment cases of one import run (read only). Kinds stay
 * explicit; a COMPLETED case says nothing about an invoice, a payment, the
 * subscription, a sale or finance. No assignment, shipping or completion
 * control exists.
 */
export function CasesPage() {
  const api = useApi()
  const ctx = useQueryContext(Object.keys(options))
  const q = { ...ctx.filters, run_id: ctx.runId, cursor: ctx.cursor, limit: 50 }
  const st = useRead(`fw-cases:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/fulfillment/cases', { params: { query: compact(q) as never } }),
    e => e.meta && e.items && e.page ? { meta: e.meta, items: e.items, page: e.page, summary: e.summary ?? [] } : noRun(e)))
  const resolvedRun = st.status === 'ok' ? st.data?.meta.query.run_id : undefined
  return (
    <>
      <PageHeader title="پرونده‌های اجرای خدمت" subtitle="پیگیری خدمت، اجرا، اجرای ارتقا و ارسال؛ واردشده از نسخه قدیمی.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <Filters options={options} values={ctx.filters} onApply={ctx.setFilters} label="فیلتر پرونده‌ها" />
      <ReadStates state={st} what="پرونده‌ها" blockedDetail={blockedDetail} />
      {st.status === 'ok' && st.data === null && <NoRun />}
      {st.status === 'ok' && st.data && (
        <div className="bw-sections">
          <RunBar run={st.data.meta.run} />
          <Card title="شمار به تفکیک نوع و وضعیت (کل مجموعه فیلترشده)">
            {st.data.summary.length === 0
              ? <StateBlock kind="empty" title="هیچ پرونده‌ای با این فیلتر در این اجرا نیست (۰)" />
              : (
                <ul className="bw-list" aria-label="شمار پرونده‌ها">
                  {st.data.summary.map(c => (
                    <li key={`${c.kind}/${c.status}`}>{kindLabel[c.kind] ?? c.kind} · {caseStatusLabel[c.status] ?? c.status}: <strong data-testid={`case-count-${c.kind}-${c.status}`}>{c.count}</strong></li>
                  ))}
                </ul>
              )}
          </Card>
          <Card title="پرونده‌ها">
            {st.data.items.length === 0
              ? <StateBlock kind="empty" title="هیچ پرونده‌ای با این فیلتر در این اجرا نیست" />
              : (
                <div className="bw-table-wrap">
                  <table className="bw-table">
                    <caption className="sr">پرونده‌های اجرای خدمت</caption>
                    <thead><tr>
                      <th scope="col">منبع</th><th scope="col">نوع</th><th scope="col">وضعیت</th><th scope="col">مسئولان فعلی</th><th scope="col">پیوند فاکتور</th>
                    </tr></thead>
                    <tbody>
                      {st.data.items.map(c => (
                        <tr key={c.case_id}>
                          <td><Link to={`/fulfillment/cases/${c.case_id}?run_id=${st.data?.meta.query.run_id ?? ''}`}>
                            <span className="bw-ltr">{`${c.source_spec.split('.').slice(-2, -1)[0] ?? ''} #${c.source_key}`}</span></Link></td>
                          <td>{kindLabel[c.kind] ?? c.kind}{c.subtype_raw ? <><br /><span className="muted bw-ltr">{c.subtype_raw}</span></> : null}</td>
                          <td>
                            <StatusPill tone={caseStatusTone[c.status_mapped] ?? 'slate'} icon={c.status_mapped === 'UNKNOWN' ? 'question' : 'check'}
                              label={caseStatusLabel[c.status_mapped] ?? c.status_mapped} />
                            <br /><span className="muted bw-ltr">{`raw: ${text(c.status_raw)}`}</span>
                          </td>
                          <td>{c.assignments.length === 0 ? <span className="muted">ثبت نشده</span>
                            : c.assignments.map(a => <span key={a.role}>{roleLabel[a.role] ?? a.role}: <span className="bw-ltr">{`#${a.user_key}`}</span>{a.user_id ? '' : ' (کاربر نامعلوم)'} </span>)}</td>
                          <td>{c.invoice_link_state === 'RESOLVED' ? 'اثبات‌شده' : 'نامعلوم'}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            <Pager page={st.data.page} depth={ctx.cursors.length} onBack={ctx.back}
              onNext={n => ctx.pinned(resolvedRun, p => p.append('cursor', n))} label="صفحه‌بندی پرونده‌ها" />
          </Card>
        </div>
      )}
    </>
  )
}
