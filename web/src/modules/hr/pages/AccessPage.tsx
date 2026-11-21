import '../hr.css'
import { useState } from 'react'
import { Link } from 'react-router'
import { useEffectiveAccess } from '../../../app/navigation/NavigationAccess'
import { isRoleKey, ROLE_CONTRACTS } from '../../../app/navigation/roles'
import { useApi, type EffectiveAccess, type PersonSummary } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { settle, useRead } from '../../../shared/read/load'
import { QueueTabs } from '../../../shared/ui/ds'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { ReadStates } from '../components/States'
import { statusLabel, text } from '../labels'
import { useCanResolveUsers } from '../workspace/access'
import { HRSectionNav } from '../workspace/SectionNav'

type Decision = EffectiveAccess['decisions'][number]
type Verdict = 'allowed' | 'denied' | 'blocked'

const verdict = (d: Decision): Verdict => (d.allowed && d.outcome === 'SUCCESS' ? 'allowed' : d.outcome === 'BLOCKED' ? 'blocked' : 'denied')

const verdictPill: Record<Verdict, { tone: 'green' | 'slate' | 'amber'; label: string }> = {
  allowed: { tone: 'green', label: 'مجاز' }, denied: { tone: 'slate', label: 'مجاز نیست' }, blocked: { tone: 'amber', label: 'مسدود (پرچم ویژگی)' },
}

/** Module of a capability key (<module>.<object>.<action>); a name, never an authority. */
const moduleLabel: Record<string, string> = {
  identity: 'هویت', org: 'سازمان و منابع انسانی', customers: 'مشتریان', cases: 'پرونده‌ها', custody: 'نگهداری پرونده', sales: 'فروش',
  billing: 'صورت‌حساب', payments: 'پرداخت', reporting: 'گزارش', finance: 'مالی', biawin: 'بیاوین', fulfillment: 'اجرای خدمت',
  integration: 'یکپارچه‌سازی', commission: 'پورسانت',
}

/**
 * "دسترسی من": the server's effective-access evaluation of the CURRENT
 * principal (the same evaluator as allowed_actions), shown read only. Five
 * things are kept apart: credential (how you signed in), grant (what the
 * server holds for you), role (a named bundle carried by a grant), position
 * (an HR fact) and capability (the evaluated permission). No grant, revoke,
 * role or position-mapping command exists (D-34, D-42, R5-SEC-01 open), and
 * no other person's effective access is available from this API.
 */
export function AccessPage() {
  const ev = useEffectiveAccess()
  const s = useSession()
  const ready = s.status === 'ready' ? s.session : undefined
  return (
    <>
      <HRSectionNav />
      <PageHeader title="دسترسی من" subtitle="ارزیابی سرور برای حساب فعلی شما؛ فقط نمایش. نقش، سمت، مجوز، اعتبارنامه و واگذاری جدا نشان داده می‌شوند.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      {!ready?.authenticated ? <Card><StateBlock kind="locked" title="ابتدا وارد شوید" detail="ارزیابی دسترسی فقط برای کاربر واردشده انجام می‌شود." /></Card>
        : ev.state === 'LOADING' || ev.state === 'IDLE' ? <Card><StateBlock kind="loading" title="در حال دریافت ارزیابی دسترسی…" /></Card>
          : ev.state === 'FAILED' ? <Card><StateBlock kind="unknown" title="ارزیابی دسترسی دریافت نشد" detail="نامعلوم به معنی مجاز نیست؛ هیچ دسترسی‌ای فرض نمی‌شود." /></Card>
            : <Evaluated data={ev.data} />}
      <Card title="آنچه این صفحه انجام نمی‌دهد">
        <ul className="hr-list">
          <li>هیچ نقش، مجوز یا واگذاری‌ای اعطا یا لغو نمی‌شود (D-34، D-42).</li>
          <li>هیچ اعتبارنامه‌ای (رمز، بازنشانی، پیامک) صادر یا نمایش داده نمی‌شود (R5-SEC-01).</li>
          <li>دسترسی هیچ‌کس از سمت یا جایگاه او در سلسله‌مراتب استنتاج نمی‌شود.</li>
          <li>دسترسی مؤثر دیگر افراد از این API در دسترس نیست و حدس زده نمی‌شود.</li>
        </ul>
      </Card>
    </>
  )
}

function Evaluated({ data }: { data: EffectiveAccess }) {
  const s = useSession()
  const session = s.status === 'ready' ? s.session : undefined
  const [queue, setQueue] = useState<'all' | Verdict>('all')
  const allowed = data.decisions.filter(d => verdict(d) === 'allowed')
  const matched = [...new Set(allowed.flatMap(d => d.matched_roles))].sort()
  const roles = matched.filter(isRoleKey)
  const direct = matched.filter(m => !isRoleKey(m))
  const counts: Record<Verdict, number> = { allowed: 0, denied: 0, blocked: 0 }
  for (const d of data.decisions) counts[verdict(d)] += 1
  const rows = queue === 'all' ? data.decisions : data.decisions.filter(d => verdict(d) === queue)
  const credential = session?.auth_mode === 'session' ? 'نشست ورود با حساب کاربری' : session?.auth_mode === 'dev_stub' ? 'هویت آزمایشی توسعه (مصنوعی؛ فقط محیط توسعه/آزمون)' : 'بدون روش ورود'
  return (
    <>
      <div className="hr-concepts" aria-label="مفاهیم دسترسی">
        <section className="hr-concept" aria-label="اعتبارنامه">
          <h2>اعتبارنامه (ورود)</h2>
          <p>{credential}</p>
          {session?.session && <p className="muted">{session.session.display_name} · <span className="hr-ltr num">{session.session.user_id}</span></p>}
          {session?.principal && <p className="muted">{session.principal.display_name} · <span className="hr-ltr num">{session.principal.principal_id}</span></p>}
          <p className="muted">فقط هویت ورود را ثابت می‌کند؛ مجوز نیست.</p>
        </section>
        <section className="hr-concept" aria-label="واگذاری">
          <h2>واگذاری (Grant)</h2>
          {matched.length === 0 ? <p>واگذاری فعالی که مجوزی بدهد ندارید.</p> : <p>{matched.length} منشأ فعال: {matched.map(m => <code key={m} className="hr-ltr">{m} </code>)}</p>}
          <p className="muted">فقط واگذاری فعال (ACTIVE) سرور مجوز می‌دهد. جزئیات واگذاری (مبنا، تاریخ) در این API ارائه نمی‌شود.</p>
        </section>
        <section className="hr-concept" aria-label="نقش">
          <h2>نقش</h2>
          {roles.length === 0 ? <p>نقش شناخته‌شده‌ای از واگذاری‌ها تطبیق نیافت.</p>
            : <p>{roles.map(r => <StatusPill key={r} tone="blue" icon="check" label={ROLE_CONTRACTS[r].label} />)}</p>}
          {direct.length > 0 && <p className="muted">واگذاری مستقیم بدون نقش: {direct.join('، ')}</p>}
          {roles.length > 1 && <p className="muted">چند نقش هم‌زمان: سیاست انتخاب زمینه (D-42) باز است؛ هیچ‌کدام برگزیده نمی‌شود.</p>}
          <p className="muted">نقش بسته‌ای از مجوزهاست که از واگذاری می‌آید، نه از سمت.</p>
        </section>
        <OwnPosition userId={session?.auth_mode === 'session' ? session.session?.user_id : undefined} />
        <section className="hr-concept" aria-label="مجوزها">
          <h2>مجوز (Capability)</h2>
          <p><span className="num">{counts.allowed}</span> مجاز · <span className="num">{counts.denied}</span> مجاز نیست · <span className="num">{counts.blocked}</span> مسدود</p>
          <p className="muted">نتیجه ارزیابی همان ارزیاب allowed_actions است؛ مشاهده ≠ نوشتن.</p>
        </section>
      </div>
      <Card title="ارزیابی مجوزها">
        <QueueTabs label="فیلتر نتیجه" value={queue} onChange={v => setQueue(v as 'all' | Verdict)} queues={[
          { id: 'all', label: 'همه', count: data.decisions.length }, { id: 'allowed', label: 'مجاز', count: counts.allowed, tone: 'green' },
          { id: 'denied', label: 'مجاز نیست', count: counts.denied }, { id: 'blocked', label: 'مسدود', count: counts.blocked, tone: 'amber' },
        ]} />
        {rows.length === 0 ? <StateBlock kind="empty" title="موردی در این دسته نیست" /> : (
          <div className="hr-table-wrap">
            <table className="hr-table hr-stack">
              <caption className="sr">ارزیابی مجوزها</caption>
              <thead><tr><th scope="col">مجوز</th><th scope="col">ماژول</th><th scope="col">نتیجه</th><th scope="col">دلیل</th><th scope="col">دامنه</th><th scope="col">منشأ</th></tr></thead>
              <tbody>
                {rows.map(d => {
                  const v = verdictPill[verdict(d)]
                  return (
                    <tr key={d.capability}>
                      <td data-label="مجوز" className="hr-ltr">{d.capability}</td>
                      <td data-label="ماژول">{moduleLabel[d.capability.split('.')[0] ?? ''] ?? '—'}</td>
                      <td data-label="نتیجه"><StatusPill tone={v.tone} icon={v.tone === 'green' ? 'check' : 'lock'} label={v.label} /></td>
                      <td data-label="دلیل" className="hr-ltr">{d.reason_code ?? '—'}</td>
                      <td data-label="دامنه">{d.scope === 'ORGANIZATION' ? 'سازمان' : 'هیچ'}</td>
                      <td data-label="منشأ" className="hr-ltr">{d.matched_roles.length ? d.matched_roles.join(', ') : '—'}</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </>
  )
}

/** The caller's own HR position (organisation evidence). Shown beside the evaluation to make clear nothing is derived from it. */
function OwnPosition({ userId }: { userId: string | undefined }) {
  const api = useApi()
  const canResolve = useCanResolveUsers()
  const run = !!userId && canResolve
  const st = useRead<PersonSummary[] | null>(`own-position:${userId ?? ''}:${run}`, async () => run && userId
    ? settle(await api.GET('/api/v1/organization/persons', { params: { query: { user_id: userId, limit: 5 } } }), e => e.items)
    : { status: 'ok', data: null, outcome: 'SUCCESS', allowed: undefined })
  return (
    <section className="hr-concept" aria-label="سمت سازمانی">
      <h2>سمت سازمانی</h2>
      {!userId ? <p>این روش ورود به حساب سازمانی متصل نیست.</p>
        : !canResolve ? <p>خواندن پرونده همکاری برای شما مجاز نیست.</p>
          : <>
            <ReadStates state={st} what="پرونده" />
            {st.status === 'ok' && st.data && (st.data.length === 0 ? <p>پرونده نیروی متصل به حساب شما یافت نشد.</p>
              : st.data.length > 1 ? <p><StatusPill tone="orange" icon="alert" label={`تعارض: حساب شما به ${st.data.length} پرونده متصل است`} /></p>
                : st.data.map(p => (
                  <p key={p.person_id}><Link to={`/hr/people/${p.person_id}`}>{text(p.display_name)}</Link> · {text(p.employment?.position?.label)}
                    {p.employment && <span className="muted"> · {statusLabel[p.employment.status]}</span>}</p>
                )))}
          </>}
      <p className="muted">سمت یک واقعیت منابع انسانی است؛ هیچ مجوزی از آن مشتق نمی‌شود.</p>
    </section>
  )
}
