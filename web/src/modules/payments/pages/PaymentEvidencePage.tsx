import '../../writes/writes.css'
import '../payments.css'
import { useApi } from '../../../shared/api'
import { useSession } from '../../../shared/auth/currentSession'
import { settle, useRead } from '../../../shared/read/load'
import { ReadStates } from '../../../shared/read/ReadStates'
import { formatInstant } from '../../../shared/ui/time'
import { Card, PageHeader, StateBlock, StatusPill } from '../../../shared/ui/primitives'
import { nextActionLabel, receiptStateLabel, receiptStateTone, reconciliationReasonLabel, uploadStateLabel } from '../labels'

/**
 * Payment evidence diagnostics (read only): receipt and upload states and
 * the evidence reconciliation records. No file is served; no fix action.
 */
export function PaymentEvidencePage() {
  const api = useApi()
  const s = useSession()
  const zone = s.status === 'ready' ? s.session.display_zone : 'Asia/Tehran'
  const v = useRead('pev', async () => settle(await api.GET('/api/v1/payments/evidence'), e => e.data))
  return (
    <>
      <PageHeader title="شواهد پرداخت" subtitle="وضعیت فنی فیش‌ها و تطبیق؛ بدون اقدام اصلاحی.">
        <StatusPill tone="slate" icon="lock" label="فقط خواندنی" />
      </PageHeader>
      <ReadStates state={v} what="شواهد" blockedDetail="شواهد پرداخت پشت پرچم payments.manual_write است." />
      {v.status === 'ok' && (
        <div className="wr-sections">
          <Card title="وضعیت فیش‌ها">
            <ul className="wr-list" aria-label="شمار وضعیت فیش‌ها">
              {Object.entries(v.data.receipts_by_state).map(([k, n]) => (
                <li key={k}><StatusPill tone={receiptStateTone[k] ?? 'slate'} icon="inbox" label={receiptStateLabel[k] ?? k} /> <strong data-testid={`rcpt-${k}`}>{n}</strong></li>
              ))}
              {Object.entries(v.data.uploads_by_state).map(([k, n]) => <li key={`u-${k}`}>{uploadStateLabel[k] ?? k}: <strong>{n}</strong></li>)}
            </ul>
          </Card>
          <Card title="تطبیق شواهد">
            {v.data.reconciliations.length === 0 ? <StateBlock kind="empty" title="مورد تطبیقی نیست" /> : (
              <div className="wr-table-wrap">
                <table className="wr-table">
                  <caption className="sr">موارد تطبیق شواهد</caption>
                  <thead><tr><th scope="col">علت</th><th scope="col">وضعیت فیش</th><th scope="col">اقدام بعدی امن</th><th scope="col">زمان</th></tr></thead>
                  <tbody>
                    {v.data.reconciliations.map(r => (
                      <tr key={r.reconciliation_id}>
                        <td>{r.kind === 'RESOLVED' ? 'حل‌شده: ' : ''}{reconciliationReasonLabel[r.reason] ?? r.reason}</td>
                        <td><StatusPill tone={receiptStateTone[r.receipt_state] ?? 'slate'} icon="inbox" label={receiptStateLabel[r.receipt_state] ?? r.receipt_state} /></td>
                        <td>{nextActionLabel[r.next_safe_action] ?? r.next_safe_action}</td>
                        <td>{formatInstant(r.opened_at, zone)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </Card>
        </div>
      )}
    </>
  )
}
