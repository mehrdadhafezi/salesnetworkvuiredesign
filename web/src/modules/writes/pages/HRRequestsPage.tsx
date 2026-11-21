import '../writes.css'
import '../../hr/hr.css'
import { useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router'
import { useApi, type CommandEnvelope } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { formatInstant } from '../../../shared/ui/time'
import { QueueTabs } from '../../../shared/ui/ds'
import { PersonResolver, UserPerson } from '../../hr/workspace/people'
import { HRSectionNav } from '../../hr/workspace/SectionNav'
import { hrStatusIcon, hrStatusTone, isQueue, REQUEST_QUEUES, requestStatusLabel, requestTypeLabel, type RequestQueue } from '../../hr/workspace/requests'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { CommandBanner, newIdempotencyKey } from '../CommandBanner'
import { hrTypeLabel, modeLabel } from '../labels'

const blockedDetail = 'درخواست‌های منابع انسانی پشت پرچم organization.hr_request_write هستند.'

interface Draft { type: 'transfer_seller' | 'terminate_seller'; subject: string; target: string; reason: string }

/**
 * HR change requests the caller requested or reviews (Slice 11). Creating a
 * request applies nothing: no hierarchy, employment, credential or
 * capability change; final HR apply is unavailable (OPD-05, SD-04).
 * RM-P04: status queues are server filters inside the caller's row scope
 * (never a claim about every request), cursor paged; people are resolved
 * from account ids only when the caller may read the employment link.
 */
export function HRRequestsPage() {
  return <PersonResolver><HRRequestsView /></PersonResolver>
}

function HRRequestsView() {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [params, setParams] = useSearchParams()
  const raw = params.get('status')
  const queue: RequestQueue = isQueue(raw) ? raw : 'all'
  const [cursors, setCursors] = useState<string[]>([])
  const cursor = cursors[cursors.length - 1]
  const [reload, setReload] = useState(0)
  const list = useRead(`hr-req:${queue}:${cursor ?? ''}:${reload}`, async () => settle(await api.GET('/api/v1/hr/requests', { params: { query: {
    limit: 25, ...(queue === 'all' ? {} : { status: queue }), ...(cursor ? { cursor } : {}) } } }),
  e => e.items ? { items: e.items, mode: e.rollout_mode, page: e.page } : undefined))
  const [draft, setDraft] = useState<Draft | null>(null)
  const [key, setKey] = useState('')
  const [result, setResult] = useState<CommandEnvelope | null>(null)
  const [busy, setBusy] = useState(false)

  function prepare(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    setDraft({ type: String(f.get('type')) as Draft['type'], subject: String(f.get('subject') ?? '').trim(),
      target: String(f.get('target') ?? '').trim(), reason: String(f.get('reason') ?? '').trim() })
    setKey(newIdempotencyKey())
    setResult(null)
  }

  async function submit() {
    if (!draft) return
    setBusy(true)
    const r = await api.POST('/api/v1/hr/requests', { params: { header: { 'Idempotency-Key': key } }, body: {
      request_type: draft.type, subject_user_id: draft.subject, reason: draft.reason,
      ...(draft.type === 'transfer_seller' ? { target_parent_user_id: draft.target } : {}) } })
    setBusy(false)
    const env = (r.data ?? r.error) as CommandEnvelope | undefined
    setResult(env && 'outcome' in env ? env : ({ outcome: 'FAILED', reasons: [], correlation_id: '' } as CommandEnvelope))
    if (env && 'outcome' in env && env.outcome !== 'UNKNOWN') {
      setDraft(null)
      setCursors([])
      setReload(n => n + 1)
    }
  }

  return (
    <>
      <HRSectionNav />
      <PageHeader title="درخواست‌های جابجایی / قطع همکاری" subtitle="ثبت درخواست اعمال واقعی نیست؛ اعمال نهایی در این مرحله در دسترس نیست.">
        <StatusPill tone="amber" icon="alert" label="دستور کنترل‌شده" />
      </PageHeader>
      <ReadStates state={list} what="درخواست‌ها" blockedDetail={blockedDetail} />
      {list.status === 'ok' && (
        <div className="wr-sections hr-fit">
          {list.data.mode && <p className="muted">{modeLabel[list.data.mode]}</p>}
          {result && <CommandBanner env={result} />}
          {actionHint(list.allowed, 'org.hr_request.create').allowed && !draft && (
            <Card title="درخواست جدید">
              <form className="wr-form" onSubmit={prepare} aria-label="ثبت درخواست منابع انسانی">
                <label className="lbl">نوع
                  <select className="input" name="type" defaultValue="terminate_seller">
                    <option value="terminate_seller">{hrTypeLabel.terminate_seller}</option>
                    <option value="transfer_seller">{hrTypeLabel.transfer_seller}</option>
                  </select>
                </label>
                <label className="lbl">شناسه کاربر فروشنده<input className="input wr-ltr" name="subject" required /></label>
                <label className="lbl">شناسه سرپرست مقصد (فقط جابجایی؛ باید در محدوده شما باشد)<input className="input wr-ltr" name="target" /></label>
                <label className="lbl">دلیل<textarea className="input" name="reason" maxLength={1000} required /></label>
                <div><Button type="submit" variant="primary">بررسی پیش از ثبت</Button></div>
              </form>
            </Card>
          )}
          {draft && (
            <div className="wr-confirm" role="dialog" aria-label="تایید ثبت درخواست">
              <dl className="wr-dl">
                <dt>اقدام</dt><dd>{hrTypeLabel[draft.type]}</dd>
                <dt>فروشنده</dt><dd className="wr-ltr">{draft.subject}</dd>
                {draft.type === 'transfer_seller' && <><dt>مقصد</dt><dd className="wr-ltr">{draft.target}</dd></>}
                <dt>دلیل</dt><dd>{draft.reason}</dd>
                <dt>اثر</dt><dd>فقط درخواست ثبت می‌شود؛ سلسله‌مراتب، استخدام، دسترسی و پرونده‌ها تغییر نمی‌کنند.</dd>
                <dt>محدودیت‌ها</dt><dd>مقصد خارج از محدوده یا برابر سرپرست فعلی مسدود است (SD-04).</dd>
              </dl>
              <div className="wr-actions">
                <Button variant="primary" disabled={busy} onClick={() => { void submit() }}>تایید و ثبت</Button>
                <Button onClick={() => setDraft(null)}>انصراف</Button>
              </div>
            </div>
          )}
          <Card title="درخواست‌ها">
            <p className="muted">فقط درخواست‌هایی که شما درخواست‌دهنده، بازبین فعلی یا اقدام‌کننده آن بوده‌اید. شمارش کل درخواست‌ها از این API در دسترس نیست.</p>
            <QueueTabs label="وضعیت درخواست" value={queue} onChange={v => {
              const next = new URLSearchParams(params)
              if (v === 'all') next.delete('status'); else next.set('status', v)
              setCursors([])
              setParams(next)
            }} queues={REQUEST_QUEUES.map(q => ({ id: q.id, label: q.label }))} />
            {list.data.items.length === 0
              ? <StateBlock kind="empty" title={queue === 'all' ? 'درخواستی برای شما وجود ندارد' : 'درخواستی با این وضعیت در محدوده شما نیست'} />
              : (
                <div className="wr-table-wrap">
                  <table className="wr-table hr-stack">
                    <caption className="sr">درخواست‌های منابع انسانی</caption>
                    <thead><tr><th scope="col">نوع</th><th scope="col">وضعیت</th><th scope="col">نیروی موضوع</th><th scope="col">درخواست‌دهنده</th>
                      <th scope="col">بازبین فعلی</th><th scope="col">ثبت</th><th scope="col">اقدام</th></tr></thead>
                    <tbody>
                      {list.data.items.map(q => (
                        <tr key={q.request_id}>
                          <td data-label="نوع"><Link to={`/hr/requests/${q.request_id}`}>{requestTypeLabel(q.request_type)}</Link></td>
                          <td data-label="وضعیت"><StatusPill tone={hrStatusTone(q.status)} icon={hrStatusIcon(q.status)} label={requestStatusLabel(q.status)} /></td>
                          <td data-label="نیروی موضوع"><UserPerson userId={q.subject_user_id} /></td>
                          <td data-label="درخواست‌دهنده"><UserPerson userId={q.requested_by_user_id} /></td>
                          <td data-label="بازبین فعلی"><UserPerson userId={q.current_reviewer_user_id} /></td>
                          <td data-label="ثبت">{formatInstant(q.created_at, zone)}</td>
                          <td data-label="اقدام">{actionHint(q.allowed_actions, 'approve_step').allowed ? 'نیازمند تصمیم شما' : <span className="muted">—</span>}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            <nav className="hr-pager" aria-label="صفحه‌بندی">
              <Button disabled={cursors.length === 0} onClick={() => setCursors(c => c.slice(0, -1))}>صفحه قبل</Button>
              <Button disabled={!list.data.page?.has_more || !list.data.page.next_cursor}
                onClick={() => { const n = list.data.page?.next_cursor; if (n) setCursors(c => [...c, n]) }}>صفحه بعد</Button>
            </nav>
          </Card>
        </div>
      )}
    </>
  )
}
