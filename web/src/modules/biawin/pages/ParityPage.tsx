import '../biawin.css'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { Count, Filters, NoRun, Pager, RunBar, ShadowBanner, When } from '../components'
import { blockedDetail, coverageLabel, parityIcon, parityLabel, parityTone, resultLabel, resultTone, text } from '../labels'
import { useQueryContext } from '../useQueryContext'

const options = { result: ['MATCHED', 'MISSING_TARGET', 'EXTRA_TARGET', 'UNKNOWN', 'CONFLICT', 'NOT_RECOVERABLE', 'INACTIVE_BOTH'] } as const

/**
 * Active subscription parity (legacy vs target) recorded offline for one
 * run; the page only reads it. A run without a check is UNKNOWN, never MATCH.
 */
export function ParityPage() {
  const api = useApi()
  const ctx = useQueryContext(Object.keys(options))
  const q = { ...ctx.filters, run_id: ctx.runId, cursor: ctx.cursor, limit: 50 }
  const st = useRead(`bw-par:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/biawin/parity', { params: { query: compact(q) as never } }),
    e => e.meta ? { meta: e.meta, check: e.check ?? null, page: e.page ?? { has_more: false } } : e.outcome === 'UNKNOWN' ? null : undefined))
  const resolvedRun = st.status === 'ok' ? st.data?.meta.query.run_id : undefined
  const c = st.status === 'ok' ? st.data?.check : null
  return (
    <>
      <PageHeader title="تطبیق اشتراک‌های فعال" subtitle="مقایسه وضعیت فعال قدیمی با اجرای هدف؛ ثبت‌شده به‌صورت آفلاین.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <Filters options={options} values={ctx.filters} onApply={ctx.setFilters} label="فیلتر تطبیق" />
      <ReadStates state={st} what="تطبیق" blockedDetail={blockedDetail} />
      {st.status === 'ok' && st.data === null && <NoRun />}
      {st.status === 'ok' && st.data && (
        <div className="bw-sections">
          <RunBar run={st.data.meta.run} />
          {!c && <StateBlock kind="unknown" title="برای این اجرا تطبیقی ثبت نشده است" detail="نامعلوم است، نه منطبق و نه صفر." />}
          {c && (
            <>
              <Card title="نتیجه تطبیق">
                <div className="bw-meta">
                  <StatusPill tone={parityTone[c.outcome] ?? 'slate'} icon={parityIcon[c.outcome] ?? 'question'} label={parityLabel[c.outcome] ?? c.outcome} />
                  <span className="muted">شواهد: <span className="bw-ltr">{c.evidence_snapshot_id}</span> · {coverageLabel[c.evidence_status] ?? c.evidence_status}</span>
                  <span className="muted">برش شواهد: <When utc={c.evidence_cutoff} /></span>
                  {c.cutoff_mismatch && <StatusPill tone="orange" icon="alert" label="برش شواهد با برش اجرای هدف یکی نیست" />}
                  {c.reasons.map(r => <StatusPill key={r} tone="slate" icon="question" label={r} />)}
                </div>
                <div className="bw-counts">
                  <Count label="قدیمی (فعال)" n={c.legacy_count} testId="par-legacy" />
                  <Count label="هدف (فعال)" n={c.target_count} testId="par-target" />
                  <Count label="منطبق" n={c.matched} testId="par-matched" />
                  <Count label="در هدف نیست" n={c.missing_target} testId="par-missing" />
                  <Count label="اضافه در هدف" n={c.extra_target} testId="par-extra" />
                  <Count label="نامعلوم" n={c.unknown} testId="par-unknown" />
                  <Count label="تعارض" n={c.conflict} testId="par-conflict" />
                  <Count label="قابل بازیابی نیست" n={c.not_recoverable} testId="par-notrec" />
                </div>
              </Card>
              <Card title="سطرهای تطبیق">
                {c.rows.length === 0
                  ? <StateBlock kind="empty" title="سطری با این فیلتر نیست" />
                  : (
                    <div className="bw-table-wrap">
                      <table className="bw-table">
                        <caption className="sr">سطرهای تطبیق</caption>
                        <thead><tr><th scope="col">منبع</th><th scope="col">نتیجه</th><th scope="col">طبقه قدیمی</th><th scope="col">وضعیت هدف</th></tr></thead>
                        <tbody>
                          {c.rows.map(r => (
                            <tr key={r.source_key}>
                              <td className="bw-ltr">{`#${r.source_key}`}</td>
                              <td><StatusPill tone={resultTone[r.result] ?? 'slate'} icon={r.result === 'MATCHED' ? 'check' : 'alert'} label={resultLabel[r.result] ?? r.result} /></td>
                              <td className="bw-ltr">{text(r.legacy_class)}</td>
                              <td className="bw-ltr">{text(r.target_state)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                <Pager page={st.data.page} depth={ctx.cursors.length} onBack={ctx.back}
                  onNext={n => ctx.pinned(resolvedRun, p => p.append('cursor', n))} label="صفحه‌بندی تطبیق" />
              </Card>
            </>
          )}
        </div>
      )}
    </>
  )
}
