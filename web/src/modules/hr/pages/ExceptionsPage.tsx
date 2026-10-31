import '../hr.css'
import { useState, type FormEvent } from 'react'
import { useApi, type FindingKind } from '../../../shared/api'
import { Button, Card, PageHeader, StateBlock } from '../../../shared/ui/primitives'
import { ReadStates } from '../components/States'
import { findingLabel, severityLabel } from '../labels'
import { compact, settle, useRead } from '../load'
import { FindingList } from './PersonPage'

type Severity = 'CONFLICT' | 'UNKNOWN' | 'INFO'

/** Hierarchy findings (SN-205 §11 exceptions), read-only: reported, never auto-fixed. */
export function ExceptionsPage() {
  const api = useApi()
  const [filter, setFilter] = useState<{ kind?: FindingKind; severity?: Severity }>({})
  const [cursors, setCursors] = useState<string[]>([])
  const cursor = cursors[cursors.length - 1]
  const state = useRead(JSON.stringify({ filter, cursor }), async () => settle(await api.GET('/api/v1/organization/hierarchy-findings', {
    params: { query: compact({ ...filter, cursor, limit: 50 }) },
  }), e => e.items && e.page ? { items: e.items, page: e.page } : undefined))

  function apply(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    const kind = form.get('kind')
    const severity = form.get('severity')
    setCursors([])
    setFilter(compact({ kind: kind ? (kind as FindingKind) : undefined, severity: severity ? (severity as Severity) : undefined }))
  }

  return (
    <>
      <PageHeader title="استثناهای سلسله‌مراتب" subtitle="یافته‌ها گزارش می‌شوند؛ هیچ موردی خودکار اصلاح نمی‌شود." />
      <Card title="فیلترها">
        <form className="hr-filters" onSubmit={apply} aria-label="فیلتر یافته‌ها">
          <label className="lbl">نوع یافته
            <select className="input" name="kind" defaultValue="">
              <option value="">همه</option>
              {(Object.keys(findingLabel) as FindingKind[]).map(k => <option key={k} value={k}>{findingLabel[k]}</option>)}
            </select>
          </label>
          <label className="lbl">شدت
            <select className="input" name="severity" defaultValue="">
              <option value="">همه</option>
              {(Object.keys(severityLabel) as Severity[]).map(s => <option key={s} value={s}>{severityLabel[s]}</option>)}
            </select>
          </label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <Card title="یافته‌ها">
        <ReadStates state={state} what="یافته‌ها" />
        {state.status === 'ok' && (state.data.items.length === 0
          ? <StateBlock kind="empty" title="یافته‌ای برای این فیلتر نیست" />
          : <FindingList findings={state.data.items} />)}
        {state.status === 'ok' && (
          <nav className="hr-pager" aria-label="صفحه‌بندی">
            <Button disabled={cursors.length === 0} onClick={() => setCursors(c => c.slice(0, -1))}>صفحه قبل</Button>
            <Button disabled={!state.data.page.has_more || !state.data.page.next_cursor}
              onClick={() => { const n = state.data.page.next_cursor; if (n) setCursors(c => [...c, n]) }}>صفحه بعد</Button>
          </nav>
        )}
      </Card>
    </>
  )
}
