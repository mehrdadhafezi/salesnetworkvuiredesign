import './writes.css'
import { useState, type FormEvent } from 'react'
import { useApi, type CommandEnvelope } from '../../shared/api'
import { useSession } from '../../shared/auth/currentSession'
import { actionHint } from '../../shared/permissions/allowedActions'
import { settle, useRead } from '../../shared/read/load'
import { ReadStates } from '../../shared/read/ReadStates'
import { formatInstant } from '../../shared/ui/time'
import { Button, Card, StateBlock, StatusPill } from '../../shared/ui/primitives'
import { CommandBanner, newIdempotencyKey } from './CommandBanner'
import { blockedLabel, modeLabel, outcomeLabel } from './labels'

const ACTION = 'cases.contact_outcome.record'
const OUTCOMES = ['NO_ANSWER', 'CALLBACK', 'NOT_PURCHASED'] as const
type Outcome = (typeof OUTCOMES)[number]

interface Intent { outcome: Outcome; reason: string; callback: string }

/**
 * The held lead's contact outcome (Slice 11). The record action appears only
 * when the server's allowed_actions (same evaluator as the command) allows
 * it; the server re-checks everything at commit. A contact outcome never
 * changes custody, the invoice, payments or finance.
 */
export function ContactOutcomePanel({ caseId }: { caseId: string }) {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [reload, setReload] = useState(0)
  const view = useRead(`co:${caseId}:${reload}`, async () => settle(await api.GET('/api/v1/cases/{case_id}/contact-outcomes', {
    params: { path: { case_id: caseId } } }), e => e.data))
  const [intent, setIntent] = useState<Intent | null>(null)
  const [key, setKey] = useState('')
  const [result, setResult] = useState<CommandEnvelope | null>(null)
  const [busy, setBusy] = useState(false)
  const unknown = result?.outcome === 'UNKNOWN'

  function prepare(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    const next: Intent = { outcome: String(f.get('outcome')) as Outcome, reason: String(f.get('reason') ?? '').trim(), callback: String(f.get('callback') ?? '') }
    // A changed intent gets a new idempotency key; the same intent keeps it.
    if (!intent || intent.outcome !== next.outcome || intent.reason !== next.reason || intent.callback !== next.callback) setKey(newIdempotencyKey())
    setIntent(next)
    setResult(null)
  }

  async function submit(version: number) {
    if (!intent || unknown) return
    setBusy(true)
    const body = { outcome: intent.outcome, ...(intent.outcome === 'NOT_PURCHASED' ? { reason: intent.reason } : {}),
      ...(intent.outcome === 'CALLBACK' ? { callback_at: new Date(intent.callback).toISOString() } : {}) }
    const r = await api.POST('/api/v1/cases/{case_id}/contact-outcomes', {
      params: { path: { case_id: caseId }, header: { 'Idempotency-Key': key, 'If-Match': String(version) } }, body })
    const env = (r.data ?? r.error) as CommandEnvelope | undefined
    setBusy(false)
    if (env && 'outcome' in env) {
      setResult(env)
      if (env.outcome === 'SUCCESS' || env.outcome === 'CONFLICT') {
        setIntent(null)
        setReload(n => n + 1) // conflict: refresh server state, never overwrite
      }
    } else {
      setResult({ outcome: 'FAILED', reasons: [], correlation_id: '' } as CommandEnvelope)
    }
  }

  return (
    <Card title="پیامد تماس (دستور کنترل‌شده)">
      <ReadStates state={view} what="پیامد تماس" blockedDetail="ثبت پیامد تماس پشت پرچم‌های cases.nonfinancial_write است." />
      {view.status === 'ok' && (
        <div className="wr-sections">
          <p className="muted">{modeLabel[view.data.rollout_mode]}</p>
          <dl className="wr-dl">
            <dt>پیامد فعلی</dt>
            <dd>{view.data.current ? outcomeLabel[view.data.current.outcome] : <span className="muted">در سامانه جدید ثبت نشده</span>}</dd>
            <dt>نسخه پرونده</dt><dd className="wr-ltr">{view.data.case_version}</dd>
          </dl>
          {result && <CommandBanner env={result} />}
          {actionHint(view.allowed, ACTION).allowed && !intent && !unknown && (
            <form className="wr-form" onSubmit={prepare} aria-label="ثبت پیامد تماس">
              <label className="lbl">پیامد
                <select className="input" name="outcome" defaultValue="NO_ANSWER">
                  {OUTCOMES.map(o => <option key={o} value={o}>{outcomeLabel[o]}</option>)}
                </select>
              </label>
              <label className="lbl">دلیل (فقط برای «عدم خرید»، اجباری)<textarea className="input" name="reason" maxLength={1000} /></label>
              <label className="lbl">زمان تماس مجدد (فقط برای «تماس مجدد»)<input className="input" type="datetime-local" name="callback" /></label>
              <div><Button type="submit" variant="primary">بررسی پیش از ثبت</Button></div>
            </form>
          )}
          {intent && !unknown && (
            <div className="wr-confirm" role="dialog" aria-label="تایید ثبت پیامد تماس">
              <dl className="wr-dl">
                <dt>موضوع</dt><dd className="wr-ltr">{caseId}</dd>
                <dt>اقدام</dt><dd>ثبت پیامد تماس</dd>
                <dt>وضعیت فعلی</dt><dd>{view.data.current ? outcomeLabel[view.data.current.outcome] : 'ثبت نشده'}</dd>
                <dt>وضعیت جدید</dt><dd>{outcomeLabel[intent.outcome]}{intent.outcome === 'NOT_PURCHASED' ? ` — ${intent.reason || 'بدون دلیل (رد می‌شود)'}` : ''}</dd>
                <dt>اثر</dt><dd>فقط پیامد تماس؛ مالکیت پرونده، فاکتور، پرداخت، مالی و اشتراک تغییر نمی‌کنند. سامانه قدیمی همچنان مرجع است.</dd>
                <dt>نسخه مورد انتظار</dt><dd className="wr-ltr">{view.data.case_version}</dd>
              </dl>
              <div className="wr-actions">
                <Button variant="primary" disabled={busy} onClick={() => { void submit(view.data.case_version) }}>تایید و ثبت</Button>
                <Button onClick={() => setIntent(null)}>انصراف</Button>
              </div>
            </div>
          )}
          {!actionHint(view.allowed, ACTION).allowed && (
            <p className="muted">{`ثبت پیامد تماس برای شما در دسترس نیست${actionHint(view.allowed, ACTION).reason_code ? ` (${actionHint(view.allowed, ACTION).reason_code})` : ''}.`}</p>
          )}
          <div>
            <strong>اقدام‌های در دسترس نیستند (تصمیم باز)</strong>
            <ul className="wr-list" aria-label="اقدام‌های مسدود">
              {view.data.blocked_actions.map(b => (
                <li key={b.action}><StatusPill tone="slate" icon="lock" label={blockedLabel[b.action] ?? b.action} /> <span className="muted">{b.blocked_by.join('، ')}</span></li>
              ))}
            </ul>
          </div>
          {view.data.history.length === 0
            ? <StateBlock kind="empty" title="هنوز پیامدی در سامانه جدید ثبت نشده است" />
            : (
              <ol className="wr-list" aria-label="تاریخچه پیامد تماس">
                {view.data.history.map(h => (
                  <li key={h.event_id}>{outcomeLabel[h.outcome] ?? h.outcome} · {formatInstant(h.occurred_at, zone)} · <span className="muted wr-ltr">{`v${h.before_version}→v${h.after_version}`}</span></li>
                ))}
              </ol>
            )}
        </div>
      )}
    </Card>
  )
}
