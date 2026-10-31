import '../hr.css'
import { Link, useParams } from 'react-router'
import { useApi, type Assignment, type Employment, type Finding, type PersonDetail } from '../../../shared/api'
import { actionHint } from '../../../shared/permissions/allowedActions'
import { useSession } from '../../../shared/auth/currentSession'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { formatInstant } from '../../../shared/ui/time'
import { ReadStates } from '../components/States'
import { findingLabel, kindLabel, linkLabel, refStateLabel, severityLabel, severityTone, statusLabel, text } from '../labels'
import { settle, useRead } from '../load'

/**
 * Person detail, read-only. Person, Employment, Position/Level, Org unit and
 * Assignment are separate sections (User ≠ Person ≠ Employment;
 * Position ≠ Role). Sections the server does not allow show a locked state.
 */
export function PersonPage() {
  const { personId = '' } = useParams()
  const api = useApi()
  const state = useRead(personId, async () => settle(await api.GET('/api/v1/organization/persons/{person_id}', {
    params: { path: { person_id: personId } },
  }), e => e.data))
  const name = state.status === 'ok' ? text(state.data.person.display_name) : 'پرونده نیرو'
  return (
    <>
      <PageHeader title={name} subtitle="نمای فقط‌خواندنی؛ ساختار فعلی سابقه تاریخی را بازنویسی نمی‌کند.">
        <Link to="/hr">بازگشت به نیروها</Link>
      </PageHeader>
      <ReadStates state={state} what="پرونده" />
      {state.status === 'ok' && <Detail d={state.data}
        canEmployment={actionHint(state.allowed, 'org.employment.read').allowed}
        canPosition={actionHint(state.allowed, 'org.position.read').allowed}
        canUnit={actionHint(state.allowed, 'org.unit.read').allowed}
        canAssignment={actionHint(state.allowed, 'org.assignment.read').allowed} />}
    </>
  )
}

function Locked({ title }: { title: string }) {
  return <Card title={title}><StateBlock kind="locked" title="دسترسی به این بخش ندارید" /></Card>
}

function Detail({ d, canEmployment, canPosition, canUnit, canAssignment }:
  { d: PersonDetail; canEmployment: boolean; canPosition: boolean; canUnit: boolean; canAssignment: boolean }) {
  const p = d.person
  const e = p.employment
  const session = useSession()
  const zone = session.status === 'ready' ? session.session.display_zone : 'Asia/Tehran'
  return (
    <div className="hr-sections">
      <Card title="شخص">
        <dl className="kv">
          <dt>نام</dt><dd>{text(p.display_name)}</dd>
          <dt>منبع</dt><dd className="num">{`${p.provenance.source_spec} #${p.provenance.source_key}`}</dd>
          <dt>آخرین ورود سایه</dt><dd>{formatInstant(p.provenance.refreshed_at, zone)}</dd>
          <dt>وضعیت در آخرین ورود</dt><dd>{p.provenance.presence === 'PRESENT' ? 'موجود' : 'در آخرین ورود کامل نبود (حذف نشده)'}</dd>
        </dl>
      </Card>
      {canEmployment && e ? <EmploymentCard e={e} /> : <Locked title="همکاری" />}
      {canPosition && e ? (
        <Card title="سمت و سطح">
          <p className="muted">سمت یک واقعیت منابع انسانی است و مجوز یا نقش نیست.</p>
          <dl className="kv">
            <dt>سمت</dt><dd>{text(e.position?.label)} {e.position_state && e.position_state !== 'RESOLVED' && <span className="muted">({refStateLabel[e.position_state]})</span>}</dd>
            <dt>سطح</dt><dd>{text(e.level?.label)} {e.level_state && e.level_state !== 'RESOLVED' && <span className="muted">({refStateLabel[e.level_state]})</span>}</dd>
          </dl>
        </Card>
      ) : <Locked title="سمت و سطح" />}
      {canUnit && e ? (
        <Card title="واحد سازمانی">
          <p className="muted">ساختار واحد با خط گزارش‌دهی (مدیر) یکی نیست.</p>
          <dl className="kv">
            {(['department', 'unit', 'team'] as const).map(k => {
              const ref = e[k]
              const st = e[`${k}_state`]
              return [<dt key={`${k}-t`}>{kindLabel[k === 'department' ? 'DEPARTMENT' : k === 'unit' ? 'UNIT' : 'TEAM']}</dt>,
                <dd key={`${k}-d`}>{text(ref?.label)} {st && st !== 'RESOLVED' && <span className="muted">({refStateLabel[st]})</span>}</dd>]
            })}
          </dl>
        </Card>
      ) : <Locked title="واحد سازمانی" />}
      {canAssignment ? <AssignmentCards d={d} /> : <Locked title="گزارش‌دهی و سابقه" />}
    </div>
  )
}

function EmploymentCard({ e }: { e: Employment }) {
  return (
    <Card title="همکاری">
      <dl className="kv">
        <dt>وضعیت همکاری</dt>
        <dd>{statusLabel[e.status]}{e.status_raw && e.status === 'UNMAPPED' && <span className="muted"> — مقدار قدیمی: {e.status_raw}</span>}</dd>
        <dt>نوع (مقدار قدیمی)</dt><dd>{text(e.type_raw)}</dd>
        <dt>کد کارمندی</dt><dd className="num">{text(e.employee_code)}</dd>
        <dt>تاریخ شروع</dt><dd>{e.hire_date ?? (e.hire_date_raw ? `نامعلوم (مقدار قدیمی: ${e.hire_date_raw})` : 'ثبت نشده')}</dd>
        <dt>تاریخ پایان</dt><dd>{e.termination_date ?? (e.termination_date_raw ? `نامعلوم (مقدار قدیمی: ${e.termination_date_raw})` : 'ثبت نشده')}</dd>
        <dt>فعال (پرچم قدیمی)</dt><dd>{e.is_active === null ? 'نامعلوم' : e.is_active ? 'بله' : 'خیر'}</dd>
        <dt>حساب کاربری</dt><dd>{linkLabel[e.user.link_state]}</dd>
      </dl>
      <p className="muted">«فعال» به معنی ورود قابل‌استفاده نیست؛ در این مرحله هیچ ورودی فعال نیست.</p>
    </Card>
  )
}

function interval(a: Assignment): string {
  return `${a.effective_from_raw ?? 'شروع نامعلوم'} ← ${a.effective_to_raw ?? (a.is_current ? 'جاری' : 'پایان نامعلوم')}`
}

function AssignmentCards({ d }: { d: PersonDetail }) {
  const p = d.person
  return (
    <>
      <Card title="مدیر جاری">
        <p className="muted">مدیر جاری اعتبار تاریخی نیست و مالکیت یا اعتبار فروش گذشته را تغییر نمی‌دهد.</p>
        {p.manager_state === 'RESOLVED' && p.manager
          ? <Link to={`/hr/people/${p.manager.person_id}`}>{text(p.manager.display_name)}</Link>
          : p.manager_state === 'MULTIPLE' ? <StatusPill tone="orange" icon="alert" label="تعارض: چند مدیر جاری" />
            : p.manager_state === 'MISSING' ? <StatusPill tone="orange" icon="alert" label="مدیر ثبت‌شده وجود ندارد" />
              : <span className="muted">بدون مدیر جاری</span>}
        <h3 className="hr-sub">زیرمجموعه مستقیم جاری</h3>
        {d.direct_reports.length === 0 ? <p className="muted">ندارد</p> : (
          <ul className="hr-list">{d.direct_reports.map(r => <li key={r.person_id}><Link to={`/hr/people/${r.person_id}`}>{text(r.display_name)}</Link></li>)}</ul>
        )}
      </Card>
      <Card title="بازه‌های گزارش‌دهی">
        <p className="muted">تاریخ‌ها مقدار خام قدیمی‌اند؛ منطقه زمانی منبع نامعلوم است. بازه گم‌شده پر نمی‌شود.</p>
        {d.assignments.length === 0 ? <StateBlock kind="empty" title="انتسابی ثبت نشده" /> : (
          <ol className="hr-timeline">
            {d.assignments.map(a => (
              <li key={a.assignment_id}>
                <b>{a.parent.person ? text(a.parent.person.display_name) : a.parent.state === 'MISSING' ? 'مدیر ناموجود' : 'بدون مدیر'}</b>
                <span className="num"> {interval(a)}</span>
                <span className="muted"> · منطقه زمانی: نامعلوم</span>
                {a.relationship_kind === 'UNMAPPED' && <span className="muted"> · نوع رابطه: {text(a.relationship_raw)}</span>}
                {a.finding_kinds.map(k => <StatusPill key={k} tone="orange" icon="alert" label={findingLabel[k]} />)}
              </li>
            ))}
          </ol>
        )}
      </Card>
      <Card title="سابقه تغییر مدیر (فقط افزودنی)">
        {d.history.length === 0 ? <StateBlock kind="unknown" title="سابقه‌ای ثبت نشده" detail="نبود سابقه یعنی نامعلوم، نه بدون تغییر." /> : (
          <ol className="hr-timeline">
            {d.history.map(h => (
              <li key={h.history_id}>
                <span className="num">{text(h.recorded_raw)}</span> · {text(h.old_parent?.display_name)} ← {text(h.new_parent?.display_name)}
                <span className="muted"> · اقدام: {text(h.action_raw)}</span>
              </li>
            ))}
          </ol>
        )}
      </Card>
      <Card title="یافته‌های سلسله‌مراتب">
        {d.findings.length === 0 ? <p className="muted">یافته‌ای نیست</p> : <FindingList findings={d.findings} />}
      </Card>
    </>
  )
}

export function FindingList({ findings }: { findings: Finding[] }) {
  return (
    <ul className="hr-list">
      {findings.map(f => (
        <li key={f.finding_id}>
          <StatusPill tone={severityTone[f.severity]} icon={f.severity === 'INFO' ? 'question' : 'alert'} label={`${severityLabel[f.severity]}: ${findingLabel[f.kind]}`} />
          <span className="muted"> {f.detail}</span>
        </li>
      ))}
    </ul>
  )
}
