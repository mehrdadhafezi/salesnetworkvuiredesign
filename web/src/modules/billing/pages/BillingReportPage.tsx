import '../billing.css'
import { useApi } from '../../../shared/api'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { Card, PageHeader, StatusPill } from '../../../shared/ui/primitives'
import { ReadOnlyNote } from '../components'
import { blockedDetail } from '../labels'

const counts = (m: Record<string, number>) => Object.entries(m).length === 0 ? '—' : Object.entries(m).map(([k, v]) => `${k}: ${v}`).join(' · ')

/**
 * Billing + payments reconciliation report (read only). Counts are
 * evidence; F02 (Open Pre-invoices) is NOT_EVALUATED with a null count —
 * never zero. Raw 'approved' and 'reconcile_error' counts stay visible.
 */
export function BillingReportPage() {
  const api = useApi()
  const rep = useRead('billing-report', async () => settle(await api.GET('/api/v1/billing/reconciliation-report'), e => e.data))
  return (
    <>
      <PageHeader title="گزارش تطبیق صورت‌حساب و پرداخت" subtitle="شواهد پوشش؛ بدون وضعیت یا جمع قطعی.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadOnlyNote />
      <ReadStates state={rep} what="گزارش" blockedDetail={blockedDetail} />
      {rep.status === 'ok' && (
        <div className="bi-sections">
          <Card title={`${rep.data.f02.label} (F02)`}>
            <StatusPill tone="slate" icon="question" label="ارزیابی نشده — مقدار تهی است، صفر نیست" />
            <p className="muted">{rep.data.f02.reason}</p>
          </Card>
          <Card title="صورت‌حساب">
            <dl className="bi-kv">
              <dt>فاکتورهای حاضر</dt><dd className="num">{rep.data.billing.invoices_present}</dd>
              <dt>اقلام / مراحل</dt><dd className="num">{`${rep.data.billing.items_present} / ${rep.data.billing.stages_present}`}</dd>
              <dt>وضعیت‌های خام ناشناخته</dt><dd className="bi-ltr">{Object.entries(rep.data.billing.unknown_raw_states).map(([f, m]) => `${f}: ${counts(m)}`).join(' | ') || '—'}</dd>
              <dt>مراحل ناشناخته (خام)</dt><dd className="bi-ltr">{counts(rep.data.billing.unknown_stage_raw_states)}</dd>
              <dt>مبالغ غیرقابل نمایش دقیق</dt><dd className="num">{rep.data.billing.unrepresentable_amounts}</dd>
              <dt>مقدار خام pre_invoice (شاهد، نه F02)</dt><dd className="num">{rep.data.billing.invoices_with_raw_pre_invoice_value}</dd>
            </dl>
          </Card>
          <Card title="پرداخت">
            <dl className="bi-kv">
              <dt>تلاش‌های حاضر</dt><dd className="num">{rep.data.payments.attempts_present}</dd>
              <dt>«approved» خام (بررسی مالی نیست)</dt><dd className="num">{rep.data.payments.approved_raw_count}</dd>
              <dt>«reconcile_error» خام</dt><dd className="num">{rep.data.payments.reconcile_error_raw_count}</dd>
              <dt>منشأ تأیید</dt><dd className="bi-ltr">{counts(rep.data.payments.approval_provenance)}</dd>
              <dt>وضعیت‌های خام ناشناخته</dt><dd className="bi-ltr">{counts(rep.data.payments.unknown_raw_attempt_states)}</dd>
              <dt>تلاش بدون فاکتور</dt><dd className="num">{rep.data.payments.attempts_without_invoice}</dd>
              <dt>بازگشت وضعیت پذیرفته‌شده</dt><dd className="num">{rep.data.payments.attempt_state_regressions}</dd>
              <dt>رسید: فقط ارجاع / نامعلوم</dt><dd className="num">{`${rep.data.payments.receipts_referenced} / ${rep.data.payments.receipts_unknown}`}</dd>
            </dl>
          </Card>
          <Card title="یادداشت‌ها">
            <ul className="bi-list">{rep.data.notes.map(n => <li key={n}>{n}</li>)}</ul>
          </Card>
        </div>
      )}
    </>
  )
}
