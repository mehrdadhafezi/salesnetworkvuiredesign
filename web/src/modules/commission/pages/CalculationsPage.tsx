import '../commission.css'
import type { FormEvent } from 'react'
import { useApi } from '../../../shared/api'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { Amount, NoRun, RunBar, ShadowBanner, blockedDetail, eligibilityLabel, eligibilityTone, engineLabel, parityLabel, parityTone, text } from '../components'
import { useQueryContext } from '../useQueryContext'

const classes = ['MATCH', 'AMOUNT_MISMATCH', 'RECIPIENT_MISMATCH', 'POLICY_MISMATCH', 'ATTRIBUTION_UNKNOWN', 'DUPLICATE_RISK', 'ENGINE_ROUTING_UNKNOWN',
  'SOURCE_INCOMPLETE', 'NOT_COMPARABLE'] as const
const engines = ['PURPOSE_MATRIX', 'RULES_AUTO', 'DRY_RUN_POSTING', 'PROJECT_CARD', 'ENGINE_ROUTING_UNKNOWN'] as const

/**
 * Stored shadow calculations with their parity and the run's findings.
 * Stored results only: nothing is (re)calculated, approved or posted here.
 */
export function CalculationsPage() {
  const api = useApi()
  const ctx = useQueryContext(['parity_class', 'engine'])
  const q = { ...ctx.filters, run_id: ctx.runId, cursor: ctx.cursor, limit: 50 }
  const st = useRead(`cm-calc:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/commission/calculations', { params: { query: compact(q) as never } }),
    e => e.run ? { run: e.run, items: e.items ?? [], page: e.page ?? { has_more: false } } : e.outcome === 'UNKNOWN' ? null : undefined))
  const fs = useRead(`cm-find:${ctx.runId ?? ''}`, async () => settle(await api.GET('/api/v1/commission/findings', { params: { query: compact({ run_id: ctx.runId, limit: 100 }) as never } }),
    e => e.items ?? (e.outcome === 'UNKNOWN' ? null : undefined)))
  const resolvedRun = st.status === 'ok' ? st.data?.run.run_id : undefined
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const out: Record<string, string> = {}
    for (const k of ['parity_class', 'engine']) {
      const v = form.get(k)
      if (typeof v === 'string' && v !== '') out[k] = v
    }
    ctx.setFilters(out)
  }
  return (
    <>
      <PageHeader title="محاسبات سایه پورسانت" subtitle="نتایج ذخیره‌شده با شواهد ورودی، انتساب و تطبیق؛ بدون اقدام.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی · SHADOW" />
      </PageHeader>
      <ShadowBanner />
      <Card title="فیلترها">
        <form className="cm-filters" onSubmit={submit} aria-label="فیلتر محاسبات">
          <label className="lbl">نتیجه تطبیق
            <select className="input" name="parity_class" defaultValue={ctx.filters.parity_class ?? ''}>
              <option value="">همه</option>
              {classes.map(c => <option key={c} value={c}>{parityLabel[c]}</option>)}
            </select>
          </label>
          <label className="lbl">موتور
            <select className="input" name="engine" defaultValue={ctx.filters.engine ?? ''}>
              <option value="">همه</option>
              {engines.map(c => <option key={c} value={c}>{engineLabel[c]}</option>)}
            </select>
          </label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <ReadStates state={st} what="محاسبات" blockedDetail={blockedDetail} />
      {st.status === 'ok' && st.data === null && <NoRun />}
      {st.status === 'ok' && st.data && (
        <div className="cm-sections">
          <RunBar run={st.data.run} />
          <Card title="محاسبات">
            {st.data.items.length === 0 ? <StateBlock kind="empty" title="محاسبه‌ای با این فیلتر نیست" /> : (
              <div className="cm-table-wrap">
                <table className="cm-table">
                  <caption className="sr">محاسبات سایه</caption>
                  <thead><tr>
                    <th scope="col">منبع</th><th scope="col">موتور</th><th scope="col">شناسه استحقاق</th><th scope="col">گیرنده (زمان رویداد)</th>
                    <th scope="col">پایه × نرخ</th><th scope="col">سایه</th><th scope="col">قدیمی</th><th scope="col">وضعیت</th><th scope="col">تطبیق</th>
                  </tr></thead>
                  <tbody>
                    {st.data.items.map(c => (
                      <tr key={c.calc_id}>
                        <td className="cm-ltr">{c.source_key}</td>
                        <td>{engineLabel[c.engine] ?? c.engine}</td>
                        <td className="cm-ltr">{c.entitlement_key ?? 'بدون کلید مصوب'}</td>
                        <td>{c.recipient_state === 'SNAPSHOT' ? <span className="cm-ltr">{`#${text(c.recipient_key)}`}</span>
                          : <StatusPill tone="slate" icon="question" label={c.recipient_state === 'CONFLICT' ? 'تعارض' : 'نامعلوم'} />}</td>
                        <td className="cm-ltr">{`${text(c.base)} × ${text(c.rate_percent)}% · ${c.policy_key ? `${c.policy_key} v${text(c.policy_version)}` : '—'}`}</td>
                        <td><Amount v={c.amount} unit={c.unit} /></td>
                        <td><Amount v={c.legacy_amount} unit={c.unit} /></td>
                        <td>
                          <StatusPill tone={eligibilityTone[c.eligibility] ?? 'slate'} icon={c.eligibility === 'BLOCKED' ? 'lock' : 'check'} label={eligibilityLabel[c.eligibility] ?? c.eligibility} />
                          {c.blocks.map(b => <span key={b} className="muted cm-ltr"> {b}</span>)}
                        </td>
                        <td><StatusPill tone={parityTone[c.parity_class] ?? 'slate'} icon={c.parity_class === 'MATCH' ? 'check' : 'alert'} label={parityLabel[c.parity_class] ?? c.parity_class} /></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            <nav className="cm-pager" aria-label="صفحه‌بندی محاسبات">
              <Button disabled={ctx.cursors.length === 0} onClick={ctx.back}>صفحه قبل</Button>
              <Button disabled={!st.data.page.has_more || !st.data.page.next_cursor}
                onClick={() => { const n = st.status === 'ok' ? st.data?.page.next_cursor : undefined; if (n) ctx.pinned(resolvedRun, p => p.append('cursor', n)) }}>صفحه بعد</Button>
            </nav>
          </Card>
          <Card title="یافته‌های تطبیق">
            {fs.status === 'ok' && fs.data && fs.data.length > 0 ? (
              <ul className="cm-list">
                {fs.data.map(f => (
                  <li key={f.finding_id}>
                    <StatusPill tone={f.severity === 'BLOCKING' ? 'red' : f.severity === 'WARNING' ? 'amber' : 'slate'} icon="alert" label={f.kind} />
                    <span className="cm-ltr"> {f.source_key} </span><span>{f.detail}</span>
                  </li>
                ))}
              </ul>
            ) : <StateBlock kind="empty" title="یافته‌ای نمایش داده نشد" />}
          </Card>
        </div>
      )}
    </>
  )
}
