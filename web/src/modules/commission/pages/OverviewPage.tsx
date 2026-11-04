import '../commission.css'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { Count, NoRun, RunBar, ShadowBanner, blockedDetail, decisionLabel, decisionTone, engineLabel, parityLabel, parityTone, runStateLabel, text } from '../components'
import { useQueryContext } from '../useQueryContext'

/**
 * Commission overview: the stored shadow run, its honest parity report,
 * findings, legacy dry runs and the blocked decisions. Reading it never
 * calculates, approves, posts or credits anything; there is no action.
 */
export function OverviewPage() {
  const api = useApi()
  const ctx = useQueryContext([])
  const st = useRead(`cm-ov:${ctx.runId ?? ''}`, async () => settle(await api.GET('/api/v1/commission/overview', { params: { query: compact({ run_id: ctx.runId }) as never } }),
    e => e.overview ? e.overview : e.outcome === 'UNKNOWN' ? null : undefined))
  const hist = useRead(`cm-hr:${ctx.runId ?? ''}`, async () => settle(await api.GET('/api/v1/commission/historical-runs', { params: { query: compact({ run_id: ctx.runId }) as never } }),
    e => e.items ?? (e.outcome === 'UNKNOWN' ? null : undefined)))
  const ov = st.status === 'ok' ? st.data : null
  return (
    <>
      <PageHeader title="پورسانت · نمای سایه" subtitle="تطبیق محاسبه سایه با موتورهای قدیمی؛ بدون هیچ اثر مالی.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <ReadStates state={st} what="نمای پورسانت" blockedDetail={blockedDetail} />
      {ov && (
        <div className="cm-sections">
          <Card title="وضعیت ثبت و تأیید">
            <div className="cm-meta">
              <StatusPill tone="red" icon="lock" label="ثبت پورسانت: مسدود — مجاز نیست" />
              <StatusPill tone="red" icon="lock" label="تأیید اجرا: مسدود (OPD-09)" />
            </div>
            <table className="cm-table">
              <caption className="sr">تصمیم‌های باز</caption>
              <thead><tr><th scope="col">تصمیم</th><th scope="col">وضعیت</th><th scope="col">اثر</th></tr></thead>
              <tbody>
                {ov.decisions.map(d => (
                  <tr key={d.id}>
                    <td className="cm-ltr">{d.id}</td>
                    <td><StatusPill tone={decisionTone[d.state] ?? 'slate'} icon="alert" label={decisionLabel[d.state] ?? d.state} /></td>
                    <td>{d.effect}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </Card>
          {!ov.run && <NoRun />}
          {ov.run && <RunBar run={ov.run} />}
          {ov.run && !ov.parity && <StateBlock kind="unknown" title="گزارش تطبیقی برای این اجرا ثبت نشده است" detail="نامعلوم است، نه منطبق و نه صفر." />}
          {ov.parity && (
            <Card title="گزارش تطبیق">
              <div className="cm-meta">
                <span className="muted">{ov.parity.cohort}</span>
                <span className="muted cm-ltr">{`${text(ov.parity.window_from_raw)} → ${text(ov.parity.window_to_raw)}`}</span>
                <StatusPill tone={ov.parity.coverage === 'FULL' ? 'green' : 'amber'} icon="alert"
                  label={`پوشش مقایسه: ${ov.parity.compared_rows} از ${ov.parity.legacy_rows} (${ov.parity.coverage})`} />
              </div>
              <div className="cm-counts">
                {Object.entries(ov.parity.classes).map(([k, n]) => (
                  <div key={k} className="cm-count">
                    <StatusPill tone={parityTone[k] ?? 'slate'} icon={k === 'MATCH' ? 'check' : 'alert'} label={parityLabel[k] ?? k} />
                    <strong data-testid={`cls-${k}`}>{n}</strong>
                  </div>
                ))}
              </div>
              <h3>جمع به تفکیک واحد</h3>
              <dl className="cm-dl">
                {Object.entries(ov.parity.legacy_totals).flatMap(([u, v]) => [<dt key={`l${u}`}>{`قدیمی (${u})`}</dt>, <dd key={`lv${u}`} className="cm-ltr">{v}</dd>])}
                {Object.entries(ov.parity.target_totals).flatMap(([u, v]) => [<dt key={`t${u}`}>{`سایه (${u})`}</dt>, <dd key={`tv${u}`} className="cm-ltr">{v}</dd>])}
                <dt>بدون مبلغ دقیق (قدیمی / سایه)</dt><dd data-testid="missing">{`${ov.parity.legacy_amount_missing} / ${ov.parity.target_amount_missing}`}</dd>
              </dl>
              <h3>به تفکیک موتور</h3>
              <div className="cm-counts">
                {Object.entries(ov.parity.engines).map(([k, n]) => <Count key={k} label={engineLabel[k] ?? k} n={n} testId={`eng-${k}`} />)}
              </div>
            </Card>
          )}
          {ov.run && (
            <Card title="یافته‌های تطبیق">
              {Object.keys(ov.findings).length === 0
                ? <StateBlock kind="empty" title="یافته‌ای ثبت نشده است" />
                : <div className="cm-counts">{Object.entries(ov.findings).map(([k, n]) => <Count key={k} label={k} n={n} testId={`f-${k}`} />)}</div>}
            </Card>
          )}
          {ov.run && hist.status === 'ok' && hist.data && (
            <Card title="اجراهای پیش‌نمایش قدیمی (منشأ تأیید و ثبت)">
              {hist.data.length === 0 ? <StateBlock kind="empty" title="اجرای پیش‌نمایش قدیمی در این برداشت نیست" /> : (
                <div className="cm-table-wrap">
                  <table className="cm-table">
                    <caption className="sr">اجراهای پیش‌نمایش قدیمی</caption>
                    <thead><tr><th scope="col">کد</th><th scope="col">بازه</th><th scope="col">وضعیت</th><th scope="col">رد سپس تأیید</th><th scope="col">آیتم / ثبت‌شده / خطا</th></tr></thead>
                    <tbody>
                      {hist.data.map(h => (
                        <tr key={h.source_key}>
                          <td className="cm-ltr">{text(h.run_code)}</td>
                          <td className="cm-ltr">{`${text(h.window_from_raw)} → ${text(h.window_to_raw)}`}</td>
                          <td>{`${runStateLabel[h.state] ?? h.state} (خام: ${text(h.approval_raw)})`}</td>
                          <td>{h.rejected_then_approved ? 'بله (هر دو حفظ شد)' : 'خیر'}</td>
                          <td>{`${h.item_count} / ${h.posted_items} / ${h.failed_items}`}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </Card>
          )}
        </div>
      )}
    </>
  )
}
