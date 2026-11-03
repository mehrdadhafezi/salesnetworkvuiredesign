import '../../writes/writes.css'
import '../payments.css'
import { Link } from 'react-router'
import { useApi, type ReviewAttempt } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { Pager, useCursors } from '../../billing/pager'
import { codes, evidenceStateLabel, evidenceStateTone, gateActionLabel, money } from '../labels'

const blockedDetail = 'صف بررسی مالی پشت پرچم finance.review_read است.'

/** The decision gate of one attempt: why no Finance decision is available. */
export function Gates({ a }: { a: ReviewAttempt }) {
  return (
    <ul className="wr-list" aria-label="دلایل در دسترس نبودن تصمیم مالی">
      {a.gates.map(g => (
        <li key={g.action}>
          <StatusPill tone={g.outcome === 'UNAUTHORIZED' ? 'red' : 'slate'} icon="lock" label={`${gateActionLabel[g.action] ?? g.action}: در دسترس نیست`} />{' '}
          <span className="muted">{codes(g.blocked_by)}</span>
        </li>
      ))}
    </ul>
  )
}

export function ReviewFlags({ a }: { a: ReviewAttempt }) {
  return (
    <span className="pay-row">
      {a.evidence_ready
        ? <StatusPill tone="green" icon="check" label="همه فیش‌ها آماده (D-43 برقرار)" />
        : <StatusPill tone="amber" icon="alert" label="فیش آماده نیست؛ تصمیم مالی مسدود (D-43)" />}
      {a.reviewer_is_submitter && <StatusPill tone="red" icon="ban" label="شما ثبت‌کننده این پرداخت هستید؛ بررسی‌کننده مستقل نیستید" />}
      <StatusPill tone={evidenceStateTone[a.evidence_state] ?? 'slate'} icon="inbox" label={evidenceStateLabel[a.evidence_state] ?? a.evidence_state} />
    </span>
  )
}

/**
 * The Finance review queue (read only). Each attempt shows its evidence
 * readiness (D-43) and the maker / checker gate. There is no approve or
 * reject control: the decision policy is not approved (D-41, OPD-08, OPD-09).
 */
export function FinanceReviewPage() {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const c = useCursors()
  const q = useRead(`frq:${c.cursor ?? ''}`, async () => settle(await api.GET('/api/v1/finance/review/attempts', {
    params: { query: { limit: 50, ...(c.cursor ? { cursor: c.cursor } : {}) } } }), e => (e.data && e.page ? { items: e.data, page: e.page } : undefined)))
  return (
    <>
      <PageHeader title="صف بررسی مالی" subtitle="فقط خواندنی؛ تایید و رد مالی در این نسخه وجود ندارد.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <StateBlock kind="locked" title="تصمیم مالی در دسترس نیست"
        detail="مبنای ثبت در دفتر (D-41)، سیاست تایید و تفکیک وظایف (OPD-08) و مسیر استحقاق (OPD-09) هنوز تصویب نشده‌اند." />
      <Card title="پرداخت‌های در انتظار بررسی">
        <ReadStates state={q} what="صف بررسی" blockedDetail={blockedDetail} />
        {q.status === 'ok' && (q.data.items.length === 0 ? <StateBlock kind="empty" title="پرداختی در انتظار بررسی نیست" /> : (
          <div className="wr-table-wrap">
            <table className="wr-table">
              <caption className="sr">پرداخت‌های در انتظار بررسی مالی</caption>
              <thead><tr><th scope="col">مبلغ</th><th scope="col">ثبت</th><th scope="col">شواهد و تفکیک وظایف</th><th scope="col">تصمیم</th></tr></thead>
              <tbody>
                {q.data.items.map(a => (
                  <tr key={a.attempt_id}>
                    <td><Link to={`/finance/review/${a.attempt_id}`}>{money(a.amount, a.unit)}</Link></td>
                    <td>{formatInstant(a.submitted_at, zone)}</td>
                    <td><ReviewFlags a={a} /></td>
                    <td><Gates a={a} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}
        {q.status === 'ok' && <Pager page={q.data.page} c={c} />}
      </Card>
    </>
  )
}
