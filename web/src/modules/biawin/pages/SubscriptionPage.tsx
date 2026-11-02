import '../biawin.css'
import { Link, useParams } from 'react-router'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { LinkCell, Pager, RawTime, RunBar, ShadowBanner } from '../components'
import { activationLabel, blockedDetail, eventClassLabel, linkLabel, subStateIcon, subStateLabel, subStateTone, text } from '../labels'
import { useQueryContext } from '../useQueryContext'

/**
 * One shadow subscription of one run with its append-only history and its
 * items (each distinct; identical titles are never merged). Detail and
 * items always come from the same pinned run.
 */
export function SubscriptionPage() {
  const api = useApi()
  const { subscriptionId = '' } = useParams()
  const ctx = useQueryContext([])
  const run = ctx.runId
  const st = useRead(`bw-sub:${subscriptionId}:${run ?? ''}`, async () => settle(await api.GET('/api/v1/biawin/subscriptions/{subscription_id}', {
    params: { path: { subscription_id: subscriptionId }, query: compact({ run_id: run }) as never },
  }), e => e.meta && e.subscription ? { meta: e.meta, sub: e.subscription, history: e.history ?? [] } : undefined))
  const pinnedRun = st.status === 'ok' ? st.data.meta.query.run_id : run
  const iq = { run_id: pinnedRun, cursor: ctx.cursor, limit: 50 }
  // Waits for the detail only when no run is pinned yet (the key then
  // changes once the detail names its run); a pinned run loads at once.
  const items = useRead(`bw-items:${subscriptionId}:${JSON.stringify(iq)}`, async () => pinnedRun === undefined
    ? { status: 'loading' as const }
    : settle(await api.GET('/api/v1/biawin/subscriptions/{subscription_id}/items', {
      params: { path: { subscription_id: subscriptionId }, query: compact(iq) as never },
    }), e => e.items && e.page ? { items: e.items, page: e.page, count: e.item_count ?? 0 } : undefined))
  return (
    <>
      <PageHeader title="اشتراک بیاوین" subtitle="جزئیات، آیتم‌ها و تاریخچه از یک اجرای ورود.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <p><Link to={`/biawin/subscriptions${pinnedRun ? `?run_id=${pinnedRun}` : ''}`}>بازگشت به فهرست</Link></p>
      <ReadStates state={st} what="اشتراک" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <div className="bw-sections">
          <RunBar run={st.data.meta.run} />
          <Card title={`اشتراک #${st.data.sub.source_key}`}>
            <dl className="bw-dl">
              <dt>وضعیت</dt>
              <dd>
                <StatusPill tone={subStateTone[st.data.sub.state_mapped] ?? 'slate'} icon={subStateIcon[st.data.sub.state_mapped] ?? 'question'}
                  label={subStateLabel[st.data.sub.state_mapped] ?? st.data.sub.state_mapped} />
                <span className="muted bw-ltr">{` raw: ${text(st.data.sub.state_raw)} · ${st.data.sub.state_namespace}`}</span>
              </dd>
              <dt>فعال‌سازی</dt><dd>{activationLabel[st.data.sub.activation_evidence]} · <RawTime v={st.data.sub.activated_at_raw} /></dd>
              <dt>محصول</dt><dd>{text(st.data.sub.product_title_snapshot)} <span className="muted bw-ltr">{`#${text(st.data.sub.product_key)}`}</span> · نسخه محصول: نامعلوم</dd>
              <dt>تأمین‌کننده</dt><dd className="bw-ltr">{st.data.sub.provider}</dd>
              <dt>مشتری</dt><dd><LinkCell l={st.data.sub.customer} /></dd>
              <dt>فاکتور منبع</dt><dd><LinkCell l={st.data.sub.invoice} /></dd>
              <dt>فاکتور اولین پرداخت</dt><dd><LinkCell l={st.data.sub.first_payment_invoice} /></dd>
              <dt>پرونده</dt><dd><LinkCell l={st.data.sub.case} /></dd>
              <dt>ایجاد در منبع</dt><dd><RawTime v={st.data.sub.created_at_raw} /></dd>
            </dl>
          </Card>
          <Card title={`آیتم‌ها (${items.status === 'ok' ? items.data.count : '…'})`}>
            <ReadStates state={items} what="آیتم‌ها" blockedDetail={blockedDetail} />
            {items.status === 'ok' && (items.data.items.length === 0
              ? <StateBlock kind="empty" title="این اشتراک در این اجرا آیتمی ندارد" />
              : (
                <div className="bw-table-wrap">
                  <table className="bw-table">
                    <caption className="sr">آیتم‌های اشتراک</caption>
                    <thead><tr><th scope="col">آیتم</th><th scope="col">خدمت (عنوان تاریخی)</th><th scope="col">چرخه عمر</th><th scope="col">پیوند فاکتور</th></tr></thead>
                    <tbody>
                      {items.data.items.map(it => (
                        <tr key={it.item_id}>
                          <td className="bw-ltr">{`#${it.source_key}`}</td>
                          <td>{text(it.service_title_snapshot)}</td>
                          <td><span className="muted">ثبت نشده (پیگیری خدمت در پرونده اجرا)</span></td>
                          <td>{it.invoice_links.length === 0 ? <span className="muted">ندارد</span>
                            : it.invoice_links.map(l => <span key={l.source_key}>{linkLabel[l.state] ?? l.state} <span className="muted bw-ltr">{text(l.status_raw)}</span> </span>)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              ))}
            {items.status === 'ok' && <Pager page={items.data.page} depth={ctx.cursors.length} onBack={ctx.back}
              onNext={n => ctx.pinned(pinnedRun, p => p.append('cursor', n))} label="صفحه‌بندی آیتم‌ها" />}
          </Card>
          <Card title="تاریخچه (فقط افزودنی)">
            {st.data.history.length === 0
              ? <StateBlock kind="empty" title="رویدادی برای این اشتراک در این اجرا ثبت نشده است" />
              : (
                <ol className="bw-list" aria-label="تاریخچه اشتراک">
                  {st.data.history.map(e => (
                    <li key={e.source_key}>
                      <span className="bw-ltr">{e.event_key_raw}</span> · {eventClassLabel[e.event_class] ?? e.event_class} · <RawTime v={e.occurred_at_raw} />
                    </li>
                  ))}
                </ol>
              )}
          </Card>
        </div>
      )}
    </>
  )
}
