import '../writes.css'
import '../../hr/hr.css'
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
import { blockedLabel, hrTypeLabel } from '../labels'
import { PersonResolver, UserPerson } from '../../hr/workspace/people'
import { HRSectionNav } from '../../hr/workspace/SectionNav'
import { hrStatusIcon, hrStatusTone, lifecycle, requestStatusLabel, stepLabel } from '../../hr/workspace/requests'

/**
 * One HR request: requester, reviewer and target are snapshots; history is
 * append-only. Step approval / rejection only for the current reviewer;
 * final apply is shown as unavailable, never as a control.
 */
export function HRRequestPage() {
  return <PersonResolver><HRRequestView /></PersonResolver>
}

function HRRequestView() {
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
      <HRSectionNav />
      <PageHeader title="درخواست منابع انسانی" subtitle="تایید مرحله‌ای ≠ اعمال تغییر.">
        <StatusPill tone="amber" icon="alert" label="دستور کنترل‌شده" />
      </PageHeader>
      <p><Link to="/hr/requests">بازگشت به فهرست</Link></p>
      <ReadStates state={st} what="درخواست" blockedDetail="درخواست‌های منابع انسانی پشت پرچم organization.hr_request_write هستند." />
      {st.status === 'ok' && (
        <div className="wr-sections hr-fit">
          {result && <CommandBanner env={result} />}
          <ol className="hr-steps" aria-label="مراحل درخواست">
            {lifecycle(st.data.request.status).map(step => (
              <li key={step.key} data-state={step.state} aria-current={step.state === 'current' ? 'step' : undefined}>
                <b>{step.label}</b> <span className="muted">{stepLabel[step.state]}</span>
              </li>
            ))}
          </ol>
          <Card title={hrTypeLabel[st.data.request.request_type] ?? st.data.request.request_type}>
            <dl className="wr-dl">
              <dt>وضعیت</dt><dd><StatusPill tone={hrStatusTone(st.data.request.status)} icon={hrStatusIcon(st.data.request.status)} label={requestStatusLabel(st.data.request.status)} />
                <span className="muted wr-ltr"> {st.data.request.status}</span></dd>
              <dt>فروشنده</dt><dd><UserPerson userId={st.data.request.subject_user_id} /></dd>
              <dt>سرپرست فعلی (لحظه ثبت)</dt><dd><UserPerson userId={st.data.request.subject_parent_user_id} /></dd>
              <dt>مقصد</dt><dd><UserPerson userId={st.data.request.target_parent_user_id} /></dd>
              <dt>درخواست‌دهنده</dt><dd><UserPerson userId={st.data.request.requested_by_user_id} /></dd>
              <dt>بازبین فعلی</dt><dd><UserPerson userId={st.data.request.current_reviewer_user_id} empty="ندارد" /></dd>
              <dt>دلیل</dt><dd>{st.data.request.reason}</dd>
              {st.data.request.reviewer_notes && <><dt>یادداشت بازبین</dt><dd>{st.data.request.reviewer_notes}</dd></>}
              <dt>ثبت · آخرین تغییر</dt><dd>{formatInstant(st.data.request.created_at, zone)} · {formatInstant(st.data.request.updated_at, zone)}</dd>
            </dl>
            <p className="muted">درخواست‌دهنده، سرپرست، مقصد و بازبین تصویر لحظه ثبت/گذار هستند؛ نام فعلی از پرونده امروز خوانده می‌شود و سابقه را بازنویسی نمی‌کند.</p>
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
          <Card title="درخواست‌شده در برابر اعمال‌شده">
            <div className="hr-rva">
              <section aria-label="تغییر درخواست‌شده">
                <h3 className="hr-sub">درخواست‌شده</h3>
                <p>{hrTypeLabel[st.data.request.request_type] ?? st.data.request.request_type}{st.data.request.request_type === 'transfer_seller' && <> ← <UserPerson userId={st.data.request.target_parent_user_id} /></>}</p>
              </section>
              <section aria-label="تغییر اعمال‌شده">
                <h3 className="hr-sub">اعمال‌شده</h3>
                <p>{st.data.request.status === 'approved' ? 'سرور وضعیت «تایید و اعمال شده» را گزارش می‌کند؛ اثر واقعی جداگانه تأیید نشده است.'
                  : st.data.request.status === 'failed' ? 'اعمال ناموفق؛ اثر واقعی نامعلوم است و باید پیش از هر تکرار تطبیق شود.'
                    : 'اعمال نشده. سلسله‌مراتب، همکاری، حساب و دسترسی تغییری نکرده‌اند.'}</p>
              </section>
            </div>
          </Card>
          <Card title="تاریخچه">
            {st.data.history.length === 0 ? <p className="muted">رویدادی ثبت نشده است.</p> : (
              <ol className="wr-list" aria-label="تاریخچه درخواست">
                {st.data.history.map(h => (
                  <li key={h.event_id}>
                    <b className="wr-ltr">{h.action}</b> · {h.from_status ? `${requestStatusLabel(h.from_status)} ← ` : ''}{requestStatusLabel(h.to_status)} · {formatInstant(h.occurred_at, zone)}
                    <div className="muted">اقدام‌کننده: <UserPerson userId={h.actor_user_id} />{h.next_reviewer_user_id && <> · بازبین بعدی: <UserPerson userId={h.next_reviewer_user_id} /></>}
                      {h.notes && <> · یادداشت: {h.notes}</>} · نسخه <span className="num">{h.after_version}</span></div>
                  </li>
                ))}
              </ol>
            )}
          </Card>
        </div>
      )}
    </>
  )
}
