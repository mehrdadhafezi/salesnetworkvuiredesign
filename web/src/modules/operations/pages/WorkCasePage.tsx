import '../operations.css'
import { useState, type FormEvent } from 'react'
import { useParams } from 'react-router'
import { useApi, type components } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Button, Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { CommandResult, ShadowNotice, newKey } from '../CommandResult'
import { actionLabel, blockedCommandLabel, blockedDetail, eventLabel, financialLabel, outcomeLabel, short, stateLabel, stateTone } from '../labels'

type Env = components['schemas']['WorkCommandEnvelope']
type WorkCase = components['schemas']['WorkCase']

type Intent =
  | { kind: 'assign'; assignee: string }
  | { kind: 'action'; action: 'NO_ANSWER' | 'FOLLOW_UP' | 'NOTE'; note: string; followUp: string }
  | { kind: 'evidence'; type: string; label: string; sha: string }
  | { kind: 'outcome'; outcome: 'EXECUTED' | 'NOT_EXECUTED'; reason: string }

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
const SHA = /^[0-9a-f]{64}$/

function describe(i: Intent): string {
  switch (i.kind) {
    case 'assign': return `تخصیص به مجری ${i.assignee}`
    case 'action': return `ثبت اقدام: ${actionLabel[i.action]}${i.note ? ` — ${i.note}` : ''}`
    case 'evidence': return `ثبت مدرک «${i.label}» (${i.type}) با اثر انگشت ${i.sha.slice(0, 12)}…`
    case 'outcome': return `ثبت نتیجه: ${outcomeLabel[i.outcome]}${i.reason ? ` — ${i.reason}` : ''}`
  }
}

/**
 * One work case: definition, evidence, history and the guarded commands
 * the server reports for the caller's scope. Viewing never assigns or
 * completes; every command needs an explicit confirmation, carries an
 * idempotency key and the expected version, and the server re-checks it.
 */
export function WorkCasePage() {
  const { caseId = '' } = useParams()
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const [reload, setReload] = useState(0)
  const st = useRead(`op-case:${caseId}:${reload}`, async () => settle(await api.GET('/api/v1/fulfillment/work-cases/{case_id}', {
    params: { path: { case_id: caseId } } }), e => e.case ? { c: e.case, history: e.history ?? [], evidence: e.evidence ?? [], blocked: e.blocked_commands ?? [] } : undefined))
  const [intent, setIntent] = useState<Intent | null>(null)
  const [key, setKey] = useState('')
  const [result, setResult] = useState<Env | null>(null)
  const [busy, setBusy] = useState(false)
  const [formError, setFormError] = useState('')
  const unknown = result?.outcome === 'UNKNOWN'

  function prepare(next: Intent) {
    if (JSON.stringify(next) !== JSON.stringify(intent)) setKey(newKey())
    setIntent(next)
    setResult(null)
    setFormError('')
  }

  async function submit(c: WorkCase) {
    if (!intent || unknown) return
    setBusy(true)
    const header = { 'Idempotency-Key': key, 'If-Match': String(c.version) }
    const path = { case_id: c.case_id }
    let r: { data?: Env; error?: unknown }
    switch (intent.kind) {
      case 'assign':
        r = await api.POST('/api/v1/fulfillment/work-cases/{case_id}/assignment', { params: { path, header }, body: { assignee_user_id: intent.assignee } })
        break
      case 'action':
        r = await api.POST('/api/v1/fulfillment/work-cases/{case_id}/operational-actions', { params: { path, header }, body: { action: intent.action,
          ...(intent.note ? { note: intent.note } : {}), ...(intent.action === 'FOLLOW_UP' && intent.followUp ? { follow_up_at: new Date(intent.followUp).toISOString() } : {}) } })
        break
      case 'evidence':
        r = await api.POST('/api/v1/fulfillment/work-cases/{case_id}/evidence-records', { params: { path, header },
          body: { evidence_type: intent.type, label: intent.label, content_sha256: intent.sha } })
        break
      case 'outcome':
        r = await api.POST('/api/v1/fulfillment/work-cases/{case_id}/outcome', { params: { path, header },
          body: { outcome: intent.outcome, ...(intent.reason ? { reason: intent.reason } : {}) } })
        break
    }
    setBusy(false)
    const env = (r.data ?? r.error) as Env | undefined
    if (env && typeof env === 'object' && 'outcome' in env) {
      setResult(env)
      if (env.outcome === 'SUCCESS' || env.outcome === 'CONFLICT') {
        setIntent(null)
        setReload(n => n + 1)
      }
    } else {
      setResult({ outcome: 'FAILED', reasons: [], correlation_id: '' })
    }
  }

  return (
    <>
      <PageHeader title="پرونده اجرای خدمت" subtitle="تعریف خدمت، مدارک، تاریخچه و دستورهای مجاز در دامنه شما.">
        <StatusPill tone="slate" icon="lock" label="SHADOW · کنترل‌شده" />
      </PageHeader>
      <ShadowNotice />
      <ReadStates state={st} what="پرونده" blockedDetail={blockedDetail} />
      {st.status === 'ok' && (() => {
        const { c, history, evidence, blocked } = st.data
        const supervisor = !!c.scope?.supervisor && actionHint(st.allowed, 'fulfillment.work.assign').allowed
        const executor = !!c.scope?.executor && actionHint(st.allowed, 'fulfillment.work.execute').allowed && c.assignee_user_id != null
        const open = c.state === 'ASSIGNED' || c.state === 'IN_PROGRESS'
        const canAssign = supervisor && (c.state === 'QUEUED' || c.state === 'ASSIGNED')
        const required = (c.evidence_requirements ?? []).filter(r => r.required)
        const missing = required.filter(r => !evidence.some(e => e.evidence_type === r.type))
        return (
          <div className="op-sections">
            <Card title={c.service_title}>
              <dl className="op-dl">
                <dt>خدمت</dt><dd className="op-ltr">{`${c.service_key} v${c.service_version} · ${c.workflow_key}`}</dd>
                <dt>وضعیت</dt><dd><StatusPill tone={stateTone[c.state] ?? 'slate'} icon="check" label={stateLabel[c.state] ?? c.state} /></dd>
                <dt>مجری فعلی</dt><dd>{c.assignee_user_id ? <span className="op-ltr">{c.assignee_user_id}</span> : <span className="muted">تخصیص نیافته</span>}</dd>
                <dt>ناظران فعلی خدمت</dt><dd>{c.current_supervisors.length ? <span className="op-ltr">{c.current_supervisors.map(short).join('، ')}</span> : <span className="muted">ثبت نشده</span>}</dd>
                <dt>اثر مالی</dt><dd>{financialLabel[c.financial_behaviour] ?? c.financial_behaviour}</dd>
                <dt>ارائه‌دهنده</dt><dd>{c.integration_provider === 'NONE' ? 'بدون ارائه‌دهنده خارجی' : 'بیاوین (اتصال پیکربندی نشده؛ تکمیل نیازمند تأیید با شواهد)'}</dd>
                <dt>اشتراک / آیتم</dt><dd className="op-ltr">{`${short(c.subscription_id)} / ${short(c.item_id)}`}</dd>
                <dt>نسخه پرونده</dt><dd className="op-ltr">{c.version}</dd>
              </dl>
            </Card>
            <Card title="مدارک لازم">
              {(c.evidence_requirements ?? []).length === 0 ? <p className="muted">این خدمت مدرک الزامی ندارد.</p> : (
                <ul className="op-list" aria-label="مدارک لازم">
                  {(c.evidence_requirements ?? []).map(r => (
                    <li key={r.type}><span className="op-ltr">{r.type}</span> · {r.required ? 'الزامی' : 'اختیاری'} · {evidence.some(e => e.evidence_type === r.type) ? 'ثبت شده' : 'ثبت نشده'}</li>
                  ))}
                </ul>
              )}
            </Card>
            <Card title="دستورها">
              {result && <CommandResult env={result} />}
              {unknown && <p className="muted">تا روشن شدن نتیجه، دستور دیگری روی این پرونده ارسال نکنید.</p>}
              {!intent && !unknown && (
                <div className="op-sections">
                  {canAssign && (
                    <form className="op-form" aria-label="تخصیص پرونده" onSubmit={(e: FormEvent<HTMLFormElement>) => {
                      e.preventDefault()
                      const v = String(new FormData(e.currentTarget).get('assignee') ?? '').trim()
                      if (!UUID.test(v)) { setFormError('شناسه مجری باید UUID کاربر سامانه جدید باشد.'); return }
                      prepare({ kind: 'assign', assignee: v })
                    }}>
                      <label className="lbl">شناسه کاربر مجری (باید اکنون مجری همین خدمت باشد)<input className="input op-ltr" name="assignee" /></label>
                      <div><Button type="submit" variant="primary">بررسی تخصیص</Button></div>
                    </form>
                  )}
                  {executor && open && (
                    <>
                      <form className="op-form" aria-label="ثبت اقدام عملیاتی" onSubmit={(e: FormEvent<HTMLFormElement>) => {
                        e.preventDefault()
                        const f = new FormData(e.currentTarget)
                        prepare({ kind: 'action', action: String(f.get('action')) as 'NO_ANSWER' | 'FOLLOW_UP' | 'NOTE', note: String(f.get('note') ?? '').trim(), followUp: String(f.get('follow') ?? '') })
                      }}>
                        <label className="lbl">اقدام
                          <select className="input" name="action" defaultValue="NO_ANSWER">
                            {Object.entries(actionLabel).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
                          </select>
                        </label>
                        <label className="lbl">یادداشت<textarea className="input" name="note" maxLength={1000} /></label>
                        <label className="lbl">زمان پیگیری (فقط برای «پیگیری بعدی»)<input className="input" type="datetime-local" name="follow" /></label>
                        <div><Button type="submit">بررسی اقدام</Button></div>
                      </form>
                      {(c.evidence_requirements ?? []).length > 0 && (
                        <form className="op-form" aria-label="ثبت مدرک" onSubmit={(e: FormEvent<HTMLFormElement>) => {
                          e.preventDefault()
                          const f = new FormData(e.currentTarget)
                          const sha = String(f.get('sha') ?? '').trim().toLowerCase()
                          const label = String(f.get('label') ?? '').trim()
                          if (!SHA.test(sha) || !label) { setFormError('عنوان و اثر انگشت SHA-256 (۶۴ نویسه هگز) لازم است.'); return }
                          prepare({ kind: 'evidence', type: String(f.get('type')), label, sha })
                        }}>
                          <label className="lbl">نوع مدرک
                            <select className="input op-ltr" name="type">{(c.evidence_requirements ?? []).map(r => <option key={r.type} value={r.type}>{r.type}</option>)}</select>
                          </label>
                          <label className="lbl">عنوان<input className="input" name="label" maxLength={200} /></label>
                          <label className="lbl">اثر انگشت SHA-256 فایل (فایل بارگذاری نمی‌شود)<input className="input op-ltr" name="sha" /></label>
                          <div><Button type="submit">بررسی مدرک</Button></div>
                        </form>
                      )}
                      <form className="op-form" aria-label="ثبت نتیجه" onSubmit={(e: FormEvent<HTMLFormElement>) => {
                        e.preventDefault()
                        const f = new FormData(e.currentTarget)
                        const outcome = String(f.get('outcome')) as 'EXECUTED' | 'NOT_EXECUTED'
                        const reason = String(f.get('reason') ?? '').trim()
                        if (outcome === 'NOT_EXECUTED' && !reason) { setFormError('برای «انجام نشد» دلیل الزامی است.'); return }
                        prepare({ kind: 'outcome', outcome, reason })
                      }}>
                        <label className="lbl">نتیجه
                          <select className="input" name="outcome" defaultValue="EXECUTED">
                            {(c.completion_outcomes ?? []).map(o => <option key={o} value={o}>{outcomeLabel[o] ?? o}</option>)}
                          </select>
                        </label>
                        <label className="lbl">دلیل (برای «انجام نشد» الزامی)<textarea className="input" name="reason" maxLength={1000} /></label>
                        {missing.length > 0 && <p className="muted">{`مدارک الزامی ثبت‌نشده: ${missing.map(m => m.type).join('، ')} — ثبت نتیجه مسدود می‌ماند.`}</p>}
                        <div><Button type="submit">بررسی نتیجه</Button></div>
                      </form>
                    </>
                  )}
                  {!canAssign && !(executor && open) && <p className="muted">برای این پرونده دستوری در دامنه فعلی شما در دسترس نیست.</p>}
                  {formError && <StateBlock kind="error" title={formError} />}
                </div>
              )}
              {intent && !unknown && (
                <div className="op-confirm" role="dialog" aria-label="تایید دستور">
                  <dl className="op-dl">
                    <dt>پرونده</dt><dd className="op-ltr">{c.case_id}</dd>
                    <dt>دستور</dt><dd>{describe(intent)}</dd>
                    <dt>وضعیت فعلی</dt><dd>{stateLabel[c.state] ?? c.state}</dd>
                    <dt>اثر</dt><dd>فقط پرونده سایه سامانه جدید؛ فاکتور، پرداخت، کیف پول، پورسانت، اشتراک و سامانه قدیمی تغییر نمی‌کنند.</dd>
                    <dt>نسخه مورد انتظار</dt><dd className="op-ltr">{c.version}</dd>
                  </dl>
                  <div className="op-actions">
                    <Button variant="primary" disabled={busy} onClick={() => { void submit(c) }}>تایید و ارسال</Button>
                    <Button onClick={() => setIntent(null)}>انصراف</Button>
                  </div>
                </div>
              )}
            </Card>
            <Card title="دستورهای در دسترس نیستند (تصمیم باز)">
              <ul className="op-list" aria-label="دستورهای مسدود">
                {blocked.map(b => (
                  <li key={b.command}><StatusPill tone="slate" icon="lock" label={blockedCommandLabel[b.command] ?? b.command} /> <span className="muted">{b.blocked_by.join('، ')}</span></li>
                ))}
              </ul>
            </Card>
            <Card title="مدارک ثبت‌شده">
              {evidence.length === 0 ? <StateBlock kind="empty" title="هنوز مدرکی ثبت نشده است" /> : (
                <ul className="op-list" aria-label="مدارک ثبت‌شده">
                  {evidence.map(e => <li key={e.evidence_id}>{e.label} · <span className="op-ltr">{`${e.evidence_type} · ${e.content_sha256.slice(0, 12)}…`}</span> · {formatInstant(e.occurred_at, zone)}</li>)}
                </ul>
              )}
            </Card>
            <Card title="تاریخچه (فقط افزودنی)">
              <ol className="op-list" aria-label="تاریخچه پرونده">
                {history.map(h => (
                  <li key={h.event_id}>{eventLabel[h.kind] ?? h.kind} · {formatInstant(h.occurred_at, zone)} · <span className="muted op-ltr">{`${h.before_version == null ? '' : `v${h.before_version}→`}v${h.after_version}${h.actor_user_id ? ` · ${short(h.actor_user_id)}` : ''}`}</span></li>
                ))}
              </ol>
            </Card>
          </div>
        )
      })()}
    </>
  )
}
