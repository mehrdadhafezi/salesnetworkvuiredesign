import { Link } from 'react-router'
import { useApi, type Employment, type UserSummary } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { formatInstant } from '../../../shared/ui/time'
import { linkLabel, text } from '../labels'
import { useCapabilities } from './access'
import { hrStatusIcon, hrStatusTone, requestStatusLabel, requestTypeLabel } from './requests'

const REQUESTS_BLOCKED = 'درخواست‌های منابع انسانی پشت پرچم organization.hr_request_write هستند.'

/**
 * Person ↔ account linkage, read only. The account is an identity record:
 * not the person, not a role, not a capability and not a credential. No
 * password, reset, login or provisioning control exists here (D-34, R5-SEC-01).
 */
export function AccountCard({ employment, canUser }: { employment: Employment; canUser: boolean }) {
  const api = useApi()
  const userId = employment.user.user_id ?? null
  const link = employment.user.link_state
  const user = useRead<UserSummary | null>(`user:${userId ?? ''}:${canUser}`, async () => userId && canUser
    ? settle(await api.GET('/api/v1/identity/users/{user_id}', { params: { path: { user_id: userId } } }), e => e.data)
    : { status: 'ok', data: null, outcome: 'SUCCESS', allowed: undefined })
  return (
    <Card title="حساب کاربری (هویت ورود)">
      <p className="muted">حساب کاربری با شخص و با همکاری یکی نیست؛ نقش، مجوز یا اعتبارنامه هم نیست. هیچ رمز، بازنشانی یا ساخت حسابی از اینجا انجام نمی‌شود.</p>
      <dl className="kv">
        <dt>وضعیت پیوند</dt>
        <dd>
          <StatusPill tone={link === 'LINKED' ? 'green' : 'orange'} icon={link === 'LINKED' ? 'check' : 'alert'} label={linkLabel[link]} />
          {link === 'SHARED_USER' && userId && <> <Link to={`/hr?user_id=${encodeURIComponent(userId)}`}>همه پرونده‌های این حساب</Link></>}
        </dd>
        <dt>شناسه حساب</dt><dd className="hr-ltr num">{userId ?? 'ثبت نشده'}</dd>
      </dl>
      {userId && !canUser && <StateBlock kind="locked" title="جزئیات حساب برای شما قابل نمایش نیست" detail="نیازمند identity.user.read" />}
      {userId && canUser && <ReadStates state={user} what="حساب" blockedDetail="خواندن هویت پشت پرچم ویژگی منابع انسانی است." />}
      {user.status === 'ok' && user.data && (
        <dl className="kv">
          <dt>نام حساب</dt><dd>{text(user.data.display_name)}</dd>
          <dt>وضعیت حساب</dt><dd>حساب سایه واردشده از سامانه قدیمی</dd>
          <dt>ورود</dt><dd><StatusPill tone="slate" icon="lock" label={user.data.login_enabled ? 'فعال' : 'ورود فعال نیست'} /></dd>
          <dt>منبع</dt><dd className="num">{`${user.data.provenance.source_spec} #${user.data.provenance.source_key}`}</dd>
        </dl>
      )}
    </Card>
  )
}

/**
 * HR requests about this person's account, within the caller's own row scope
 * (requester, current reviewer or history actor). Never a claim about every
 * request that exists for the person. Not requested at all when the caller's
 * evaluation does not allow org.hr_request.read.
 */
export function PersonRequestsCard({ userId, linkState }: { userId: string | null; linkState: Employment['user']['link_state'] }) {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const feature = s.status === 'ready' && s.session.features?.hr_request_write === true
  const cap = useCapabilities(['org.hr_request.read'] as const)['org.hr_request.read']
  const run = feature && cap === 'allowed' && userId !== null
  const list = useRead(`person-req:${userId ?? ''}:${run}`, async () => run && userId
    ? settle(await api.GET('/api/v1/hr/requests', { params: { query: { subject_user_id: userId, limit: 50 } } }),
      e => e.items ? { items: e.items, more: e.page?.has_more === true } : undefined)
    : { status: 'ok', data: { items: [], more: false }, outcome: 'SUCCESS', allowed: undefined })
  return (
    <Card title="درخواست‌های منابع انسانی این نیرو">
      <p className="muted">فقط درخواست‌هایی که شما درخواست‌دهنده، بازبین فعلی یا اقدام‌کننده آن بوده‌اید؛ این فهرست همه درخواست‌های این نیرو نیست.</p>
      {userId === null ? <StateBlock kind="unknown" title="حساب کاربری متصل ثبت نشده است" detail="درخواست‌ها با حساب کاربری ثبت می‌شوند؛ بدون حساب، ارتباطی قابل نمایش نیست." />
        : !feature ? <StateBlock kind="locked" title="این بخش هنوز فعال نشده است" detail={REQUESTS_BLOCKED} />
          : cap !== 'allowed' ? <StateBlock kind="locked" title="دسترسی به درخواست‌ها ندارید" detail={cap === 'unknown' ? 'ارزیابی دسترسی در دسترس نیست.' : 'نیازمند org.hr_request.read'} />
            : <>
              <ReadStates state={list} what="درخواست‌ها" blockedDetail={REQUESTS_BLOCKED} />
              {list.status === 'ok' && (list.data.items.length === 0
                ? <StateBlock kind="empty" title="درخواستی در محدوده شما برای این نیرو نیست" />
                : (
                  <ul className="hr-list" aria-label="درخواست‌های این نیرو">
                    {list.data.items.map(q => (
                      <li key={q.request_id}>
                        <Link to={`/hr/requests/${q.request_id}`}>{requestTypeLabel(q.request_type)}</Link>{' '}
                        <StatusPill tone={hrStatusTone(q.status)} icon={hrStatusIcon(q.status)} label={requestStatusLabel(q.status)} />
                        <span className="muted"> · {formatInstant(q.created_at, zone)}</span>
                      </li>
                    ))}
                  </ul>
                ))}
              {list.status === 'ok' && list.data.more && <p className="muted">موارد بیشتری وجود دارد؛ فهرست کامل در صفحه درخواست‌ها.</p>}
            </>}
      {linkState === 'SHARED_USER' && <p className="muted">حساب این نیرو مشترک است؛ درخواست‌های حساب ممکن است به پرونده دیگری مربوط باشد.</p>}
    </Card>
  )
}
