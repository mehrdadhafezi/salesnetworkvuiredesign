import '../workspace/workspace.css'
import { Link } from 'react-router'
import { useSession } from '../../../shared/auth/currentSession'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { blockedDetail as billingBlocked } from '../../billing/labels'
import { nextActionLabel, receiptStateLabel, receiptStateTone, reconciliationReasonLabel, uploadStateLabel } from '../../payments/labels'
import { receiptStateLabel as legacyReceiptLabel } from '../../billing/labels'
import { Count } from '../workspace/components'
import { C, useFinanceReports, type CountSource } from '../workspace/reports'
import { FinanceSectionNav } from '../workspace/sections'
import { Counters } from './FinanceHomePage'

const evidenceBlocked = 'شواهد پرداخت سامانه جدید پشت پرچم payments.manual_write است (FEATURE_DISABLED).'

const LEGACY: { label: string; source: CountSource }[] = [
  { label: legacyReceiptLabel.REFERENCE_ONLY ?? 'REFERENCE_ONLY', source: C.receiptsReferenced },
  { label: legacyReceiptLabel.UNKNOWN ?? 'UNKNOWN', source: C.receiptsUnknown },
  { label: legacyReceiptLabel.MISSING ?? 'MISSING', source: C.receiptsMissing },
]
const TARGET_STATES = ['QUARANTINED', 'FINALIZING', 'READY', 'STORAGE_ERROR', 'NOT_RECOVERABLE'] as const

/**
 * RM-P07 receipt evidence (read only). Legacy receipt references (metadata
 * of a legacy file link, never inspected) and target-system receipts
 * (quarantined uploads finalised with a hash check) are different evidence
 * and are shown apart, as counts. A READY receipt proves only that a file
 * was stored intact: it is not a confirmed receipt of funds, a Finance
 * decision or a settlement. No file is served and no fix action exists.
 */
export function FinanceReceiptsPage() {
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const reports = useFinanceReports()
  const ev = reports.evidence
  return (
    <>
      <PageHeader title="شواهد رسید" subtitle="ارجاع رسید قدیمی و فیش سامانه جدید، جدا از هم؛ فیش آماده به معنای وصول وجه نیست.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <FinanceSectionNav />
      <div className="fw-sections">
        <div className="fw-two">
          <Card title="ارجاع‌های رسید قدیمی (شاهد قدیمی)">
            {reports.billing.status !== 'ok' && <ReadStates state={reports.billing} what="گزارش تطبیق صورت‌حساب" blockedDetail={billingBlocked} />}
            <Counters label="شمارنده‌های رسید قدیمی" counters={LEGACY} reports={reports} />
            <p className="muted fw-hint">فقط ارجاع به فایل قدیمی ثبت شده و فایل بررسی نشده است. فهرست سراسری ارجاع رسید در قرارداد نیست؛ ارجاع‌های هر فاکتور در بخش «رسیدها»ی جزئیات همان فاکتور دیده می‌شوند.</p>
          </Card>
          <Card title="فیش‌های سامانه جدید">
            <ReadStates state={ev} what="شواهد پرداخت" blockedDetail={evidenceBlocked} />
            {ev.status === 'ok' && (
              <ul className="fw-counters" aria-label="شمارنده‌های فیش سامانه جدید">
                {TARGET_STATES.map(k => (
                  <li key={k}>
                    <StatusPill tone={receiptStateTone[k] ?? 'slate'} icon="inbox" label={receiptStateLabel[k] ?? k} />
                    <Count value={ev.data.receipts_by_state[k] ?? 0} />
                    <span className="fw-src fw-ltr">{`GET /api/v1/payments/evidence · receipts_by_state.${k} · تعداد فیش، نه پرداخت`}</span>
                  </li>
                ))}
                {Object.entries(ev.data.uploads_by_state).map(([k, n]) => (
                  <li key={`u-${k}`}>
                    <span>{uploadStateLabel[k] ?? k}</span>
                    <Count value={n} />
                    <span className="fw-src fw-ltr">{`GET /api/v1/payments/evidence · uploads_by_state.${k}`}</span>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
        <Card title="پرداخت‌های ثبت‌شده بر اساس آمادگی فیش">
          <p className="muted">صف بررسی با فیلتر سرور evidence_state؛ هر صف جداگانه مجوز finance.review.read را بررسی می‌کند.</p>
          <ul className="fw-list">
            <li><Link to="/finance/review?evidence_state=PENDING_FINALISATION">فیش در انتظار نهایی‌سازی (تصمیم مسدود، D-43)</Link></li>
            <li><Link to="/finance/review?evidence_state=READY">همه فیش‌ها آماده (هنوز بررسی نشده)</Link></li>
            <li><Link to="/finance/review?evidence_state=NOT_RECOVERABLE">شواهد غیرقابل بازیابی</Link></li>
          </ul>
        </Card>
        <Card title="تطبیق شواهد فیش (ثبت‌شده توسط سرور)">
          <ReadStates state={ev} what="تطبیق شواهد" blockedDetail={evidenceBlocked} />
          {ev.status === 'ok' && (ev.data.reconciliations.length === 0 ? <StateBlock kind="empty" title="مورد تطبیقی ثبت نشده است" /> : (
            <div className="fw-table-wrap">
              <table className="fw-table fw-stack">
                <caption className="sr">موارد تطبیق شواهد فیش</caption>
                <thead><tr><th scope="col">علت</th><th scope="col">وضعیت فیش</th><th scope="col">اقدام بعدی امن</th><th scope="col">زمان</th><th scope="col">پرداخت ثبت‌شده</th></tr></thead>
                <tbody>
                  {ev.data.reconciliations.map(r => (
                    <tr key={r.reconciliation_id}>
                      <td data-label="علت">{r.kind === 'RESOLVED' ? 'حل‌شده: ' : ''}{reconciliationReasonLabel[r.reason] ?? r.reason}</td>
                      <td data-label="وضعیت فیش"><StatusPill tone={receiptStateTone[r.receipt_state] ?? 'slate'} icon="inbox" label={receiptStateLabel[r.receipt_state] ?? r.receipt_state} /></td>
                      <td data-label="اقدام بعدی امن">{nextActionLabel[r.next_safe_action] ?? r.next_safe_action}</td>
                      <td data-label="زمان">{formatInstant(r.opened_at, zone)}</td>
                      <td data-label="پرداخت ثبت‌شده">{r.attempt_id ? <Link to={`/finance/review/${r.attempt_id}`}>مورد بررسی</Link> : <span className="muted">—</span>}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
          <p className="muted fw-hint">اقدام بعدی امن را سرور تعیین کرده است؛ این صفحه هیچ تلاش دوباره، اصلاح یا حل دستی ندارد.</p>
        </Card>
      </div>
    </>
  )
}
