import '../commission.css'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ShadowBanner, blockedDetail, text } from '../components'
import { useQueryContext } from '../useQueryContext'

const evidenceLabel: Record<string, string> = {
  OBSERVED_UNVERSIONED: 'مشاهده‌شده در برداشت (بدون تاریخچه نسخه)', EVENT_SNAPSHOT: 'تصویر زمان رویداد', INVALID: 'نامعتبر (حفظ شد)',
}

/** Observed policy versions. None is approved; nothing is editable here. */
export function PoliciesPage() {
  const api = useApi()
  const ctx = useQueryContext([])
  const q = { cursor: ctx.cursor, limit: 100 }
  const st = useRead(`cm-pol:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/commission/policies', { params: { query: compact(q) as never } }),
    e => e.items ? { items: e.items, page: e.page ?? { has_more: false } } : e.outcome === 'UNKNOWN' ? null : undefined))
  return (
    <>
      <PageHeader title="نسخه‌های سیاست پورسانت" subtitle="مشاهده‌شده از شواهد قدیمی؛ هیچ نسخه‌ای تأیید نشده است.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <ReadStates state={st} what="سیاست‌ها" blockedDetail={blockedDetail} />
      {st.status === 'ok' && st.data === null && <StateBlock kind="unknown" title="هنوز نسخه سیاستی مشاهده نشده است" detail="نامعلوم است، نه صفر." />}
      {st.status === 'ok' && st.data && (
        <Card title="نسخه‌ها">
          <div className="cm-table-wrap">
            <table className="cm-table">
              <caption className="sr">نسخه‌های سیاست</caption>
              <thead><tr><th scope="col">کلید</th><th scope="col">نسخه</th><th scope="col">تابع</th><th scope="col">نرخ / ثابت</th><th scope="col">بازه اعتبار</th>
                <th scope="col">شاهد</th><th scope="col">تأیید</th></tr></thead>
              <tbody>
                {st.data.items.map(p => (
                  <tr key={`${p.policy_key}#${p.version}`}>
                    <td className="cm-ltr">{p.policy_key}</td>
                    <td>{p.version}</td>
                    <td className="cm-ltr">{p.function_id}</td>
                    <td className="cm-ltr">{`${text(p.rate_percent)}% · ${text(p.fixed)} ${p.unit}`}</td>
                    <td className="cm-ltr">{p.effective_from || p.effective_to ? `${text(p.effective_from)} → ${text(p.effective_to)}` : 'نامعلوم (همیشگی فرض نمی‌شود)'}</td>
                    <td>{evidenceLabel[p.evidence] ?? p.evidence}</td>
                    <td><StatusPill tone="slate" icon="lock" label="تأیید نشده" /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <nav className="cm-pager" aria-label="صفحه‌بندی سیاست‌ها">
            <Button disabled={ctx.cursors.length === 0} onClick={ctx.back}>صفحه قبل</Button>
            <Button disabled={!st.data.page.has_more || !st.data.page.next_cursor}
              onClick={() => { const n = st.status === 'ok' ? st.data?.page.next_cursor : undefined; if (n) ctx.update(p => p.append('cursor', n)) }}>صفحه بعد</Button>
          </nav>
        </Card>
      )}
    </>
  )
}
