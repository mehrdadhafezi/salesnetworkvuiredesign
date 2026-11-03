import '../../writes/writes.css'
import '../payments.css'
import { useParams } from 'react-router'
import { useApi } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { money } from '../labels'
import { AttemptList } from '../PaymentSubmissionPanel'

/** One committed payment batch of the caller with its attempts and receipt states. */
export function PaymentBatchPage() {
  const { batchId = '' } = useParams()
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const v = useRead(`pb:${batchId}`, async () => settle(await api.GET('/api/v1/payments/batches/{batch_id}', { params: { path: { batch_id: batchId } } }),
    e => e.data))
  return (
    <>
      <PageHeader title="دسته پرداخت" subtitle="ثبت در سامانه جدید؛ بررسی مالی انجام نشده و سامانه قدیمی مرجع است.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadStates state={v} what="دسته پرداخت" blockedDetail="ثبت پرداخت پشت پرچم payments.manual_write است." />
      {v.status === 'ok' && (
        <Card title="دسته">
          <dl className="wr-dl">
            <dt>جمع</dt><dd>{money(v.data.total_amount, v.data.unit)} · {v.data.item_count} تراکنش</dd>
            <dt>مرحله</dt><dd>{v.data.stage_no}</dd>
            <dt>نسخه حساب وصول</dt><dd className="wr-ltr">{v.data.collection_version_before} → {v.data.collection_version_after}</dd>
            <dt>زمان</dt><dd>{formatInstant(v.data.created_at, zone)}</dd>
          </dl>
          <AttemptList attempts={v.data.attempts} zone={zone} />
        </Card>
      )}
    </>
  )
}
