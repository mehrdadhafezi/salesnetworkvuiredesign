import '../../writes/writes.css'
import '../payments.css'
import { Link, useParams } from 'react-router'
import { useApi } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { attemptStateLabel, money, receiptStateLabel, receiptStateTone } from '../labels'
import { Gates, ReviewFlags } from './FinanceReviewPage'

/** One attempt awaiting review with its receipts and decision gate (read only). */
export function FinanceReviewAttemptPage() {
  const { attemptId = '' } = useParams()
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const v = useRead(`fra:${attemptId}`, async () => settle(await api.GET('/api/v1/finance/review/attempts/{attempt_id}', {
    params: { path: { attempt_id: attemptId } } }), e => e.data))
  return (
    <>
      <PageHeader title="بررسی پرداخت" subtitle="فقط خواندنی؛ بدون تایید یا رد.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <p><Link to="/finance/review">بازگشت به صف بررسی</Link></p>
      <ReadStates state={v} what="پرداخت" blockedDetail="صف بررسی مالی پشت پرچم finance.review_read است." />
      {v.status === 'ok' && (
        <div className="wr-sections">
          <Card title="پرداخت">
            <dl className="wr-dl">
              <dt>مبلغ</dt><dd>{money(v.data.amount, v.data.unit)}</dd>
              <dt>وضعیت</dt><dd>{attemptStateLabel[v.data.state] ?? v.data.state}</dd>
              <dt>مرحله</dt><dd>{v.data.stage_no}</dd>
              <dt>ثبت</dt><dd>{formatInstant(v.data.submitted_at, zone)}</dd>
              <dt>شواهد</dt><dd><ReviewFlags a={v.data} /></dd>
            </dl>
          </Card>
          <Card title="فیش‌ها">
            <ul className="wr-list" aria-label="فیش‌ها">
              {v.data.receipts.map(r => (
                <li key={r.receipt_id}><StatusPill tone={receiptStateTone[r.state] ?? 'slate'} icon="inbox" label={receiptStateLabel[r.state] ?? r.state} />{' '}
                  <span className="muted">{r.media_type} · {formatInstant(r.updated_at, zone)}</span></li>
              ))}
            </ul>
            <p className="muted">فایل فیش از این صفحه ارائه نمی‌شود.</p>
          </Card>
          <Card title="تصمیم مالی"><Gates a={v.data} /></Card>
        </div>
      )}
    </>
  )
}
