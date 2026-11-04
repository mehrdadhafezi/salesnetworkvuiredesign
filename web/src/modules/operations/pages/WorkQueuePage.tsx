import '../operations.css'
import type { FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { compact, settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ShadowNotice } from '../CommandResult'
import { blockedDetail, short, stateLabel, stateTone } from '../labels'

const states = ['QUEUED', 'ASSIGNED', 'IN_PROGRESS', 'COMPLETED', 'NOT_EXECUTED'] as const

/**
 * The caller's work queue: cases of services the caller currently
 * supervises or executes (catalog scope, never HR hierarchy). Opening the
 * queue never creates, assigns or completes anything.
 */
export function WorkQueuePage() {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [params, setParams] = useSearchParams()
  const q = { state: params.get('state') ?? undefined, service_key: params.get('service_key') ?? undefined, cursor: params.get('cursor') ?? undefined, limit: 50 }
  const st = useRead(`op-queue:${JSON.stringify(q)}`, async () => settle(await api.GET('/api/v1/fulfillment/work-cases', { params: { query: compact(q) as never } }),
    e => e.items ? { items: e.items, page: e.page ?? { has_more: false } } : undefined))
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    const next = new URLSearchParams()
    for (const k of ['state', 'service_key']) {
      const v = String(f.get(k) ?? '').trim()
      if (v) next.set(k, v)
    }
    setParams(next)
  }
  return (
    <>
      <PageHeader title="صف کار اجرای خدمت" subtitle="فقط پرونده‌های خدماتی که اکنون ناظر یا مجری آن‌ها هستید.">
        <StatusPill tone="slate" icon="lock" label="SHADOW · کنترل‌شده" />
      </PageHeader>
      <ShadowNotice />
      <Card title="فیلترها">
        <form className="op-filters" onSubmit={submit} aria-label="فیلتر صف کار">
          <label className="lbl">وضعیت
            <select className="input" name="state" defaultValue={q.state ?? ''}>
              <option value="">همه</option>
              {states.map(x => <option key={x} value={x}>{stateLabel[x]}</option>)}
            </select>
          </label>
          <label className="lbl">کلید خدمت<input className="input op-ltr" name="service_key" defaultValue={q.service_key ?? ''} /></label>
          <div><Button type="submit" variant="primary">اعمال فیلتر</Button></div>
        </form>
      </Card>
      <ReadStates state={st} what="صف کار" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (
        <Card title="پرونده‌ها">
          {st.data.items.length === 0
            ? <StateBlock kind="empty" title="هیچ پرونده‌ای در دامنه خدمات شما نیست" />
            : (
              <div className="op-table-wrap">
                <table className="op-table">
                  <caption className="sr">صف کار اجرای خدمت</caption>
                  <thead><tr>
                    <th scope="col">خدمت</th><th scope="col">وضعیت</th><th scope="col">مجری</th><th scope="col">نقش شما</th><th scope="col">آخرین تغییر</th>
                  </tr></thead>
                  <tbody>
                    {st.data.items.map(c => (
                      <tr key={c.case_id}>
                        <td><Link to={`/operations/work/${c.case_id}`}>{c.service_title}</Link><br /><span className="muted op-ltr">{`${c.service_key} v${c.service_version}`}</span></td>
                        <td><StatusPill tone={stateTone[c.state] ?? 'slate'} icon="check" label={stateLabel[c.state] ?? c.state} /></td>
                        <td>{c.assignee_user_id ? <span className="op-ltr">{short(c.assignee_user_id)}</span> : <span className="muted">تخصیص نیافته</span>}</td>
                        <td>{[c.scope?.supervisor && 'ناظر خدمت', c.scope?.executor && 'مجری خدمت'].filter(Boolean).join('، ') || '—'}</td>
                        <td>{formatInstant(c.updated_at, zone)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          <div className="op-pager">
            {params.get('cursor') && <Button onClick={() => { const n = new URLSearchParams(params); n.delete('cursor'); setParams(n) }}>صفحه اول</Button>}
            {st.data.page.has_more && st.data.page.next_cursor && (
              <Button onClick={() => { const n = new URLSearchParams(params); n.set('cursor', st.data.page.next_cursor ?? ''); setParams(n) }}>صفحه بعد</Button>
            )}
          </div>
        </Card>
      )}
    </>
  )
}
