import '../writes.css'
import { useState } from 'react'
import { Link, useParams } from 'react-router'
import { useApi, type CommandEnvelope } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Button, Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { CommandBanner, newIdempotencyKey } from '../CommandBanner'
import { blockedLabel, hrStatusLabel, hrTypeLabel } from '../labels'

/**
 * One HR request: requester, reviewer and target are snapshots; history is
 * append-only. Step approval / rejection only for the current reviewer;
 * final apply is shown as unavailable, never as a control.
 */
export function HRRequestPage() {
  const { requestId = '' } = useParams()
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [reload, setReload] = useState(0)
  const st = useRead(`hr-one:${requestId}:${reload}`, async () => settle(await api.GET('/api/v1/hr/requests/{request_id}', {
    params: { path: { request_id: requestId } } }), e => e.request ? { request: e.request, history: e.history ?? [] } : undefined))
  const [pending, setPending] = useState<{ decision: 'approve' | 'reject'; key: string } | null>(null)
  const [result, setResult] = useState<CommandEnvelope | null>(null)

  async function decide(version: number) {
    if (!pending) return
    const r = await api.POST('/api/v1/hr/requests/{request_id}/decisions', {
      params: { path: { request_id: requestId }, header: { 'Idempotency-Key': pending.key, 'If-Match': String(version) } },
      body: { decision: pending.decision } })
    const env = (r.data ?? r.error) as CommandEnvelope | undefined
    setResult(env && 'outcome' in env ? env : ({ outcome: 'FAILED', reasons: [], correlation_id: '' } as CommandEnvelope))
    if (env && 'outcome' in env && env.outcome !== 'UNKNOWN') {
      setPending(null)
      setReload(n => n + 1)
    }
  }

  return (
    <>
      <PageHeader title="درخواست منابع انسانی" subtitle="تایید مرحله‌ای ≠ اعمال تغییر.">
        <StatusPill tone="amber" icon="alert" label="دستور کنترل‌شده" />
      </PageHeader>
      <p><Link to="/hr/requests">بازگشت به فهرست</Link></p>
      <ReadStates state={st} what="درخواست" blockedDetail="درخواست‌های منابع انسانی پشت پرچم organization.hr_request_write هستند." />
      {st.status === 'ok' && (
        <div className="wr-sections">
          {result && <CommandBanner env={result} />}
          <Card title={hrTypeLabel[st.data.request.request_type] ?? st.data.request.request_type}>
            <dl className="wr-dl">
              <dt>وضعیت</dt><dd>{hrStatusLabel[st.data.request.status] ?? st.data.request.status}</dd>
              <dt>فروشنده</dt><dd className="wr-ltr">{st.data.request.subject_user_id}</dd>
              <dt>سرپرست فعلی (لحظه ثبت)</dt><dd className="wr-ltr">{st.data.request.subject_parent_user_id ?? '—'}</dd>
              <dt>مقصد</dt><dd className="wr-ltr">{st.data.request.target_parent_user_id ?? '—'}</dd>
              <dt>درخواست‌دهنده</dt><dd className="wr-ltr">{st.data.request.requested_by_user_id}</dd>
              <dt>بازبین فعلی</dt><dd className="wr-ltr">{st.data.request.current_reviewer_user_id ?? '—'}</dd>
              <dt>دلیل</dt><dd>{st.data.request.reason}</dd>
            </dl>
            {actionHint(st.data.request.allowed_actions, 'approve_step').allowed && !pending && (
              <div className="wr-actions">
                <Button variant="primary" onClick={() => setPending({ decision: 'approve', key: newIdempotencyKey() })}>تایید مرحله</Button>
                <Button onClick={() => setPending({ decision: 'reject', key: newIdempotencyKey() })}>رد درخواست</Button>
              </div>
            )}
            {pending && (
              <div className="wr-confirm" role="dialog" aria-label="تایید تصمیم">
                <p>{pending.decision === 'approve' ? 'تایید این مرحله درخواست را به بازبین بعدی (یا منابع انسانی) می‌فرستد؛ چیزی اعمال نمی‌شود.' : 'درخواست رد می‌شود.'}</p>
                <div className="wr-actions">
                  <Button variant="primary" onClick={() => { void decide(st.data.request.version) }}>تایید</Button>
                  <Button onClick={() => setPending(null)}>انصراف</Button>
                </div>
              </div>
            )}
            <ul className="wr-list" aria-label="اقدام‌های مسدود">
              {st.data.request.blocked_actions.map(b => (
                <li key={b.action}><StatusPill tone="slate" icon="lock" label={blockedLabel[b.action] ?? b.action} /> <span className="muted">{b.blocked_by.join('، ')}</span></li>
              ))}
            </ul>
          </Card>
          <Card title="تاریخچه">
            <ol className="wr-list" aria-label="تاریخچه درخواست">
              {st.data.history.map(h => (
                <li key={h.event_id}>{h.action} · {hrStatusLabel[h.to_status] ?? h.to_status} · {formatInstant(h.occurred_at, zone)}</li>
              ))}
            </ol>
          </Card>
        </div>
      )}
    </>
  )
}
