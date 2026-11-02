import '../writes.css'
import { useState } from 'react'
import { useApi } from '../../../shared/api'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { parityLabel, writebackLabel, writebackTone } from '../labels'

const STATES = ['PENDING', 'WAITING_DEPENDENCY', 'APPLIED', 'UNKNOWN', 'REJECTED', 'RECONCILIATION_REQUIRED', 'COMPARED'] as const

/**
 * Legacy write-back diagnostics (read only): pending, unknown, reconciliation
 * backlog and parity. No payload, no PII, no fix / retry action —
 * resolution is owned by the domain.
 */
export function WritebackPage() {
  const api = useApi()
  const [state, setState] = useState<string>('')
  const summary = useRead('wb-sum', async () => settle(await api.GET('/api/v1/writeback/summary'), e => e.data))
  const list = useRead(`wb-list:${state}`, async () => settle(await api.GET('/api/v1/writeback/commands', {
    params: { query: { limit: 100, ...(state ? { state: state as (typeof STATES)[number] } : {}) } } }), e => e.items))
  const blocked = 'تشخیص بازنویسی پشت پرچم integration.writeback_read است.'
  return (
    <>
      <PageHeader title="بازنویسی به سامانه قدیمی" subtitle="وضعیت فنی؛ بدون اقدام اصلاحی.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadStates state={summary} what="خلاصه" blockedDetail={blocked} />
      {summary.status === 'ok' && (
        <Card title="خلاصه">
          <ul className="wr-list" aria-label="شمار وضعیت‌ها">
            {summary.data.states.map(c => <li key={c.key}>{writebackLabel[c.key] ?? c.key}: <strong data-testid={`wb-${c.key}`}>{c.count}</strong></li>)}
            {summary.data.parity.map(c => <li key={`p-${c.key}`}>{parityLabel[c.key] ?? c.key}: <strong>{c.count}</strong></li>)}
            {summary.data.reconciliations.map(c => <li key={`r-${c.key}`}>{c.key}: <strong>{c.count}</strong></li>)}
          </ul>
        </Card>
      )}
      <Card title="فرمان‌ها">
        <label className="lbl">وضعیت
          <select className="input" value={state} onChange={e => setState(e.target.value)} aria-label="فیلتر وضعیت">
            <option value="">همه</option>
            {STATES.map(s => <option key={s} value={s}>{writebackLabel[s]}</option>)}
          </select>
        </label>
        <ReadStates state={list} what="فرمان‌ها" blockedDetail={blocked} />
        {list.status === 'ok' && (list.data.length === 0 ? <StateBlock kind="empty" title="فرمانی نیست" /> : (
          <div className="wr-table-wrap">
            <table className="wr-table">
              <caption className="sr">فرمان‌های بازنویسی</caption>
              <thead><tr><th scope="col">نوع</th><th scope="col">وضعیت</th><th scope="col">تطابق</th><th scope="col">تلاش‌ها</th></tr></thead>
              <tbody>
                {list.data.map(c => (
                  <tr key={c.command_id}>
                    <td className="wr-ltr">{c.command_type}</td>
                    <td><StatusPill tone={writebackTone[c.state] ?? 'slate'} icon="alert" label={writebackLabel[c.state] ?? c.state} /></td>
                    <td>{c.parity ? parityLabel[c.parity] : '—'}</td>
                    <td className="wr-ltr">{c.attempts}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}
      </Card>
    </>
  )
}
