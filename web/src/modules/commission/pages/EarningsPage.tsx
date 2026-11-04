import '../commission.css'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { Amount, NoRun, RunBar, ShadowBanner, blockedDetail, engineLabel, text } from '../components'
import { useQueryContext } from '../useQueryContext'

/**
 * The caller's own historical commission credits (own scope only; the
 * server filters to the authenticated user). Legacy wallet records, not a
 * payable balance; no withdrawal, settlement or dispute action exists.
 */
export function EarningsPage() {
  const api = useApi()
  const ctx = useQueryContext([])
  const q = { run_id: ctx.runId, cursor: ctx.cursor, limit: 50 }
  const st = useRead(`cm-own:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/commission/me/earnings', { params: { query: compact(q) as never } }),
    e => e.run ? { run: e.run, items: e.items ?? [], totals: e.totals ?? {}, page: e.page ?? { has_more: false } } : e.outcome === 'UNKNOWN' ? null : undefined))
  const resolvedRun = st.status === 'ok' ? st.data?.run.run_id : undefined
  return (
    <>
      <PageHeader title="پورسانت‌های من (سوابق قدیمی)" subtitle="فقط سوابق خود شما؛ موجودی قابل برداشت نیست.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <ReadStates state={st} what="سوابق پورسانت" blockedDetail={blockedDetail} />
      {st.status === 'ok' && st.data === null && <NoRun />}
      {st.status === 'ok' && st.data && (
        <div className="cm-sections">
          <RunBar run={st.data.run} />
          <Card title="جمع سوابق (اعتبار منهای برگشت)">
            <dl className="cm-dl">
              {Object.entries(st.data.totals).flatMap(([k, v]) => [<dt key={`k${k}`} className="cm-ltr">{k}</dt>, <dd key={`v${k}`} className="cm-ltr" data-testid={`total-${k}`}>{v}</dd>])}
            </dl>
          </Card>
          <Card title="سوابق">
            {st.data.items.length === 0 ? <StateBlock kind="empty" title="سابقه‌ای برای شما در این اجرا نیست" /> : (
              <div className="cm-table-wrap">
                <table className="cm-table">
                  <caption className="sr">سوابق پورسانت من</caption>
                  <thead><tr><th scope="col">تراکنش</th><th scope="col">موتور</th><th scope="col">جهت</th><th scope="col">هدف / سهم / مرحله</th>
                    <th scope="col">مبلغ</th><th scope="col">برگشت‌خورده</th><th scope="col">زمان خام</th></tr></thead>
                  <tbody>
                    {st.data.items.map(e => (
                      <tr key={e.source_key}>
                        <td className="cm-ltr">{`#${e.source_key}`}</td>
                        <td>{engineLabel[e.engine] ?? e.engine}</td>
                        <td>{e.direction === 'CREDIT' ? 'اعتبار' : e.direction === 'DEBIT' ? 'برگشت' : 'نامعلوم'}</td>
                        <td className="cm-ltr">{`${text(e.purpose_raw)} / ${text(e.share_facet_raw)} / ${text(e.stage_raw)}`}</td>
                        <td><Amount v={e.amount} unit={e.unit} /></td>
                        <td>{e.reversed ? 'بله' : 'خیر'}</td>
                        <td className="cm-ltr">{text(e.occurred_at_raw)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <nav className="cm-pager" aria-label="صفحه‌بندی سوابق">
              <Button disabled={ctx.cursors.length === 0} onClick={ctx.back}>صفحه قبل</Button>
              <Button disabled={!st.data.page.has_more || !st.data.page.next_cursor}
                onClick={() => { const n = st.status === 'ok' ? st.data?.page.next_cursor : undefined; if (n) ctx.pinned(resolvedRun, p => p.append('cursor', n)) }}>صفحه بعد</Button>
            </nav>
          </Card>
        </div>
      )}
    </>
  )
}
